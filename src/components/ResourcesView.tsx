import React, { useMemo, useRef, useState } from 'react';
import {
  Plus,
  Trash2,
  ArrowUpDown,
  Upload,
  Ban,
  Bot,
  X,
  Eraser,
} from 'lucide-react';
import type { Project } from '../types';
import { api } from '../api';
import { I18N } from '../i18n';

interface Props {
  project: Project;
  activeWeek?: number;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
}

export const ResourcesView: React.FC<Props> = ({ project, activeWeek = 1, onUpdate }) => {
  const t = I18N[project.language === 'en' ? 'en' : 'es'];
  const isEn = project.language === 'en';
  const [search, setSearch] = useState('');
  const [catFilter, setCatFilter] = useState(0);
  const [selectedId, setSelectedId] = useState(project.resources[0]?.id ?? '');
  const [creating, setCreating] = useState(false);
  const [sortAsc, setSortAsc] = useState(true);
  const [allWeeksBlock, setAllWeeksBlock] = useState(true);
  const [blockLabel, setBlockLabel] = useState(isEn ? 'Unavailable' : 'No disponible');
  const [showImport, setShowImport] = useState(false);
  const [importText, setImportText] = useState(
    'P-GARCIA;Dra. Laura García;20;Prefiere horario de mañana\nP-PEREZ;Prof. Daniel Pérez;18;No puede los viernes\nP-LOPEZ;MSc. Ana López;24;Requiere aula taller'
  );
  const [importCat, setImportCat] = useState(2);
  const csvInputRef = useRef<HTMLInputElement>(null);

  const [form, setForm] = useState({
    code: '',
    name: '',
    category: 1,
    planned: 60,
    memo: '',
    mcpNotes: '',
  });

  const selected = useMemo(
    () => project.resources.find((r) => r.id === selectedId) ?? null,
    [project.resources, selectedId]
  );

  React.useEffect(() => {
    if (!creating && selected) {
      setForm({
        code: selected.code,
        name: selected.name,
        category: selected.category,
        planned: selected.planned,
        memo: selected.memo || '',
        mcpNotes: selected.mcpNotes || '',
      });
    }
  }, [selected?.id, selected?.mcpNotes, creating]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return project.resources.filter((r) => {
      if (catFilter > 0 && r.category !== catFilter) return false;
      if (!q) return true;
      return (
        r.code.toLowerCase().includes(q) ||
        r.name.toLowerCase().includes(q) ||
        (r.mcpNotes || '').toLowerCase().includes(q)
      );
    });
  }, [project.resources, search, catFilter]);

  const linkedCourses = useMemo(() => {
    if (!selected) return [];
    return project.courses.filter((c) => c.resources.includes(selected.id));
  }, [project.courses, selected]);

  // Fast lookup of unavailable cells for selected resource on activeWeek
  const blockedCells = useMemo(() => {
    const m = new Map<string, string>();
    if (!selected) return m;
    for (const b of project.bookings) {
      if (
        b.targetType === 'resource' &&
        b.targetId === selected.id &&
        b.scope === 'timetable' &&
        b.week === activeWeek
      ) {
        m.set(`${b.day}_${b.period}`, b.label || '/////');
      }
    }
    return m;
  }, [project.bookings, selected, activeWeek]);

  const startCreate = () => {
    setCreating(true);
    setForm({
      code: catFilter === 2 ? `P-${project.resources.length + 1}` : `RES-${project.resources.length + 1}`,
      name: '',
      category: catFilter > 0 ? catFilter : 2,
      planned: 24,
      memo: '',
      mcpNotes: '',
    });
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.code.trim()) return;
    if (creating) {
      await onUpdate(
        api.createResource({
          code: form.code.trim(),
          name: form.name.trim() || form.code.trim(),
          category: form.category,
          planned: form.planned,
          memo: form.memo,
          mcpNotes: form.mcpNotes,
        }),
        isEn ? 'Resource created' : 'Recurso creado'
      );
      setCreating(false);
    } else if (selected) {
      await onUpdate(
        api.updateResource({
          ...selected,
          code: form.code.trim(),
          name: form.name.trim() || form.code.trim(),
          category: form.category,
          planned: form.planned,
          memo: form.memo,
          mcpNotes: form.mcpNotes,
        }),
        isEn ? 'Saved changes' : 'Cambios guardados'
      );
    }
  };

  const handleSort = (by: string) => {
    const nextAsc = !sortAsc;
    setSortAsc(nextAsc);
    onUpdate(api.sortResources(by, false, nextAsc));
  };

  const handleImportTeachers = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!importText.trim()) return;
    const res = await onUpdate(
      api.importTeachers(importText, importCat),
      isEn ? 'Teachers imported' : 'Profesores importados correctamente'
    );
    if (res) setShowImport(false);
  };

  const handleCsvUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    e.target.value = '';
    setImportText(await f.text());
  };

  return (
    <>
      <div className="view-subbar">
        <div className="control-group">
          <button className="btn btn-primary" onClick={startCreate}>
            <Plus size={14} /> {isEn ? 'New resource / teacher' : 'Nuevo recurso / profesor'}
          </button>
          <button className="btn" onClick={() => setShowImport(true)}>
            <Upload size={14} /> {isEn ? 'Import teachers (CSV)' : 'Importar profesores'}
          </button>
          <input
            className="input"
            placeholder={isEn ? 'Search resource or MCP note...' : 'Buscar recurso o nota MCP...'}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ width: 200 }}
          />
          <div className="pill-list">
            <button
              className={`btn btn-sm ${catFilter === 0 ? 'btn-primary' : ''}`}
              onClick={() => setCatFilter(0)}
            >
              {isEn ? 'All' : 'Todos'} ({project.resources.length})
            </button>
            {project.resourceCategories.map((c) => {
              const cnt = project.resources.filter((r) => r.category === c.index).length;
              if (cnt === 0 && c.index > 4) return null;
              return (
                <button
                  key={c.index}
                  className={`btn btn-sm ${catFilter === c.index ? 'btn-primary' : ''}`}
                  onClick={() => setCatFilter(c.index)}
                >
                  {c.shortCode}: {c.name} ({cnt})
                </button>
              );
            })}
          </div>
        </div>

        {selected && !creating && (
          <button
            className="btn btn-sm btn-danger"
            onClick={() => {
              onUpdate(
                api.deleteResource(selected.id),
                isEn ? 'Resource deleted' : 'Recurso eliminado'
              );
              setSelectedId('');
            }}
          >
            <Trash2 size={13} /> {isEn ? 'Delete' : 'Eliminar'}
          </button>
        )}
      </div>

      <div className="split-right-panel" style={{ gridTemplateColumns: '1fr 390px' }}>
        <div className="panel">
          <div className="panel-body">
            <table className="data-table">
              <thead>
                <tr>
                  <th onClick={() => handleSort('code')}>
                    {t.cols.code} <ArrowUpDown size={10} />
                  </th>
                  <th onClick={() => handleSort('name')}>
                    {t.cols.name} <ArrowUpDown size={10} />
                  </th>
                  <th>{t.cols.category}</th>
                  <th>{isEn ? 'MCP Notes / Blocks' : 'Notas MCP / Bloqueos'}</th>
                  <th className="num-cell" onClick={() => handleSort('scheduled')}>
                    {t.cols.scheduled}
                  </th>
                  <th className="num-cell" onClick={() => handleSort('links')}>
                    {t.cols.links}
                  </th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((r) => {
                  const cat = project.resourceCategories.find((c) => c.index === r.category);
                  const bkCount = project.bookings.filter(
                    (b) => b.targetType === 'resource' && b.targetId === r.id && b.week === activeWeek
                  ).length;
                  return (
                    <tr
                      key={r.id}
                      className={!creating && selectedId === r.id ? 'selected' : ''}
                      style={{ cursor: 'pointer' }}
                      onClick={() => {
                        setCreating(false);
                        setSelectedId(r.id);
                      }}
                    >
                      <td>
                        <span className="code-badge resource">{r.code}</span>
                      </td>
                      <td style={{ fontWeight: 500 }}>{r.name}</td>
                      <td style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>
                        {cat ? `${cat.shortCode}: ${cat.name}` : r.category}
                      </td>
                      <td style={{ fontSize: 11, color: 'var(--text-muted)', maxWidth: 210 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 5, flexWrap: 'wrap' }}>
                          {bkCount > 0 && (
                            <span className="stat-chip err" title={isEn ? 'Blocked slots' : 'Horas bloqueadas'}>
                              <Ban size={9} style={{ marginRight: 2 }} />
                              {bkCount}h
                            </span>
                          )}
                          {r.mcpNotes && (
                            <span
                              style={{
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                                maxWidth: 150,
                                color: '#1d4ed8',
                              }}
                              title={r.mcpNotes}
                            >
                              {r.mcpNotes}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="num-cell" style={{ color: 'var(--accent)' }}>
                        {r.scheduled}
                      </td>
                      <td className="num-cell">
                        <span className="stat-chip ok">{r.links}</span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right Panel: Edit Teacher/Resource + MCP Notes + Visual Unavailability Grid */}
        <div className="panel">
          <div className="panel-header">
            <span>
              {creating
                ? isEn
                  ? 'New Teacher / Resource'
                  : 'Nuevo Profesor / Recurso'
                : selected
                ? `${isEn ? 'Edit' : 'Editar'}: ${selected.code}`
                : isEn
                ? 'Select a resource'
                : 'Selecciona un recurso'}
            </span>
          </div>
          <form className="panel-body" style={{ padding: 12 }} onSubmit={handleSave}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <div className="form-grid">
                <div className="field">
                  <label>{t.cols.code}</label>
                  <input
                    className="input"
                    required
                    value={form.code}
                    onChange={(e) => setForm({ ...form, code: e.target.value })}
                  />
                </div>
                <div className="field">
                  <label>{t.cols.planned} (MAX)</label>
                  <input
                    className="input"
                    type="number"
                    min={0}
                    max={9999}
                    value={form.planned}
                    onChange={(e) => setForm({ ...form, planned: Number(e.target.value) })}
                  />
                </div>
              </div>
              <div className="field">
                <label>{t.cols.name}</label>
                <input
                  className="input"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </div>
              <div className="field">
                <label>{t.cols.category}</label>
                <select
                  className="select"
                  value={form.category}
                  onChange={(e) => setForm({ ...form, category: Number(e.target.value) })}
                >
                  {project.resourceCategories.map((c) => (
                    <option key={c.index} value={c.index}>
                      {c.shortCode}: {c.name}
                    </option>
                  ))}
                </select>
              </div>

              {/* MCP Notes for AI Agent */}
              <div className="field">
                <label style={{ display: 'flex', alignItems: 'center', gap: 5, color: '#1d4ed8' }}>
                  <Bot size={13} />
                  {isEn ? 'Notes for MCP / AI Agent' : 'Notas e instrucciones para MCP (Agente)'}
                </label>
                <textarea
                  className="input"
                  rows={2}
                  placeholder={
                    isEn
                      ? 'e.g. Prefers mornings, max 3 consecutive hours, needs projector...'
                      : 'Ej. Prefiere mañanas, no dar clase los viernes a última hora, necesita taller...'
                  }
                  value={form.mcpNotes}
                  onChange={(e) => setForm({ ...form, mcpNotes: e.target.value })}
                />
              </div>

              <button className="btn btn-primary" type="submit" style={{ justifyContent: 'center' }}>
                {creating
                  ? isEn
                    ? 'Create'
                    : 'Crear'
                  : isEn
                  ? 'Save changes'
                  : 'Guardar cambios'}
              </button>

              {/* Visual Unavailability / "No puede un día a una hora" Matrix */}
              {!creating && selected && (
                <div
                  style={{
                    marginTop: 4,
                    padding: 10,
                    borderRadius: 8,
                    border: '1px solid var(--border)',
                    background: '#fafbfd',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      marginBottom: 6,
                      flexWrap: 'wrap',
                      gap: 4,
                    }}
                  >
                    <strong style={{ fontSize: 11.5, display: 'flex', alignItems: 'center', gap: 4 }}>
                      <Ban size={12} color="#dc2626" />
                      {isEn ? 'Unavailability (click to block)' : 'Horas no disponibles (clic para bloquear)'}
                    </strong>
                    <label style={{ fontSize: 11, display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={allWeeksBlock}
                        onChange={(e) => setAllWeeksBlock(e.target.checked)}
                      />
                      {isEn ? 'All weeks' : 'Todas las sem.'}
                    </label>
                  </div>

                  <div style={{ display: 'flex', gap: 4, marginBottom: 7, flexWrap: 'wrap' }}>
                    <input
                      className="input"
                      style={{ flex: 1, minWidth: 95, padding: '3px 6px', fontSize: 11 }}
                      value={blockLabel}
                      onChange={(e) => setBlockLabel(e.target.value)}
                      placeholder={isEn ? 'Reason...' : 'Motivo...'}
                    />
                    <button
                      type="button"
                      className="btn btn-sm"
                      onClick={() =>
                        onUpdate(
                          api.batchBookings({
                            targetId: selected.id,
                            mode: 'block_afternoons',
                            week: activeWeek,
                            allWeeks: allWeeksBlock,
                            label: blockLabel,
                          })
                        )
                      }
                    >
                      {isEn ? 'Block PM' : 'Bloq. tardes'}
                    </button>
                    <button
                      type="button"
                      className="btn btn-sm"
                      title={isEn ? 'Clear all blocks' : 'Quitar bloqueos'}
                      onClick={() =>
                        onUpdate(
                          api.batchBookings({
                            targetId: selected.id,
                            mode: 'clear_all',
                            week: activeWeek,
                            allWeeks: allWeeksBlock,
                          })
                        )
                      }
                    >
                      <Eraser size={11} />
                    </button>
                  </div>

                  <div style={{ overflowX: 'auto' }}>
                    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 10.5 }}>
                      <thead>
                        <tr>
                          <th style={{ width: 32, textAlign: 'left', color: 'var(--text-faint)' }}>B</th>
                          {Array.from({ length: project.time.maxDays }, (_, di) => (
                            <th
                              key={di}
                              style={{
                                textAlign: 'center',
                                cursor: 'pointer',
                                padding: '2px 0',
                                color: 'var(--text-muted)',
                              }}
                              title={isEn ? 'Click to block entire day' : 'Clic para bloquear el día entero'}
                              onClick={() =>
                                onUpdate(
                                  api.batchBookings({
                                    targetId: selected.id,
                                    mode: 'block_day',
                                    week: activeWeek,
                                    allWeeks: allWeeksBlock,
                                    day: di + 1,
                                    label: blockLabel,
                                  })
                                )
                              }
                            >
                              {(project.time.days[di] || `D${di + 1}`).slice(0, 3)}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {Array.from({ length: project.time.maxPeriods }, (_, pi) => {
                          const h = pi + 1;
                          return (
                            <tr key={h}>
                              <td style={{ fontWeight: 700, color: 'var(--text-faint)', padding: '2px 0' }}>
                                B{h}
                              </td>
                              {Array.from({ length: project.time.maxDays }, (_, di) => {
                                const d = di + 1;
                                const lbl = blockedCells.get(`${d}_${h}`);
                                const isBlocked = Boolean(lbl);
                                return (
                                  <td key={d} style={{ padding: 2 }}>
                                    <button
                                      type="button"
                                      onClick={() =>
                                        onUpdate(
                                          api.batchBookings({
                                            targetId: selected.id,
                                            mode: 'toggle_cell',
                                            week: activeWeek,
                                            allWeeks: allWeeksBlock,
                                            day: d,
                                            period: h,
                                            label: blockLabel,
                                          })
                                        )
                                      }
                                      title={
                                        isBlocked
                                          ? `${lbl} (${isEn ? 'click to unblock' : 'clic para liberar'})`
                                          : isEn
                                          ? 'Available (click to block)'
                                          : 'Disponible (clic para marcar no disponible)'
                                      }
                                      style={{
                                        width: '100%',
                                        height: 20,
                                        borderRadius: 4,
                                        border: isBlocked ? '1px solid #ef4444' : '1px solid #cbd5e1',
                                        background: isBlocked ? '#fee2e2' : '#ffffff',
                                        color: isBlocked ? '#b91c1c' : '#94a3b8',
                                        fontSize: 10,
                                        fontWeight: 700,
                                        cursor: 'pointer',
                                        padding: 0,
                                      }}
                                    >
                                      {isBlocked ? '✕' : '·'}
                                    </button>
                                  </td>
                                );
                              })}
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {!creating && selected && linkedCourses.length > 0 && (
                <div>
                  <div style={{ fontSize: 11.5, fontWeight: 700, color: 'var(--text-muted)', marginBottom: 5 }}>
                    {isEn ? 'Linked courses' : 'Cursos vinculados'} ({linkedCourses.length})
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                    {linkedCourses.map((c) => (
                      <div
                        key={c.id}
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          padding: '4px 8px',
                          borderRadius: 6,
                          background: '#f8fafc',
                          border: '1px solid var(--border)',
                          fontSize: 11.5,
                        }}
                      >
                        <span>
                          <strong>{c.code}</strong> · {c.name}
                        </span>
                        <span className={`stat-chip ${c.left === 0 ? 'ok' : 'warn'}`}>
                          {c.scheduled}/{c.allocated}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </form>
        </div>
      </div>

      {/* Bulk Import Teachers Modal */}
      {showImport && (
        <div className="modal-overlay" onClick={() => setShowImport(false)}>
          <div className="modal-card" style={{ maxWidth: 620 }} onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>
                <Upload size={16} color="#2563eb" />
                {isEn ? 'Import Teachers / Resources (CSV or Text)' : 'Importar Profesores / Recursos (CSV o Texto)'}
              </h3>
              <button className="btn btn-icon" onClick={() => setShowImport(false)}>
                <X size={15} />
              </button>
            </div>
            <form onSubmit={handleImportTeachers}>
              <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8 }}>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                    {isEn
                      ? 'One per line: CODE;Name;MaxHours;MCP Notes (or just full names).'
                      : 'Uno por línea: CODIGO;Nombre;HorasMax;Notas MCP (o simplemente un nombre por línea).'}
                  </div>
                  <input
                    ref={csvInputRef}
                    type="file"
                    accept=".csv,.txt,.tsv"
                    style={{ display: 'none' }}
                    onChange={handleCsvUpload}
                  />
                  <button
                    type="button"
                    className="btn btn-sm"
                    onClick={() => csvInputRef.current?.click()}
                  >
                    <Upload size={12} /> {isEn ? 'Load .csv/.txt' : 'Cargar .csv/.txt'}
                  </button>
                </div>
                <div className="field">
                  <label>{isEn ? 'Target category' : 'Categoría destino'}</label>
                  <select className="select" value={importCat} onChange={(e) => setImportCat(Number(e.target.value))}>
                    {project.resourceCategories.map((c) => (
                      <option key={c.index} value={c.index}>
                        {c.shortCode}: {c.name}
                      </option>
                    ))}
                  </select>
                </div>
                <textarea
                  className="input"
                  rows={8}
                  style={{ fontFamily: 'monospace', fontSize: 12 }}
                  value={importText}
                  onChange={(e) => setImportText(e.target.value)}
                />
              </div>
              <div className="modal-footer">
                <button type="button" className="btn" onClick={() => setShowImport(false)}>
                  {isEn ? 'Cancel' : 'Cancelar'}
                </button>
                <button type="submit" className="btn btn-primary">
                  {isEn ? 'Import list' : 'Importar lista'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
};
