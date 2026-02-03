# Task: T080 - Caddy: produce deployment artifacts and validate LAN-only config

## Goal

Provide a ready-to-copy Caddy example (in README) and verify it works with the chosen ports.

## Required reading (MUST read before starting)

- `README.md`
- `docs/architecture.md`

## Deliverables

- Caddy example in README
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
