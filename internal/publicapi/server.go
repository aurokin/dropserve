package publicapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"dropserve/internal/config"
	"dropserve/internal/control"
	"dropserve/internal/pathsafe"
	"dropserve/internal/webassets"
)

type Server struct {
	store       *control.Store
	logger      *log.Logger
	tempDirName string
	assets      fs.FS
	indexHTML   []byte
}

type errorResponse struct {
	Error string `json:"error"`
}

type ClaimPortalResponse struct {
	PortalID    string      `json:"portal_id"`
	ClientToken string      `json:"client_token"`
	ExpiresAt   string      `json:"expires_at"`
	Policy      ClaimPolicy `json:"policy"`
	Reusable    bool        `json:"reusable"`
}

type PortalInfoResponse struct {
	PortalID  string      `json:"portal_id"`
	ExpiresAt string      `json:"expires_at"`
	Policy    ClaimPolicy `json:"policy"`
	Reusable  bool        `json:"reusable"`
}

type ClosePortalResponse struct {
	Status string `json:"status"`
}

type ClaimPolicy struct {
	Overwrite  bool `json:"overwrite"`
	Autorename bool `json:"autorename"`
}

type InitUploadRequest struct {
	UploadID     string  `json:"upload_id"`
	Relpath      string  `json:"relpath"`
	Size         int64   `json:"size"`
	ClientSHA256 *string `json:"client_sha256"`
	Policy       string  `json:"policy"`
}

type InitUploadResponse struct {
	UploadID string `json:"upload_id"`
	PutURL   string `json:"put_url"`
}

type PreflightItem struct {
	Relpath string `json:"relpath"`
	Size    int64  `json:"size"`
}

type PreflightRequest struct {
	Items []PreflightItem `json:"items"`
}

type PreflightConflict struct {
	Relpath string `json:"relpath"`
	Reason  string `json:"reason"`
}

type PreflightResponse struct {
	TotalFiles int                 `json:"total_files"`
	TotalBytes int64               `json:"total_bytes"`
	Conflicts  []PreflightConflict `json:"conflicts"`
}

type UploadCommitResponse struct {
	Status        string `json:"status"`
	Relpath       string `json:"relpath"`
	ServerSHA256  string `json:"server_sha256"`
	BytesReceived int64  `json:"bytes_received"`
	FinalRelpath  string `json:"final_relpath"`
}

type UploadStatusResponse struct {
	UploadID      string  `json:"upload_id"`
	Status        string  `json:"status"`
	ServerSHA256  *string `json:"server_sha256"`
	FinalRelpath  *string `json:"final_relpath"`
	BytesReceived int64   `json:"bytes_received"`
}

type requestIDKey struct{}

const landingPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>DropServe</title>
  <style>
    :root {
      color-scheme: light;
    }
    body {
      margin: 0;
      font-family: "Inter", "Segoe UI", system-ui, -apple-system, sans-serif;
      background: #f5f7fb;
      color: #1a1d21;
    }
    main {
      max-width: 720px;
      margin: 48px auto;
      padding: 0 20px;
    }
    .card {
      background: #ffffff;
      border-radius: 16px;
      padding: 28px;
      box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
    }
    h1 {
      margin: 0 0 12px;
      font-size: 32px;
    }
    p {
      margin: 0 0 12px;
      color: #475467;
      line-height: 1.5;
    }
    code {
      background: #eef2f6;
      padding: 2px 6px;
      border-radius: 6px;
      font-family: "SFMono-Regular", ui-monospace, monospace;
      font-size: 0.95em;
    }
    ol {
      margin: 16px 0 0 20px;
      padding: 0;
      color: #344054;
    }
    li {
      margin-bottom: 8px;
    }
  </style>
</head>
<body>
  <main>
    <div class="card">
      <h1>DropServe</h1>
      <p>Open a portal from your server terminal and share the URL with someone on your LAN.</p>
      <ol>
        <li>Run <code>dropserve open</code> inside the destination folder.</li>
        <li>Copy the URL shown in the CLI.</li>
        <li>Open that URL in a LAN browser to upload files.</li>
      </ol>
    </div>
  </main>
</body>
</html>`

const portalPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>DropServe Portal</title>
  <style>
    :root {
      color-scheme: light;
    }
    body {
      margin: 0;
      font-family: "Inter", "Segoe UI", system-ui, -apple-system, sans-serif;
      background: #f5f7fb;
      color: #1a1d21;
    }
    main {
      max-width: 960px;
      margin: 40px auto;
      padding: 0 20px 60px;
      display: grid;
      gap: 20px;
    }
    .card {
      background: #ffffff;
      border-radius: 16px;
      padding: 24px;
      box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
    }
    h1 {
      margin: 0 0 4px;
      font-size: 28px;
    }
    .muted {
      color: #475467;
      font-size: 14px;
    }
    .status {
      margin-top: 8px;
      font-size: 14px;
      font-weight: 600;
    }
    .status[data-tone="error"] {
      color: #b42318;
    }
    .status[data-tone="ok"] {
      color: #027a48;
    }
    .status[data-tone="info"] {
      color: #364152;
    }
    .controls {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
      align-items: center;
      margin-top: 16px;
    }
    .button {
      border: none;
      background: #2e90fa;
      color: #fff;
      padding: 10px 16px;
      border-radius: 10px;
      font-weight: 600;
      cursor: pointer;
    }
    .button:disabled {
      background: #94c5fd;
      cursor: not-allowed;
    }
    .drop-zone {
      border: 2px dashed #cbd5e1;
      border-radius: 14px;
      padding: 20px;
      text-align: center;
      color: #667085;
      background: #f8fafc;
    }
    .drop-zone.dragging {
      border-color: #2e90fa;
      color: #1d4ed8;
      background: #eff6ff;
    }
    .drop-zone.disabled {
      opacity: 0.6;
    }
    .stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 12px;
      margin-top: 12px;
    }
    .stat {
      background: #f8fafc;
      border-radius: 12px;
      padding: 12px 14px;
    }
    .stat-label {
      color: #667085;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }
    .stat-value {
      margin-top: 4px;
      font-size: 18px;
      font-weight: 600;
    }
    .queue {
      display: grid;
      gap: 8px;
      margin-top: 12px;
    }
    .queue-row {
      display: flex;
      justify-content: space-between;
      gap: 16px;
      border: 1px solid #e2e8f0;
      border-radius: 10px;
      padding: 10px 12px;
      font-size: 14px;
      background: #ffffff;
    }
    .queue-name {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      flex: 1;
    }
    .queue-status {
      color: #475467;
      min-width: 120px;
      text-align: right;
      font-weight: 600;
    }
    .conflict-panel {
      margin-top: 16px;
      border-radius: 12px;
      border: 1px solid #fecdca;
      background: #fff4ed;
      padding: 12px 14px;
      color: #7a271a;
    }
    .conflict-title {
      font-weight: 600;
      margin-bottom: 6px;
    }
    .conflict-message {
      font-size: 14px;
      color: #7a271a;
    }
    .conflict-toggle {
      display: flex;
      gap: 8px;
      align-items: center;
      margin-top: 8px;
      font-size: 14px;
      color: #5c2a1b;
    }
    .conflict-toggle input {
      accent-color: #2e90fa;
    }
    .hidden {
      display: none;
    }
  </style>
</head>
<body>
  <main>
    <section class="card">
      <h1>Upload Portal</h1>
      <div class="muted">Portal ID: <span id="portal-id">...</span></div>
      <div class="status" id="portal-status" data-tone="info">Claiming portal...</div>
      <div class="controls">
        <input id="file-input" type="file" multiple>
        <input id="folder-input" type="file" webkitdirectory directory>
        <button id="start-upload" class="button" disabled>Start upload</button>
      </div>
      <div id="drop-zone" class="drop-zone">Drop files or folders here</div>
      <div id="conflict-panel" class="conflict-panel hidden">
        <div class="conflict-title">Filename conflicts detected</div>
        <div class="conflict-message" id="conflict-message"></div>
        <label class="conflict-toggle">
          <input type="checkbox" id="autorename-toggle">
          Auto-rename conflicts instead
        </label>
      </div>
      <div class="stats">
        <div class="stat">
          <div class="stat-label">Files queued</div>
          <div class="stat-value" id="file-count">0</div>
        </div>
        <div class="stat">
          <div class="stat-label">Total bytes</div>
          <div class="stat-value" id="total-bytes">0 B</div>
        </div>
        <div class="stat">
          <div class="stat-label">Bytes uploaded</div>
          <div class="stat-value" id="uploaded-bytes">0 B</div>
        </div>
        <div class="stat">
          <div class="stat-label">Rolling speed</div>
          <div class="stat-value" id="speed">0 B/s</div>
        </div>
      </div>
    </section>
    <section class="card">
      <h2>Queue</h2>
      <div class="queue" id="queue"></div>
    </section>
  </main>
  <script>
    (function () {
      const portalId = getPortalId();
      const portalLabel = document.getElementById("portal-id");
      const statusEl = document.getElementById("portal-status");
      const fileInput = document.getElementById("file-input");
      const folderInput = document.getElementById("folder-input");
      const startButton = document.getElementById("start-upload");
      const dropZone = document.getElementById("drop-zone");
      const queueEl = document.getElementById("queue");
      const totalEl = document.getElementById("total-bytes");
      const uploadedEl = document.getElementById("uploaded-bytes");
      const speedEl = document.getElementById("speed");
      const fileCountEl = document.getElementById("file-count");
      const conflictPanel = document.getElementById("conflict-panel");
      const conflictMessage = document.getElementById("conflict-message");
      const autorenameToggle = document.getElementById("autorename-toggle");

      const state = {
        portalId: portalId,
        clientToken: "",
        defaultPolicy: "overwrite",
        claimed: false,
        queue: [],
        conflicts: [],
        running: false,
        totalBytes: 0,
        completedBytes: 0,
        currentLoaded: 0,
        uploadedBytes: 0,
        lastSpeedBytes: 0,
        lastSpeedTime: 0,
        speedBps: 0,
        speedTimer: null
      };

      portalLabel.textContent = portalId || "unknown";

      function getPortalId() {
        const parts = window.location.pathname.split("/").filter(Boolean);
        if (parts.length < 2) {
          return "";
        }
        return parts[1];
      }

      function setStatus(message, tone) {
        statusEl.textContent = message;
        statusEl.dataset.tone = tone || "info";
      }

      function formatBytes(bytes) {
        if (!Number.isFinite(bytes)) {
          return "0 B";
        }
        const units = ["B", "KB", "MB", "GB", "TB"];
        let value = bytes;
        let unitIndex = 0;
        while (value >= 1024 && unitIndex < units.length - 1) {
          value /= 1024;
          unitIndex += 1;
        }
        const digits = value >= 10 || unitIndex === 0 ? 0 : 1;
        return value.toFixed(digits) + " " + units[unitIndex];
      }

      function updateSummary() {
        totalEl.textContent = formatBytes(state.totalBytes);
        uploadedEl.textContent = formatBytes(state.uploadedBytes);
        speedEl.textContent = formatBytes(state.speedBps) + "/s";
        fileCountEl.textContent = String(state.queue.length);
      }

      function updateControls() {
        const canStart = state.claimed && state.queue.length > 0 && !state.running;
        startButton.disabled = !canStart;
        fileInput.disabled = !state.claimed || state.running;
        folderInput.disabled = !state.claimed || state.running;
        dropZone.classList.toggle("disabled", !state.claimed || state.running);
        if (autorenameToggle) {
          autorenameToggle.disabled = !state.claimed || state.running || state.conflicts.length === 0;
        }
      }

      function startSpeedTimer() {
        if (state.speedTimer) {
          return;
        }
        state.lastSpeedBytes = state.uploadedBytes;
        state.lastSpeedTime = performance.now();
        state.speedTimer = setInterval(() => {
          const now = performance.now();
          const deltaBytes = state.uploadedBytes - state.lastSpeedBytes;
          const deltaTime = (now - state.lastSpeedTime) / 1000;
          if (deltaTime > 0) {
            state.speedBps = Math.max(0, deltaBytes / deltaTime);
          }
          state.lastSpeedBytes = state.uploadedBytes;
          state.lastSpeedTime = now;
          updateSummary();
        }, 500);
      }

      function stopSpeedTimer() {
        if (!state.speedTimer) {
          return;
        }
        clearInterval(state.speedTimer);
        state.speedTimer = null;
        state.speedBps = 0;
        updateSummary();
      }

      function updateQueueItem(item, statusText) {
        item.status = statusText;
        item.statusEl.textContent = statusText;
      }

      function updateConflictPanel(conflicts) {
        const count = conflicts.length;
        state.conflicts = conflicts;
        if (!conflictPanel || !conflictMessage || !autorenameToggle) {
          updateControls();
          return;
        }
        if (count === 0) {
          conflictPanel.classList.add("hidden");
          conflictMessage.textContent = "";
          updateControls();
          return;
        }
        const verb = state.defaultPolicy === "autorename" ? "auto-renamed" : "overwritten";
        const label = count === 1 ? "file" : "files";
        conflictMessage.textContent = count + " " + label + " already exist and will be " + verb + ".";
        autorenameToggle.checked = state.defaultPolicy === "autorename";
        conflictPanel.classList.remove("hidden");
        updateControls();
      }

      async function runPreflight(showError) {
        if (!state.claimed || state.running || state.queue.length === 0) {
          updateConflictPanel([]);
          return true;
        }
        const payload = {
          items: state.queue.map((item) => ({
            relpath: item.relpath,
            size: item.file.size
          }))
        };
        try {
          const response = await fetch("/api/portals/" + state.portalId + "/preflight", {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "X-Client-Token": state.clientToken
            },
            body: JSON.stringify(payload)
          });
          if (!response.ok) {
            if (showError) {
              const message = await readError(response);
              setStatus("Preflight failed: " + message, "error");
            }
            return false;
          }
          const data = await response.json();
          const conflicts = Array.isArray(data.conflicts) ? data.conflicts : [];
          updateConflictPanel(conflicts);
          return true;
        } catch (error) {
          if (showError) {
            setStatus("Preflight failed.", "error");
          }
          return false;
        }
      }

      function normalizeQueueItems(items) {
        return Array.from(items || [])
          .map((item) => {
            if (!item) {
              return null;
            }
            if (item.file) {
              return item;
            }
            return { file: item, relpath: item.webkitRelativePath || item.name };
          })
          .filter(Boolean);
      }

      function addQueueItems(items) {
        const queueItems = normalizeQueueItems(items);
        queueItems.forEach((item) => {
          const file = item.file;
          if (!file) {
            return;
          }
          const relpath = item.relpath || file.webkitRelativePath || file.name;
          const row = document.createElement("div");
          row.className = "queue-row";
          const nameEl = document.createElement("div");
          nameEl.className = "queue-name";
          nameEl.textContent = relpath;
          const statusEl = document.createElement("div");
          statusEl.className = "queue-status";
          statusEl.textContent = "queued";
          row.appendChild(nameEl);
          row.appendChild(statusEl);
          queueEl.appendChild(row);
          state.queue.push({
            file: file,
            relpath: relpath,
            status: "queued",
            rowEl: row,
            statusEl: statusEl
          });
          state.totalBytes += file.size;
        });
        updateSummary();
        updateControls();
        runPreflight(false);
      }

      function stripLeadingSlash(value) {
        if (!value) {
          return "";
        }
        return value.replace(/^\/+/, "");
      }

      function readDirectoryEntries(reader) {
        return new Promise((resolve) => {
          const entries = [];
          const readBatch = () => {
            reader.readEntries((batch) => {
              if (!batch.length) {
                resolve(entries);
                return;
              }
              entries.push(...batch);
              readBatch();
            }, () => resolve(entries));
          };
          readBatch();
        });
      }

      async function readEntryFiles(entry) {
        if (!entry) {
          return [];
        }
        if (entry.isFile) {
          return new Promise((resolve) => {
            entry.file((file) => {
              const relpath = stripLeadingSlash(entry.fullPath) || file.webkitRelativePath || file.name;
              resolve([{ file: file, relpath: relpath }]);
            }, () => resolve([]));
          });
        }
        if (entry.isDirectory) {
          const reader = entry.createReader();
          const entries = await readDirectoryEntries(reader);
          const files = [];
          for (const child of entries) {
            const childFiles = await readEntryFiles(child);
            files.push(...childFiles);
          }
          return files;
        }
        return [];
      }

      async function collectDropItems(dataTransfer) {
        const items = Array.from((dataTransfer && dataTransfer.items) || []);
        if (items.length === 0) {
          return normalizeQueueItems(dataTransfer ? dataTransfer.files : []);
        }
        const files = [];
        for (const item of items) {
          if (item.kind !== "file") {
            continue;
          }
          if (item.webkitGetAsEntry) {
            const entry = item.webkitGetAsEntry();
            if (entry) {
              const entryFiles = await readEntryFiles(entry);
              files.push(...entryFiles);
              continue;
            }
          }
          const file = item.getAsFile ? item.getAsFile() : null;
          if (file) {
            files.push({
              file: file,
              relpath: file.webkitRelativePath || file.name
            });
          }
        }
        return files;
      }

      function makeUploadID() {
        if (window.crypto && window.crypto.randomUUID) {
          return window.crypto.randomUUID();
        }
        return "u_" + Math.random().toString(16).slice(2) + Date.now().toString(16);
      }

      async function claimPortal() {
        if (!state.portalId) {
          setStatus("Invalid portal URL.", "error");
          return;
        }
        setStatus("Claiming portal...", "info");
        try {
          const response = await fetch("/api/portals/" + state.portalId + "/claim", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: "{}"
          });
          if (!response.ok) {
            const message = await readError(response);
            setStatus(message, "error");
            return;
          }
          const data = await response.json();
          state.clientToken = data.client_token;
          if (data.policy && data.policy.autorename) {
            state.defaultPolicy = "autorename";
          } else {
            state.defaultPolicy = "overwrite";
          }
          state.claimed = true;
          setStatus("Portal ready. Add files to upload.", "ok");
          updateControls();
          runPreflight(false);
        } catch (error) {
          setStatus("Failed to claim portal.", "error");
        }
      }

      async function readError(response) {
        try {
          const data = await response.json();
          if (data && data.error) {
            return data.error;
          }
        } catch (error) {
        }
        return response.statusText || "request failed";
      }

      async function initUpload(item) {
        const payload = {
          upload_id: makeUploadID(),
          relpath: item.relpath,
          size: item.file.size,
          client_sha256: null,
          policy: state.defaultPolicy
        };
        const response = await fetch("/api/portals/" + state.portalId + "/uploads", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Client-Token": state.clientToken
          },
          body: JSON.stringify(payload)
        });
        if (!response.ok) {
          const message = await readError(response);
          throw new Error(message);
        }
        return response.json();
      }

      function putUpload(item, putUrl) {
        return new Promise((resolve, reject) => {
          const request = new XMLHttpRequest();
          request.open("PUT", putUrl);
          request.setRequestHeader("X-Client-Token", state.clientToken);
          request.upload.onprogress = (event) => {
            if (!event.lengthComputable) {
              return;
            }
            state.currentLoaded = event.loaded;
            state.uploadedBytes = state.completedBytes + event.loaded;
            updateSummary();
            const percent = item.file.size > 0
              ? Math.round((event.loaded / item.file.size) * 100)
              : 100;
            updateQueueItem(item, "uploading " + percent + "%");
          };
          request.onload = () => {
            if (request.status >= 200 && request.status < 300) {
              resolve(request.responseText);
              return;
            }
            let message = request.statusText || "upload failed";
            try {
              const data = JSON.parse(request.responseText);
              if (data && data.error) {
                message = data.error;
              }
            } catch (error) {
            }
            reject(new Error(message));
          };
          request.onerror = () => reject(new Error("network error"));
          request.send(item.file);
        });
      }

      async function runQueue() {
        if (state.running || !state.claimed || state.queue.length === 0) {
          return;
        }
        const preflightOk = await runPreflight(true);
        if (!preflightOk) {
          updateControls();
          return;
        }
        state.running = true;
        state.completedBytes = 0;
        state.currentLoaded = 0;
        state.uploadedBytes = 0;
        updateSummary();
        updateControls();
        startSpeedTimer();

        for (let index = 0; index < state.queue.length; index += 1) {
          const item = state.queue[index];
          updateQueueItem(item, "initializing");
          let initResponse;
          try {
            initResponse = await initUpload(item);
          } catch (error) {
            updateQueueItem(item, "failed");
            setStatus("Upload failed: " + error.message, "error");
            state.running = false;
            stopSpeedTimer();
            updateControls();
            return;
          }

          updateQueueItem(item, "uploading 0%");
          try {
            await putUpload(item, initResponse.put_url);
          } catch (error) {
            updateQueueItem(item, "failed");
            setStatus("Upload failed: " + error.message, "error");
            state.running = false;
            stopSpeedTimer();
            updateControls();
            return;
          }

          updateQueueItem(item, "done");
          state.completedBytes += item.file.size;
          state.currentLoaded = 0;
          state.uploadedBytes = state.completedBytes;
          updateSummary();
        }

        state.running = false;
        stopSpeedTimer();
        setStatus("All uploads complete.", "ok");
        updateControls();
      }

      fileInput.addEventListener("change", (event) => {
        if (!state.claimed || state.running) {
          return;
        }
        addQueueItems(event.target.files);
        event.target.value = "";
      });

      folderInput.addEventListener("change", (event) => {
        if (!state.claimed || state.running) {
          return;
        }
        addQueueItems(event.target.files);
        event.target.value = "";
      });

      startButton.addEventListener("click", () => {
        runQueue();
      });

      if (autorenameToggle) {
        autorenameToggle.addEventListener("change", () => {
          state.defaultPolicy = autorenameToggle.checked ? "autorename" : "overwrite";
          updateConflictPanel(state.conflicts);
        });
      }

      dropZone.addEventListener("dragover", (event) => {
        if (!state.claimed || state.running) {
          return;
        }
        event.preventDefault();
        dropZone.classList.add("dragging");
      });

      dropZone.addEventListener("dragleave", () => {
        dropZone.classList.remove("dragging");
      });

      dropZone.addEventListener("drop", async (event) => {
        if (!state.claimed || state.running) {
          return;
        }
        event.preventDefault();
        dropZone.classList.remove("dragging");
        const droppedItems = await collectDropItems(event.dataTransfer);
        addQueueItems(droppedItems);
      });

      updateSummary();
      updateControls();
      claimPortal();
    })();
  </script>
</body>
</html>`

const notFoundHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>DropServe | 404</title>
  <link rel="preload" as="image" href="/assets/404.png">
  <style>
    :root {
      color-scheme: dark;
      --ink: #e1ecff;
      --glow: #a8e5ff;
      --deep: #12172e;
      --shade: rgba(7, 9, 19, 0.62);
      --mist: rgba(120, 168, 255, 0.18);
    }
    * {
      box-sizing: border-box;
    }
    html,
    body {
      height: 100%;
    }
    body {
      margin: 0;
      font-family: "Roboto", "Helvetica Neue", Arial, sans-serif;
      background-color: var(--deep);
      color: var(--ink);
      display: grid;
      place-items: center;
      overflow: hidden;
      text-rendering: optimizeLegibility;
      letter-spacing: 0.04em;
    }
    body::before {
      content: "";
      position: fixed;
      inset: -6%;
      background-image: url("/assets/404.png");
      background-size: cover;
      background-position: center;
      image-rendering: -webkit-optimize-contrast;
      filter: saturate(1.06) contrast(1.05) brightness(0.99);
      transform: scale(1.01);
      z-index: 0;
      animation: slow-drift 30s ease-in-out infinite;
    }
    body::after {
      content: "";
      position: fixed;
      inset: 0;
      background:
        radial-gradient(circle at 52% 44%, rgba(168, 229, 255, 0.18), transparent 42%),
        radial-gradient(circle at 18% 70%, rgba(111, 168, 255, 0.2), transparent 55%),
        linear-gradient(180deg, rgba(6, 8, 18, 0.35), rgba(6, 8, 18, 0.55));
      mix-blend-mode: screen;
      opacity: 0.45;
      pointer-events: none;
      z-index: 1;
    }
    main {
      position: relative;
      z-index: 2;
      width: 100%;
      height: 100%;
      display: flex;
      align-items: flex-end;
      justify-content: flex-start;
      padding: clamp(24px, 5vw, 56px);
    }
    .number {
      margin: 0;
      font-size: clamp(72px, 18vw, 240px);
      font-weight: 600;
      color: #f7fbff;
      text-shadow:
        0 12px 40px rgba(9, 14, 32, 0.75),
        0 2px 12px rgba(20, 24, 50, 0.5),
        0 0 40px rgba(168, 229, 255, 0.35);
      -webkit-text-stroke: 1px rgba(10, 20, 40, 0.45);
      letter-spacing: 0.12em;
      mix-blend-mode: normal;
      animation: rise-in 1.4s ease-out both;
    }
    @keyframes slow-drift {
      0% { transform: scale(1.04) translate3d(0, 0, 0); }
      50% { transform: scale(1.06) translate3d(-1.5%, 1%, 0); }
      100% { transform: scale(1.04) translate3d(0, 0, 0); }
    }
    @keyframes rise-in {
      0% { opacity: 0; transform: translate3d(0, 16px, 0); }
      100% { opacity: 1; transform: translate3d(0, 0, 0); }
    }
    @media (prefers-reduced-motion: reduce) {
      body::before {
        animation: none;
      }
      .number {
        animation: none;
      }
    }
  </style>
</head>
<body>
  <main aria-label="404">
    <h1 class="number">404</h1>
  </main>
</body>
</html>`

func NewServer(store *control.Store, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.New(os.Stdout, "public ", log.LstdFlags)
	}

	assets, err := webassets.Dist()
	if err != nil {
		logger.Printf("failed to load web assets: %v", err)
	}
	indexHTML, err := webassets.ReadIndex()
	if err != nil {
		logger.Printf("failed to load index.html: %v", err)
	}

	return &Server{
		store:       store,
		logger:      logger,
		tempDirName: config.TempDirName(),
		assets:      assets,
		indexHTML:   indexHTML,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/assets/", s.assetsHandler())
	mux.Handle("/favicon.svg", s.assetsHandler())
	mux.HandleFunc("/api/portals/", s.handlePortals)
	mux.HandleFunc("/api/uploads/", s.handleUploads)
	mux.HandleFunc("/p/", s.handlePortalPage)
	mux.HandleFunc("/", s.handleLanding)
	return s.withRequestID(mux)
}

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.serveNotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	s.serveIndex(w, r)
}

func (s *Server) handlePortalPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	pathValue := strings.TrimPrefix(r.URL.Path, "/p/")
	if pathValue == r.URL.Path {
		http.NotFound(w, r)
		return
	}

	segments := strings.Split(strings.Trim(pathValue, "/"), "/")
	if len(segments) < 1 || len(segments) > 2 || strings.TrimSpace(segments[0]) == "" {
		s.serveNotFound(w, r)
		return
	}
	if len(segments) == 2 && strings.TrimSpace(segments[1]) != "claimed" {
		s.serveNotFound(w, r)
		return
	}

	portalID := segments[0]
	if _, err := s.store.PortalByID(portalID); err != nil {
		s.serveNotFound(w, r)
		return
	}

	s.serveIndex(w, r)
}

func (s *Server) handlePortals(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/portals/")
	if path == r.URL.Path {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) != 2 {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	portalID := segments[0]
	action := segments[1]

	switch action {
	case "info":
		s.handleInfo(w, r, portalID)
	case "claim":
		s.handleClaim(w, r, portalID)
	case "uploads":
		s.handleInitUpload(w, r, portalID)
	case "preflight":
		s.handlePreflight(w, r, portalID)
	case "close":
		s.handleClose(w, r, portalID)
	default:
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
	}
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request, portalID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	portal, err := s.store.PortalByID(portalID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load portal"})
		}
		return
	}

	if portal.State == control.PortalClosing {
		writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		return
	}

	resp := PortalInfoResponse{
		PortalID:  portal.ID,
		ExpiresAt: portal.OpenUntil.Format(time.RFC3339),
		Policy: ClaimPolicy{
			Overwrite:  portal.DefaultPolicy == "overwrite",
			Autorename: portal.DefaultPolicy == "autorename",
		},
		Reusable: portal.Reusable,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleClaim(w http.ResponseWriter, r *http.Request, portalID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if err := decodeEmptyJSON(r); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	result, err := s.store.ClaimPortal(portalID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalAlreadyClaimed):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "Portal already claimed"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to claim portal"})
		}
		return
	}

	resp := ClaimPortalResponse{
		PortalID:    result.Portal.ID,
		ClientToken: result.ClientToken,
		ExpiresAt:   result.Portal.OpenUntil.Format(time.RFC3339),
		Policy: ClaimPolicy{
			Overwrite:  result.Portal.DefaultPolicy == "overwrite",
			Autorename: result.Portal.DefaultPolicy == "autorename",
		},
		Reusable: result.Portal.Reusable,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePreflight(w http.ResponseWriter, r *http.Request, portalID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	portal, err := s.store.PortalByID(portalID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load portal"})
		}
		return
	}

	if portal.State == control.PortalClosing {
		writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		return
	}

	if !s.requireClientToken(w, r, portalID) {
		return
	}

	var req PreflightRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	totalBytes := int64(0)
	conflicts := make([]PreflightConflict, 0)
	for _, item := range req.Items {
		if item.Size < 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "size must be non-negative"})
			return
		}
		cleanedRelpath, err := pathsafe.SanitizeRelpath(item.Relpath)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid relpath"})
			return
		}
		finalAbs, err := pathsafe.JoinAndVerify(portal.DestAbs, cleanedRelpath)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid relpath"})
			return
		}
		totalBytes += item.Size
		if _, err := os.Stat(finalAbs); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to preflight upload"})
			return
		}
		conflicts = append(conflicts, PreflightConflict{Relpath: cleanedRelpath, Reason: "exists"})
	}

	writeJSON(w, http.StatusOK, PreflightResponse{
		TotalFiles: len(req.Items),
		TotalBytes: totalBytes,
		Conflicts:  conflicts,
	})
}

func (s *Server) handleInitUpload(w http.ResponseWriter, r *http.Request, portalID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	portal, err := s.store.PortalByID(portalID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load portal"})
		}
		return
	}

	if portal.State == control.PortalClosing {
		writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		return
	}

	if !s.requireClientToken(w, r, portalID) {
		return
	}

	var req InitUploadRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	if strings.TrimSpace(req.UploadID) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "upload_id required"})
		return
	}
	if req.Size < 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "size must be non-negative"})
		return
	}

	cleanedRelpath, err := pathsafe.SanitizeRelpath(req.Relpath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid relpath"})
		return
	}
	if _, err := pathsafe.JoinAndVerify(portal.DestAbs, cleanedRelpath); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid relpath"})
		return
	}

	policy := strings.TrimSpace(req.Policy)
	if policy == "" {
		policy = portal.DefaultPolicy
	}
	policy, err = control.NormalizePolicy(policy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	clientSHA := ""
	if req.ClientSHA256 != nil {
		clientSHA = strings.TrimSpace(*req.ClientSHA256)
	}

	if _, err := s.store.CreateUpload(control.CreateUploadInput{
		PortalID:     portal.ID,
		UploadID:     req.UploadID,
		Relpath:      cleanedRelpath,
		Size:         req.Size,
		ClientSHA256: clientSHA,
		Policy:       policy,
	}); err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		case errors.Is(err, control.ErrUploadAlreadyCommitted):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "upload already committed"})
		case errors.Is(err, control.ErrUploadAlreadyExists):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "upload already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to initialize upload"})
		}
		return
	}

	tempDir := s.uploadTempDir(portal.DestAbs, portal.ID)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		s.store.DeleteUpload(req.UploadID)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to prepare upload"})
		return
	}

	_, metaPath := uploadTempPaths(tempDir, req.UploadID)
	meta := uploadMetadata{
		PortalID:     portal.ID,
		UploadID:     req.UploadID,
		Relpath:      cleanedRelpath,
		Size:         req.Size,
		Policy:       policy,
		ClientSHA256: clientSHA,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeUploadMetadata(metaPath, meta); err != nil {
		cleanupUploadArtifacts("", metaPath)
		s.store.DeleteUpload(req.UploadID)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to prepare upload"})
		return
	}

	writeJSON(w, http.StatusOK, InitUploadResponse{
		UploadID: req.UploadID,
		PutURL:   "/api/uploads/" + req.UploadID,
	})
}

func (s *Server) handleClose(w http.ResponseWriter, r *http.Request, portalID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if err := decodeEmptyJSON(r); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	if !s.requireClientToken(w, r, portalID) {
		return
	}

	portal, err := s.store.ClosePortal(portalID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to close portal"})
		}
		return
	}

	if portal.State == control.PortalClosing && portal.ActiveUploads > 0 {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "portal has active uploads"})
		return
	}

	s.cleanupPortalTempDir(portal)

	writeJSON(w, http.StatusOK, ClosePortalResponse{Status: "closed"})
}

func (s *Server) handleUploads(w http.ResponseWriter, r *http.Request) {
	pathValue := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	if pathValue == r.URL.Path {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	segments := strings.Split(strings.Trim(pathValue, "/"), "/")
	if len(segments) == 1 {
		s.handleUploadStream(w, r, segments[0])
		return
	}
	if len(segments) == 2 && segments[1] == "status" {
		s.handleUploadStatus(w, r, segments[0])
		return
	}

	writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
}

func (s *Server) handleUploadStream(w http.ResponseWriter, r *http.Request, uploadID string) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	upload, err := s.store.GetUpload(uploadID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrUploadNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "upload not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load upload"})
		}
		return
	}

	if upload.Status == control.UploadCommitted {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "upload already committed"})
		return
	}

	portal, err := s.store.PortalByID(upload.PortalID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load portal"})
		}
		return
	}

	if !s.requireClientToken(w, r, portal.ID) {
		return
	}

	tempDir := s.uploadTempDir(portal.DestAbs, portal.ID)
	partPath, metaPath := uploadTempPaths(tempDir, uploadID)

	if _, err := s.store.StartUpload(uploadID); err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		case errors.Is(err, control.ErrUploadNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "upload not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to start upload"})
		}
		return
	}

	if r.ContentLength < 0 || r.ContentLength != upload.Size {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "size mismatch"})
		return
	}

	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to prepare upload"})
		return
	}

	file, err := os.OpenFile(partPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to write upload"})
		return
	}
	defer func() {
		_ = file.Close()
	}()
	defer func() {
		_ = r.Body.Close()
	}()

	hasher := sha256.New()
	bytesWritten, err := io.Copy(io.MultiWriter(file, hasher), r.Body)
	if err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to stream upload"})
		return
	}
	if bytesWritten != upload.Size {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "size mismatch"})
		return
	}

	serverSHA := hex.EncodeToString(hasher.Sum(nil))
	if upload.ClientSHA256 != "" && !strings.EqualFold(serverSHA, upload.ClientSHA256) {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "sha256 mismatch"})
		return
	}

	finalRelpath, finalAbs, err := resolveFinalRelpath(portal.DestAbs, upload.Relpath, upload.Policy)
	if err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to finalize upload"})
		return
	}
	if err := os.MkdirAll(filepath.Dir(finalAbs), 0o755); err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to finalize upload"})
		return
	}

	if err := os.Rename(partPath, finalAbs); err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to commit upload"})
		return
	}

	if err := os.Remove(metaPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.logger.Printf("failed to remove metadata: %v", err)
	}

	committed, err := s.store.MarkUploadCommitted(uploadID, serverSHA, finalRelpath, bytesWritten)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to commit upload"})
		return
	}

	writeJSON(w, http.StatusOK, UploadCommitResponse{
		Status:        string(committed.Status),
		Relpath:       committed.Relpath,
		ServerSHA256:  committed.ServerSHA256,
		BytesReceived: committed.BytesReceived,
		FinalRelpath:  committed.FinalRelpath,
	})
}

func (s *Server) handleUploadStatus(w http.ResponseWriter, r *http.Request, uploadID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	upload, err := s.store.GetUpload(uploadID)
	if err != nil {
		if errors.Is(err, control.ErrUploadNotFound) {
			writeJSON(w, http.StatusOK, UploadStatusResponse{
				UploadID:      uploadID,
				Status:        "not_found",
				ServerSHA256:  nil,
				FinalRelpath:  nil,
				BytesReceived: 0,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load upload"})
		return
	}

	var serverSHA *string
	if upload.ServerSHA256 != "" {
		serverSHA = &upload.ServerSHA256
	}
	var finalRelpath *string
	if upload.FinalRelpath != "" {
		finalRelpath = &upload.FinalRelpath
	}

	writeJSON(w, http.StatusOK, UploadStatusResponse{
		UploadID:      upload.ID,
		Status:        string(upload.Status),
		ServerSHA256:  serverSHA,
		FinalRelpath:  finalRelpath,
		BytesReceived: upload.BytesReceived,
	})
}

func (s *Server) assetsHandler() http.Handler {
	if s.assets == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}
	fileServer := http.FileServer(http.FS(s.assets))
	return http.StripPrefix("/", fileServer)
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	if len(s.indexHTML) == 0 && s.assets != nil {
		indexHTML, err := fs.ReadFile(s.assets, "index.html")
		if err == nil {
			s.indexHTML = indexHTML
		}
	}
	if len(s.indexHTML) == 0 {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "web ui not available"})
		return
	}
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(s.indexHTML))
}

func (s *Server) serveNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = io.WriteString(w, notFoundHTML)
}

func (s *Server) failUpload(uploadID, partPath, metaPath string) {
	_, _ = s.store.MarkUploadFailed(uploadID)
	cleanupUploadArtifacts(partPath, metaPath)
}

func (s *Server) requireClientToken(w http.ResponseWriter, r *http.Request, portalID string) bool {
	token := strings.TrimSpace(r.Header.Get("X-Client-Token"))
	if err := s.store.RequireClientToken(portalID, token); err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalClosed):
			writeJSON(w, http.StatusGone, errorResponse{Error: "portal closed"})
		case errors.Is(err, control.ErrClientTokenRequired):
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "client token required"})
		case errors.Is(err, control.ErrClientTokenInvalid):
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "client token invalid"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to validate client token"})
		}
		return false
	}

	return true
}

func decodeEmptyJSON(r *http.Request) error {
	if r.Body == nil {
		return nil
	}

	var payload struct{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if decoder.More() {
		return errors.New("invalid json")
	}

	return nil
}

type uploadMetadata struct {
	PortalID     string `json:"portal_id"`
	UploadID     string `json:"upload_id"`
	Relpath      string `json:"relpath"`
	Size         int64  `json:"size"`
	Policy       string `json:"policy"`
	ClientSHA256 string `json:"client_sha256,omitempty"`
	CreatedAt    string `json:"created_at"`
}

func (s *Server) cleanupPortalTempDir(portal control.Portal) {
	if strings.TrimSpace(portal.DestAbs) == "" {
		return
	}
	portalPath := filepath.Join(portal.DestAbs, s.tempDirName, portal.ID)
	if err := os.RemoveAll(portalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.logger.Printf("failed to remove portal temp dir: %v", err)
	}
}

func (s *Server) uploadTempDir(destAbs, portalID string) string {
	return filepath.Join(destAbs, s.tempDirName, portalID, "uploads")
}

func uploadTempPaths(tempDir, uploadID string) (string, string) {
	partPath := filepath.Join(tempDir, uploadID+".part")
	metaPath := filepath.Join(tempDir, uploadID+".json")
	return partPath, metaPath
}

func writeUploadMetadata(path string, meta uploadMetadata) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := json.NewEncoder(file)
	return encoder.Encode(meta)
}

func cleanupUploadArtifacts(partPath, metaPath string) {
	if partPath != "" {
		_ = os.Remove(partPath)
	}
	if metaPath != "" {
		_ = os.Remove(metaPath)
	}
}

func resolveFinalRelpath(destAbs, relpath, policy string) (string, string, error) {
	finalRelpath := relpath
	finalAbs, err := pathsafe.JoinAndVerify(destAbs, relpath)
	if err != nil {
		return "", "", err
	}

	if policy != "autorename" {
		return finalRelpath, finalAbs, nil
	}

	if _, err := os.Stat(finalAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return finalRelpath, finalAbs, nil
		}
		return "", "", err
	}

	dir, base := path.Split(relpath)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	timestamp := time.Now().Format("2006-01-02_150405")

	for i := 0; ; i++ {
		suffix := ""
		if i > 0 {
			suffix = fmt.Sprintf("_%d", i+1)
		}
		candidate := fmt.Sprintf("%s_%s%s%s", name, timestamp, suffix, ext)
		candidateRelpath := path.Join(dir, candidate)
		candidateAbs, err := pathsafe.JoinAndVerify(destAbs, candidateRelpath)
		if err != nil {
			return "", "", err
		}
		if _, err := os.Stat(candidateAbs); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return candidateRelpath, candidateAbs, nil
			}
			return "", "", err
		}
	}
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = newRequestID()
		}

		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		w.Header().Set("X-Request-Id", requestID)
		s.logger.Printf("request_id=%s method=%s path=%s", requestID, r.Method, r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	return "r_" + time.Now().UTC().Format("20060102T150405.000000000")
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
