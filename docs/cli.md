# CLI

The CLI runs on the destination server.

## Commands

### `dropserve` (default: open)

Uses the current working directory as the destination.

### `dropserve open`

- Resolves `dest_abs` to the canonical current directory.
- Calls `POST http://127.0.0.1:8080/api/control/portals`.
- Detects the primary LAN IPv4 address.
- Prints a portal link:
  - HTTP: `http://{primary_ipv4}:{PUBLIC_PORT}/p/{portal_id}`
  - HTTPS (if Caddy configured): `https://{host}/p/{portal_id}`

Flags:
- `--minutes <N>` (default 15; alias `-m`)
- `--reusable` (alias `--reuseable`, `-r`)
- `--policy overwrite|autorename`
- `--host <HOST>` override LAN host/IP in the printed link
- `--port <N>` override server port for control call + printed link

### `dropserve serve`

- Starts the HTTP service on `DROPSERVE_ADDR` (default `0.0.0.0:8080`).
- `--port <N>` overrides the port.

### `dropserve version`

Prints version info.

## LAN IPv4 detection

1. Default route method: UDP "connect" to a public IP and read the local socket address.
2. Fallback: choose the first UP, non-loopback, private RFC1918 IPv4.
3. Override: `--host` for unusual setups.

## Config

Optional config file (example `~/.config/dropserve/config.toml`):

- `base_url`: `http://127.0.0.1:8080`
- `public_base_url`: optional override (e.g., `https://dropserve.lan`)
- `defaults`: minutes, policy, reusable
