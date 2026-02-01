
# Caddy Deployment (Caddy-only)

This document describes the required setup for running DropServe behind Caddy.

## Overview

- Caddy owns ports 80 and optionally 443.
- DropServe Public Service listens on localhost (default 127.0.0.1:8080).
- DropServe Control Service listens on localhost (default 127.0.0.1:9090) and MUST NOT be proxied.

## LAN-only access

Caddy should enforce LAN-only access using a request matcher (private IP ranges). This is important because the app itself is bound to localhost and only reachable through Caddy.

## Option A — HTTPS on LAN using Caddy internal CA (default)

Use a stable LAN hostname if possible (recommended), e.g. `dropserve.lan`. The repository `Caddyfile`
expects a hostname or IP via `DROPSERVE_LAN_HOST`.

Create `/etc/caddy/Caddyfile`:

```caddyfile
{$DROPSERVE_LAN_HOST:dropserve.lan} {
    @denied not remote_ip private_ranges
    respond @denied "LAN only" 403

    tls internal
    reverse_proxy 127.0.0.1:8080
}
```

Notes:
- Set `DROPSERVE_LAN_HOST` to your LAN IP or hostname (example: `192.168.1.23`).
- Clients may need to trust Caddy’s internal CA to avoid browser warnings.
- On Ubuntu, you may use `caddy trust` to install Caddy’s CA in local trust stores (requires privileges).
- Each client machine may need to trust the CA as well.

Then reload Caddy.

## Option B — HTTP only (fallback)

If you need HTTP-only operation, use the provided `Caddyfile.http`:

```caddyfile
http://:80 {
    @denied not remote_ip private_ranges
    respond @denied "LAN only" 403

    reverse_proxy 127.0.0.1:8080
}
```

### Using an IP address in HTTPS

You MAY configure Caddy with a specific IP address:

```caddyfile
https://192.168.1.23 {
    @denied not remote_ip private_ranges
    respond @denied "LAN only" 403

    tls internal
    reverse_proxy 127.0.0.1:8080
}
```

However, hostname-based TLS is typically less fragile long-term.

## systemd notes

- Run DropServe as a user service or system service, binding only to localhost.
- Let Caddy be the only service bound to 80/443.

## What NOT to do

- Do not expose the Control Service (9090) via Caddy.
- Do not bind DropServe directly to 0.0.0.0 unless you have a specific reason and understand the security implications.
