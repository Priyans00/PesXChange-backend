# Use the official Go image as base
FROM golang:1.22-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (required for go mod download)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Start fresh from alpine for smaller image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh pesxchange

WORKDIR /home/pesxchange

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Change ownership to non-root user
RUN chown pesxchange:pesxchange main

# Switch to non-root user
USER pesxchange

# Expose port
EXPOSE 8080

# Command to run
CMD ["./main"]