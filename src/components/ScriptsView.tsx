import React, { useRef, useState } from 'react';
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
} from 'lucide-react';
import type { Project, UserScript } from '../types';
import { api } from '../api';

interface Props {
  project: Project;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
}

function generateScriptFromPrompt(prompt: string, isEn: boolean): { name: string; description: string; code: string } {
  const low = prompt.toLowerCase();

  if (low.includes('viernes') || low.includes('friday') || low.includes('tarde') || low.includes('afternoon') || low.includes('bloqu') || low.includes('unavail')) {
    const day = low.includes('lunes') || low.includes('monday') ? 1
      : low.includes('martes') || low.includes('tuesday') ? 2
      : low.includes('miércoles') || low.includes('miercoles') || low.includes('wednesday') ? 3
      : low.includes('jueves') || low.includes('thursday') ? 4 : 5;
    const minPeriod = low.includes('mañana') || low.includes('morning') ? 1 : 6;
    const maxPeriod = low.includes('mañana') || low.includes('morning') ? 5 : 10;
    return {
      name: prompt.slice(0, 46),
      description: isEn ? `AI generated: ${prompt}` : `Generado con IA: ${prompt}`,
      code: `// ${prompt}
const teachers = project.resources.filter(r => r.category === 2);
let added = 0;
for (const t of teachers) {
  for (let w = 1; w <= project.time.maxWeeks; w++) {
    for (let p = ${minPeriod}; p <= Math.min(${maxPeriod}, project.time.maxPeriods); p++) {
      const exists = project.bookings.some(b =>
        b.targetId === t.id && b.week === w && b.day === ${day} && b.period === p
      );
      if (!exists) {
        project.bookings.push({
          id: 'b_ai_' + t.id + '_' + w + '_' + p,
          targetType: 'resource',
          targetId: t.id,
          scope: 'timetable',
          week: w,
          day: ${day},
          period: p,
          kind: 1,
          label: 'Bloqueado (IA)'
        });
        added++;
      }
    }
  }
}
log('Bloqueos añadidos: ' + added + ' para ' + teachers.length + ' docentes.');
return added > 0;`,
    };
  }

  if (low.includes('color') || low.includes('pintar') || low.includes('paleta')) {
    return {
      name: prompt.slice(0, 46),
      description: isEn ? `AI generated: ${prompt}` : `Generado con IA: ${prompt}`,
      code: `// ${prompt}
const palette = ['#2563eb', '#059669', '#d97706', '#7c3aed', '#e11d48', '#0891b2', '#4f46e5'];
project.courses.forEach((c, idx) => {
  c.color = palette[idx % palette.length];
});
log('Colores actualizados en ' + project.courses.length + ' sesiones.');
return true;`,
    };
  }

  if (low.includes('guia') || low.includes('guía') || low.includes('guide') || low.includes('astro')) {
    return {
      name: prompt.slice(0, 46),
      description: isEn ? `AI generated: ${prompt}` : `Generado con IA: ${prompt}`,
      code: `// ${prompt}
let created = 0;
for (const c of project.courses) {
  if (!c.guideId) {
    const id = 'g_ai_' + c.code.toLowerCase().replace(/[^a-z0-9]/g, '');
    if (!project.guides.some(g => g.id === id)) {
      project.guides.push({
        id,
        slug: id,
        title: c.name,
        degree: project.institution,
        degreeLevel: 'Grado',
        courseType: 'Obligatoria',
        character: 'Teórico-práctica',
        specialty: '',
        subjectArea: '',
        academicYear: '1º',
        semester: '1º Semestre',
        creditsEcts: 6,
        totalHours: 150,
        inPersonHours: 60,
        department: project.department,
        prerequisites: '',
        languages: 'Español',
        description: '',
        groups: [],
        competencies: '',
        learningOutcomes: '',
        contents: '',
        theoryActivities: '',
        practiceActivities: '',
        otherActivities: '',
        studentWorkHours: '',
        evalAttitudePct: 20,
        evalWrittenPct: 30,
        evalPracticalPct: 50,
        evalCriteria: '',
        bibliography: '',
        updatedAt: new Date().toISOString()
      });
      created++;
    }
    c.guideId = id;
  }
}
log('Guías creadas/vinculadas: ' + created);
return created > 0;`,
    };
  }

  if (low.includes('nota') || low.includes('mcp') || low.includes('note')) {
    return {
      name: prompt.slice(0, 46),
      description: isEn ? `AI generated: ${prompt}` : `Generado con IA: ${prompt}`,
      code: `// ${prompt}
const note = ${JSON.stringify(prompt)};
let count = 0;
for (const r of project.resources) {
  if (r.category === 2) {
    r.mcpNotes = r.mcpNotes ? r.mcpNotes + ' | ' + note : note;
    count++;
  }
}
log('Nota MCP añadida a ' + count + ' docentes.');
return count > 0;`,
    };
  }

  // Default smart audit & summary script
  return {
    name: prompt.slice(0, 46) || (isEn ? 'AI Script' : 'Script IA'),
    description: isEn ? `AI generated: ${prompt}` : `Generado con IA: ${prompt}`,
    code: `// ${prompt}
log('Proyecto: ' + project.name + ' (' + project.courses.length + ' sesiones, ' + project.resources.length + ' recursos)');
const unscheduled = project.courses.filter(c => c.allocated > c.scheduled);
log('Sesiones con horas pendientes: ' + unscheduled.length);
for (const c of unscheduled.slice(0, 15)) {
  log('  - ' + c.code + ' ' + c.name + ': ' + c.scheduled + '/' + c.allocated + ' h');
}
return false; // Cambia a return true si modificas project`,
  };
}

export const ScriptsView: React.FC<Props> = ({ project, onUpdate }) => {
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
  const [aiEndpoint, setAiEndpoint] = useState('');
  const [aiBusy, setAiBusy] = useState(false);
  const [logs, setLogs] = useState<string[]>([]);
  const [runError, setRunError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const importRef = useRef<HTMLInputElement>(null);

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
      if (aiEndpoint.trim()) {
        // Optional OpenAI-compatible local/remote LLM endpoint (Ollama, LM Studio, Agente)
        const resp = await fetch(aiEndpoint.trim(), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            model: 'default',
            messages: [
              {
                role: 'system',
                content:
                  'Return ONLY valid JavaScript body for `function(project, log)` operating on an OpenMimo Project JSON. Return true if mutated, false if read-only.',
              },
              { role: 'user', content: aiPrompt },
            ],
          }),
        });
        const json = await resp.json();
        const code = json?.choices?.[0]?.message?.content?.replace(/```[a-z]*\n?|```/gi, '').trim();
        if (code) {
          setSelectedId('__new__');
          setDraft({
            id: '',
            name: aiPrompt.slice(0, 44),
            description: aiPrompt,
            author: 'IA',
            code,
            updatedAt: '',
          });
          setAiPrompt('');
          return;
        }
      }
      const gen = generateScriptFromPrompt(aiPrompt.trim(), isEn);
      setSelectedId('__new__');
      setDraft({
        id: '',
        name: gen.name,
        description: gen.description,
        author: 'IA',
        code: gen.code,
        updatedAt: '',
      });
      setAiPrompt('');
    } catch (err: any) {
      setRunError(`AI Error: ${err?.message || String(err)}`);
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

      {/* Top Studio Bar + AI Generator */}
      <div className="view-subbar">
        <div className="control-group">
          <Code2 size={16} color="#e11d48" />
          <strong>{isEn ? 'Script Library' : 'Librería de Scripts'}</strong>
          <span className="stat-chip ok">{scripts.length}</span>
        </div>

        <form onSubmit={handleGenerateAI} style={{ display: 'flex', gap: 6, flex: 1, maxWidth: 620, minWidth: 240 }}>
          <input
            className="input"
            style={{ flex: 1 }}
            placeholder={
              isEn
                ? 'Generate with AI (e.g. "Block Friday afternoons for all teachers")...'
                : 'Generar con IA (ej. "Bloquear viernes tarde a profesores", "Colorear sesiones")...'
            }
            value={aiPrompt}
            onChange={(e) => setAiPrompt(e.target.value)}
          />
          <button className="btn btn-primary btn-sm" type="submit" disabled={aiBusy || !aiPrompt.trim()}>
            <Sparkles size={13} /> {isEn ? 'Generate AI' : 'Generar IA'}
          </button>
        </form>

        <div className="control-group">
          <button className="btn btn-sm" onClick={() => importRef.current?.click()}>
            <Upload size={12} /> {isEn ? 'Import' : 'Importar'}
          </button>
          <button className="btn btn-sm" onClick={handleShareCopy}>
            {copied ? <Check size={12} color="#16a34a" /> : <Copy size={12} />} {isEn ? 'Copy JSON' : 'Copiar'}
          </button>
          <button className="btn btn-sm" onClick={handleExportScript}>
            <Share2 size={12} /> {isEn ? 'Share file' : 'Compartir'}
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

            {/* Optional custom LLM endpoint row */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11, color: 'var(--text-faint)' }}>
              <span>API IA (opcional):</span>
              <input
                className="input"
                style={{ flex: 1, fontSize: 11, padding: '3px 7px' }}
                placeholder="http://127.0.0.1:11434/v1/chat/completions (Ollama / Agente)"
                value={aiEndpoint}
                onChange={(e) => setAiEndpoint(e.target.value)}
              />
            </div>

            {/* Output & Debug Console */}
            <div className="script-console">
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4, color: '#94a3b8', fontSize: 11, fontWeight: 700 }}>
                <Terminal size={12} /> {isEn ? 'Output & Debug Console' : 'Consola de salida y debug'}
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
