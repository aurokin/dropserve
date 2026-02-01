
# Configuration Reference

## Server configuration

Suggested environment variables (names may vary by implementation):

- `DROPSERVE_PUBLIC_ADDR` (default `127.0.0.1:8080`)
- `DROPSERVE_CONTROL_ADDR` (default `127.0.0.1:9090`)
- `DROPSERVE_TMP_DIR_NAME` (default `.dropserve_tmp`)
- `DROPSERVE_SWEEP_INTERVAL_SECONDS` (default 120)
- `DROPSERVE_PART_MAX_AGE_SECONDS` (default 600)
- `DROPSERVE_PORTAL_IDLE_MAX_SECONDS` (default 1800)
- `DROPSERVE_SWEEP_ROOTS` (default current working directory; colon-separated paths)
- `DROPSERVE_MAX_UPLOAD_BYTES` (optional; default unlimited)
- `DROPSERVE_LOG_LEVEL` (default `info`)

## CLI configuration

- Control API base URL (default `http://127.0.0.1:9090`)
- Optional public base URL override:
  - `http://{primary_ipv4}` (HTTP mode)
  - `https://dropserve.lan` (TLS mode)

## Caddy configuration

- `DROPSERVE_LAN_HOST` (default `dropserve.lan`): hostname or LAN IP used in the HTTPS Caddyfile.
