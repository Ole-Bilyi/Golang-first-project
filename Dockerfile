# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create necessary directories with proper permissions
RUN mkdir -p /data /app/static /app/templates && \
    chmod -R 755 /app && \
    chmod -R 777 /data

# Copy binary from builder
COPY --from=builder /build/main .

# Copy static files and templates
COPY static/ ./static/
COPY templates/ ./templates/

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./main"] 