# --- Stage 1: Build Stage ---
# Upgraded to match your local project's toolchain requirements
FROM golang:1.26-alpine AS builder

# Install build dependencies if needed
RUN apk add --no-cache git

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker caching for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code and templates
COPY . .

# Build the Go application as a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o xpath-hunter .

# --- Stage 2: Final Runtime Stage ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/xpath-hunter .

# Copy the templates directory
COPY --from=builder /app/templates ./templates

# Expose the port your Go server runs on
EXPOSE 8080

# Run the web application
CMD ["./xpath-hunter"]