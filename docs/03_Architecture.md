
# Architecture

## Overview

DropServe consists of:

1. **Public HTTP service** (proxied by Caddy)
   - Serves the landing page and portal upload UI
   - Implements public API for claiming portals, uploading files, checking status, and closing portals

2. **Control HTTP service** (local-only)
   - Used by CLI to create/manage portals with an absolute destination path
   - Must **not** be exposed via Caddy

3. **CLI**
   - Runs on the destination server
   - Creates portals through the Control API
   - Detects the primary LAN IPv4 address to print a usable link

4. **Caddy**
   - Owns ports 80/443
   - Enforces LAN-only access and proxies requests to the Public service

## Ports

Recommended defaults:

- Public service: `127.0.0.1:8080`
- Control service: `127.0.0.1:9090` (local-only; CLI uses this)
- Caddy: `:80` and optionally `:443`

## Trust boundaries

- Only the **Control API** is allowed to choose destination paths (`dest_abs`).
- Browsers never supply an absolute destination path; they only supply `relpath`.
- Caddy enforces LAN-only at the edge.
- Portal IDs are high-entropy capability tokens.

## Data model

- Portal record:
  - portal_id
  - dest_abs (absolute, canonical)
  - open_until timestamp
  - reusable boolean
  - policy defaults
  - state (open/claimed/closing/closed/expired)
  - active_uploads counter
  - last_activity timestamp
  - claimed_client_token (for one-time portals)

- Upload record:
  - upload_id (client-generated UUID)
  - portal_id
  - relpath
  - expected size
  - optional client sha256
  - status (writing/committed/failed)
  - server sha256
  - temp path (.part)
  - final path

## Persistence

v1 can use in-memory maps plus filesystem sidecar metadata in the portal temp directory. The startup sweeper can recover state from `.dropserve_tmp/` if desired.

If persistence is added later:
- use SQLite or a small embedded DB
- keep file safety invariants unchanged
