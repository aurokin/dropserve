
# File I/O and Cleanup Specification

This is the source of truth for file safety and cleanup.

## Core invariants (MUST)

1. The server MUST NOT write partial uploads to the final destination path.
2. The server MUST stream uploads to disk without buffering whole files in memory.
3. The final destination file MUST only appear when the upload is complete and verified.
4. On any upload failure/cancel, the server MUST delete partial temp artifacts as soon as possible.
5. On startup and periodically, the server MUST sweep and delete orphaned temp artifacts.

## Temp layout

For destination directory `DEST` and portal `P`:

- Portal temp root:
  - `DEST/.dropserve_tmp/P/`

- Upload temp files:
  - `DEST/.dropserve_tmp/P/uploads/{upload_id}.part`

- Upload metadata:
  - `DEST/.dropserve_tmp/P/uploads/{upload_id}.json`

The temp layout is inside `DEST` to ensure the commit step can use an atomic rename within a single filesystem.

## Upload write algorithm (per file)

1. On init:
   - Create portal temp root if needed.
   - Create metadata sidecar `{upload_id}.json` (contains portal_id, relpath, size, policy, timestamps).

2. On PUT stream:
   - Open `{upload_id}.part` for write (create new; truncate if exists only when safe).
   - Stream request body -> temp file.
   - While streaming:
     - count bytes written
     - compute server SHA-256
   - If stream errors or is canceled:
     - close file handle
     - delete `.part` and `.json`
     - mark upload as failed

3. Verification:
   - bytes_written MUST equal expected size
   - if client_sha256 is present, it MUST match server hash

4. Resolve final destination name:
   - If policy == overwrite: final_relpath = relpath
   - If policy == autorename and final exists: compute renamed relpath per rules below

5. Commit:
   - Ensure parent directories exist
   - Atomically rename `.part` -> final_abs_path
   - Delete `.json` metadata

## Auto-rename rule (date/time, only when necessary)

When a conflict exists at the final destination path:

Given `name.ext`:
- Try: `name_YYYY-MM-DD_HHMMSS.ext`
- If still exists (same-second multiple):
  - `name_YYYY-MM-DD_HHMMSS_2.ext`
  - `name_YYYY-MM-DD_HHMMSS_3.ext`
  - etc.

Rule: Keep the original name whenever possible.

## Cleanup strategy

### Immediate cleanup (request-level)

On any PUT failure:
- delete `.part` and `.json`
- decrement portal active_uploads

### Portal close/expire cleanup

When a portal closes or expires:
- delete `DEST/.dropserve_tmp/P/` entirely

### Startup + periodic sweeper (crash-proof)

On server startup:
- scan for `DEST/**/.dropserve_tmp/*` or track known destinations
- remove any portal temp dirs that are:
  - expired/closed, or
  - have no corresponding in-memory portal record, or
  - have been idle past a threshold (configurable)

Because resumability is NOT required, the sweeper may be aggressive.

Recommended defaults:
- sweep interval: 2 minutes
- delete `.part` older than: 10 minutes (if not active)
- delete portal temp dirs idle older than: 30 minutes

## Exactly-once / idempotency

- Client supplies `upload_id` (UUID) per file.
- Server stores status for each upload_id:
  - writing/committed/failed
- If a PUT is retried after an uncertain failure:
  - client checks `GET /api/uploads/{upload_id}/status`
  - if committed, do not re-upload
