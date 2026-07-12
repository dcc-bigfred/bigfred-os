import { useCallback, useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

import { terminalStreamURL } from "../api/client";

type StreamStatus = "idle" | "connecting" | "connected" | "error";

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

export default function TerminalPage() {
  const [status, setStatus] = useState<StreamStatus>("idle");
  const [reconnectKey, setReconnectKey] = useState(0);

  const containerRef = useRef<HTMLDivElement>(null);

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
      fontSize: 13,
      theme: {
        background: "#0a0e12",
        foreground: "#e8eef5",
        cursor: "#3d8bfd",
      },
      cursorBlink: true,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(container);
    fit.fit();

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
    };
  }, [reconnectKey]);

  return (
    <div className="terminal-layout">
      <section className="terminal-viewer">
        <div className="logs-toolbar">
          <span>Interactive shell</span>
          <div className="terminal-toolbar-actions">
            <span className={`logs-status ${status}`}>
              {status === "connected" && "Connected"}
              {status === "connecting" && "Connecting…"}
              {status === "error" && "Connection error"}
              {status === "idle" && "Disconnected"}
            </span>
            <button type="button" className="btn-ghost" onClick={reconnect}>
              Reconnect
            </button>
          </div>
        </div>
        <div ref={containerRef} className="terminal-container" />
      </section>
    </div>
  );
}
