# PRD: DropServe LAN Upload Portal

## Introduction

DropServe is a self-hosted LAN-only web app and CLI that lets a server operator open a short-lived upload portal tied to a specific destination directory. Users on the same LAN can drag and drop files or folders in a browser, with the server guaranteeing that incomplete files never appear in the final destination and that temporary artifacts are aggressively cleaned up.

## Goals

- Provide a “just works” LAN upload experience for desktop browsers.
- Ensure no partial files appear in destination directories.
- Clean up incomplete uploads immediately and on startup via a sweeper.
- Support short-lived, one-time portals by default with safe expiration behavior.
- Deliver a reliable sequential upload UX with progress and speed feedback.

## User Stories

### US-001: Open a portal from the CLI
**Description:** As a server operator, I want to open a portal from the destination directory so that uploads land in the correct location without exposing absolute paths to browsers.

**Acceptance Criteria:**
- [ ] `dropserve open` uses the canonical current directory as `dest_abs`.
- [ ] CLI calls `POST /api/control/portals` and prints `http://{lan-ip}/p/{portal_id}`.
- [ ] CLI supports `--minutes`, `--reusable`, and `--policy` flags.
- [ ] Control endpoints are blocked by the proxy.

### US-002: Claim a portal from the browser
**Description:** As a browser user, I want to claim a portal so that I can upload files securely with a client token.

**Acceptance Criteria:**
- [ ] `POST /api/portals/{portal_id}/claim` returns `client_token` and policy defaults.
- [ ] One-time portals reject subsequent claims with HTTP 409.
- [ ] State-changing endpoints require `X-Client-Token` after claim.

### US-003: Upload files with sequential progress
**Description:** As a browser user, I want to upload a queue of files sequentially so that large transfers are reliable and I can see overall progress.

**Acceptance Criteria:**
- [ ] UI queues multiple files/folders and uploads one at a time.
- [ ] UI displays total bytes, bytes uploaded, and rolling transfer speed.
- [ ] UI calls init then PUT for each file using the public API.
- [ ] Verify in browser using dev-browser skill.

### US-004: Upload folders with preserved structure
**Description:** As a browser user, I want to upload folders so that nested structure is preserved in the destination directory.

**Acceptance Criteria:**
- [ ] UI accepts drag-and-drop files and folder selection via `webkitdirectory`.
- [ ] The client uses `webkitRelativePath` when available.
- [ ] Empty directories are not uploaded.
- [ ] Verify in browser using dev-browser skill.

### US-005: Prevent path traversal
**Description:** As a system owner, I want all user-supplied paths sanitized so that uploads never escape the destination directory.

**Acceptance Criteria:**
- [ ] Paths containing `..`, absolute prefixes, or Windows drive letters are rejected.
- [ ] Backslashes are normalized to forward slashes before validation.
- [ ] Final absolute path is verified to be contained within `dest_abs`.

### US-006: Stream uploads safely to disk
**Description:** As a system owner, I want uploads streamed to disk with verification so that large files are handled safely and consistently.

**Acceptance Criteria:**
- [ ] `PUT /api/uploads/{upload_id}` streams bytes without buffering entire files.
- [ ] Server verifies `Content-Length` and optional client SHA-256.
- [ ] Uploads only appear in the final destination after atomic rename.
- [ ] Server returns `server_sha256` and `bytes_received` on commit.

### US-007: Resolve filename conflicts
**Description:** As a browser user, I want clear overwrite behavior and automatic renaming for conflicts so I can choose how collisions are handled.

**Acceptance Criteria:**
- [ ] Preflight reports existing conflicts before uploads begin.
- [ ] UI shows warning and toggle for autorename policy.
- [ ] Autorename uses `name_YYYY-MM-DD_HHMMSS.ext` and adds `_2`, `_3` when needed.
- [ ] Verify in browser using dev-browser skill.

### US-008: Cleanup incomplete uploads
**Description:** As a system owner, I want immediate cleanup on failed or canceled uploads so that no partial files remain.

**Acceptance Criteria:**
- [ ] `.part` and `.json` files are deleted on any upload failure or cancel.
- [ ] Final destination never contains partial files.
- [ ] Portal `active_uploads` is decremented on all error paths.

### US-009: Sweep orphaned temp artifacts
**Description:** As a system owner, I want a startup and periodic sweeper so that crashed uploads are removed automatically.

**Acceptance Criteria:**
- [ ] Startup sweep removes stale portal temp directories and old `.part` files.
- [ ] Periodic sweep runs on a configurable interval.
- [ ] Sweeper never deletes committed files.

### US-010: Expire portals safely
**Description:** As a server operator, I want portals to expire without interrupting active uploads so users can finish transfers safely.

**Acceptance Criteria:**
- [ ] Unused portals expire after `open_until` and are cleaned.
- [ ] Used portals remain valid until `active_uploads` drains to zero.
- [ ] One-time portals close after the queue completes.

### US-011: Deploy behind Caddy on LAN
**Description:** As a server operator, I want a simple Caddy configuration so that the app is LAN-only and safe.

**Acceptance Criteria:**
- [ ] Caddyfile enforces LAN-only access using private IP ranges.
- [ ] Public service is proxied to `127.0.0.1:8080`.
- [ ] Control endpoints are not proxied.

## Functional Requirements

- FR-1: The system must expose an HTTP service on `0.0.0.0:8080` (configurable).
- FR-2: The control endpoints must live under `/api/control/*` and never be proxied.
- FR-3: The CLI must create portals via `POST /api/control/portals` and print a LAN URL.
- FR-4: The public API must implement portal claim, preflight, upload init, upload PUT, upload status, and close endpoints.
- FR-5: Uploads must stream to `.dropserve_tmp/{portal_id}/uploads/{upload_id}.part` and commit via atomic rename.
- FR-6: The server must enforce relpath sanitization and containment checks before any write.
- FR-7: The server must delete temp artifacts on any failure, cancellation, close, or expiration.
- FR-8: The server must sweep temp artifacts on startup and on a periodic interval.
- FR-9: The portal state machine must follow OPEN → CLAIMED → IN_USE → CLOSING → CLOSED/EXPIRED rules.
- FR-10: The web UI must support drag-and-drop, folder selection, sequential uploads, and progress/speed display.
- FR-11: The web UI must call preflight and allow overwrite vs autorename policy selection.
- FR-12: Caddy must enforce LAN-only access and proxy only the public endpoints.
- FR-13: The server must serve the web UI from embedded build assets produced from `web/`.
- FR-14: HTTPS via Caddy internal CA is the default; configuration must allow HTTP-only operation and disabling HTTPS-only features.

## Non-Goals (Out of Scope)

- No resumable uploads or partial resume protocol in v1.
- No mobile browser support in v1.
- No preservation of permissions, ownership, timestamps, or extended attributes.
- No sync/backup or continuous transfer semantics.
- No account system or internet-facing access.
- No NGINX or alternative reverse proxies.

## Design Considerations

- Landing page (`/`) explains how to run `dropserve open`.
- Portal page (`/p/{portal_id}`) shows status, queue, progress, speed, and conflict warnings.
- Desktop-first layout with clear drag-and-drop affordances.
- Provide a simple toggle for autorename when conflicts are detected.

## Technical Considerations

- OS target: Ubuntu LTS.
- Implementation language: Go for server and CLI.
- Repo layout: `cmd/` (CLI/server entrypoint), `internal/` (server/CLI packages), `web/` (React UI).
- Web UI stack: React + Vite + TypeScript, managed with Bun.
- Vite builds static assets that are embedded into the Go server binary.
- Default deployment uses HTTPS via Caddy internal CA; configuration must allow HTTP-only operation and disabling HTTPS-only features.
- Typical uploads are <100MB, but outliers up to 50GB must be supported.
- Folder uploads may include hundreds of files; sequential uploads minimize complexity.
- Control endpoints live under `/api/control/*` and should be blocked by Caddy.
- Temp directory lives inside destination for atomic rename on commit.
- SHA-256 is computed during streaming for verification and response payloads.
- Portal IDs and client tokens must be high-entropy and unguessable.
- Configurable sweep intervals and age thresholds are required.
- License: MIT.

## Success Metrics

- Uploading a folder with 100–300 files succeeds reliably.
- Canceling a large upload leaves no `.part` files in the destination.
- Restart after a crash removes temp artifacts within one sweep interval.
- Users can complete a typical upload without refreshing the portal page.

## Open Questions

- None.
