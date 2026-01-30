
# CLI Specification

The CLI runs on the destination server.

## Commands

### `dropserve open`
Opens a portal for the current working directory.

Behavior:
- Detect `dest_abs` as the current directory (canonical).
- Call Control API: `POST http://127.0.0.1:9090/api/control/portals`
- Detect primary LAN IPv4 address.
- Print a link to the portal page:
  - HTTP: `http://{primary_ipv4}:{PUBLIC_PORT}/p/{portal_id}`
  - HTTPS (if enabled via Caddy and configured): `https://{host}/p/{portal_id}`

Flags:
- `--minutes <N>` (default 15)
- `--reusable` (default false)
- `--policy overwrite|autorename` (default overwrite)
- `--open-browser` (optional; tries to open the URL)

### `dropserve version`
Print version info.

### `dropserve help`
Show help.

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

- control_api: `http://127.0.0.1:9090`
- public_base_url: optional override (e.g., `https://dropserve.lan`)
- defaults: minutes, policy, reusable
