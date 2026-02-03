# Web UI (v1, desktop-first)

## Pages

- Landing (`/`): explains how to run `dropserve`.
- Portal upload (`/p/{portal_id}`): main upload experience.

## Inputs

- Drag and drop files.
- "Select folder" using `<input type="file" webkitdirectory>`.
- Preserve folder structure via `webkitRelativePath`.
- Empty directories are not uploaded.

## Upload queue behavior

- Queue accepts multiple drops/selections; items append.
- Uploads are sequential in v1.
- UI shows total bytes, bytes uploaded, and rolling speed.

## Preflight collisions

- Call `/api/portals/{portal_id}/preflight` before upload.
- Warn on conflicts and allow auto-rename toggle.
- Default policy remains overwrite with warning.

## Client upload protocol

Per file:

1. `POST /api/portals/{portal_id}/uploads` init.
2. `PUT /api/uploads/{upload_id}` stream bytes (use XHR for progress).
3. On error, mark failed and allow retry from scratch.

## Close behavior

- UI does not auto-close portals after queue completion.
- Show completion message; portal remains open until expiry.
