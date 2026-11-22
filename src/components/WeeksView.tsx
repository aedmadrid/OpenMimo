import React, { useMemo, useState } from 'react';
import { Sparkles, AlertTriangle } from 'lucide-react';
import type { Project } from '../types';
import { api } from '../api';
import { I18N } from '../i18n';

interface Props {
  project: Project;
  activeWeek: number;
  setActiveWeek: (w: number) => void;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
}

export const WeeksView: React.FC<Props> = ({
  project,
  activeWeek,
  setActiveWeek,
  onUpdate,
}) => {
  const t = I18N[project.language === 'en' ? 'en' : 'es'];
  const [mode, setMode] = useState<'numbers' | 'pct' | 'sched' | 'gantt'>('numbers');
  const [editing, setEditing] = useState<{ courseId: string; week: number; val: string } | null>(
    null
  );
  const [search, setSearch] = useState('');

  const allocLookup = useMemo(() => {
    const m = new Map<string, number>();
    for (const wa of project.weekAllocations) {
      m.set(`${wa.courseId}_${wa.week}`, wa.periods);
    }
    return m;
  }, [project.weekAllocations]);

  const schedLookup = useMemo(() => {
    const m = new Map<string, number>();
    for (const s of project.slots) {
      const k = `${s.courseId}_${s.week}`;
      m.set(k, (m.get(k) ?? 0) + 1);
    }
    return m;
  }, [project.slots]);

  const filteredCourses = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return project.courses;
    return project.courses.filter(
      (c) => c.code.toLowerCase().includes(q) || c.name.toLowerCase().includes(q)
    );
  }, [project.courses, search]);

  // Resource weekly load summary
  const resWeeklyLoad = useMemo(() => {
    const out: Record<string, Record<number, number>> = {};
    for (const r of project.resources) {
      out[r.id] = {};
    }
    for (const c of project.courses) {
      for (let w = 1; w <= project.time.maxWeeks; w++) {
        const per = allocLookup.get(`${c.id}_${w}`) ?? 0;
        if (per > 0) {
          for (const rid of c.resources) {
            if (!out[rid]) out[rid] = {};
            out[rid][w] = (out[rid][w] ?? 0) + per;
          }
        }
      }
    }
    return out;
  }, [project.resources, project.courses, project.time.maxWeeks, allocLookup]);

  const commitEdit = async () => {
    if (!editing) return;
    const n = Math.max(0, parseInt(editing.val, 10) || 0);
    const { courseId, week } = editing;
    setEditing(null);
    await onUpdate(api.setWeekAllocation(courseId, week, n));
  };

  return (
    <>
      <div className="view-subbar">
        <div className="control-group">
          <strong style={{ fontSize: 13 }}>{t.weeks.title}</strong>
          <input
            className="input"
            placeholder={project.language === 'en' ? 'Filter events...' : 'Filtrar cursos...'}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ width: 190 }}
          />
        </div>

        <div className="control-group">
          <div className="view-switcher">
            {(
              [
                ['numbers', t.weeks.showNumbers],
                ['sched', t.weeks.showSched],
                ['pct', t.weeks.showPct],
                ['gantt', t.weeks.showGantt],
              ] as const
            ).map(([k, lbl]) => (
              <button
                key={k}
                className={`view-tab ${mode === k ? 'active' : ''}`}
                style={{ padding: '4px 9px', fontSize: 11.5 }}
                onClick={() => setMode(k)}
              >
                {lbl}
              </button>
            ))}
          </div>

          <button
            className="btn btn-sm btn-primary"
            onClick={() =>
              onUpdate(
                api.courseAction({ action: 'auto_allocate', maxWeek: project.time.maxWeeks }),
                project.language === 'en'
                  ? 'Auto-allocated remaining periods across weeks'
                  : 'Periodos restantes distribuidos automáticamente'
              )
            }
          >
            <Sparkles size={13} /> {t.weeks.autoAlloc}
          </button>
        </div>
      </div>

      <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column', gap: 10 }}>
        {/* Main Course × Week Matrix */}
        <div className="panel" style={{ flex: 2 }}>
          <div className="panel-body">
            <table className="data-table">
              <thead>
                <tr>
                  <th style={{ width: 115 }}>{t.cols.code}</th>
                  <th style={{ minWidth: 180 }}>{t.cols.name}</th>
                  <th className="num-cell" style={{ width: 58 }}>
                    {t.cols.planned}
                  </th>
                  <th className="num-cell" style={{ width: 58 }}>
                    {t.cols.allocated}
                  </th>
                  <th className="num-cell" style={{ width: 58 }}>
                    {t.cols.scheduled}
                  </th>
                  {project.time.weeks.map((w) => (
                    <th
                      key={w.index}
                      style={{
                        textAlign: 'center',
                        background: activeWeek === w.index ? '#eff6ff' : undefined,
                        color: activeWeek === w.index ? 'var(--accent)' : undefined,
                      }}
                      onClick={() => setActiveWeek(w.index)}
                    >
                      {w.name}
                      {w.length > 1 ? ` ×${w.length}` : ''}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {filteredCourses.map((c) => {
                  const unalloc = c.planned - c.allocated;
                  return (
                    <tr key={c.id}>
                      <td>
                        <span className="code-badge course">{c.code}</span>
                      </td>
                      <td style={{ fontWeight: 500 }}>{c.name}</td>
                      <td className="num-cell">{c.planned}</td>
                      <td
                        className="num-cell"
                        style={{
                          color:
                            unalloc === 0
                              ? 'var(--ok-text)'
                              : unalloc < 0
                              ? 'var(--err-text)'
                              : '#b45309',
                        }}
                      >
                        {c.allocated}
                      </td>
                      <td className="num-cell" style={{ color: 'var(--accent)' }}>
                        {c.scheduled}
                      </td>
                      {project.time.weeks.map((w) => {
                        const val = allocLookup.get(`${c.id}_${w.index}`) ?? 0;
                        const sc = schedLookup.get(`${c.id}_${w.index}`) ?? 0;
                        const isEd = editing?.courseId === c.id && editing?.week === w.index;

                        if (isEd) {
                          return (
                            <td key={w.index} style={{ textAlign: 'center', padding: 3 }}>
                              <input
                                autoFocus
                                type="number"
                                min={0}
                                max={99}
                                className="input"
                                style={{ width: 52, textAlign: 'center', padding: '3px 4px' }}
                                value={editing.val}
                                onChange={(e) => setEditing({ ...editing, val: e.target.value })}
                                onBlur={commitEdit}
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter') commitEdit();
                                  if (e.key === 'Escape') setEditing(null);
                                }}
                              />
                            </td>
                          );
                        }

                        if (mode === 'gantt') {
                          const ratio = c.planned > 0 ? Math.min(1, val / Math.max(1, c.planned / project.time.maxWeeks * 1.5)) : 0;
                          return (
                            <td
                              key={w.index}
                              style={{ textAlign: 'center', cursor: 'pointer' }}
                              onClick={() =>
                                setEditing({ courseId: c.id, week: w.index, val: String(val) })
                              }
                            >
                              {val > 0 ? (
                                <div
                                  style={{
                                    height: 18,
                                    borderRadius: 4,
                                    background: c.color || '#3b82f6',
                                    opacity: 0.35 + ratio * 0.65,
                                    color: '#fff',
                                    fontSize: 10.5,
                                    fontWeight: 700,
                                    lineHeight: '18px',
                                  }}
                                >
                                  {val}
                                </div>
                              ) : (
                                <span style={{ color: '#cbd5e1' }}>·</span>
                              )}
                            </td>
                          );
                        }

                        let label = val > 0 ? String(val) : '—';
                        if (mode === 'sched') {
                          label = val > 0 || sc > 0 ? `${sc}/${val}` : '—';
                        } else if (mode === 'pct') {
                          label = val > 0 ? `${Math.round((sc * 100) / val)}%` : '—';
                        }

                        return (
                          <td key={w.index} style={{ textAlign: 'center', padding: 4 }}>
                            <button
                              className={`week-cell-btn ${
                                val === 0 ? 'empty' : sc >= val && val > 0 ? 'done' : ''
                              }`}
                              onClick={() =>
                                setEditing({ courseId: c.id, week: w.index, val: String(val) })
                              }
                            >
                              {label}
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

        {/* Bottom Linked Resource Weekly Load Panel (Mimosa SumGrid) */}
        <div className="panel" style={{ flex: 1 }}>
          <div className="panel-header">
            <span>{t.weeks.resourceTotals}</span>
            <span style={{ fontSize: 11, color: 'var(--text-faint)' }}>
              {project.resources.length} {t.views.resources.toLowerCase()}
            </span>
          </div>
          <div className="panel-body">
            <table className="data-table">
              <thead>
                <tr>
                  <th style={{ width: 115 }}>{t.cols.code}</th>
                  <th>{t.cols.name}</th>
                  <th style={{ width: 90 }}>{t.cols.category}</th>
                  {project.time.weeks.map((w) => (
                    <th key={w.index} style={{ textAlign: 'center' }}>
                      {w.name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {project.resources
                  .filter((r) => !project.resourceCategories.find((c) => c.index === r.category)?.noTimetables)
                  .map((r) => {
                    const cat = project.resourceCategories.find((c) => c.index === r.category);
                    const lim = cat?.weeklyLimit ?? 0;
                    return (
                      <tr key={r.id}>
                        <td>
                          <span className="code-badge resource">{r.code}</span>
                        </td>
                        <td>{r.name}</td>
                        <td style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>
                          {cat?.name ?? r.category}
                        </td>
                        {project.time.weeks.map((w) => {
                          const load = resWeeklyLoad[r.id]?.[w.index] ?? 0;
                          const over = lim > 0 && load > lim;
                          return (
                            <td
                              key={w.index}
                              style={{
                                textAlign: 'center',
                                fontWeight: 700,
                                color: over ? 'var(--err-text)' : load > 0 ? 'var(--text-main)' : '#cbd5e1',
                                background: over ? 'var(--err-bg)' : undefined,
                              }}
                              title={over ? `${t.weeks.overLimit} (${load} > ${lim})` : ''}
                            >
                              {over && <AlertTriangle size={11} style={{ marginRight: 2 }} />}
                              {load > 0 ? load : '—'}
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
      </div>
    </>
  );
};
