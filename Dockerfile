# Stage 1: Builder
FROM --platform=$BUILDPLATFORM golang:1.25.5 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

RUN apt-get update && apt-get install -y git bash curl ca-certificates && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o commission-tracker .

# Stage 2: Runtime
FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y bash netcat-openbsd ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/commission-tracker .

COPY templates/ templates/
COPY sql/schema/ sql/schema/
COPY static/ static/
COPY docker/entrypoint.sh .

RUN mkdir -p /app/outputs

RUN chmod +x entrypoint.sh

ENV PATH="/usr/local/bin:${PATH}"

ENTRYPOINT ["./entrypoint.sh"]
