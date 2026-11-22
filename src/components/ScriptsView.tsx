import React, { useEffect, useRef, useState } from 'react';
import {
  Play,
  Plus,
  Save,
  Trash2,
  Sparkles,
  Share2,
  Upload,
  Terminal,
  Copy,
  Check,
  Code2,
  KeyRound,
} from 'lucide-react';
import type { Project, UserScript } from '../types';
import { api } from '../api';

interface Props {
  project: Project;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
  onOpenAISettings: () => void;
}

export const ScriptsView: React.FC<Props> = ({ project, onUpdate, onOpenAISettings }) => {
  const isEn = project.language === 'en';
  const scripts = project.scripts || [];
  const [selectedId, setSelectedId] = useState<string>(scripts[0]?.id || '__new__');
  const [draft, setDraft] = useState<UserScript>(() => {
    const first = scripts[0];
    if (first) return { ...first };
    return {
      id: '',
      name: isEn ? 'New script' : 'Nuevo script',
      description: '',
      author: 'OpenMimo',
      code: `// Acceso directo a \`project\` y \`log(msg)\`.\n// Devuelve true para guardar los cambios con soporte Deshacer.\nlog('Sesiones totales: ' + project.courses.length);\nreturn false;`,
      updatedAt: '',
    };
  });
  const [aiPrompt, setAiPrompt] = useState('');
  const [aiBusy, setAiBusy] = useState(false);
  const [hasApiKey, setHasApiKey] = useState(false);
  const [aiModel, setAiModel] = useState('gpt-4o-mini');
  const [logs, setLogs] = useState<string[]>([]);
  const [runError, setRunError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const importRef = useRef<HTMLInputElement>(null);

  const refreshAiStatus = () => {
    api
      .getAIConfig()
      .then((c) => {
        setHasApiKey(Boolean(c.hasApiKey));
        setAiModel(c.model || 'gpt-4o-mini');
      })
      .catch(() => {});
  };

  useEffect(() => {
    refreshAiStatus();
    const onFocus = () => refreshAiStatus();
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, []);

  const selectScript = (sc: UserScript) => {
    setSelectedId(sc.id);
    setDraft({ ...sc });
    setRunError(null);
  };

  const startNew = () => {
    setSelectedId('__new__');
    setDraft({
      id: '',
      name: isEn ? 'Untitled script' : 'Script sin título',
      description: '',
      author: 'Usuario',
      code: `// Modifica \`project\` y devuelve true para aplicar cambios.\nlog('Recursos: ' + project.resources.length);\nreturn false;`,
      updatedAt: '',
    });
    setRunError(null);
  };

  const handleSave = async () => {
    if (!draft.name.trim()) return;
    const res = await onUpdate(api.saveScript(draft), isEn ? 'Script saved' : 'Script guardado');
    const saved = res?.project?.scripts?.find((s: UserScript) => s.name === draft.name);
    if (saved) {
      setSelectedId(saved.id);
      setDraft({ ...saved });
    }
  };

  const handleDelete = async () => {
    if (!draft.id) return;
    await onUpdate(api.deleteScript(draft.id), isEn ? 'Script deleted' : 'Script eliminado');
    startNew();
  };

  const handleRun = async () => {
    setRunError(null);
    const out: string[] = [];
    const logFn = (...args: any[]) => {
      out.push(args.map((a) => (typeof a === 'object' ? JSON.stringify(a) : String(a))).join(' '));
    };
    try {
      const cloned: Project = JSON.parse(JSON.stringify(project));
      // eslint-disable-next-line no-new-func
      const fn = new Function('project', 'log', draft.code);
      const changed = fn(cloned, logFn);
      if (changed === true) {
        await onUpdate(
          api.replaceProject(cloned),
          isEn ? 'Script executed & applied (Undo ready)' : 'Script ejecutado y aplicado (Deshacer disponible)'
        );
        out.push(isEn ? '✓ Changes committed to project.' : '✓ Cambios guardados en el proyecto.');
      } else {
        out.push(isEn ? '✓ Script finished (read-only).' : '✓ Script finalizado (sin cambios).');
      }
      setLogs(out);
    } catch (err: any) {
      setLogs(out);
      setRunError(`${err?.name || 'Error'}: ${err?.message || String(err)}\n${err?.stack || ''}`);
    }
  };

  const handleGenerateAI = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!aiPrompt.trim()) return;
    setAiBusy(true);
    setRunError(null);
    try {
      const gen = await api.generateScriptAI(aiPrompt.trim());
      setSelectedId('__new__');
      setDraft({
        id: '',
        name: gen.name,
        description: gen.description,
        author: `IA (${gen.model})`,
        code: gen.code,
        updatedAt: '',
      });
      setLogs([isEn ? `✓ Script generated with ${gen.model}` : `✓ Script generado con ${gen.model}`]);
      setAiPrompt('');
      refreshAiStatus();
    } catch (err: any) {
      setRunError(err?.message || String(err));
    } finally {
      setAiBusy(false);
    }
  };

  const handleShareCopy = () => {
    const payload = JSON.stringify(
      { format: 'openmimo-script/1', name: draft.name, description: draft.description, author: draft.author, code: draft.code },
      null,
      2
    );
    navigator.clipboard?.writeText(payload);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  const handleExportScript = () => {
    const payload = JSON.stringify(
      { format: 'openmimo-script/1', name: draft.name, description: draft.description, author: draft.author, code: draft.code },
      null,
      2
    );
    const blob = new Blob([payload], { type: 'application/json;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${(draft.name || 'script').replace(/\s+/g, '_')}.omimo-script.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleImportScript = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = '';
    try {
      const parsed = JSON.parse(await file.text());
      setSelectedId('__new__');
      setDraft({
        id: '',
        name: parsed.name || file.name,
        description: parsed.description || '',
        author: parsed.author || 'Compartido',
        code: parsed.code || '',
        updatedAt: '',
      });
    } catch (err: any) {
      setRunError(`Import error: ${err?.message || String(err)}`);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, gap: 10 }}>
      <input ref={importRef} type="file" accept=".json,.omimo-script.json" style={{ display: 'none' }} onChange={handleImportScript} />

      {/* Top Studio Bar + AI Generator backed by OpenAI API Key in Settings */}
      <div className="view-subbar">
        <div className="control-group">
          <Code2 size={16} color="#e11d48" />
          <strong>{isEn ? 'Script Library' : 'Librería de Scripts'}</strong>
          <span className="stat-chip ok">{scripts.length}</span>
        </div>

        <form onSubmit={handleGenerateAI} style={{ display: 'flex', gap: 6, flex: 1, maxWidth: 580, minWidth: 220 }}>
          <input
            className="input"
            style={{ flex: 1 }}
            placeholder={
              isEn
                ? 'Generate with AI (uses your OpenAI API key in Settings)...'
                : 'Generar con IA (usa tu API Key de Ajustes)...'
            }
            value={aiPrompt}
            onChange={(e) => setAiPrompt(e.target.value)}
          />
          <button className="btn btn-primary btn-sm" type="submit" disabled={aiBusy || !aiPrompt.trim()}>
            <Sparkles size={13} /> {aiBusy ? '...' : isEn ? 'Generate AI' : 'Generar IA'}
          </button>
          <button
            type="button"
            className={`btn btn-sm ${hasApiKey ? 'btn-ok' : 'btn-amber'}`}
            onClick={onOpenAISettings}
            title={hasApiKey ? `API Key OK (${aiModel})` : isEn ? 'Set OpenAI API Key in Settings' : 'Configurar API Key en Ajustes'}
          >
            <KeyRound size={12} /> {hasApiKey ? aiModel : 'API Key'}
          </button>
        </form>

        <div className="control-group">
          <button className="btn btn-sm" onClick={() => importRef.current?.click()}>
            <Upload size={12} /> {isEn ? 'Import' : 'Importar'}
          </button>
          <button className="btn btn-sm" onClick={handleShareCopy}>
            {copied ? <Check size={12} color="#16a34a" /> : <Copy size={12} />} {isEn ? 'Copy' : 'Copiar'}
          </button>
          <button className="btn btn-sm" onClick={handleExportScript}>
            <Share2 size={12} /> {isEn ? 'Share' : 'Compartir'}
          </button>
        </div>
      </div>

      {/* Main Split: Library List | Editor + Console */}
      <div className="split-main">
        <div className="panel">
          <div className="panel-header">
            <span>{isEn ? 'Saved scripts' : 'Scripts guardados'}</span>
            <button className="btn btn-sm btn-primary" onClick={startNew}>
              <Plus size={12} />
            </button>
          </div>
          <div className="panel-body">
            {scripts.map((sc) => (
              <div
                key={sc.id}
                data-ctx="script"
                data-ctx-id={sc.id}
                className={`event-row ${selectedId === sc.id ? 'selected' : ''}`}
                onClick={() => selectScript(sc)}
              >
                <div style={{ minWidth: 0 }}>
                  <div style={{ fontWeight: 700, fontSize: 12.5 }}>{sc.name}</div>
                  {sc.description && (
                    <div style={{ fontSize: 11, color: 'var(--text-faint)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {sc.description}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="panel" style={{ minWidth: 0 }}>
          <div className="panel-header" style={{ gap: 8, flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', gap: 6, flex: 1, minWidth: 200 }}>
              <input
                className="input"
                style={{ fontWeight: 700, width: 220 }}
                placeholder={isEn ? 'Script name' : 'Nombre del script'}
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              />
              <input
                className="input"
                style={{ flex: 1 }}
                placeholder={isEn ? 'Short description' : 'Descripción breve'}
                value={draft.description}
                onChange={(e) => setDraft({ ...draft, description: e.target.value })}
              />
            </div>
            <div style={{ display: 'flex', gap: 6 }}>
              <button className="btn btn-sm btn-ok" onClick={handleRun}>
                <Play size={12} /> {isEn ? 'Run' : 'Ejecutar'}
              </button>
              <button className="btn btn-sm btn-primary" onClick={handleSave}>
                <Save size={12} /> {isEn ? 'Save' : 'Guardar'}
              </button>
              {draft.id && (
                <button className="btn btn-sm btn-danger" onClick={handleDelete}>
                  <Trash2 size={12} />
                </button>
              )}
            </div>
          </div>

          <div className="panel-body" style={{ display: 'flex', flexDirection: 'column', gap: 8, padding: 10 }}>
            <textarea
              className="script-editor"
              spellCheck={false}
              value={draft.code}
              onChange={(e) => setDraft({ ...draft, code: e.target.value })}
            />

            {/* Output & Debug Console */}
            <div className="script-console">
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: '#94a3b8', fontSize: 11, fontWeight: 700 }}>
                  <Terminal size={12} /> {isEn ? 'Output & Debug Console' : 'Consola de salida y debug'}
                </div>
                {runError && !hasApiKey && (
                  <button className="btn btn-sm btn-primary" onClick={onOpenAISettings}>
                    <KeyRound size={11} /> {isEn ? 'Configure API Key' : 'Configurar API Key'}
                  </button>
                )}
              </div>
              {runError && <pre className="script-console-err">{runError}</pre>}
              {logs.length > 0 ? (
                <pre className="script-console-out">{logs.join('\n')}</pre>
              ) : !runError ? (
                <div style={{ color: '#64748b', fontSize: 11.5 }}>
                  {isEn ? 'Press Run to execute script against the active schedule.' : 'Pulsa Ejecutar para correr el script sobre el horario actual.'}
                </div>
              ) : null}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
