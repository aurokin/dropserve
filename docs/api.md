# API

This document defines the HTTP API surface.

## Service shape

- Single HTTP service (proxied by Caddy).
- Bind: `DROPSERVE_ADDR` (default `0.0.0.0:8080`).
- Control endpoints live under `/api/control/*` and must be blocked by the proxy.

## Common headers

- `X-Client-Token`: required for state-changing public endpoints once a one-time portal is claimed.
- `X-Request-Id`: optional; if present, server logs it and returns the same ID.

## Public endpoints

- `GET /` landing page.
- `GET /p/{portal_id}` portal UI.
- `GET /api/portals/{portal_id}/info` portal metadata.
- `POST /api/portals/{portal_id}/claim` issue `client_token` (one-time only).
- `POST /api/portals/{portal_id}/preflight` collision check.
- `POST /api/portals/{portal_id}/uploads` init upload.
- `PUT /api/uploads/{upload_id}` stream upload bytes.
- `GET /api/uploads/{upload_id}/status` check upload state.
- `POST /api/portals/{portal_id}/close` close portal.

## Control endpoints (CLI-only)

- `POST /api/control/portals` create portal.
- `POST /api/control/portals/{portal_id}/close` admin close.
- `GET /api/control/health` basic health check.

## Notes

- All paths are relative to the same server.
- See `file-safety.md` for file handling rules.
