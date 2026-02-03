
# Troubleshooting

## I can’t access the site from another machine

- Confirm Caddy is running and listening on :80 (and :443 if using HTTPS).
- Confirm the firewall allows inbound 80/443 on the LAN.
- Confirm your Caddyfile includes the LAN-only matcher and the reverse_proxy target is correct (see README example).

## Caddy says “LAN only” even from my LAN machine

- Ensure the client is actually in RFC1918 private space (10/8, 172.16/12, 192.168/16).
- If using VPNs or special routing, you may need to expand allowed ranges.

## Uploads fail immediately

- Check DropServe public service is running and reachable on 127.0.0.1:8080.
- Check Caddy logs for reverse proxy errors.
- Check server logs for size mismatches or path rejection.

## HTTPS shows browser warnings

- If using `tls internal`, clients must trust Caddy’s internal CA.
- Consider using a stable hostname (e.g., dropserve.lan) and installing trust on clients.

## Temp files accumulating

- Confirm the sweeper is enabled and running.
- Ensure the temp directory is inside the destination (see `06_File_IO_and_Cleanup.md`).
- Check that on request cancel/error, the code deletes `.part` and `.json`.
