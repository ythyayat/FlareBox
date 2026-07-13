FROM golang:1.21-alpine AS builder

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o flarebox .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata sqlite-libs wget

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/flarebox .

# Create data directory
RUN mkdir -p data

# Expose port
EXPOSE 2525

# Run the application
CMD ["./flarebox"]
