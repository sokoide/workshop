# --- Stage 1: Build ---
FROM --platform=linux/amd64 golang:1.26.1-alpine AS builder

# Install necessary tools (git, ca-certificates, etc.)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependencies first to leverage caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
# Use CGO_ENABLED=0 for static linking and strip symbol information (-ldflags="-s -w") for a smaller binary
# Explicitly set GOARCH=amd64 to ensure an amd64 binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main .

# --- Stage 2: Runtime ---
# Use a minimal alpine image for security and size efficiency
FROM --platform=linux/amd64 alpine:3.19
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the built binary
COPY --from=builder /app/main .

# Expose port (App Service default is 8080)
EXPOSE 8080

# Execute
CMD ["./main"]
