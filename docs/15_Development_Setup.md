 
# Development Environment Setup

This guide walks a junior developer through setting up a full DropServe dev environment: the Go server + CLI, the web UI build pipeline, and optional Caddy for LAN testing.

## Components (what you are installing and why)

- Go 1.22: builds and runs the DropServe server and CLI.
- Bun + Vite: builds the React web UI into `internal/webassets/dist` so Go can embed it.
- Caddy (optional): serves HTTPS on your LAN using the provided Caddyfile.

## Prerequisites

- Git
- Go 1.22 (`go version` should report 1.22.x)
- Bun (recommended for this repo because `web/bun.lock` is present)
- Caddy (optional; only needed for LAN HTTPS testing)

## One-time setup

1) Clone the repo and enter it.

```bash
git clone <your-repo-url>
cd dropserve
```

2) Install web dependencies.

```bash
cd web
bun install
```

3) Build the web UI assets so the Go server can embed them.

```bash
bun run build
```

This writes to `internal/webassets/dist` which the Go server embeds at build/runtime.

## Run the backend and CLI (local dev)

1) Start the DropServe server (control + public APIs).

```bash
go run ./cmd/dropserve serve
```

By default this starts:

- Control API: `127.0.0.1:9090` (CLI talks to this)
- Public API + UI: `127.0.0.1:8080`

2) In a second terminal, open a portal from any directory you want to upload into.

```bash
go run ./cmd/dropserve open --minutes 15
```

The CLI prints a URL. Open it in a browser and upload files.

## Optional: LAN HTTPS with Caddy

Use this when you want HTTPS and a stable LAN hostname.

1) Start Caddy using the repo Caddyfile.

```bash
DROPSERVE_LAN_HOST=dropserve.lan caddy run --config Caddyfile
```

2) (Optional) Trust Caddy’s local CA so browsers accept the cert.

```bash
caddy trust
```

Notes:

- Caddy only proxies the public service on `127.0.0.1:8080`.
- The control service (`127.0.0.1:9090`) must stay local-only.

## Environment variables you can customize

- `DROPSERVE_PUBLIC_ADDR` (default `127.0.0.1:8080`)
- `DROPSERVE_CONTROL_ADDR` (default `127.0.0.1:9090`)
- `DROPSERVE_TMP_DIR_NAME` (default `.dropserve_tmp`)
- `DROPSERVE_SWEEP_INTERVAL_SECONDS` (default 120)
- `DROPSERVE_PART_MAX_AGE_SECONDS` (default 600)
- `DROPSERVE_PORTAL_IDLE_MAX_SECONDS` (default 1800)
- `DROPSERVE_SWEEP_ROOTS` (default current directory; colon-separated paths)
- `DROPSERVE_MAX_UPLOAD_BYTES` (optional; default unlimited)
- `DROPSERVE_LOG_LEVEL` (default `info`)

## Troubleshooting quick hits

- “web ui not available”: run `bun run build` again to refresh `internal/webassets/dist`.
- Port already in use: change `DROPSERVE_PUBLIC_ADDR` or `DROPSERVE_CONTROL_ADDR`.
- Caddy can’t bind 80/443: stop other services or run without Caddy.
