FROM golang:1.25-alpine AS builder
# Set environment variables
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
# Set working directory inside the container
WORKDIR /build
# Copy go.mod and go.sum files for dependency installation
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download
# Copy the entire application source
COPY . .
# Build the Go binary
RUN go build -o /app ./cmd/

# Final lightweight stage
FROM alpine:3.17 AS final
# Copy the compiled binary from the builder stage
COPY --from=builder /app /bin/app
#RUN chmod +x /bin/app
# Copy the index.html to the final stage
COPY --from=builder /build/index.html /index.html
# Expose the application's port
EXPOSE 8080
# Run the application
CMD ["bin/app"]