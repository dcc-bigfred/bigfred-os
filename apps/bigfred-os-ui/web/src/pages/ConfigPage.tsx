import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ApiError,
  fetchEtcFile,
  fetchEtcFiles,
  saveEtcFile,
  type EtcFile,
  type EtcFileContent,
} from "../api/client";

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}

function groupByDir(files: EtcFile[]): [string, EtcFile[]][] {
  const map = new Map<string, EtcFile[]>();
  for (const file of files) {
    const slash = file.path.lastIndexOf("/");
    const dir = slash >= 0 ? file.path.slice(0, slash) : ".";
    const list = map.get(dir) ?? [];
    list.push(file);
    map.set(dir, list);
  }
  return [...map.entries()].sort(([a], [b]) => a.localeCompare(b));
}

function listErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 404) {
      return "Config API not found — restart bigfred-os-ui to load the latest backend.";
    }
    return err.detail ?? err.code;
  }
  return "Could not load the file list.";
}

export default function ConfigPage() {
  const [files, setFiles] = useState<EtcFile[]>([]);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [content, setContent] = useState("");
  const [savedContent, setSavedContent] = useState("");
  const [meta, setMeta] = useState<EtcFileContent | null>(null);
  const [listLoading, setListLoading] = useState(true);
  const [fileLoading, setFileLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [listError, setListError] = useState<string | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveOk, setSaveOk] = useState(false);

  const grouped = useMemo(() => groupByDir(files), [files]);
  const dirty = content !== savedContent;

  const loadList = useCallback(async () => {
    setListError(null);
    setListLoading(true);
    try {
      const list = await fetchEtcFiles();
      setFiles(Array.isArray(list) ? list : []);
    } catch (err) {
      setListError(listErrorMessage(err));
      setFiles([]);
    } finally {
      setListLoading(false);
    }
  }, []);

  const loadFile = useCallback(async (path: string) => {
    setFileError(null);
    setSaveError(null);
    setSaveOk(false);
    setFileLoading(true);
    try {
      const body = await fetchEtcFile(path);
      setMeta(body);
      setContent(body.content);
      setSavedContent(body.content);
    } catch (err) {
      if (err instanceof ApiError) {
        setFileError(err.detail ?? err.code);
      } else {
        setFileError("Could not load file.");
      }
      setMeta(null);
      setContent("");
      setSavedContent("");
    } finally {
      setFileLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  useEffect(() => {
    if (!selectedPath) {
      setMeta(null);
      setContent("");
      setSavedContent("");
      return;
    }
    void loadFile(selectedPath);
  }, [loadFile, selectedPath]);

  const onSelect = (path: string) => {
    if (dirty && !window.confirm("Discard unsaved changes?")) return;
    setSelectedPath(path);
  };

  const onSave = async () => {
    if (!selectedPath) return;
    setSaving(true);
    setSaveError(null);
    setSaveOk(false);
    try {
      const body = await saveEtcFile(selectedPath, content);
      setMeta(body);
      setContent(body.content);
      setSavedContent(body.content);
      setSaveOk(true);
      await loadList();
    } catch (err) {
      if (err instanceof ApiError) {
        setSaveError(err.detail ?? err.code);
      } else {
        setSaveError("Could not save file.");
      }
    } finally {
      setSaving(false);
    }
  };

  const onReload = () => {
    if (!selectedPath) return;
    if (dirty && !window.confirm("Discard unsaved changes?")) return;
    void loadFile(selectedPath);
  };

  return (
    <div className="etc-layout">
      <aside className="etc-sidebar">
        <div className="etc-sidebar-header">
          <h3>/data/etc</h3>
          <button type="button" className="btn-ghost" onClick={() => void loadList()} disabled={listLoading}>
            Refresh
          </button>
        </div>
        {listError ? <p className="etc-empty">{listError}</p> : null}
        {listLoading ? <p className="etc-empty">Loading…</p> : null}
        {!listLoading && !listError && files.length === 0 ? (
          <p className="etc-empty">No files in /data/etc.</p>
        ) : null}
        {grouped.map(([dir, items]) => (
          <div key={dir} className="etc-group">
            <h4 className="etc-group-title">{dir}</h4>
            <ul className="etc-list">
              {items.map((file) => (
                <li key={file.path}>
                  <button
                    type="button"
                    className={file.path === selectedPath ? "active" : ""}
                    onClick={() => onSelect(file.path)}
                  >
                    <span className="etc-name">{file.name}</span>
                    <span className="etc-size">{formatSize(file.size)}</span>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </aside>

      <section className="etc-editor">
        <div className="etc-toolbar">
          <span className="etc-path">{selectedPath ?? "Select a file"}</span>
          <div className="etc-toolbar-actions">
            {dirty ? <span className="etc-dirty">Unsaved changes</span> : null}
            {saveOk ? <span className="etc-saved">Saved</span> : null}
            <button
              type="button"
              className="btn-ghost"
              onClick={onReload}
              disabled={!selectedPath || fileLoading || saving}
            >
              Reload
            </button>
            <button
              type="button"
              className="btn-action"
              onClick={() => void onSave()}
              disabled={!selectedPath || !dirty || fileLoading || saving}
            >
              {saving ? "Saving…" : "Save"}
            </button>
          </div>
        </div>

        {fileError ? <div className="etc-error">{fileError}</div> : null}
        {saveError ? <div className="etc-error">{saveError}</div> : null}

        {selectedPath ? (
          <textarea
            className="etc-textarea"
            value={content}
            onChange={(e) => {
              setContent(e.target.value);
              setSaveOk(false);
            }}
            disabled={fileLoading || saving}
            spellCheck={false}
            placeholder={fileLoading ? "Loading…" : ""}
          />
        ) : (
          <p className="etc-empty etc-placeholder">Choose a configuration file to edit.</p>
        )}

        {meta ? (
          <div className="etc-meta">
            {formatSize(meta.size)}
            {meta.modified ? ` · modified ${new Date(meta.modified).toLocaleString()}` : null}
          </div>
        ) : null}
      </section>
    </div>
  );
}
