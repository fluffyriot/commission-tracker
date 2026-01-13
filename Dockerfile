# Stage 1: Builder
FROM golang:1.25.5 AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y git bash curl ca-certificates && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o commission-tracker .

# Install Goose CLI into /go/bin
ENV GOBIN=/go/bin
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Stage 2: Runtime
FROM debian:bookworm-slim

WORKDIR /app

# Runtime deps
RUN apt-get update && apt-get install -y bash netcat-openbsd ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy app binary
COPY --from=builder /app/commission-tracker .

# Copy Goose CLI from builder
COPY --from=builder /go/bin/goose /usr/local/bin/goose

# Copy runtime assets
COPY templates/ templates/
COPY sql/schema/ sql/schema/
COPY static/ static/
COPY docker/entrypoint.sh .

RUN mkdir -p /app/outputs

RUN chmod +x entrypoint.sh

# Add Goose to PATH just in case
ENV PATH="/usr/local/bin:${PATH}"

ENTRYPOINT ["./entrypoint.sh"]
