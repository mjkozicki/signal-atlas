# syntax=docker/dockerfile:1
FROM node:24-bookworm-slim AS frontend
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci
COPY svelte.config.js tsconfig.json vite.config.ts ./
COPY src/web/ ./src/web/
RUN npm run build:web

FROM golang:1.26.4-bookworm AS backend
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY src/ ./src/
COPY --from=frontend /build/src/internal/web/dist/ ./src/internal/web/dist/
RUN CGO_ENABLED=0 go build -trimpath -o /out/ ./src/cmd/...

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 atlas \
    && useradd --uid 10001 --gid atlas --no-create-home atlas \
    && mkdir -p /app/.data \
    && chown atlas:atlas /app/.data
WORKDIR /app
COPY --from=backend /out/ /app/bin/
COPY LICENSE ATTRIBUTION.md THIRD_PARTY_NOTICES.md /app/
COPY licenses/ /app/licenses/
COPY fixtures/ /app/fixtures/
ENV PATH="/app/bin:${PATH}"
USER 10001:10001
EXPOSE 8787
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
    CMD curl -fsS --max-time 1 http://127.0.0.1:8787/api/scans > /dev/null \
    && curl -fsS --max-time 1 http://127.0.0.1:8787/api/bluetooth/scans > /dev/null \
    && curl -fsS --max-time 1 http://127.0.0.1:8787/api/nfc/scans > /dev/null
ENTRYPOINT ["/app/bin/signal-atlas", "--container"]
CMD ["--demo"]
