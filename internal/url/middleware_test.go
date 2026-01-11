package url

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func resetLimiters(t *testing.T) {
	t.Helper()

	clientLimitersMu.Lock()
	defer clientLimitersMu.Unlock()

	clientLimiters = map[string]*clientLimiter{}
}

func buildTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func createRequestByIP(ip string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = ip

	return request
}

func Test_rateLimitByIP_ImmediateRequestDenied(t *testing.T) {
	resetLimiters(t)

	mw := rateLimitByIP(60, 1)

	h := mw(buildTestHandler())

	firstRr := httptest.NewRecorder()
	h.ServeHTTP(firstRr, createRequestByIP("127.0.0.1:8080"))
	if firstRr.Code != http.StatusOK {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusOK, firstRr.Code)
	}

	secondRr := httptest.NewRecorder()
	h.ServeHTTP(secondRr, createRequestByIP("127.0.0.1:8080"))
	if secondRr.Code != http.StatusTooManyRequests {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusTooManyRequests, secondRr.Code)
	}
}

func Test_rateLimitByIp_DifferentIPRequestAccepted(t *testing.T) {
	resetLimiters(t)

	mw := rateLimitByIP(60, 1)

	h := mw(buildTestHandler())

	firstRr := httptest.NewRecorder()
	h.ServeHTTP(firstRr, createRequestByIP("127.0.0.1:8080"))
	if firstRr.Code != http.StatusOK {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusOK, firstRr.Code)
	}

	secondRr := httptest.NewRecorder()
	h.ServeHTTP(secondRr, createRequestByIP("127.0.0.1:8080"))
	if secondRr.Code != http.StatusTooManyRequests {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusTooManyRequests, secondRr.Code)
	}

	thirdRr := httptest.NewRecorder()
	h.ServeHTTP(thirdRr, createRequestByIP("192.0.0.1:8080"))
	if thirdRr.Code != http.StatusOK {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusOK, thirdRr.Code)
	}
}

func TestRateLimitByIP_UsesXForwardedForFirstIP(t *testing.T) {
	resetLimiters(t)

	mw := rateLimitByIP(60, 1)

	h := mw(buildTestHandler())

	firstRr := httptest.NewRecorder()
	firstR := createRequestByIP("127.0.0.1:8080")
	firstR.Header.Set("X-Forwarded-For", "222.50.30.11, 232.50.20.11, 124.54.45.12")
	h.ServeHTTP(firstRr, firstR)
	if firstRr.Code != http.StatusOK {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusOK, firstRr.Code)
	}

	secondRr := httptest.NewRecorder()
	secondR := createRequestByIP("127.0.0.1:8080")
	secondR.Header.Set("X-Forwarded-For", "222.50.30.11, 233.50.20.11, 127.54.45.12")
	h.ServeHTTP(secondRr, secondR)
	if secondRr.Code != http.StatusTooManyRequests {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusTooManyRequests, secondRr.Code)
	}
}

func TestRateLimitByIP_RPMDisabledIsNoOp(t *testing.T) {
	resetLimiters(t)

	mw := rateLimitByIP(0, 1)

	h := mw(buildTestHandler())

	firstRr := httptest.NewRecorder()
	h.ServeHTTP(firstRr, createRequestByIP("127.0.0.1:8080"))
	if firstRr.Code != http.StatusOK {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusOK, firstRr.Code)
	}

	secondRr := httptest.NewRecorder()
	h.ServeHTTP(secondRr, createRequestByIP("127.0.0.1:8080"))
	if secondRr.Code != http.StatusOK {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusOK, secondRr.Code)
	}
}

func TestRateLimitByIP_BurstDefaultedToOneWhenNonPositive(t *testing.T) {
	resetLimiters(t)

	mw := rateLimitByIP(60, 0)

	h := mw(buildTestHandler())

	firstRr := httptest.NewRecorder()
	h.ServeHTTP(firstRr, createRequestByIP("127.0.0.1:8080"))
	if firstRr.Code != http.StatusOK {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusOK, firstRr.Code)
	}

	secondRr := httptest.NewRecorder()
	h.ServeHTTP(secondRr, createRequestByIP("127.0.0.1:8080"))
	if secondRr.Code != http.StatusTooManyRequests {
		t.Fatalf("expecting status code to be %d, got %d", http.StatusTooManyRequests, secondRr.Code)
	}
}

func TestRateLimitByIP_ConcurrentAccessDoesNotRace(t *testing.T) {
	resetLimiters(t)

	var tooMany int32

	mw := rateLimitByIP(60000, 1000)
	h := mw(buildTestHandler())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr := httptest.NewRecorder()
			req := createRequestByIP("127.0.0.1:8080")
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusTooManyRequests {
				atomic.AddInt32(&tooMany, 1)
			}
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&tooMany) > 0 {
		t.Fatalf("expected no 429 responses in permissive concurrent test, got %d", tooMany)
	}
}
