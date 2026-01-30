
# Web UI Specification (v1, desktop-first)

## Pages

### Landing (`/`)
- Explains: “Run `dropserve open` on the server in the destination directory.”
- May show example commands.

### Portal upload page (`/p/{portal_id}`)
- Claims portal
- Shows:
  - portal status (active / expired / closing)
  - queue list
  - overall progress
  - transfer speed
  - collision warnings (preflight)

## Inputs: files and folders

- Provide two ways to add items:
  1. Drag and drop files
  2. “Select folder” using `<input type="file" webkitdirectory>` (recommended)

- Folder structure is preserved using each file’s `webkitRelativePath`.
- Empty directories are not uploaded; only directories that contain files will be recreated server-side.

## Upload queue behavior

- Queue can accept multiple drops/selections; items append.
- Upload strategy v1:
  - sequential (1 at a time) to maximize reliability and simplify cleanup.
  - optional small concurrency (2) if desired later.

- Overall progress:
  - total bytes of all files
  - bytes uploaded so far
- Speed:
  - rolling average over last 1–3 seconds
  - display as MB/s or Mbps

## Preflight collision warning

Before the first upload begins:
- UI calls `/api/portals/{portal_id}/preflight` with relpaths and sizes.
- If conflicts exist:
  - Show warning: “N files already exist and will be overwritten.”
  - Provide toggle: “Auto-rename conflicts instead”
  - Default remains overwrite (with warning), as decided in Product Brief.

## Upload protocol (client)

Per file:

1. POST init:
   - `/api/portals/{portal_id}/uploads` with `upload_id`, relpath, size, and policy

2. PUT bytes:
   - `PUT /api/uploads/{upload_id}`
   - Use `XMLHttpRequest` to get upload progress events.

3. On error:
   - Mark file as failed
   - Allow “retry” (restart from scratch)
   - If uncertainty, call `/api/uploads/{upload_id}/status` before retrying

## Close behavior

- For one-time portals:
  - after queue finishes, UI calls `/api/portals/{portal_id}/close` (optional; server may auto-close)
- UI should show a “Portal closed” message after completion.
