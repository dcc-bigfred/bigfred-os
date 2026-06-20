import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  fetchServices,
  serviceAction,
  type HubService,
  type ServiceAction,
} from "../api/client";

export default function ServicesPage() {
  const [services, setServices] = useState<HubService[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      setServices(await fetchServices());
    } catch {
      setError("Nie udało się pobrać listy usług.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

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
        setActionError("Operacja nie powiodła się.");
      }
    } finally {
      setPending(null);
    }
  };

  const isPending = (id: string, action: ServiceAction) => pending === `${id}:${action}`;

  return (
    <div className="services-page">
      <div className="services-header">
        <h2>Usługi SysV</h2>
        <button type="button" className="btn-ghost" onClick={() => void load()} disabled={loading}>
          Odśwież
        </button>
      </div>
      <p className="services-hint">Skrypty z <code>/etc/init.d/S??-*</code> wykrywane dynamicznie.</p>

      {actionError ? <div className="services-error">{actionError}</div> : null}
      {loading ? <p className="services-empty">Ładowanie…</p> : null}
      {!loading && error ? <p className="services-empty">{error}</p> : null}
      {!loading && !error && services.length === 0 ? (
        <p className="services-empty">Brak skryptów init w /etc/init.d.</p>
      ) : null}

      {!loading && !error && services.length > 0 ? (
        <div className="services-table-wrap">
          <table className="services-table">
            <thead>
              <tr>
                <th>Usługa</th>
                <th>Status</th>
                <th>Skrypt</th>
                <th>Akcje</th>
              </tr>
            </thead>
            <tbody>
              {services.map((svc) => (
                <tr key={svc.id}>
                  <td>
                    <div className="services-name">{svc.name}</div>
                    <div className="services-id">{svc.id}</div>
                  </td>
                  <td>
                    <span className={`services-badge ${svc.running ? "running" : "stopped"}`}>
                      {svc.running ? "działa" : "zatrzymana"}
                    </span>
                  </td>
                  <td className="services-script">{svc.script}</td>
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
  );
}
