import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  fetchServices,
  serviceAction,
  serviceLogStreamURL,
  type HubService,
  type LogWSMessage,
  type ServiceAction,
} from "../api/client";

type StreamStatus = "idle" | "connecting" | "connected" | "error";

function badgeClass(state: string, running: boolean): string {
  const s = state.toLowerCase();
  if (s === "failed") return "fatal";
  if (s === "disabled" || s === "stopped" || s === "succeeded") return "stopped";
  if (running || s === "running" || s === "starting" || s === "restarting") return "running";
  return "unknown";
}

export default function ServicesPage() {
  const [services, setServices] = useState<HubService[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const [logService, setLogService] = useState<HubService | null>(null);
  const [lines, setLines] = useState<string[]>([]);
  const [streamStatus, setStreamStatus] = useState<StreamStatus>("idle");
  const outputRef = useRef<HTMLPreElement>(null);
  const stickToBottom = useRef(true);

  const load = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      setServices(await fetchServices());
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.detail ?? err.code);
      } else {
        setError("Could not load the service list.");
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!logService) {
      setLines([]);
      setStreamStatus("idle");
      return;
    }

    setLines([]);
    setStreamStatus("connecting");
    stickToBottom.current = true;
    const ws = new WebSocket(serviceLogStreamURL(logService.id));

    ws.onopen = () => setStreamStatus("connected");
    ws.onerror = () => setStreamStatus("error");
    ws.onclose = () => setStreamStatus((s) => (s === "error" ? "error" : "idle"));

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data as string) as LogWSMessage;
        if (msg.type === "history") {
          setLines(msg.lines ?? []);
          stickToBottom.current = true;
        } else if (msg.type === "line") {
          setLines((prev) => [...(prev ?? []), msg.text ?? ""]);
        } else if (msg.type === "error") {
          setStreamStatus("error");
        }
      } catch {
        setStreamStatus("error");
      }
    };

    return () => ws.close();
  }, [logService]);

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

  const runAction = async (id: string, action: ServiceAction) => {
    const key = `${id}:${action}`;
    setPending(key);
    setActionError(null);
    try {
      await serviceAction(id, action);
      await load();
    } catch (err) {
      if (err instanceof ApiError) {
        setActionError(err.detail ?? err.code);
      } else {
        setActionError("The operation failed.");
      }
    } finally {
      setPending(null);
    }
  };

  const isPending = (id: string, action: ServiceAction) => pending === `${id}:${action}`;

  return (
    <div className={`services-page${logService ? " services-page-with-logs" : ""}`}>
      <div className="services-main">
        <div className="services-header">
          <h2>Services</h2>
          <button type="button" className="btn-ghost" onClick={() => void load()} disabled={loading}>
            Refresh
          </button>
        </div>
        <p className="services-hint">
          Managed by <code>microinit</code> via <code>/run/microinit.sock</code>.
        </p>

        {actionError ? <div className="services-error">{actionError}</div> : null}
        {loading ? <p className="services-empty">Loading…</p> : null}
        {!loading && error ? <p className="services-empty">{error}</p> : null}
        {!loading && !error && services.length === 0 ? (
          <p className="services-empty">No services reported by microinit.</p>
        ) : null}

        {!loading && !error && services.length > 0 ? (
          <div className="services-table-wrap">
            <table className="services-table">
              <thead>
                <tr>
                  <th>Service</th>
                  <th>State</th>
                  <th>PID</th>
                  <th>Restarts</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {services.map((svc) => (
                  <tr key={svc.id} className={logService?.id === svc.id ? "services-row-active" : ""}>
                    <td>
                      <div className="services-name-row">
                        <div>
                          <div className="services-name">{svc.name}</div>
                          <div className="services-id">{svc.id}</div>
                        </div>
                        <button
                          type="button"
                          className="btn-icon"
                          title="Show logs"
                          aria-label={`Show logs for ${svc.id}`}
                          onClick={() => setLogService(svc)}
                        >
                          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
                            <path
                              fill="currentColor"
                              d="M4 4h16a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1zm1 2v12h14V6H5zm2 2h10v2H7V8zm0 4h7v2H7v-2z"
                            />
                          </svg>
                        </button>
                      </div>
                    </td>
                    <td>
                      <span className={`services-badge ${badgeClass(svc.state, svc.running)}`}>
                        {svc.state}
                        {!svc.enabled ? " · disabled" : ""}
                      </span>
                    </td>
                    <td className="services-pid">{svc.pid ?? "—"}</td>
                    <td>{svc.restarts}</td>
                    <td className="services-actions">
                      {(["start", "stop", "restart"] as const).map((action) => (
                        <button
                          key={action}
                          type="button"
                          className="btn-action"
                          disabled={pending !== null}
                          onClick={() => void runAction(svc.id, action)}
                        >
                          {isPending(svc.id, action)
                            ? "…"
                            : action === "start"
                              ? "Start"
                              : action === "stop"
                                ? "Stop"
                                : "Restart"}
                        </button>
                      ))}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </div>

      {logService ? (
        <section className="services-logs-viewer logs-viewer">
          <div className="logs-toolbar">
            <span>
              Logs — <strong>{logService.id}</strong>
            </span>
            <span className={`logs-status ${streamStatus}`}>
              {streamStatus === "connected" && "Connected — live stream"}
              {streamStatus === "connecting" && "Connecting…"}
              {streamStatus === "error" && "Stream error"}
              {streamStatus === "idle" && "Disconnected"}
            </span>
            <button type="button" className="btn-ghost" onClick={() => setLogService(null)}>
              Close
            </button>
          </div>
          <pre ref={outputRef} className="logs-output" onScroll={onScroll}>
            {(lines ?? []).length === 0 ? "Waiting for data…" : (lines ?? []).join("\n")}
          </pre>
        </section>
      ) : null}
    </div>
  );
}
