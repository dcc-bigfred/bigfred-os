import { useEffect, useMemo, useRef, useState } from "react";
import { fetchLogs, logStreamURL, type LogEntry, type LogWSMessage } from "../api/client";

type StreamStatus = "idle" | "connecting" | "connected" | "error";

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

export default function LogsPage() {
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [lines, setLines] = useState<string[]>([]);
  const [status, setStatus] = useState<StreamStatus>("idle");
  const [listError, setListError] = useState<string | null>(null);
  const outputRef = useRef<HTMLPreElement>(null);
  const stickToBottom = useRef(true);

  const grouped = useMemo(() => groupByRoot(entries), [entries]);

  useEffect(() => {
    fetchLogs()
      .then((list) => {
        setEntries(list);
        if (list.length > 0) {
          setSelectedId((prev) => prev ?? list[0].id);
        }
      })
      .catch(() => setListError("Nie udało się pobrać listy logów."));
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
          setLines(msg.lines);
          stickToBottom.current = true;
        } else if (msg.type === "line") {
          setLines((prev) => [...prev, msg.text]);
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

  const onScroll = () => {
    const el = outputRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 48;
    stickToBottom.current = nearBottom;
  };

  const selected = entries.find((e) => e.id === selectedId);

  return (
    <div className="logs-layout">
      <aside className="logs-sidebar">
        <h3>Pliki logów</h3>
        {listError ? <p className="logs-empty">{listError}</p> : null}
        {!listError && entries.length === 0 ? (
          <p className="logs-empty">Brak plików logów w skonfigurowanych katalogach.</p>
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

      <section className="logs-viewer">
        <div className="logs-toolbar">
          <span>
            {selected
              ? `${selected.root} — ${selected.service}/${selected.name} (${formatSize(selected.size)})`
              : "—"}
          </span>
          <span className={`logs-status ${status}`}>
            {status === "connected" && "Połączono — strumień na żywo"}
            {status === "connecting" && "Łączenie…"}
            {status === "error" && "Błąd strumienia"}
            {status === "idle" && "Rozłączono"}
          </span>
        </div>
        <pre ref={outputRef} className="logs-output" onScroll={onScroll}>
          {lines.length === 0 ? "Oczekiwanie na dane…" : lines.join("\n")}
        </pre>
      </section>
    </div>
  );
}
