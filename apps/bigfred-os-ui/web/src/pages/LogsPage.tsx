import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { fetchLogs, logStreamURL, type LogEntry, type LogWSMessage } from "../api/client";
import ViewerToolbar, {
  FONT_MAX,
  FONT_MIN,
  loadStoredBool,
  loadStoredNumber,
  storeBool,
  storeNumber,
  type StreamStatus,
} from "../components/ViewerToolbar";

const FONT_KEY = "bigfred.logs.fontSize";
const WRAP_KEY = "bigfred.logs.wrap";

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}

function groupByRoot(entries: LogEntry[]): [string, LogEntry[]][] {
  const map = new Map<string, LogEntry[]>();
  for (const entry of entries) {
    const list = map.get(entry.root) ?? [];
    list.push(entry);
    map.set(entry.root, list);
  }
  return [...map.entries()].sort(([a], [b]) => a.localeCompare(b));
}

function statusLabel(status: StreamStatus): string {
  if (status === "connected") return "Connected — live stream";
  if (status === "connecting") return "Connecting…";
  if (status === "error") return "Stream error";
  return "Disconnected";
}

export default function LogsPage() {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [lines, setLines] = useState<string[]>([]);
  const [status, setStatus] = useState<StreamStatus>("idle");
  const [listError, setListError] = useState<string | null>(null);
  const [fontSize, setFontSize] = useState(() => loadStoredNumber(FONT_KEY, 13, FONT_MIN, FONT_MAX));
  const [wrap, setWrap] = useState(() => loadStoredBool(WRAP_KEY, true));
  const [fullscreen, setFullscreen] = useState(false);
  const outputRef = useRef<HTMLPreElement>(null);
  const viewerRef = useRef<HTMLElement>(null);
  const stickToBottom = useRef(true);

  const grouped = useMemo(() => groupByRoot(entries), [entries]);

  useEffect(() => {
    fetchLogs()
      .then((list) => {
        const entries = list ?? [];
        setEntries(entries);
        if (entries.length > 0) {
          setSelectedId((prev) => prev ?? entries[0].id);
        }
      })
      .catch(() => setListError("Could not load the log file list."));
  }, []);

  useEffect(() => {
    if (!selectedId) {
      setLines([]);
      setStatus("idle");
      return;
    }

    setLines([]);
    setStatus("connecting");
    const ws = new WebSocket(logStreamURL(selectedId));

    ws.onopen = () => setStatus("connected");
    ws.onerror = () => setStatus("error");
    ws.onclose = () => setStatus((s) => (s === "error" ? "error" : "idle"));

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data as string) as LogWSMessage;
        if (msg.type === "history") {
          setLines(msg.lines ?? []);
          stickToBottom.current = true;
        } else if (msg.type === "line") {
          setLines((prev) => [...(prev ?? []), msg.text ?? ""]);
        } else if (msg.type === "error") {
          setStatus("error");
        }
      } catch {
        setStatus("error");
      }
    };

    return () => ws.close();
  }, [selectedId]);

  useEffect(() => {
    const el = outputRef.current;
    if (!el || !stickToBottom.current) return;
    el.scrollTop = el.scrollHeight;
  }, [lines]);

  useEffect(() => {
    const onFsChange = () => {
      const el = viewerRef.current;
      setFullscreen(!!el && document.fullscreenElement === el);
    };
    document.addEventListener("fullscreenchange", onFsChange);
    return () => document.removeEventListener("fullscreenchange", onFsChange);
  }, []);

  const onScroll = () => {
    const el = outputRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 48;
    stickToBottom.current = nearBottom;
  };

  const onFontSizeChange = useCallback((size: number) => {
    setFontSize(size);
    storeNumber(FONT_KEY, size);
  }, []);

  const onWrapChange = useCallback((next: boolean) => {
    setWrap(next);
    storeBool(WRAP_KEY, next);
  }, []);

  const onFullscreenToggle = useCallback(() => {
    const el = viewerRef.current;
    if (!el) return;
    if (document.fullscreenElement === el) {
      void document.exitFullscreen();
    } else {
      void el.requestFullscreen().catch(() => {
        /* ignore — browser may deny */
      });
    }
  }, []);

  const selected = entries.find((e) => e.id === selectedId);

  return (
    <div className="logs-layout">
      <aside className="logs-sidebar">
        <h3>Log files</h3>
        {listError ? <p className="logs-empty">{listError}</p> : null}
        {!listError && entries.length === 0 ? (
          <p className="logs-empty">No log files in the configured directories.</p>
        ) : null}
        {grouped.map(([root, items]) => (
          <div key={root} className="logs-group">
            <h4 className="logs-group-title">{root}</h4>
            <ul className="logs-list">
              {items.map((entry) => (
                <li key={entry.id}>
                  <button
                    type="button"
                    className={entry.id === selectedId ? "active" : ""}
                    onClick={() => setSelectedId(entry.id)}
                  >
                    <span className="service">{entry.service}</span>
                    {entry.name}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </aside>

      <section
        ref={viewerRef}
        className={`logs-viewer${fullscreen ? " is-fullscreen" : ""}`}
      >
        <ViewerToolbar
          status={status}
          statusLabel={statusLabel(status)}
          fontSize={fontSize}
          wrap={wrap}
          fullscreen={fullscreen}
          onFontSizeChange={onFontSizeChange}
          onWrapChange={onWrapChange}
          onFullscreenToggle={onFullscreenToggle}
        >
          <span>
            {selected
              ? `${selected.root} — ${selected.service}/${selected.name} (${formatSize(selected.size)})`
              : "—"}
          </span>
        </ViewerToolbar>
        <pre
          ref={outputRef}
          className={`logs-output${wrap ? "" : " logs-output--nowrap"}`}
          style={{ fontSize: `${fontSize}px` }}
          onScroll={onScroll}
        >
          {(lines ?? []).length === 0 ? "Waiting for data…" : (lines ?? []).join("\n")}
        </pre>
      </section>
    </div>
  );
}
