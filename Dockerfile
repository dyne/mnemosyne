# Multi-stage build for Mnemosyne
# Stage 1: Build the Go binary
FROM golang:1.26-bookworm AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
    libsqlite3-dev gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /mnemosyne ./cmd/mnemosyne

# Stage 2: Minimal runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates libsqlite3-0 curl unzip \
    && rm -rf /var/lib/apt/lists/*

# Install Zenroom from GitHub releases
ENV ZENROOM_VERSION=5.31.2
RUN curl -fsSL -o /tmp/zenroom.zip \
    https://github.com/dyne/Zenroom/releases/download/v${ZENROOM_VERSION}/zenroom-linux.zip \
    && unzip -o /tmp/zenroom.zip -d /usr/local/bin/ \
    && chmod +x /usr/local/bin/zenroom \
    && rm /tmp/zenroom.zip

COPY --from=build /mnemosyne /usr/local/bin/mnemosyne
COPY zenflows/ /zenflows/
COPY web/ /web/

ENV MNEMOSYNE_ADDR=:8546
ENV MNEMOSYNE_WEB=/web
ENV MNEMOSYNE_CONTRACTS=/zenflows
ENV MNEMOSYNE_DB=/data/mnemosyne.db
ENV ZENROOM_BIN=/usr/local/bin/zenroom

RUN mkdir -p /data
VOLUME /data

EXPOSE 8546

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -fsS http://localhost:8546/health || exit 1

ENTRYPOINT ["mnemosyne"]
