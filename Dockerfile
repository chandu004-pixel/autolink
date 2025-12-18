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

# Copy binaries and scripts
COPY --from=builder /app/platform .
COPY --from=builder /app/bot .
COPY --from=builder /app/start.sh .
RUN chmod +x start.sh

# Copy assets and environment
COPY --from=builder /app/simulated-app ./simulated-app
COPY --from=builder /app/.env.example .env

# Expose the platform port
EXPOSE 8080

# Run both services via the start script
CMD ["./start.sh"]
