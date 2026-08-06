import type { ReactNode } from "react";

const FONT_MIN = 10;
const FONT_MAX = 24;

function IconZoomOut() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle cx="10.5" cy="10.5" r="6.5" stroke="currentColor" strokeWidth="2" />
      <path d="M16 16l5 5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      <path d="M8 10.5h5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  );
}

function IconZoomIn() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle cx="10.5" cy="10.5" r="6.5" stroke="currentColor" strokeWidth="2" />
      <path d="M16 16l5 5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      <path d="M10.5 8v5M8 10.5h5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  );
}

function IconWrap() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M4 7h16M4 12h11a3 3 0 010 6h-3"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M14 15l-2 3 2 3"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function IconFullscreen({ active }: { active: boolean }) {
  if (active) {
    return (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="M9 3H5a2 2 0 00-2 2v4M15 3h4a2 2 0 012 2v4M9 21H5a2 2 0 01-2-2v-4M15 21h4a2 2 0 002-2v-4"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
        />
      </svg>
    );
  }
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M9 3v4a2 2 0 01-2 2H3M15 3v4a2 2 0 002 2h4M9 21v-4a2 2 0 00-2-2H3M15 21v-4a2 2 0 012-2h4"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
}

export type StreamStatus = "idle" | "connecting" | "connected" | "error";

export type ViewerToolbarProps = {
  children?: ReactNode;
  status: StreamStatus;
  statusLabel: string;
  fontSize: number;
  wrap: boolean;
  fullscreen: boolean;
  onFontSizeChange: (size: number) => void;
  onWrapChange: (wrap: boolean) => void;
  onFullscreenToggle: () => void;
  onReconnect?: () => void;
  showWrap?: boolean;
};

export { FONT_MIN, FONT_MAX };

export default function ViewerToolbar({
  children,
  status,
  statusLabel,
  fontSize,
  wrap,
  fullscreen,
  onFontSizeChange,
  onWrapChange,
  onFullscreenToggle,
  onReconnect,
  showWrap = true,
}: ViewerToolbarProps) {
  return (
    <div className="logs-toolbar viewer-toolbar">
      <div className="viewer-toolbar-left">{children}</div>
      <div className="viewer-toolbar-actions">
        <span className={`logs-status ${status}`}>{statusLabel}</span>
        {onReconnect ? (
          <button type="button" className="btn-ghost" onClick={onReconnect}>
            Reconnect
          </button>
        ) : null}
        <div className="viewer-toolbar-icons" role="group" aria-label="Viewer controls">
          <button
            type="button"
            className="btn-icon"
            title="Decrease font size"
            aria-label="Decrease font size"
            disabled={fontSize <= FONT_MIN}
            onClick={() => onFontSizeChange(Math.max(FONT_MIN, fontSize - 1))}
          >
            <IconZoomOut />
          </button>
          <button
            type="button"
            className="btn-icon"
            title="Increase font size"
            aria-label="Increase font size"
            disabled={fontSize >= FONT_MAX}
            onClick={() => onFontSizeChange(Math.min(FONT_MAX, fontSize + 1))}
          >
            <IconZoomIn />
          </button>
          {showWrap ? (
            <button
              type="button"
              className={`btn-icon${wrap ? " is-active" : ""}`}
              title={wrap ? "Disable line wrap" : "Enable line wrap"}
              aria-label={wrap ? "Disable line wrap" : "Enable line wrap"}
              aria-pressed={wrap}
              onClick={() => onWrapChange(!wrap)}
            >
              <IconWrap />
            </button>
          ) : null}
          <button
            type="button"
            className={`btn-icon${fullscreen ? " is-active" : ""}`}
            title={fullscreen ? "Exit fullscreen" : "Enter fullscreen"}
            aria-label={fullscreen ? "Exit fullscreen" : "Enter fullscreen"}
            aria-pressed={fullscreen}
            onClick={onFullscreenToggle}
          >
            <IconFullscreen active={fullscreen} />
          </button>
        </div>
      </div>
    </div>
  );
}

/** Read a persisted number from localStorage with fallback. */
export function loadStoredNumber(key: string, fallback: number, min: number, max: number): number {
  try {
    const raw = localStorage.getItem(key);
    if (raw == null) return fallback;
    const n = Number(raw);
    if (!Number.isFinite(n)) return fallback;
    return Math.min(max, Math.max(min, Math.round(n)));
  } catch {
    return fallback;
  }
}

/** Read a persisted boolean from localStorage with fallback. */
export function loadStoredBool(key: string, fallback: boolean): boolean {
  try {
    const raw = localStorage.getItem(key);
    if (raw == null) return fallback;
    return raw === "1" || raw === "true";
  } catch {
    return fallback;
  }
}

export function storeNumber(key: string, value: number) {
  try {
    localStorage.setItem(key, String(value));
  } catch {
    /* ignore quota / private mode */
  }
}

export function storeBool(key: string, value: boolean) {
  try {
    localStorage.setItem(key, value ? "1" : "0");
  } catch {
    /* ignore */
  }
}
