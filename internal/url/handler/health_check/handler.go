package health_check

import (
	"encoding/json"
	"net/http"
	"url-shortener/internal/url/handler"
)

func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(`{"status": "ok"}`))
	if err != nil {
		writeJSONError(w, 500, "internal error")
	}
}

func writeJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(handler.ErrorResponse{
		Message: message,
	})
}
