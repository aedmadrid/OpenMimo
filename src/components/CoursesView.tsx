import React, { useMemo, useState } from 'react';
import {
  Plus,
  Trash2,
  Copy,
  Split,
  Lock,
  Unlock,
  ArrowUpDown,
  Eraser,
  BookOpenCheck,
} from 'lucide-react';
import type { Course, Project } from '../types';
import { api } from '../api';
import { I18N } from '../i18n';

interface Props {
  project: Project;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
  onOpenGuide: (guideId?: string, linkCourseId?: string) => void;
}

const PALETTE = ['#f59e0b', '#3b82f6', '#10b981', '#8b5cf6', '#ec4899', '#0ea5e9', '#ef4444', '#14b8a6', '#6366f1', '#64748b'];

export const CoursesView: React.FC<Props> = ({ project, onUpdate, onOpenGuide }) => {
  const t = I18N[project.language === 'en' ? 'en' : 'es'];
  const isEn = project.language === 'en';
  const [search, setSearch] = useState('');
  const [catFilter, setCatFilter] = useState<number>(0);
  const [selectedId, setSelectedId] = useState<string>(project.courses[0]?.id ?? '');
  const [creating, setCreating] = useState(false);
  const [sortAsc, setSortAsc] = useState(true);

  const [form, setForm] = useState({
    code: '',
    name: '',
    category: 1,
    planned: 16,
    minConsecutive: 0,
    maxConsecutive: 0,
    color: '#3b82f6',
    locked: false,
    memo: '',
    guideId: '',
    resources: [] as string[],
  });

  const selectedCourse = useMemo(
    () => project.courses.find((c) => c.id === selectedId) ?? null,
    [project.courses, selectedId]
  );

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return project.courses.filter((c) => {
      if (catFilter > 0 && c.category !== catFilter) return false;
      if (!q) return true;
      return c.code.toLowerCase().includes(q) || c.name.toLowerCase().includes(q);
    });
  }, [project.courses, search, catFilter]);

  const resMap = useMemo(
    () => new Map(project.resources.map((r) => [r.id, r])),
    [project.resources]
  );

  const loadIntoForm = (c: Course) => {
    setCreating(false);
    setSelectedId(c.id);
    setForm({
      code: c.code,
      name: c.name,
      category: c.category,
      planned: c.planned,
      minConsecutive: c.minConsecutive || 0,
      maxConsecutive: c.maxConsecutive || 0,
      color: c.color || '#3b82f6',
      locked: c.locked,
      memo: c.memo || '',
      guideId: c.guideId || '',
      resources: [...c.resources],
    });
  };

  // Sync form when selectedCourse changes and not creating
  React.useEffect(() => {
    if (!creating && selectedCourse) {
      setForm({
        code: selectedCourse.code,
        name: selectedCourse.name,
        category: selectedCourse.category,
        planned: selectedCourse.planned,
        minConsecutive: selectedCourse.minConsecutive || 0,
        maxConsecutive: selectedCourse.maxConsecutive || 0,
        color: selectedCourse.color || '#3b82f6',
        locked: selectedCourse.locked,
        memo: selectedCourse.memo || '',
        guideId: selectedCourse.guideId || '',
        resources: [...selectedCourse.resources],
      });
    }
  }, [selectedCourse?.id, creating]);

  const startCreate = () => {
    setCreating(true);
    setForm({
      code: `CUR-${project.courses.length + 1}`,
      name: '',
      category: 1,
      planned: project.time.maxWeeks * 2,
      minConsecutive: 0,
      maxConsecutive: 0,
      color: PALETTE[project.courses.length % PALETTE.length],
      locked: false,
      memo: '',
      guideId: '',
      resources: [],
    });
  };

  const toggleResource = (rid: string) => {
    setForm((prev) => ({
      ...prev,
      resources: prev.resources.includes(rid)
        ? prev.resources.filter((x) => x !== rid)
        : [...prev.resources, rid],
    }));
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.code.trim()) return;
    if (creating) {
      await onUpdate(
        api.createCourse({
          code: form.code.trim(),
          name: form.name.trim() || form.code.trim(),
          category: form.category,
          planned: form.planned,
          minConsecutive: form.minConsecutive || 0,
          maxConsecutive: form.maxConsecutive || 0,
          color: form.color,
          locked: form.locked,
          memo: form.memo,
          guideId: form.guideId || undefined,
          resources: form.resources,
        }),
        isEn ? 'Event created' : 'Curso creado'
      );
      setCreating(false);
    } else if (selectedCourse) {
      await onUpdate(
        api.updateCourse({
          ...selectedCourse,
          code: form.code.trim(),
          name: form.name.trim() || form.code.trim(),
          category: form.category,
          planned: form.planned,
          minConsecutive: form.minConsecutive || 0,
          maxConsecutive: form.maxConsecutive || 0,
          color: form.color,
          locked: form.locked,
          memo: form.memo,
          guideId: form.guideId || undefined,
          resources: form.resources,
        }),
        isEn ? 'Event saved' : 'Cambios guardados'
      );
    }
  };

  const handleSort = (by: string) => {
    const nextAsc = !sortAsc;
    setSortAsc(nextAsc);
    onUpdate(api.courseAction({ action: 'sort', by, byCategory: false, ascending: nextAsc }));
  };

  return (
    <>
      <div className="view-subbar">
        <div className="control-group">
          <button className="btn btn-primary" onClick={startCreate}>
            <Plus size={14} /> {isEn ? 'New event' : 'Nuevo curso'}
          </button>
          <input
            className="input"
            placeholder={isEn ? 'Search code or name...' : 'Buscar código o nombre...'}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ width: 210 }}
          />
          <select
            className="select"
            value={catFilter}
            onChange={(e) => setCatFilter(Number(e.target.value))}
          >
            <option value={0}>{isEn ? 'All categories' : 'Todas las categorías'}</option>
            {project.courseCategories.map((c) => (
              <option key={c.index} value={c.index}>
                {c.shortCode}: {c.name}
              </option>
            ))}
          </select>
        </div>

        <div className="control-group">
          {selectedCourse && !creating && (
            <>
              <button
                className="btn btn-sm"
                onClick={() =>
                  onUpdate(
                    api.courseAction({ action: 'duplicate', courseId: selectedCourse.id }),
                    isEn ? 'Event duplicated' : 'Curso duplicado'
                  )
                }
              >
                <Copy size={13} /> {isEn ? 'Duplicate' : 'Duplicar'}
              </button>
              <button
                className="btn btn-sm"
                onClick={() =>
                  onUpdate(
                    api.courseAction({ action: 'split', courseId: selectedCourse.id }),
                    isEn ? 'Event split in two' : 'Curso dividido en dos'
                  )
                }
              >
                <Split size={13} /> {isEn ? 'Split' : 'Dividir'}
              </button>
              <button
                className="btn btn-sm"
                onClick={() =>
                  onUpdate(
                    api.courseAction({ action: 'unschedule', courseId: selectedCourse.id }),
                    isEn ? 'Event unscheduled' : 'Horario del curso limpiado'
                  )
                }
              >
                <Eraser size={13} /> {isEn ? 'Unschedule' : 'Desprogramar'}
              </button>
              <button
                className="btn btn-sm btn-danger"
                onClick={() => {
                  onUpdate(
                    api.deleteCourse(selectedCourse.id),
                    isEn ? 'Event deleted' : 'Curso eliminado'
                  );
                  setSelectedId('');
                }}
              >
                <Trash2 size={13} /> {isEn ? 'Delete' : 'Eliminar'}
              </button>
            </>
          )}
        </div>
      </div>

      <div className="split-right-panel">
        {/* Left Table */}
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
                  <th>{isEn ? 'Resources (Collection)' : 'Recursos vinculados'}</th>
                  <th className="num-cell" onClick={() => handleSort('planned')}>
                    {t.cols.planned}
                  </th>
                  <th className="num-cell" onClick={() => handleSort('allocated')}>
                    {t.cols.allocated}
                  </th>
                  <th className="num-cell" onClick={() => handleSort('scheduled')}>
                    {t.cols.scheduled}
                  </th>
                  <th className="num-cell" onClick={() => handleSort('left')}>
                    {t.cols.left}
                  </th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((c) => (
                  <tr
                    key={c.id}
                    className={!creating && selectedId === c.id ? 'selected' : ''}
                    style={{ cursor: 'pointer' }}
                    onClick={() => loadIntoForm(c)}
                  >
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                        <span
                          style={{
                            width: 8,
                            height: 8,
                            borderRadius: '50%',
                            background: c.color || '#f59e0b',
                            flexShrink: 0,
                          }}
                        />
                        <span className="code-badge course">{c.code}</span>
                        {c.locked && <Lock size={11} color="#64748b" />}
                      </div>
                    </td>
                    <td style={{ fontWeight: 500 }}>
                      {c.name}
                      {((c.minConsecutive ?? 0) > 0 || (c.maxConsecutive ?? 0) > 0) && (
                        <span className="stat-chip ok" style={{ marginLeft: 6, fontSize: 10 }}>
                          {c.minConsecutive || 1}–{c.maxConsecutive || '∞'}h
                        </span>
                      )}
                    </td>
                    <td>
                      <div className="pill-list">
                        {c.resources.map((rid) => {
                          const r = resMap.get(rid);
                          if (!r) return null;
                          return (
                            <span
                              key={rid}
                              className="code-badge resource"
                              style={{ padding: '1px 5px', fontSize: 10.5 }}
                            >
                              {r.code}
                            </span>
                          );
                        })}
                      </div>
                    </td>
                    <td className="num-cell">{c.planned}</td>
                    <td className="num-cell">{c.allocated}</td>
                    <td className="num-cell" style={{ color: 'var(--accent)' }}>
                      {c.scheduled}
                    </td>
                    <td className="num-cell">
                      <span
                        className={`stat-chip ${
                          c.left === 0 ? 'ok' : c.left > 0 ? 'warn' : 'err'
                        }`}
                      >
                        {c.left}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right Editor & Resource Collection Picker */}
        <div className="panel">
          <div className="panel-header">
            <span>
              {creating
                ? isEn
                  ? 'New Event'
                  : 'Nuevo Curso'
                : selectedCourse
                ? `${isEn ? 'Edit' : 'Editar'}: ${selectedCourse.code}`
                : isEn
                ? 'Select an event'
                : 'Selecciona un curso'}
            </span>
            {!creating && selectedCourse && (
              <button
                className="btn btn-sm"
                type="button"
                onClick={() => setForm((f) => ({ ...f, locked: !f.locked }))}
              >
                {form.locked ? <Lock size={12} /> : <Unlock size={12} />}
                {form.locked ? (isEn ? 'Locked' : 'Bloqueado') : isEn ? 'Unlocked' : 'Libre'}
              </button>
            )}
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
                    max={999}
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

              <div className="form-grid">
                <div className="field">
                  <label>{isEn ? 'Min consecutive hrs (0=auto)' : 'Mín. horas seguidas (0=auto)'}</label>
                  <input
                    className="input"
                    type="number"
                    min={0}
                    max={12}
                    value={form.minConsecutive}
                    onChange={(e) => setForm({ ...form, minConsecutive: Math.max(0, Number(e.target.value)) })}
                  />
                </div>
                <div className="field">
                  <label>{isEn ? 'Max consecutive hrs (0=auto)' : 'Máx. horas seguidas (0=auto)'}</label>
                  <input
                    className="input"
                    type="number"
                    min={0}
                    max={12}
                    value={form.maxConsecutive}
                    onChange={(e) => setForm({ ...form, maxConsecutive: Math.max(0, Number(e.target.value)) })}
                  />
                </div>
              </div>

              <div className="form-grid">
                <div className="field">
                  <label>{t.cols.category}</label>
                  <select
                    className="select"
                    value={form.category}
                    onChange={(e) => setForm({ ...form, category: Number(e.target.value) })}
                  >
                    {project.courseCategories.map((c) => (
                      <option key={c.index} value={c.index}>
                        {c.shortCode}: {c.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="field">
                  <label>Color</label>
                  <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', paddingTop: 4 }}>
                    {PALETTE.map((col) => (
                      <button
                        type="button"
                        key={col}
                        onClick={() => setForm({ ...form, color: col })}
                        style={{
                          width: 18,
                          height: 18,
                          borderRadius: '50%',
                          background: col,
                          border: form.color === col ? '2px solid #0f172a' : '1px solid #cbd5e1',
                          cursor: 'pointer',
                        }}
                      />
                    ))}
                  </div>
                </div>
              </div>

              {/* Visual Collection Picker grouped by Resource Category */}
              <div className="field">
                <label>
                  {isEn ? 'Assigned Resources' : 'Recursos asignados (Colección)'} ({form.resources.length})
                </label>
                <div
                  style={{
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    padding: 8,
                    maxHeight: 240,
                    overflowY: 'auto',
                    background: '#fafbfd',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 8,
                  }}
                >
                  {project.resourceCategories.map((cat) => {
                    const items = project.resources.filter((r) => r.category === cat.index);
                    if (items.length === 0) return null;
                    return (
                      <div key={cat.index}>
                        <div
                          style={{
                            fontSize: 10.5,
                            fontWeight: 700,
                            color: 'var(--text-faint)',
                            textTransform: 'uppercase',
                            marginBottom: 4,
                          }}
                        >
                          {cat.shortCode} · {cat.name}
                        </div>
                        <div className="pill-list">
                          {items.map((r) => {
                            const active = form.resources.includes(r.id);
                            return (
                              <button
                                type="button"
                                key={r.id}
                                className={`btn btn-sm ${active ? 'btn-primary' : ''}`}
                                style={{ padding: '2px 7px', fontSize: 11 }}
                                onClick={() => toggleResource(r.id)}
                                title={r.name}
                              >
                                {r.code}
                              </button>
                            );
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Global Subject (Guía Docente) containing specific sessions like DNOP1, DNOP2, DNOP3 */}
              <div className="field">
                <label>
                  {isEn ? 'Global subject (Teaching Guide)' : 'Asignatura global (Guía docente)'}
                </label>
                <div style={{ display: 'flex', gap: 6 }}>
                  <select
                    className="select"
                    style={{ flex: 1 }}
                    value={form.guideId}
                    onChange={(e) => setForm({ ...form, guideId: e.target.value })}
                  >
                    <option value="">{isEn ? '— None —' : '— Ninguna —'}</option>
                    {(project.guides || []).map((g) => (
                      <option key={g.id} value={g.id}>
                        {g.title} ({g.creditsEcts} ECTS)
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    className="btn btn-sm"
                    onClick={() => onOpenGuide(form.guideId || undefined, selectedCourse?.id)}
                  >
                    <BookOpenCheck size={13} /> {isEn ? 'Guide' : 'Guía'}
                  </button>
                </div>
              </div>

              <button className="btn btn-primary" type="submit" style={{ justifyContent: 'center' }}>
                {creating
                  ? isEn
                    ? 'Create event'
                    : 'Crear curso'
                  : isEn
                  ? 'Save changes'
                  : 'Guardar cambios'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
};
