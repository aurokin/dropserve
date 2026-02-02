
# API Specification

This document defines the API surface. The implementation must follow this spec.

## Service split

- **Public Service** (proxied by Caddy): user browsers access this.
  - Bind: `PUBLIC_ADDR` (default `127.0.0.1:8080`)

- **Control Service** (NOT proxied): CLI uses this to create portals.
  - Bind: `CONTROL_ADDR` (default `127.0.0.1:9090`, loopback only)

## Common headers

- `X-Client-Token`: required for state-changing public endpoints once a one-time portal is claimed (reusable portals ignore client tokens).
- `X-Request-Id`: optional; if present, server logs it.

## Public Service Endpoints

### GET /
Landing page. Explains how to run the CLI to open a portal.

- 200 text/html

### GET /p/{portal_id}
Portal upload page.

- 200 text/html
- 404 if portal not found or invalid

### GET /api/portals/{portal_id}/info
Returns portal metadata without claiming.

Response 200:
```json
{
  "portal_id": "p_...",
  "expires_at": "RFC3339 timestamp",
  "policy": {
    "overwrite": true,
    "autorename": true
  },
  "reusable": false
}
```

Errors:
- 404 portal not found
- 410 portal closed/expired

### POST /api/portals/{portal_id}/claim
Claims a portal for a browser client.

Notes:
- Reusable portals do not require claiming; clients should skip this call.

Request:
```json
{}
```

Response 200:
```json
{
  "portal_id": "p_...",
  "client_token": "ct_...",
  "expires_at": "RFC3339 timestamp",
  "policy": {
    "overwrite": true,
    "autorename": true
  },
  "reusable": false
}
```

Errors:
- 404 portal not found
- 409 already claimed (one-time portal)

### POST /api/portals/{portal_id}/preflight
Checks collisions and totals.

Request:
```json
{
  "items": [
    {"relpath": "photos/2024/img1.jpg", "size": 123}
  ]
}
```

Response 200:
```json
{
  "total_files": 1,
  "total_bytes": 123,
  "conflicts": [
    {"relpath": "photos/2024/img1.jpg", "reason": "exists"}
  ]
}
```

### POST /api/portals/{portal_id}/uploads
Initializes an upload and reserves a temp path.

Request:
```json
{
  "upload_id": "uuid-v4",
  "relpath": "photos/2024/img1.jpg",
  "size": 12345678,
  "client_sha256": null,
  "policy": "overwrite"  // or "autorename"
}
```

Response 200:
```json
{
  "upload_id": "uuid-v4",
  "put_url": "/api/uploads/uuid-v4"
}
```

Errors:
- 404 portal invalid
- 409 upload_id already committed (idempotency)

### PUT /api/uploads/{upload_id}
Streams bytes for the upload.

Requirements:
- Request body is raw bytes.
- `Content-Length` must match the expected `size` from init.

Response 200:
```json
{
  "status": "committed",
  "relpath": "photos/2024/img1.jpg",
  "server_sha256": "hex",
  "bytes_received": 12345678,
  "final_relpath": "photos/2024/img1.jpg"
}
```

Errors:
- 400 size mismatch
- 409 already committed
- 410 portal closed/expired

### GET /api/uploads/{upload_id}/status
Returns status for idempotency and UX.

Response 200:
```json
{
  "upload_id": "uuid-v4",
  "status": "writing|committed|failed|not_found",
  "server_sha256": null,
  "final_relpath": null,
  "bytes_received": 0
}
```

### POST /api/portals/{portal_id}/close
Closes a portal explicitly. Portals otherwise close when the open window expires.

Response 200:
```json
{ "status": "closed" }
```

Errors:
- 404 portal invalid
- 409 active uploads present AND close requested (implementation may mark closing_requested instead)

## Control Service Endpoints (CLI-only)

### POST /api/control/portals
Creates a portal bound to a destination directory.

Request:
```json
{
  "dest_abs": "/srv/media/incoming",
  "open_minutes": 15,
  "reusable": false,
  "default_policy": "overwrite",
  "autorename_on_conflict": true
}
```

Response 200:
```json
{
  "portal_id": "p_...",
  "expires_at": "RFC3339 timestamp"
}
```

Notes:
- Control service must bind to loopback only.
- This endpoint MUST NOT be exposed via Caddy.

### POST /api/control/portals/{portal_id}/close
Administrative close, if needed.

### GET /api/control/health
Returns 200 if control service is running.
