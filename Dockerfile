# Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
# Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
# All rights reserved. See LICENSE for details.

# =============================================================================
# YaaiCMS — Production Multi-Stage Dockerfile
# =============================================================================
# Stage 1: Compile TailwindCSS (standalone CLI) + vendor frontend JS
# Stage 2: Compile Go binary (with embedded assets + libvips for image processing)
# Stage 3: Minimal runtime image with libvips shared libraries
# =============================================================================

# ---------------------------------------------------------------------------
# Stage 1: Frontend assets (no Node.js — uses Tailwind standalone CLI)
# ---------------------------------------------------------------------------
FROM alpine:3.21 AS frontend

ARG TAILWIND_VERSION=3.4.19

WORKDIR /build

# Download Tailwind CSS standalone CLI. Bundles all first-party plugins
# (@tailwindcss/forms, @tailwindcss/typography) — no npm needed.
RUN apk add --no-cache curl \
    && curl -sLo /usr/local/bin/tailwindcss \
       "https://github.com/tailwindlabs/tailwindcss/releases/download/v${TAILWIND_VERSION}/tailwindcss-linux-x64" \
    && chmod +x /usr/local/bin/tailwindcss

# Copy Tailwind config and admin templates to scan for CSS classes.
COPY tailwind.config.js ./
COPY web/static/css/input.css ./web/static/css/
COPY internal/render/templates/ ./internal/render/templates/

# Compile admin CSS (minified, tree-shaken to only used classes).
RUN tailwindcss -i ./web/static/css/input.css \
    -o ./web/static/css/admin.css --minify

# Vendor HTMX, AlpineJS, and EasyMDE for offline serving.
RUN mkdir -p web/static/js web/static/css \
    && curl -sLo web/static/js/htmx.min.js \
       "https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js" \
    && curl -sLo web/static/js/alpine.min.js \
       "https://unpkg.com/alpinejs@3.14.8/dist/cdn.min.js" \
    && curl -sLo web/static/js/easymde.min.js \
       "https://unpkg.com/easymde@2.20.0/dist/easymde.min.js" \
    && curl -sLo web/static/css/easymde.min.css \
       "https://unpkg.com/easymde@2.20.0/dist/easymde.min.css"

# Vendor Ace Editor for HTML template editing.
ARG ACE_VERSION=1.36.5
ARG ACE_CDN=https://cdn.jsdelivr.net/npm/ace-builds@${ACE_VERSION}/src-min-noconflict

RUN mkdir -p web/static/js/ace \
    && for f in ace.js ext-language_tools.js ext-searchbox.js \
         mode-html.js mode-css.js mode-javascript.js \
         worker-html.js worker-css.js worker-javascript.js \
         theme-chrome.js theme-monokai.js; do \
         curl -sLo "web/static/js/ace/$f" "${ACE_CDN}/$f"; \
       done

# ---------------------------------------------------------------------------
# Stage 2: Go binary build (CGO enabled for libvips image processing)
# ---------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates build-base pkgconfig vips-dev

WORKDIR /app

# Download dependencies first (cached unless go.mod/go.sum change).
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source tree.
COPY . .

# Overlay the compiled frontend assets so they get embedded via //go:embed.
COPY --from=frontend /build/web/static/ ./web/static/

# Build with CGO enabled for libvips bindings.
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /yaaicms \
    ./cmd/yaaicms

# ---------------------------------------------------------------------------
# Stage 3: Runtime
# ---------------------------------------------------------------------------
FROM alpine:3.21

# Install CA certificates, timezone data, and libvips runtime libraries.
RUN apk add --no-cache ca-certificates tzdata \
    vips libwebp libpng libjpeg-turbo tiff giflib \
    && adduser -D -H -s /sbin/nologin yaaicms

COPY --from=builder /yaaicms /usr/local/bin/yaaicms

# Limit glib malloc arenas to prevent memory fragmentation under load.
ENV MALLOC_ARENA_MAX=2

USER yaaicms

EXPOSE 8080

ENTRYPOINT ["yaaicms"]
