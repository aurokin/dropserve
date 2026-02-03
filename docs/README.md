# DropServe Docs

This folder contains the source-of-truth documentation for DropServe. Start here, then follow the links below.

## Read order

1. `overview.md`
2. `architecture.md`
3. `portal-lifecycle.md`
4. `api.md`
5. `cli.md`
6. `file-safety.md`
7. `web-ui.md`
8. `operations.md`
9. `licensing.md`

## Source-of-truth rules

- **Portal behavior** must match `portal-lifecycle.md`.
- **Server endpoints** must match `api.md`.
- **File write + cleanup invariants** must match `file-safety.md`.
- **Deployment** uses Caddy-only; see the README example (no NGINX guidance).

When docs conflict:
1. `file-safety.md` wins for file safety and cleanup.
2. `portal-lifecycle.md` wins for lifecycle/timers.
3. `api.md` wins for endpoints and payloads.

## Doc update policy

- Update docs in `docs/` whenever behavior changes.
- Keep `docs/README.md` as the canonical map and ruleset.
- Avoid duplicating content across multiple files; consolidate instead.
- If removing a doc, update references in `README.md` and `agents/` files.

## Glossary

- **Portal**: A short-lived capability link that authorizes uploads to a specific destination directory.
- **Control API**: CLI-only endpoints under `/api/control/*`; block these in Caddy.
- **Public API**: Proxied by Caddy; used by browsers to claim portals and upload files.
- **Commit**: The atomic rename/move step that makes a file appear in the final destination path.
