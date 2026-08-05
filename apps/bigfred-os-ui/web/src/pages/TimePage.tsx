import { FormEvent, useCallback, useEffect, useState } from "react";
import {
  ApiError,
  fetchTime,
  fetchTimezone,
  HubTime,
  setTime,
  setTimezone,
} from "../api/client";

function toLocalInputValue(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function localInputToISO(value: string): string {
  const d = new Date(value);
  return d.toISOString();
}

export default function TimePage() {
  const [current, setCurrent] = useState<HubTime | null>(null);
  const [timezone, setTimezoneState] = useState("");
  const [available, setAvailable] = useState<string[]>([]);
  const [timeInput, setTimeInput] = useState("");
  const [tzInput, setTzInput] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [pendingTime, setPendingTime] = useState(false);
  const [pendingTz, setPendingTz] = useState(false);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    const [t, tz] = await Promise.all([fetchTime(), fetchTimezone()]);
    setCurrent(t);
    setTimeInput(toLocalInputValue(t.iso));
    setTimezoneState(tz.timezone);
    setTzInput(tz.timezone);
    setAvailable(tz.available);
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await refresh();
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof ApiError ? err.code : "Failed to load time settings.");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [refresh]);

  useEffect(() => {
    const id = window.setInterval(() => {
      void fetchTime()
        .then((t) => {
          setCurrent(t);
        })
        .catch(() => {
          /* ignore poll errors */
        });
    }, 5000);
    return () => window.clearInterval(id);
  }, []);

  const onSetTime = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    if (!timeInput) {
      setError("Choose a date and time.");
      return;
    }
    setPendingTime(true);
    try {
      const t = await setTime({ iso: localInputToISO(timeInput) });
      setCurrent(t);
      setTimeInput(toLocalInputValue(t.iso));
      setSuccess("System time updated and saved to /data/etc/fake-hwclock.");
    } catch (err) {
      setError(err instanceof ApiError ? (err.detail ?? err.code) : "Could not set time.");
    } finally {
      setPendingTime(false);
    }
  };

  const onSetTimezone = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    if (!tzInput) {
      setError("Choose a timezone.");
      return;
    }
    setPendingTz(true);
    try {
      const tz = await setTimezone(tzInput);
      setTimezoneState(tz.timezone);
      setTzInput(tz.timezone);
      setAvailable(tz.available);
      const t = await fetchTime();
      setCurrent(t);
      setTimeInput(toLocalInputValue(t.iso));
      setSuccess(`Timezone set to ${tz.timezone}. New processes use it immediately.`);
    } catch (err) {
      setError(err instanceof ApiError ? (err.detail ?? err.code) : "Could not set timezone.");
    } finally {
      setPendingTz(false);
    }
  };

  if (loading) {
    return (
      <div className="account-page">
        <h2>Time &amp; timezone</h2>
        <p className="account-lead">Loading…</p>
      </div>
    );
  }

  return (
    <div className="account-page">
      <h2>Time &amp; timezone</h2>
      <p className="account-lead">
        The hub has no RTC and no NTP. Wall clock is restored from{" "}
        <code>/data/etc/fake-hwclock</code> at boot and refreshed every 10 minutes. Timezone is
        persisted under <code>/data/etc</code> and bind-mounted over read-only{" "}
        <code>/etc/localtime</code>.
      </p>

      {error ? <div className="login-error">{error}</div> : null}
      {success ? <div className="account-success">{success}</div> : null}

      <div className="time-grid">
        <form className="account-card" onSubmit={(e) => void onSetTime(e)}>
          <h3 className="time-card-title">System time</h3>
          <p className="time-now">
            Now: <strong>{current?.iso ?? "—"}</strong>
          </p>
          <label htmlFor="system-time">Set date &amp; time</label>
          <input
            id="system-time"
            name="system-time"
            type="datetime-local"
            value={timeInput}
            onChange={(e) => setTimeInput(e.target.value)}
            required
          />
          <button type="submit" disabled={pendingTime}>
            {pendingTime ? "Applying…" : "Apply time"}
          </button>
        </form>

        <form className="account-card" onSubmit={(e) => void onSetTimezone(e)}>
          <h3 className="time-card-title">Timezone</h3>
          <p className="time-now">
            Current: <strong>{timezone || "—"}</strong>
          </p>
          <label htmlFor="timezone">Timezone</label>
          <select
            id="timezone"
            name="timezone"
            value={tzInput}
            onChange={(e) => setTzInput(e.target.value)}
            required
          >
            {available.length === 0 ? <option value="">No zoneinfo installed</option> : null}
            {available.map((z) => (
              <option key={z} value={z}>
                {z}
              </option>
            ))}
          </select>
          <button type="submit" disabled={pendingTz || available.length === 0}>
            {pendingTz ? "Applying…" : "Apply timezone"}
          </button>
        </form>
      </div>
    </div>
  );
}
