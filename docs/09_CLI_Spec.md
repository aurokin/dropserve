
# CLI Specification

The CLI runs on the destination server.

## Commands

### `dropserve`
Defaults to `open` for the current working directory.

### `dropserve open`
Opens a portal for the current working directory.

Behavior:
- Detect `dest_abs` as the current directory (canonical).
- Call Control API: `POST http://127.0.0.1:8080/api/control/portals`
- Detect primary LAN IPv4 address.
- Print a link to the portal page:
  - HTTP: `http://{primary_ipv4}:{PUBLIC_PORT}/p/{portal_id}`
  - HTTPS (if enabled via Caddy and configured): `https://{host}/p/{portal_id}`

Flags:
- `--minutes <N>` (default 15; alias: `-m`)
- `--reusable` (default false; aliases: `--reuseable`, `-r`)
- `--policy overwrite|autorename` (default overwrite)
- `--host <HOST>` (optional; override LAN host/IP in the printed link)
- `--port <N>` (optional; override server port for control call + printed link)

### `dropserve serve`
Starts the DropServe server services for local development.

Behavior:
- Starts the HTTP service on `DROPSERVE_ADDR` (default `0.0.0.0:8080`).
- `--port <N>` overrides the port.

### `dropserve version`
Print version info.

### `dropserve help`
Show help (aliases: `-h`, `--help`, `-help`).

## Primary LAN IPv4 detection

The CLI should attempt:

1. Default route method:
   - Create a UDP socket and “connect” to a public IP:port (no data needs to be sent).
   - Read the local socket address; use that IPv4 as the primary IP.

2. Fallback:
   - Enumerate interfaces and choose first UP, non-loopback, private RFC1918 IPv4.

3. Override:
   - `--host` or env config for unusual network setups.

## Config

Optional config file (e.g., `~/.config/dropserve/config.toml`):

- base_url: `http://127.0.0.1:8080`
- public_base_url: optional override (e.g., `https://dropserve.lan`)
- defaults: minutes, policy, reusable
