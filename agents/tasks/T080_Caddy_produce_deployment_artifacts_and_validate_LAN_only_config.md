# Task: T080 - Caddy: produce deployment artifacts and validate LAN-only config

## Goal

Provide ready-to-copy Caddyfile templates and verify they work with the chosen ports.

## Required reading (MUST read before starting)

- `docs/10_Caddy_Deployment.md`
- `docs/03_Architecture.md`

## Deliverables

- Caddyfile templates in docs
- A short quickstart checklist

## Steps

1. Confirm reverse_proxy points to 127.0.0.1:8080.
2. Confirm Control API is not proxied.
3. Include LAN-only matcher examples.

## Acceptance criteria

- [ ] Docs are copy/paste runnable.
- [ ] LAN-only enforced at Caddy level.

## Tests to run

- Manual test: access from LAN works; public IP (if any) denied.

## Notes

- None.
