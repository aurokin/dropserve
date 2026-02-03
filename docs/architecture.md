# Architecture

## Components

- **HTTP service** (proxied by Caddy)
  - Serves the landing page and portal UI.
  - Exposes public API for portal info, claims, uploads, and close.
  - Hosts control endpoints under `/api/control/*` for the CLI.
- **CLI**
  - Runs on the destination server.
  - Creates portals through `/api/control/portals`.
  - Detects the primary LAN IPv4 to print a usable link.
- **Caddy**
  - Owns ports 80/443.
  - Enforces LAN-only access.
  - Blocks `/api/control/*` from public access.

## Ports

- HTTP service: `0.0.0.0:8080` by default.
- Caddy: `:80` and optionally `:443`.

## Trust boundaries

- Only the Control API chooses destination paths (`dest_abs`).
- Browsers only send `relpath` values.
- Caddy enforces LAN-only access.
- Portal IDs and client tokens are high entropy.

## Data model (high level)

- **Portal**: `portal_id`, `dest_abs`, `open_until`, `reusable`, `policy`, `state`, `active_uploads`, `last_activity`, `claimed_client_token`.
- **Upload**: `upload_id`, `portal_id`, `relpath`, `size`, `client_sha256`, `status`, `server_sha256`, `temp_path`, `final_path`.

## Persistence

- v1 uses in-memory state plus filesystem sidecar metadata under `.dropserve_tmp/`.
- Startup sweeper can recover or clean stale portal state.
- If persistence is added later, keep file safety invariants unchanged.
