# Build stage
FROM golang:1.24.6-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o treasurehunt main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS and wget for healthcheck
RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/treasurehunt .

# Copy .env file if it exists (optional)
COPY .env* ./ 

# Expose port
EXPOSE 8080

# Run the application
CMD ["./treasurehunt"]
