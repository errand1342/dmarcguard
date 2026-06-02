# syntax=docker/dockerfile:1
FROM oven/bun:1 AS frontend-builder

# Install system dependencies needed for SVG processing
RUN apt-get update && apt-get install -y \
    fontconfig \
    libfontconfig1 \
    libfontconfig1-dev \
    libcairo2 \
    libcairo2-dev \
    libpango1.0-0 \
    libpango1.0-dev \
    libjpeg62-turbo \
    libjpeg62-turbo-dev \
    libpng16-16 \
    libpng-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build/frontend

RUN --mount=type=bind,source=package.json,target=package.json \
    --mount=type=bind,source=bun.lock,target=bun.lock \
    --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile

COPY . .
RUN bun run build

FROM golang:1.26 AS mod

WORKDIR /app

RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM golang:1.26 AS backend-builder

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
ARG BUILT_BY=docker

WORKDIR /app

ENV CGO_ENABLED=0

COPY . .
COPY --from=frontend-builder /build/frontend/dist ./internal/api/dist

RUN --mount=type=cache,target=/go/pkg/mod \
    go build -ldflags="-s -w -extldflags '-static' -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -X main.builtBy=${BUILT_BY}" -trimpath -o parse-dmarc . && \
    mkdir -p /data /tmp

FROM scratch AS final

VOLUME /data
ENV PARSE_DMARC_CONFIG=/app/config.json \
    DATABASE_PATH=/data/parse-dmarc.db

COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend-builder /app/parse-dmarc /usr/local/bin/parse-dmarc
COPY --from=backend-builder /data /data
COPY --from=backend-builder /tmp /tmp

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/parse-dmarc"]
CMD ["--config=/app/config.json"]
