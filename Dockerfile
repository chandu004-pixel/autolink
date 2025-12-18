# Build Stage
FROM golang:1.23-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o platform ./simulated-app/server/main.go
RUN go build -o bot ./cmd/bot/main.go

# Run Stage
FROM debian:bookworm-slim
WORKDIR /app

# Install Chromium and dependencies for ROD/Stealth
RUN apt-get update && apt-get install -y \
    chromium \
    ca-certificates \
    fonts-liberation \
    libnss3 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    && rm -rf /var/lib/apt/lists/*

# Copy binaries and assets
COPY --from=builder /app/platform .
COPY --from=builder /app/bot .
COPY --from=builder /app/simulated-app/templates ./simulated-app/templates
COPY --from=builder /app/simulated-app/static ./simulated-app/static
COPY --from=builder /app/.env.example .env

# Expose the platform port
EXPOSE 8080

# Default to running the platform
CMD ["./platform"]
