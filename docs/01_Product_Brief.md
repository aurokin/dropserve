
# Product Brief

## Summary

DropServe is a self-hosted LAN tool that lets you quickly open an upload “portal” to a specific server directory, then upload files/folders from any browser on the LAN.

The upload destination is chosen on the server by the CLI (run inside the destination directory). The browser never sends or chooses an absolute server path.

## Primary goals

1. **“Just works” LAN uploads** for desktop browsers.
2. **No partial files in final destination**: incomplete uploads must never appear at the final path.
3. **Aggressive cleanup**: incomplete temp files should be removed immediately when possible, and always removed on restart via a sweeper.
4. **Short-lived, one-time portals by default**:
   - Default open window: 15 minutes
   - Portal closes when used or expires
   - If an upload is active when the timer ends, portal stays open until the transfer completes
5. **Upload queue UX**:
   - Drag & drop multiple files and folders
   - Upload sequentially (v1) with overall progress and transfer speed display

## Non-goals (v1)

- Resumable uploads (tus) or partial resume
- Mobile browser support
- Preserving permissions, ownership, timestamps, extended attributes
- Sync/backup semantics; this is an on-demand transfer tool

## Key constraints/assumptions

- OS: Ubuntu LTS
- Reverse proxy: **Caddy only** (ports 80/443)
- App binds to localhost on a high port (e.g., 127.0.0.1:8080) and is proxied by Caddy
- Typical uploads < 100MB; outliers up to 50GB
- Folder uploads: hundreds of files, not thousands
- Storage is local disk (no network mounts assumed)
- Security posture: “LAN only” (no accounts); portal capability links are unguessable; portals are short-lived

## Success criteria

- Uploading a folder with hundreds of files succeeds reliably.
- Canceling a transfer leaves **no incomplete files** in the destination directory.
- If the server crashes mid-upload, only temp artifacts exist and are cleaned on the next start.
- Overwrites warn the user; auto-rename creates a timestamp suffix only when needed.
