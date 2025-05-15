FROM golang:1.21-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Create necessary directories
RUN mkdir -p /data /app/static /app/templates

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the static files and templates first
COPY static/ ./static/
COPY templates/ ./templates/

# Copy the rest of the source code
COPY . .

# Set proper permissions
RUN chmod -R 755 /app
RUN chmod -R 777 /data

# Build the application
RUN go build -o main .

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./main"] 