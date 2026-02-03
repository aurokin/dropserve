# Operations

## Environment configuration

Server:
- `DROPSERVE_ADDR` (default `0.0.0.0:8080`)
- `DROPSERVE_PORT` (optional; overrides port for `DROPSERVE_ADDR`)
- `DROPSERVE_PUBLIC_ADDR` (legacy alias)
- `DROPSERVE_TMP_DIR_NAME` (default `.dropserve_tmp`)
- `DROPSERVE_SWEEP_INTERVAL_SECONDS` (default 120)
- `DROPSERVE_PART_MAX_AGE_SECONDS` (default 600)
- `DROPSERVE_PORTAL_IDLE_MAX_SECONDS` (default 1800)
- `DROPSERVE_SWEEP_ROOTS` (default current directory; colon-separated)
- `DROPSERVE_MAX_UPLOAD_BYTES` (optional; default unlimited)
- `DROPSERVE_LOG_LEVEL` (default `info`)

CLI:
- `DROPSERVE_URL` overrides the CLI base URL.
- `--port` on `dropserve open` overrides the base URL port.
- Optional public base URL override:
  - `http://{primary_ipv4}` (HTTP)
  - `https://dropserve.lan` (TLS via Caddy)

Caddy:
- `DROPSERVE_LAN_HOST` (default `dropserve.lan`) for the README example.

## Development setup (short form)

1. Install Go 1.22 and Bun.
2. Build web assets:
   - `cd web`
   - `bun install`
   - `bun run build`
3. Run server:
   - `go run ./cmd/dropserve serve`
4. Open portal:
   - `go run ./cmd/dropserve --minutes 15`

## Troubleshooting

- **Cannot access from another machine**: verify Caddy is running on :80/:443 and firewall allows inbound LAN traffic.
- **"LAN only" from a LAN machine**: verify client IP is RFC1918 private space.
- **Uploads fail immediately**: check server logs and Caddy proxy target; ensure server is listening.
- **HTTPS warnings**: with `tls internal`, clients must trust Caddy's internal CA.
- **Temp files accumulate**: verify sweeper settings and failure cleanup paths.

## Test plan (manual highlights)

- Upload a folder with 100–300 files and verify integrity.
- Cancel a large upload; confirm no partial files remain.
- Kill server mid-upload; on restart sweeper cleans temp artifacts.
- Expire a portal mid-upload; upload completes, then portal closes.
- Verify overwrite warnings and autorename behavior.
