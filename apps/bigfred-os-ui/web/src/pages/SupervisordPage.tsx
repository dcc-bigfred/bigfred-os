import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  fetchSupervisordPrograms,
  supervisordProgramAction,
  type HubSupervisordProgram,
  type SupervisordAction,
} from "../api/client";

function statusLabel(status: string): string {
  switch (status) {
    case "RUNNING":
      return "running";
    case "STOPPED":
      return "stopped";
    case "STARTING":
      return "starting";
    case "BACKOFF":
      return "retrying";
    case "FATAL":
      return "failed";
    case "EXITED":
      return "exited";
    default:
      return status.toLowerCase();
  }
}

function statusClass(status: string): string {
  switch (status) {
    case "RUNNING":
      return "running";
    case "STOPPED":
    case "EXITED":
      return "stopped";
    case "FATAL":
    case "BACKOFF":
      return "fatal";
    default:
      return "unknown";
  }
}

export default function SupervisordPage() {
  const [programs, setPrograms] = useState<HubSupervisordProgram[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      setPrograms(await fetchSupervisordPrograms());
    } catch {
      setError("Could not load the supervisord program list.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const runAction = async (name: string, action: SupervisordAction) => {
    const key = `${name}:${action}`;
    setPending(key);
    setActionError(null);
    try {
      await supervisordProgramAction(name, action);
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

  const isPending = (name: string, action: SupervisordAction) => pending === `${name}:${action}`;

  return (
    <div className="services-page">
      <div className="services-header">
        <h2>Supervisord</h2>
        <button type="button" className="btn-ghost" onClick={() => void load()} disabled={loading}>
          Refresh
        </button>
      </div>
      <p className="services-hint">
        Programs from <code>/data/etc/supervisord/supervisord.conf</code> — controlled via{" "}
        <code>supervisorctl</code>.
      </p>

      {actionError ? <div className="services-error">{actionError}</div> : null}
      {loading ? <p className="services-empty">Loading…</p> : null}
      {!loading && error ? <p className="services-empty">{error}</p> : null}
      {!loading && !error && programs.length === 0 ? (
        <p className="services-empty">No programs in the supervisord configuration.</p>
      ) : null}

      {!loading && !error && programs.length > 0 ? (
        <div className="services-table-wrap">
          <table className="services-table">
            <thead>
              <tr>
                <th>Program</th>
                <th>Status</th>
                <th>Group</th>
                <th>Command</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {programs.map((prog) => (
                <tr key={prog.name}>
                  <td>
                    <div className="services-name">{prog.name}</div>
                    {prog.pid ? <div className="services-id">PID {prog.pid}</div> : null}
                  </td>
                  <td>
                    <span className={`services-badge ${statusClass(prog.status)}`}>
                      {statusLabel(prog.status)}
                    </span>
                  </td>
                  <td className="services-script">{prog.group || "—"}</td>
                  <td className="services-script">{prog.command || "—"}</td>
                  <td className="services-actions">
                    {(["start", "stop", "restart"] as const).map((action) => (
                      <button
                        key={action}
                        type="button"
                        className="btn-action"
                        disabled={pending !== null}
                        onClick={() => void runAction(prog.name, action)}
                      >
                        {isPending(prog.name, action)
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
  );
}
