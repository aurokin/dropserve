import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";

type StatusTone = "info" | "ok" | "error" | "warn";

type StatusState = {
  message: string;
  tone: StatusTone;
};

type QueueItem = {
  id: string;
  file: File;
  relpath: string;
  status: string;
};

type QueueCandidate = {
  file: File;
  relpath: string;
};

type ClaimPolicy = {
  overwrite: boolean;
  autorename: boolean;
};

type ClaimResponse = {
  portal_id: string;
  client_token: string;
  expires_at: string;
  policy: ClaimPolicy;
  reusable?: boolean;
};

type PreflightConflict = {
  relpath: string;
  reason: string;
};

type DropServeFileSystemEntry = {
  isFile: boolean;
  isDirectory: boolean;
  name: string;
  fullPath: string;
  file: (success: (file: File) => void, error?: (err: DOMException) => void) => void;
  createReader: () => DropServeDirectoryReader;
};

type DropServeDirectoryReader = {
  readEntries: (
    success: (entries: DropServeFileSystemEntry[]) => void,
    error?: (err: DOMException) => void
  ) => void;
};

type DataTransferItemWithEntry = DataTransferItem & {
  webkitGetAsEntry?: () => DropServeFileSystemEntry | null;
};

const fileInputProps = {
  multiple: true
};

const folderInputProps = {
  multiple: true,
  webkitdirectory: "true",
  directory: "true"
} as React.InputHTMLAttributes<HTMLInputElement>;

const defaultStatus: StatusState = {
  message: "Preparing portal...",
  tone: "info"
};

function App() {
  const portalId = useMemo(() => getPortalId(window.location.pathname), []);
  const isPortalPage = portalId.length > 0;

  return (
    <div className="page">
      <header className="topbar">
        <div className="brand">
          <div className="brand-mark" aria-hidden="true"></div>
          <div>
            <div className="brand-title">DropServe</div>
            <div className="brand-subtitle">LAN Upload Portal</div>
          </div>
        </div>
        <div className="topbar-note">Secure LAN-only uploads, no partials.</div>
      </header>
      <main className="content">
        {isPortalPage ? <PortalPage portalId={portalId} /> : <LandingPage />}
      </main>
    </div>
  );
}

function LandingPage() {
  return (
    <section className="card hero">
      <div className="hero-copy">
        <p className="eyebrow">Quick start</p>
        <h1>Open a portal from your server, then share the link on your LAN.</h1>
        <p className="lead">
          DropServe keeps uploads sequential and safe. Files land only after the
          transfer completes and verifies.
        </p>
        <div className="steps">
          <div className="step">
            <span>1</span>
            <div>
              Run <code>dropserve open</code> in the destination folder.
            </div>
          </div>
          <div className="step">
            <span>2</span>
            <div>Copy the LAN URL from the CLI output.</div>
          </div>
          <div className="step">
            <span>3</span>
            <div>Open the link on a LAN desktop to upload.</div>
          </div>
        </div>
        <div className="callout">
          <div className="callout-title">Tip</div>
          <div className="callout-body">
            Keep the terminal open while uploads are in progress. Close it after
            the portal finishes.
          </div>
        </div>
      </div>
      <div className="hero-panel">
        <div className="hero-panel-header">Example</div>
        <pre>
          <code>$ dropserve open --minutes 20 --policy autorename</code>
        </pre>
        <div className="hero-panel-footer">
          Your CLI prints a short URL like:
          <div className="chip">http://192.168.1.42/p/p_abc123</div>
        </div>
      </div>
    </section>
  );
}

function PortalPage({ portalId }: { portalId: string }) {
  const [status, setStatus] = useState<StatusState>(defaultStatus);
  const [claimed, setClaimed] = useState(false);
  const [running, setRunning] = useState(false);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [conflicts, setConflicts] = useState<PreflightConflict[]>([]);
  const [defaultPolicy, setDefaultPolicy] = useState<"overwrite" | "autorename">(
    "overwrite"
  );
  const [totalBytes, setTotalBytes] = useState(0);
  const [uploadedBytes, setUploadedBytes] = useState(0);
  const [speedBps, setSpeedBps] = useState(0);
  const [expiresAt, setExpiresAt] = useState<string | null>(null);
  const [portalReusable, setPortalReusable] = useState(true);

  const clientTokenRef = useRef("");
  const queueRef = useRef<QueueItem[]>([]);
  const uploadedBytesRef = useRef(0);
  const completedBytesRef = useRef(0);
  const speedTimerRef = useRef<number | null>(null);
  const lastSpeedBytesRef = useRef(0);
  const lastSpeedTimeRef = useRef(0);

  useEffect(() => {
    queueRef.current = queue;
  }, [queue]);

  const updateUploadedBytes = useCallback((value: number) => {
    uploadedBytesRef.current = value;
    setUploadedBytes(value);
  }, []);

  const updateStatus = useCallback((message: string, tone: StatusTone = "info") => {
    setStatus({ message, tone });
  }, []);

  const updateQueueStatus = useCallback((id: string, statusText: string) => {
    setQueue((items) =>
      items.map((item) =>
        item.id === id ? { ...item, status: statusText } : item
      )
    );
  }, []);

  const startSpeedTimer = useCallback(() => {
    if (speedTimerRef.current !== null) {
      return;
    }
    lastSpeedBytesRef.current = uploadedBytesRef.current;
    lastSpeedTimeRef.current = performance.now();
    speedTimerRef.current = window.setInterval(() => {
      const now = performance.now();
      const deltaBytes = uploadedBytesRef.current - lastSpeedBytesRef.current;
      const deltaSeconds = (now - lastSpeedTimeRef.current) / 1000;
      if (deltaSeconds > 0) {
        setSpeedBps(Math.max(0, deltaBytes / deltaSeconds));
      }
      lastSpeedBytesRef.current = uploadedBytesRef.current;
      lastSpeedTimeRef.current = now;
    }, 600);
  }, []);

  const stopSpeedTimer = useCallback(() => {
    if (speedTimerRef.current === null) {
      return;
    }
    window.clearInterval(speedTimerRef.current);
    speedTimerRef.current = null;
    setSpeedBps(0);
  }, []);

  const runPreflight = useCallback(
    async (items: QueueItem[] = queueRef.current, showError = false) => {
      if (!claimed || running || items.length === 0) {
        setConflicts([]);
        return true;
      }
      const payload = {
        items: items.map((item) => ({
          relpath: item.relpath,
          size: item.file.size
        }))
      };
      try {
        const response = await fetch(`/api/portals/${portalId}/preflight`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Client-Token": clientTokenRef.current
          },
          body: JSON.stringify(payload)
        });
        if (!response.ok) {
          if (showError) {
            const message = await readError(response);
            updateStatus(`Preflight failed: ${message}`, "error");
          }
          return false;
        }
        const data = await response.json();
        const incoming = Array.isArray(data.conflicts) ? data.conflicts : [];
        setConflicts(incoming);
        return true;
      } catch {
        if (showError) {
          updateStatus("Preflight failed.", "error");
        }
        return false;
      }
    },
    [claimed, portalId, running, updateStatus]
  );

  const addQueueItems = useCallback(
    (items: QueueCandidate[]) => {
      if (!claimed || running) {
        return;
      }
      const normalized = items.filter((item) => item.file);
      if (normalized.length === 0) {
        return;
      }
      setQueue((existing) => {
        const next = [...existing];
        let addedBytes = 0;
        for (const item of normalized) {
          const relpath = item.relpath || item.file.webkitRelativePath || item.file.name;
          next.push({
            id: makeLocalID(),
            file: item.file,
            relpath,
            status: "queued"
          });
          addedBytes += item.file.size;
        }
        setTotalBytes((value) => value + addedBytes);
        runPreflight(next, false);
        return next;
      });
    },
    [claimed, runPreflight, running]
  );

  const claimPortal = useCallback(async () => {
    if (!portalId) {
      updateStatus("Invalid portal URL.", "error");
      return;
    }
    updateStatus("Claiming portal...", "info");
    try {
      const response = await fetch(`/api/portals/${portalId}/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: "{}"
      });
      if (!response.ok) {
        const message = await readError(response);
        updateStatus(message, "error");
        return;
      }
      const data: ClaimResponse = await response.json();
      clientTokenRef.current = data.client_token;
      setExpiresAt(data.expires_at || null);
      const policy = data.policy && data.policy.autorename ? "autorename" : "overwrite";
      setDefaultPolicy(policy);
      setPortalReusable(data.reusable !== false);
      setClaimed(true);
      updateStatus("Portal ready. Add files to upload.", "ok");
      runPreflight(queueRef.current, false);
    } catch {
      updateStatus("Failed to claim portal.", "error");
    }
  }, [portalId, runPreflight, updateStatus]);

  const initUpload = useCallback(
    async (item: QueueItem) => {
      const payload = {
        upload_id: makeUploadID(),
        relpath: item.relpath,
        size: item.file.size,
        client_sha256: null,
        policy: defaultPolicy
      };
      const response = await fetch(`/api/portals/${portalId}/uploads`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Client-Token": clientTokenRef.current
        },
        body: JSON.stringify(payload)
      });
      if (!response.ok) {
        const message = await readError(response);
        throw new Error(message);
      }
      return response.json();
    },
    [defaultPolicy, portalId]
  );

  const putUpload = useCallback(
    (item: QueueItem, putUrl: string) =>
      new Promise<void>((resolve, reject) => {
        const request = new XMLHttpRequest();
        request.open("PUT", putUrl);
        request.setRequestHeader("X-Client-Token", clientTokenRef.current);
        request.upload.onprogress = (event) => {
          if (!event.lengthComputable) {
            return;
          }
          const current = completedBytesRef.current + event.loaded;
          updateUploadedBytes(current);
          const percent = item.file.size > 0 ? Math.round((event.loaded / item.file.size) * 100) : 100;
          updateQueueStatus(item.id, `uploading ${percent}%`);
        };
        request.onload = () => {
          if (request.status >= 200 && request.status < 300) {
            resolve();
            return;
          }
          let message = request.statusText || "upload failed";
          try {
            const data = JSON.parse(request.responseText);
            if (data && data.error) {
              message = data.error;
            }
          } catch {
          }
          reject(new Error(message));
        };
        request.onerror = () => reject(new Error("network error"));
        request.send(item.file);
      }),
    [updateQueueStatus, updateUploadedBytes]
  );

  const closePortal = useCallback(async () => {
    try {
      const response = await fetch(`/api/portals/${portalId}/close`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Client-Token": clientTokenRef.current
        },
        body: "{}"
      });
      if (!response.ok) {
        const message = await readError(response);
        updateStatus(`Uploads complete. Portal close failed: ${message}`, "warn");
        return;
      }
      updateStatus("All uploads complete. Portal closed.", "ok");
    } catch {
      updateStatus("Uploads complete. Portal close failed.", "warn");
    }
  }, [portalId, updateStatus]);

  const runQueue = useCallback(async () => {
    if (running || !claimed || queueRef.current.length === 0) {
      return;
    }
    const preflightOk = await runPreflight(queueRef.current, true);
    if (!preflightOk) {
      return;
    }
    setRunning(true);
    completedBytesRef.current = 0;
    updateUploadedBytes(0);
    setSpeedBps(0);
    startSpeedTimer();

    for (const item of queueRef.current) {
      updateQueueStatus(item.id, "initializing");
      let initResponse;
      try {
        initResponse = await initUpload(item);
      } catch (error) {
        const message = error instanceof Error ? error.message : "upload failed";
        updateQueueStatus(item.id, "failed");
        updateStatus(`Upload failed: ${message}`, "error");
        setRunning(false);
        stopSpeedTimer();
        return;
      }

      updateQueueStatus(item.id, "uploading 0%");
      try {
        await putUpload(item, initResponse.put_url);
      } catch (error) {
        const message = error instanceof Error ? error.message : "upload failed";
        updateQueueStatus(item.id, "failed");
        updateStatus(`Upload failed: ${message}`, "error");
        setRunning(false);
        stopSpeedTimer();
        return;
      }

      updateQueueStatus(item.id, "done");
      completedBytesRef.current += item.file.size;
      updateUploadedBytes(completedBytesRef.current);
    }

    setRunning(false);
    stopSpeedTimer();
    if (!portalReusable) {
      await closePortal();
    } else {
      updateStatus("All uploads complete. Portal remains open.", "ok");
    }
  }, [
    claimed,
    closePortal,
    initUpload,
    portalReusable,
    putUpload,
    runPreflight,
    running,
    startSpeedTimer,
    stopSpeedTimer,
    updateQueueStatus,
    updateStatus,
    updateUploadedBytes
  ]);

  useEffect(() => {
    claimPortal();
  }, [claimPortal]);

  useEffect(() => () => stopSpeedTimer(), [stopSpeedTimer]);

  const handleFileInput = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const items = normalizeFileList(event.target.files);
      addQueueItems(items);
      event.target.value = "";
    },
    [addQueueItems]
  );

  const handleDrop = useCallback(
    async (event: React.DragEvent<HTMLDivElement>) => {
      if (!claimed || running) {
        return;
      }
      event.preventDefault();
      event.currentTarget.classList.remove("dragging");
      const items = await collectDropItems(event.dataTransfer);
      addQueueItems(items);
    },
    [addQueueItems, claimed, running]
  );

  const handleDragOver = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      if (!claimed || running) {
        return;
      }
      event.preventDefault();
      event.currentTarget.classList.add("dragging");
    },
    [claimed, running]
  );

  const handleDragLeave = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.currentTarget.classList.remove("dragging");
  }, []);

  const canStart = claimed && queue.length > 0 && !running;
  const conflictCount = conflicts.length;
  const conflictVerb = defaultPolicy === "autorename" ? "auto-renamed" : "overwritten";
  const expiryLabel = expiresAt ? formatTimestamp(expiresAt) : "";

  return (
    <div className="portal-layout">
      <section className="card">
        <div className="portal-header">
          <div>
            <p className="eyebrow">Upload portal</p>
            <h1>Portal {portalId}</h1>
            {expiryLabel && <div className="meta">Expires {expiryLabel}</div>}
          </div>
          <div className={`status status-${status.tone}`}>{status.message}</div>
        </div>

        <div className="action-row">
          <label className="file-button">
            Select files
            <input
              {...fileInputProps}
              type="file"
              disabled={!claimed || running}
              onChange={handleFileInput}
            />
          </label>
          <label className="file-button ghost">
            Select folder
            <input
              {...folderInputProps}
              type="file"
              disabled={!claimed || running}
              onChange={handleFileInput}
            />
          </label>
          <button
            className="primary-button"
            onClick={runQueue}
            disabled={!canStart}
          >
            Start upload
          </button>
        </div>

        <div
          className={`drop-zone ${!claimed || running ? "disabled" : ""}`}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
        >
          <div className="drop-title">Drop files or folders here</div>
          <div className="drop-subtitle">Sequential uploads keep large transfers steady.</div>
        </div>

        <div className="stats-grid">
          <Stat label="Files queued" value={String(queue.length)} />
          <Stat label="Total bytes" value={formatBytes(totalBytes)} />
          <Stat label="Bytes uploaded" value={formatBytes(uploadedBytes)} />
          <Stat label="Rolling speed" value={`${formatBytes(speedBps)}/s`} />
        </div>

        <div className={`conflict-panel ${conflictCount === 0 ? "hidden" : ""}`}>
          <div className="conflict-title">Filename conflicts detected</div>
          <div className="conflict-message">
            {conflictCount} {conflictCount === 1 ? "file" : "files"} already exist and will be {conflictVerb}.
          </div>
          <label className="toggle">
            <input
              type="checkbox"
              checked={defaultPolicy === "autorename"}
              disabled={!claimed || running || conflictCount === 0}
              onChange={(event) =>
                setDefaultPolicy(event.target.checked ? "autorename" : "overwrite")
              }
            />
            Auto-rename conflicts instead
          </label>
        </div>
      </section>

      <section className="card">
        <div className="queue-header">
          <h2>Queue</h2>
          <span className="queue-meta">{queue.length} items</span>
        </div>
        <div className="queue-list">
          {queue.length === 0 && (
            <div className="queue-empty">Add files or folders to begin.</div>
          )}
          {queue.map((item) => (
            <div key={item.id} className="queue-row">
              <div className="queue-name" title={item.relpath}>
                {item.relpath}
              </div>
              <div className="queue-status">{item.status}</div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="stat">
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
    </div>
  );
}

function getPortalId(pathname: string) {
  const match = pathname.match(/^\/p\/([^/]+)/);
  return match ? match[1] : "";
}

function makeLocalID() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `q_${Math.random().toString(16).slice(2)}${Date.now().toString(16)}`;
}

function makeUploadID() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `u_${Math.random().toString(16).slice(2)}${Date.now().toString(16)}`;
}

function normalizeFileList(files: FileList | null): QueueCandidate[] {
  if (!files) {
    return [];
  }
  return Array.from(files).map((file) => ({
    file,
    relpath: file.webkitRelativePath || file.name
  }));
}

async function collectDropItems(dataTransfer: DataTransfer | null): Promise<QueueCandidate[]> {
  if (!dataTransfer) {
    return [];
  }
  const items = Array.from(dataTransfer.items || []);
  if (items.length === 0) {
    return normalizeFileList(dataTransfer.files);
  }
  const files: QueueCandidate[] = [];
  for (const item of items) {
    if (item.kind !== "file") {
      continue;
    }
    const entry = (item as DataTransferItemWithEntry).webkitGetAsEntry?.() as
      | DropServeFileSystemEntry
      | null;
    if (entry) {
      const entryFiles = await readEntryFiles(entry);
      files.push(...entryFiles);
      continue;
    }
    const file = item.getAsFile?.();
    if (file) {
      files.push({ file, relpath: file.webkitRelativePath || file.name });
    }
  }
  return files;
}

async function readEntryFiles(entry: DropServeFileSystemEntry): Promise<QueueCandidate[]> {
  if (entry.isFile) {
    return new Promise((resolve) => {
      entry.file(
        (file) => {
          const relpath = stripLeadingSlash(entry.fullPath) || file.webkitRelativePath || file.name;
          resolve([{ file, relpath }]);
        },
        () => resolve([])
      );
    });
  }
  if (entry.isDirectory) {
    const reader = entry.createReader();
    const entries = await readDirectoryEntries(reader);
    const files: QueueCandidate[] = [];
    for (const child of entries) {
      const childFiles = await readEntryFiles(child);
      files.push(...childFiles);
    }
    return files;
  }
  return [];
}

function readDirectoryEntries(reader: DropServeDirectoryReader): Promise<DropServeFileSystemEntry[]> {
  return new Promise((resolve) => {
    const entries: DropServeFileSystemEntry[] = [];
    const readBatch = () => {
      reader.readEntries(
        (batch) => {
          if (batch.length === 0) {
            resolve(entries);
            return;
          }
          entries.push(...batch);
          readBatch();
        },
        () => resolve(entries)
      );
    };
    readBatch();
  });
}

function stripLeadingSlash(value: string) {
  return value.replace(/^\/+/, "");
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) {
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
  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

function formatTimestamp(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) {
    return value;
  }
  return date.toLocaleString();
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    if (data && data.error) {
      return data.error as string;
    }
  } catch {
  }
  return response.statusText || "request failed";
}

export default App;
