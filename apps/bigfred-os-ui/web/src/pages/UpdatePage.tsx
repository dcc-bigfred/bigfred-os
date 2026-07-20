import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  ApiError,
  fetchUpdateReleases,
  runUpdate,
  type UpdateRelease,
  type UpdateResult,
  type UpdateTarget,
} from "../api/client";

type PendingTarget = UpdateTarget | null;

interface ConfirmState {
  target: UpdateTarget;
  tag: string;
  title: string;
  body: string;
}

const LABELS: Record<UpdateTarget, { title: string; button: string; service: string }> = {
  bigfred: {
    title: "Update BigFred?",
    button: "Update BigFred",
    service: "BigFred",
  },
  "bigfred-remote-icmp": {
    title: "Update remote-icmp?",
    button: "Update remote-icmp",
    service: "remote-icmp",
  },
  "bigfred-ui": {
    title: "Update BigFred UI?",
    button: "Update BigFred UI",
    service: "bigfred-os-ui",
  },
};

function formatReleaseLabel(rel: UpdateRelease): string {
  const bits = [rel.tag];
  if (rel.name && rel.name !== rel.tag) {
    bits.push(`— ${rel.name}`);
  }
  if (rel.prerelease) {
    bits.push("(pre-release)");
  }
  return bits.join(" ");
}

function TargetRow({
  target,
  pending,
  onConfirm,
}: {
  target: UpdateTarget;
  pending: PendingTarget;
  onConfirm: (target: UpdateTarget, tag: string) => void;
}) {
  const [releases, setReleases] = useState<UpdateRelease[]>([]);
  const [tag, setTag] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const list = await fetchUpdateReleases(target);
      setReleases(list);
      setTag((prev) => {
        if (prev && list.some((r) => r.tag === prev)) {
          return prev;
        }
        return list[0]?.tag ?? "";
      });
    } catch (err) {
      setReleases([]);
      setTag("");
      if (err instanceof ApiError) {
        setLoadError(err.detail ?? err.code);
      } else {
        setLoadError("Could not load releases.");
      }
    } finally {
      setLoading(false);
    }
  }, [target]);

  useEffect(() => {
    void load();
  }, [load]);

  const busy = pending !== null;
  const canUpdate = !busy && !loading && tag !== "";

  return (
    <div className="update-row">
      <div className="update-row-main">
        <label className="update-select-label" htmlFor={`release-${target}`}>
          {LABELS[target].button}
        </label>
        <select
          id={`release-${target}`}
          className="update-select"
          value={tag}
          disabled={busy || loading || releases.length === 0}
          onChange={(e) => setTag(e.target.value)}
        >
          {releases.length === 0 ? (
            <option value="">{loading ? "Loading releases…" : "No releases available"}</option>
          ) : (
            releases.map((rel) => (
              <option key={rel.tag} value={rel.tag}>
                {formatReleaseLabel(rel)}
              </option>
            ))
          )}
        </select>
        <button
          type="button"
          className="btn-ghost update-refresh"
          disabled={busy || loading}
          onClick={() => void load()}
        >
          Refresh
        </button>
      </div>
      {loadError ? <div className="update-row-error">{loadError}</div> : null}
      <button
        type="button"
        className="btn-action update-btn"
        disabled={!canUpdate}
        onClick={() => onConfirm(target, tag)}
      >
        {pending === target ? "Updating…" : LABELS[target].button}
      </button>
    </div>
  );
}

export default function UpdatePage() {
  const [confirm, setConfirm] = useState<ConfirmState | null>(null);
  const [pending, setPending] = useState<PendingTarget>(null);
  const [error, setError] = useState<string | null>(null);
  const [last, setLast] = useState<UpdateResult | null>(null);

  const openConfirm = (target: UpdateTarget, tag: string) => {
    const dest =
      target === "bigfred"
        ? "/data/opt/bigfred/bin/bigfred"
        : target === "bigfred-remote-icmp"
          ? "/data/opt/bigfred/bin/bigfred-remote-icmp"
          : "/data/opt/bigfred/bin/bigfred-os-ui";
    setError(null);
    setConfirm({
      target,
      tag,
      title: LABELS[target].title,
      body: `Download ${tag} and install it to ${dest}. After the update, restart ${LABELS[target].service} from the Services tab so the new binary is loaded.`,
    });
  };

  const run = async (target: UpdateTarget, tag: string) => {
    setConfirm(null);
    setPending(target);
    setError(null);
    setLast(null);
    try {
      const res = await runUpdate(target, tag);
      setLast(res);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.detail ?? err.code);
      } else {
        setError("The update failed.");
      }
    } finally {
      setPending(null);
    }
  };

  return (
    <div className="update-page">
      <div className="update-header">
        <h2>Update</h2>
      </div>
      <p className="update-hint">
        Choose a GitHub release, then install the binary into <code>/data/opt/bigfred/bin</code>.
        After updating BigFred, restart it from the <Link to="/services">Services</Link> tab.
      </p>

      {error ? <div className="update-error">{error}</div> : null}
      {last ? (
        <div className="update-success">
          Installed <code>{last.asset}</code> ({last.tag}) → <code>{last.path}</code>. Restart{" "}
          <strong>{last.restart}</strong> from the <Link to="/services">Services</Link> tab.
        </div>
      ) : null}

      <div className="update-actions">
        <TargetRow target="bigfred" pending={pending} onConfirm={openConfirm} />
        <TargetRow target="bigfred-remote-icmp" pending={pending} onConfirm={openConfirm} />
        <TargetRow target="bigfred-ui" pending={pending} onConfirm={openConfirm} />
      </div>

      {confirm ? (
        <div
          className="update-modal-backdrop"
          role="presentation"
          onClick={() => pending === null && setConfirm(null)}
        >
          <div
            className="update-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="update-dialog-title"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="update-modal-header">
              <h3 id="update-dialog-title">{confirm.title}</h3>
            </div>
            <div className="update-modal-body">
              <p>{confirm.body}</p>
              <p className="update-modal-note">
                Selected release: <code>{confirm.tag}</code>. Use the{" "}
                <Link to="/services">Services</Link> tab to restart the process after the download
                finishes.
              </p>
            </div>
            <div className="update-modal-actions">
              <button type="button" className="btn-ghost" onClick={() => setConfirm(null)}>
                Cancel
              </button>
              <button
                type="button"
                className="btn-action"
                onClick={() => void run(confirm.target, confirm.tag)}
              >
                Download &amp; install
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
