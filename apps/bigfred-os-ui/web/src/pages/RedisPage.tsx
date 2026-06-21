import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  deleteRedisKey,
  fetchRedisKeys,
  redisKeyStreamURL,
  type RedisKeyDetail,
  type RedisKeySummary,
  type RedisKeyWSMessage,
} from "../api/client";

type StreamStatus = "idle" | "connecting" | "connected" | "error";

function formatTTL(ttl: number): string {
  if (ttl === -1 || ttl === 0) return "no expiry";
  if (ttl === -2) return "missing";
  if (ttl < 60) return `${ttl}s`;
  if (ttl < 3600) return `${Math.floor(ttl / 60)}m ${ttl % 60}s`;
  const h = Math.floor(ttl / 3600);
  const m = Math.floor((ttl % 3600) / 60);
  return `${h}h ${m}m`;
}

function formatValue(value: unknown): string {
  if (typeof value === "string") return value;
  return JSON.stringify(value, null, 2);
}

export default function RedisPage() {
  const [pattern, setPattern] = useState("*");
  const [searchPattern, setSearchPattern] = useState("*");
  const [keys, setKeys] = useState<RedisKeySummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [detail, setDetail] = useState<RedisKeyDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [streamStatus, setStreamStatus] = useState<StreamStatus>("idle");
  const [deleted, setDeleted] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async (p: string) => {
    setError(null);
    setLoading(true);
    try {
      setKeys(await fetchRedisKeys(p));
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.detail ?? err.code);
      } else {
        setError("Could not load Redis keys.");
      }
      setKeys([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load(searchPattern);
  }, [load, searchPattern]);

  useEffect(() => {
    if (!selectedKey) {
      setDetail(null);
      setDetailLoading(false);
      setDetailError(null);
      setStreamStatus("idle");
      setDeleted(false);
      return;
    }

    setDetail(null);
    setDetailLoading(true);
    setDetailError(null);
    setDeleted(false);
    setStreamStatus("connecting");

    const ws = new WebSocket(redisKeyStreamURL(selectedKey));

    ws.onopen = () => setStreamStatus("connected");
    ws.onerror = () => setStreamStatus("error");
    ws.onclose = () => setStreamStatus((s) => (s === "error" ? "error" : "idle"));

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data as string) as RedisKeyWSMessage;
        if (msg.type === "snapshot" || msg.type === "update") {
          setDetail(msg.detail);
          setDetailLoading(false);
          setDeleted(false);
          setDetailError(null);
        } else if (msg.type === "deleted") {
          setDetail(null);
          setDetailLoading(false);
          setDeleted(true);
        } else if (msg.type === "error") {
          setDetailLoading(false);
          setDetailError(msg.error);
          setStreamStatus("error");
        }
      } catch {
        setDetailLoading(false);
        setDetailError("Invalid stream message.");
        setStreamStatus("error");
      }
    };

    return () => ws.close();
  }, [selectedKey]);

  const onSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSearchPattern(pattern.trim() || "*");
  };

  const openKey = (key: string) => {
    setSelectedKey(key);
  };

  const closeModal = () => {
    setSelectedKey(null);
  };

  const onDelete = async () => {
    if (!selectedKey) return;
    if (!window.confirm(`Delete key "${selectedKey}"?`)) return;
    setDeleting(true);
    setDetailError(null);
    try {
      await deleteRedisKey(selectedKey);
      closeModal();
      await load(searchPattern);
    } catch (err) {
      if (err instanceof ApiError) {
        setDetailError(err.detail ?? err.code);
      } else {
        setDetailError("Could not delete key.");
      }
    } finally {
      setDeleting(false);
    }
  };

  const streamLabel =
    streamStatus === "connecting"
      ? "Connecting…"
      : streamStatus === "connected"
        ? "Live"
        : streamStatus === "error"
          ? "Stream error"
          : "";

  return (
    <div className="redis-page">
      <div className="redis-header">
        <h2>Redis</h2>
        <button
          type="button"
          className="btn-ghost"
          onClick={() => void load(searchPattern)}
          disabled={loading}
        >
          Refresh
        </button>
      </div>
      <p className="redis-hint">
        Keys from <code>127.0.0.1:6379</code> — use Redis glob patterns (e.g. <code>bigfred:*</code>).
      </p>

      <form className="redis-search" onSubmit={onSearch}>
        <input
          type="text"
          value={pattern}
          onChange={(e) => setPattern(e.target.value)}
          placeholder="Pattern (e.g. *)"
          spellCheck={false}
        />
        <button type="submit" className="btn-action" disabled={loading}>
          Search
        </button>
      </form>

      {error ? <div className="redis-error">{error}</div> : null}
      {loading ? <p className="redis-empty">Loading…</p> : null}
      {!loading && !error && keys.length === 0 ? (
        <p className="redis-empty">No keys match the pattern.</p>
      ) : null}

      {!loading && !error && keys.length > 0 ? (
        <div className="redis-table-wrap">
          <table className="redis-table">
            <thead>
              <tr>
                <th>Key</th>
                <th>TTL</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((row) => (
                <tr key={row.key}>
                  <td>
                    <button type="button" className="redis-key-btn" onClick={() => openKey(row.key)}>
                      {row.key}
                    </button>
                  </td>
                  <td className="redis-ttl">{formatTTL(row.ttl)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {selectedKey ? (
        <div className="redis-modal-backdrop" onClick={closeModal} role="presentation">
          <div
            className="redis-modal"
            onClick={(e) => e.stopPropagation()}
            role="dialog"
            aria-labelledby="redis-modal-title"
          >
            <div className="redis-modal-header">
              <div>
                <h3 id="redis-modal-title">{selectedKey}</h3>
                {streamLabel ? (
                  <span className={`redis-stream-status ${streamStatus}`}>{streamLabel}</span>
                ) : null}
              </div>
              <button type="button" className="btn-ghost" onClick={closeModal}>
                Close
              </button>
            </div>

            {detailLoading ? <p className="redis-empty">Loading…</p> : null}
            {detailError ? <div className="redis-error">{detailError}</div> : null}
            {deleted ? <p className="redis-empty">Key was deleted or expired.</p> : null}

            {detail ? (
              <div className="redis-modal-body">
                <div className="redis-meta">
                  <span>Type: {detail.type}</span>
                  <span>TTL: {formatTTL(detail.ttl)}</span>
                </div>
                <pre className="redis-value">{formatValue(detail.value)}</pre>
              </div>
            ) : null}

            <div className="redis-modal-actions">
              <button
                type="button"
                className="btn-danger"
                onClick={() => void onDelete()}
                disabled={deleting || detailLoading || deleted}
              >
                {deleting ? "Deleting…" : "Delete key"}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
