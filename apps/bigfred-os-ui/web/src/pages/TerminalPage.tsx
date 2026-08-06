import { useCallback, useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

import { terminalStreamURL } from "../api/client";
import ViewerToolbar, {
  FONT_MAX,
  FONT_MIN,
  loadStoredBool,
  loadStoredNumber,
  storeBool,
  storeNumber,
  type StreamStatus,
} from "../components/ViewerToolbar";

const FONT_KEY = "bigfred.term.fontSize";
const WRAP_KEY = "bigfred.term.wrap";

function sendResize(ws: WebSocket, cols: number, rows: number) {
  if (ws.readyState !== WebSocket.OPEN) {
    return;
  }
  ws.send(
    JSON.stringify({
      type: "resize",
      cols,
      rows,
    }),
  );
}

function statusLabel(status: StreamStatus): string {
  if (status === "connected") return "Connected";
  if (status === "connecting") return "Connecting…";
  if (status === "error") return "Connection error";
  return "Disconnected";
}

export default function TerminalPage() {
  const [status, setStatus] = useState<StreamStatus>("idle");
  const [reconnectKey, setReconnectKey] = useState(0);
  const [fontSize, setFontSize] = useState(() => loadStoredNumber(FONT_KEY, 13, FONT_MIN, FONT_MAX));
  const [wrap, setWrap] = useState(() => loadStoredBool(WRAP_KEY, true));
  const [fullscreen, setFullscreen] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);
  const viewerRef = useRef<HTMLElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const fontSizeRef = useRef(fontSize);

  fontSizeRef.current = fontSize;

  const reconnect = useCallback(() => {
    setReconnectKey((k) => k + 1);
  }, []);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    const term = new Terminal({
      fontFamily: "JetBrains Mono, monospace",
      fontSize: fontSizeRef.current,
      theme: {
        background: "#0a0e12",
        foreground: "#e8eef5",
        cursor: "#3d8bfd",
      },
      cursorBlink: true,
      scrollback: 5000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(container);
    fit.fit();

    termRef.current = term;
    fitRef.current = fit;

    setStatus("connecting");
    const ws = new WebSocket(terminalStreamURL());
    ws.binaryType = "arraybuffer";

    const onData = term.onData((data) => {
      if (ws.readyState !== WebSocket.OPEN) {
        return;
      }
      ws.send(new TextEncoder().encode(data));
    });

    const onResize = term.onResize(({ cols, rows }) => {
      sendResize(ws, cols, rows);
    });

    ws.onopen = () => {
      setStatus("connected");
      fit.fit();
      sendResize(ws, term.cols, term.rows);
      term.focus();
    };

    ws.onerror = () => setStatus("error");
    ws.onclose = () => setStatus((s) => (s === "error" ? "error" : "idle"));

    ws.onmessage = (ev) => {
      if (ev.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(ev.data));
        return;
      }
      if (typeof ev.data === "string") {
        try {
          const msg = JSON.parse(ev.data) as { type?: string; error?: string };
          if (msg.type === "error") {
            setStatus("error");
          }
        } catch {
          term.write(ev.data);
        }
      }
    };

    const resizeObserver = new ResizeObserver(() => {
      fit.fit();
    });
    resizeObserver.observe(container);

    const onWindowResize = () => fit.fit();
    window.addEventListener("resize", onWindowResize);

    return () => {
      onData.dispose();
      onResize.dispose();
      resizeObserver.disconnect();
      window.removeEventListener("resize", onWindowResize);
      ws.close();
      term.dispose();
      termRef.current = null;
      fitRef.current = null;
    };
  }, [reconnectKey]);

  useEffect(() => {
    const term = termRef.current;
    const fit = fitRef.current;
    if (!term) return;
    term.options.fontSize = fontSize;
    fit?.fit();
  }, [fontSize]);

  useEffect(() => {
    const onFsChange = () => {
      const el = viewerRef.current;
      setFullscreen(!!el && document.fullscreenElement === el);
      // Refit after fullscreen transition.
      requestAnimationFrame(() => fitRef.current?.fit());
    };
    document.addEventListener("fullscreenchange", onFsChange);
    return () => document.removeEventListener("fullscreenchange", onFsChange);
  }, []);

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
        /* ignore */
      });
    }
  }, []);

  return (
    <div className="terminal-layout">
      <section
        ref={viewerRef}
        className={`terminal-viewer${fullscreen ? " is-fullscreen" : ""}`}
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
          onReconnect={reconnect}
        >
          <span>Interactive shell</span>
        </ViewerToolbar>
        <div
          ref={containerRef}
          className={`terminal-container${wrap ? "" : " terminal-container--nowrap"}`}
        />
      </section>
    </div>
  );
}
