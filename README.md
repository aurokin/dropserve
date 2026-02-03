
# DropServe Documentation Pack

This folder is meant to be zipped and shared with both **humans** and **AI coding agents** to implement the DropServe project.

## What DropServe is

A self-hosted LAN-only web app + CLI pair:

- You run a CLI **on the destination server** inside the directory where uploaded files should be written.
- The CLI opens a short-lived **portal** and prints a unique link.
- Anyone on your LAN can open that link and drag & drop files/folders to upload.
- The server ensures **no partial files** land in the final destination and performs aggressive cleanup of temporary files.

## Key constraints (from project decisions)

- OS: Ubuntu LTS
- Reverse proxy: **Caddy only** (ports 80/443)
- App binds to all interfaces on a high port (e.g., 0.0.0.0:8080) and is proxied by Caddy
- Typical upload < 100MB; outliers up to 50GB
- Desktop-first for v1
- Restarting an upload is acceptable (no resumable protocol required for v1)
- Cleanup of incomplete files is critical

## Where to start

- Humans: read `docs/README.md`
- AI agents: read `agents/00_Read_First.md`
  - Doc update policy lives in `docs/README.md`.

## Build and install (local)

Build the web assets and Go binary:

```bash
./scripts/build.sh
```

Install the binary to `~/.local/bin/dropserve`:

```bash
./scripts/install.sh
```

If `~/.local/bin` is not on your PATH, add it in your shell profile.

## Caddy example (LAN-only)

DropServe expects to run behind Caddy for LAN-only access. Example Caddyfile (HTTPS with internal CA):

```caddyfile
{$DROPSERVE_LAN_HOST:dropserve.lan} {
    @denied not remote_ip private_ranges
    respond @denied "LAN only" 403

    @control path /api/control/*
    respond @control "Not Found" 404

    tls internal
    reverse_proxy 127.0.0.1:8080
}
```

HTTP-only variant:

```caddyfile
http://:80 {
    @denied not remote_ip private_ranges
    respond @denied "LAN only" 403

    @control path /api/control/*
    respond @control "Not Found" 404

    reverse_proxy 127.0.0.1:8080
}
```

## Date & timezone reference

- Pack created: 2026-01-29
- Expected user timezone: America/Denver
