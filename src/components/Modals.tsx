import React, { useEffect, useState } from 'react';
import {
  X,
  Sparkles,
  DoorOpen,
  Settings,
  BarChart3,
  FileText,
  Download,
  AlertTriangle,
  CheckCircle2,
  Wand2,
  Copy,
  Trash2,
} from 'lucide-react';
import type {
  ModalKind,
  OptimizeOptions,
  OptimizeResult,
  Project,
  ProjectStats,
  Resource,
  SlotAssignment,
} from '../types';
import { api } from '../api';

interface Props {
  modal: ModalKind;
  onClose: () => void;
  project: Project;
  stats: ProjectStats;
  activeWeek: number;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
}

function downloadText(fileName: string, content: string, mime = 'text/plain;charset=utf-8') {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = fileName;
  a.click();
  URL.revokeObjectURL(url);
}

export const Modals: React.FC<Props> = ({
  modal,
  onClose,
  project,
  stats,
  activeWeek,
  onUpdate,
}) => {
  if (!modal) return null;
  const isEn = project.language === 'en';

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className={`modal-card ${modal === 'reports' || modal === 'rooms' ? 'wide' : ''}`}
        onClick={(e) => e.stopPropagation()}
      >
        {modal === 'optimize' && (
          <OptimizeModal
            project={project}
            stats={stats}
            activeWeek={activeWeek}
            isEn={isEn}
            onUpdate={onUpdate}
            onClose={onClose}
          />
        )}
        {modal === 'rooms' && (
          <RoomsModal
            project={project}
            activeWeek={activeWeek}
            isEn={isEn}
            onUpdate={onUpdate}
            onClose={onClose}
          />
        )}
        {modal === 'options' && (
          <OptionsModal project={project} isEn={isEn} onUpdate={onUpdate} onClose={onClose} />
        )}
        {modal === 'stats' && (
          <StatsModal project={project} stats={stats} isEn={isEn} onClose={onClose} />
        )}
        {modal === 'reports' && (
          <ReportsModal
            project={project}
            activeWeek={activeWeek}
            isEn={isEn}
            onClose={onClose}
          />
        )}
      </div>
    </div>
  );
};

/* ---------------- 1. OPTIMISE MODAL (TOptimisationDlg) ---------------- */
const OptimizeModal: React.FC<{
  project: Project;
  stats: ProjectStats;
  activeWeek: number;
  isEn: boolean;
  onUpdate: (p: Promise<any>, msg?: string) => Promise<any>;
  onClose: () => void;
}> = ({ project, stats, activeWeek, isEn, onUpdate, onClose }) => {
  const [tab, setTab] = useState<'tools' | 'selections' | 'stats'>('tools');
  const [opts, setOpts] = useState<OptimizeOptions>({
    action: 'initial',
    fromWeek: activeWeek,
    toWeek: project.time.maxWeeks,
    minDay: 1,
    maxDay: project.time.maxDays,
    minPeriod: 1,
    maxPeriod: project.time.maxPeriods,
    minBlock: 1,
    maxBlock: 2,
    passes: 6,
    includeLocked: false,
    bookingsWithGaps: false,
    gapWeights: Object.fromEntries(project.resourceCategories.map((c) => [c.index, c.gapWeight || 1])),
    randSeed: 38746,
  });
  const [lastResult, setLastResult] = useState<OptimizeResult | null>(null);
  const [busy, setBusy] = useState(false);

  const run = async (action: OptimizeOptions['action']) => {
    setBusy(true);
    try {
      const payload = await onUpdate(api.optimize({ ...opts, action }));
      if (payload?.result) {
        setLastResult(payload.result);
        setTab('stats');
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="modal-header">
        <h3>
          <Sparkles size={17} color="#2563eb" />
          {isEn ? 'Optimise Timetables' : 'Optimizar Horarios'}
        </h3>
        <button className="btn btn-icon" onClick={onClose}>
          <X size={15} />
        </button>
      </div>
      <div className="modal-body">
        <div className="tab-bar">
          <button className={`tab-btn ${tab === 'tools' ? 'active' : ''}`} onClick={() => setTab('tools')}>
            {isEn ? 'Tools' : 'Herramientas'}
          </button>
          <button
            className={`tab-btn ${tab === 'selections' ? 'active' : ''}`}
            onClick={() => setTab('selections')}
          >
            {isEn ? 'Selections & Weights' : 'Selecciones y Pesos'}
          </button>
          <button className={`tab-btn ${tab === 'stats' ? 'active' : ''}`} onClick={() => setTab('stats')}>
            {isEn ? 'Results' : 'Estadísticas'}
          </button>
        </div>

        {tab === 'tools' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div className="kpi-grid">
              <div className="kpi-card">
                <div className="kpi-val">{stats.totalScheduled}/{stats.totalAllocated}</div>
                <div className="kpi-lbl">{isEn ? 'Scheduled / Allocated' : 'Programados / Asignados'}</div>
              </div>
              <div className="kpi-card">
                <div className="kpi-val" style={{ color: stats.totalUnscheduled > 0 ? '#b45309' : '#15803d' }}>
                  {stats.totalUnscheduled}
                </div>
                <div className="kpi-lbl">{isEn ? 'Unscheduled' : 'Sin programar'}</div>
              </div>
              <div className="kpi-card">
                <div className="kpi-val">{stats.totalGaps}</div>
                <div className="kpi-lbl">{isEn ? 'Current gaps' : 'Huecos actuales'}</div>
              </div>
              <div className="kpi-card">
                <div className="kpi-val" style={{ color: stats.conflictCount > 0 ? '#dc2626' : '#15803d' }}>
                  {stats.conflictCount}
                </div>
                <div className="kpi-lbl">{isEn ? 'Conflicts' : 'Conflictos'}</div>
              </div>
            </div>

            <button
              className="btn btn-primary action-tile"
              style={{
                background: 'linear-gradient(135deg, #2563eb, #4f46e5)',
                borderColor: '#4338ca',
              }}
              disabled={busy}
              onClick={() => run('auto_improve')}
            >
              <span className="action-tile-title">
                <Sparkles size={15} style={{ flexShrink: 0 }} />{' '}
                {isEn
                  ? `Auto-Improve (${opts.passes ?? 6} passes)`
                  : `Automejora inteligente (${opts.passes ?? 6} pasadas)`}
              </span>
              <span className="action-tile-desc">
                {isEn
                  ? 'Schedules remaining blocks, runs multi-pass gap minimization, and enforces per-course min/max consecutive hours.'
                  : 'Programa huecos pendientes, ejecuta varias pasadas reduciendo ventanas y respeta el mínimo/máximo de horas seguidas por curso.'}
              </span>
            </button>

            <div className="form-grid">
              <button
                className="btn btn-primary action-tile"
                disabled={busy}
                onClick={() => run('initial')}
              >
                <span className="action-tile-title">
                  <Wand2 size={15} style={{ flexShrink: 0 }} />{' '}
                  {isEn ? '1. Initial Solution' : '1. Solución inicial'}
                </span>
                <span className="action-tile-desc">
                  {isEn
                    ? 'Schedules all remaining unscheduled periods conflict-free.'
                    : 'Programa automáticamente todos los bloques pendientes sin choques.'}
                </span>
              </button>

              <button
                className="btn btn-ok action-tile"
                disabled={busy}
                onClick={() => run('optimise')}
              >
                <span className="action-tile-title">
                  <Sparkles size={15} style={{ flexShrink: 0 }} />{' '}
                  {isEn ? '2. Optimise Timetables (Compact Gaps)' : '2. Optimizar horarios (Reducir huecos)'}
                </span>
                <span className="action-tile-desc">
                  {isEn
                    ? 'Moves already-scheduled blocks to eliminate idle gaps.'
                    : 'Compacta las ventanas libres de grupos y docentes respetando restricciones.'}
                </span>
              </button>

              <button
                className="btn action-tile"
                disabled={busy || project.time.maxWeeks <= 1}
                onClick={() => run('copy')}
              >
                <span className="action-tile-title">
                  <Copy size={15} style={{ flexShrink: 0 }} />{' '}
                  {isEn ? '3. Copy Across Weeks' : '3. Copiar horarios a semanas siguientes'}
                </span>
                <span className="action-tile-desc" style={{ color: 'var(--text-muted)' }}>
                  {isEn
                    ? `Copies Week ${opts.fromWeek} to Weeks ${opts.fromWeek + 1}..${opts.toWeek}.`
                    : `Replica Semana ${opts.fromWeek} hacia Semanas ${opts.fromWeek + 1}..${opts.toWeek}.`}
                </span>
              </button>

              <button
                className="btn btn-danger action-tile"
                disabled={busy}
                onClick={() => run('remove')}
              >
                <span className="action-tile-title">
                  <Trash2 size={15} style={{ flexShrink: 0 }} />{' '}
                  {isEn ? '4. Remove Scheduled Periods' : '4. Limpiar horarios en rango'}
                </span>
                <span className="action-tile-desc">
                  {isEn
                    ? 'Unschedules non-locked events in the selected week/time range.'
                    : 'Desprograma eventos no bloqueados en el rango de semanas y horas.'}
                </span>
              </button>
            </div>
          </div>
        )}

        {tab === 'selections' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            <div className="form-grid">
              <div className="field">
                <label>{isEn ? 'From week' : 'Desde semana'}</label>
                <select
                  className="select"
                  value={opts.fromWeek}
                  onChange={(e) => setOpts({ ...opts, fromWeek: Number(e.target.value) })}
                >
                  {project.time.weeks.map((w) => (
                    <option key={w.index} value={w.index}>
                      {w.index} ({w.name})
                    </option>
                  ))}
                </select>
              </div>
              <div className="field">
                <label>{isEn ? 'To week' : 'Hasta semana'}</label>
                <select
                  className="select"
                  value={opts.toWeek}
                  onChange={(e) => setOpts({ ...opts, toWeek: Number(e.target.value) })}
                >
                  {project.time.weeks.map((w) => (
                    <option key={w.index} value={w.index}>
                      {w.index} ({w.name})
                    </option>
                  ))}
                </select>
              </div>
              <div className="field">
                <label>{isEn ? 'Days range (min – max)' : 'Rango de días (mín – máx)'}</label>
                <div style={{ display: 'flex', gap: 6 }}>
                  <input
                    className="input"
                    type="number"
                    min={1}
                    max={project.time.maxDays}
                    value={opts.minDay}
                    onChange={(e) => setOpts({ ...opts, minDay: Number(e.target.value) })}
                    style={{ width: '50%' }}
                  />
                  <input
                    className="input"
                    type="number"
                    min={1}
                    max={project.time.maxDays}
                    value={opts.maxDay}
                    onChange={(e) => setOpts({ ...opts, maxDay: Number(e.target.value) })}
                    style={{ width: '50%' }}
                  />
                </div>
              </div>
              <div className="field">
                <label>{isEn ? 'Periods range (min – max)' : 'Rango de bloques horarios (mín – máx)'}</label>
                <div style={{ display: 'flex', gap: 6 }}>
                  <input
                    className="input"
                    type="number"
                    min={1}
                    max={project.time.maxPeriods}
                    value={opts.minPeriod}
                    onChange={(e) => setOpts({ ...opts, minPeriod: Number(e.target.value) })}
                    style={{ width: '50%' }}
                  />
                  <input
                    className="input"
                    type="number"
                    min={1}
                    max={project.time.maxPeriods}
                    value={opts.maxPeriod}
                    onChange={(e) => setOpts({ ...opts, maxPeriod: Number(e.target.value) })}
                    style={{ width: '50%' }}
                  />
                </div>
              </div>
              <div className="field">
                <label>{isEn ? 'Block length (min – max)' : 'Duración de bloque consecutivo (mín – máx)'}</label>
                <div style={{ display: 'flex', gap: 6 }}>
                  <input
                    className="input"
                    type="number"
                    min={1}
                    max={10}
                    value={opts.minBlock}
                    onChange={(e) => setOpts({ ...opts, minBlock: Number(e.target.value) })}
                    style={{ width: '50%' }}
                  />
                  <input
                    className="input"
                    type="number"
                    min={1}
                    max={10}
                    value={opts.maxBlock}
                    onChange={(e) => setOpts({ ...opts, maxBlock: Number(e.target.value) })}
                    style={{ width: '50%' }}
                  />
                </div>
              </div>
              <div className="field">
                <label>{isEn ? 'Auto-Improve passes (2–15)' : 'Pasadas de Automejora (2–15)'}</label>
                <input
                  className="input"
                  type="number"
                  min={2}
                  max={15}
                  value={opts.passes ?? 6}
                  onChange={(e) => setOpts({ ...opts, passes: Math.max(2, Math.min(15, Number(e.target.value))) })}
                />
              </div>
              <div className="field" style={{ justifyContent: 'center', gap: 8 }}>
                <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={opts.includeLocked}
                    onChange={(e) => setOpts({ ...opts, includeLocked: e.target.checked })}
                  />
                  {isEn ? 'Include locked events' : 'Incluir cursos bloqueados'}
                </label>
                <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={opts.bookingsWithGaps}
                    onChange={(e) => setOpts({ ...opts, bookingsWithGaps: e.target.checked })}
                  />
                  {isEn ? 'Treat bookings like events' : 'Contar reservas al medir huecos'}
                </label>
              </div>
            </div>

            <div>
              <strong style={{ fontSize: 12 }}>
                {isEn ? 'Gap weights by resource category' : 'Peso de huecos por categoría de recurso'}
              </strong>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8, marginTop: 8 }}>
                {project.resourceCategories.slice(0, 6).map((c) => (
                  <div key={c.index} className="field">
                    <label>
                      {c.shortCode}: {c.name}
                    </label>
                    <input
                      className="input"
                      type="number"
                      min={0}
                      max={10}
                      value={opts.gapWeights[c.index] ?? c.gapWeight}
                      onChange={(e) =>
                        setOpts({
                          ...opts,
                          gapWeights: { ...opts.gapWeights, [c.index]: Number(e.target.value) },
                        })
                      }
                    />
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {tab === 'stats' && (
          <div>
            {lastResult ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <div className="kpi-grid">
                  <div className="kpi-card">
                    <div className="kpi-val">{lastResult.gapsBefore}</div>
                    <div className="kpi-lbl">{isEn ? 'Gaps initially' : 'Huecos iniciales'}</div>
                  </div>
                  <div className="kpi-card">
                    <div className="kpi-val" style={{ color: '#15803d' }}>
                      {lastResult.gapsAfter}
                    </div>
                    <div className="kpi-lbl">{isEn ? 'Gaps now' : 'Huecos ahora'}</div>
                  </div>
                  <div className="kpi-card">
                    <div className="kpi-val" style={{ color: '#2563eb' }}>
                      {lastResult.gapsChange >= 0 ? `-${lastResult.gapsChange}` : `+${-lastResult.gapsChange}`} (
                      {lastResult.gapsPercent.toFixed(1)}%)
                    </div>
                    <div className="kpi-lbl">{isEn ? 'Gap reduction' : 'Reducción'}</div>
                  </div>
                  <div className="kpi-card">
                    <div className="kpi-val">{lastResult.placed || lastResult.removed}</div>
                    <div className="kpi-lbl">
                      {lastResult.action === 'remove'
                        ? isEn
                          ? 'Removed'
                          : 'Quitados'
                        : isEn
                        ? 'Blocks placed'
                        : 'Bloques colocados'}
                    </div>
                  </div>
                </div>
                <div>
                  <strong style={{ fontSize: 12 }}>
                    {isEn ? 'Modified events' : 'Cursos modificados'} ({lastResult.changedCourses.length}):
                  </strong>
                  <div className="pill-list" style={{ marginTop: 6 }}>
                    {lastResult.changedCourses.map((c) => (
                      <span key={c} className="code-badge course">
                        {c}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            ) : (
              <p style={{ color: 'var(--text-muted)' }}>
                {isEn
                  ? 'Run an optimisation tool from the Tools tab to view statistics.'
                  : 'Ejecuta una herramienta en la pestaña Herramientas para ver los resultados.'}
              </p>
            )}
          </div>
        )}
      </div>
      <div className="modal-footer">
        <button className="btn" onClick={onClose}>
          {isEn ? 'Close' : 'Cerrar'}
        </button>
      </div>
    </>
  );
};

/* ---------------- 2. MANUAL ROOMS MODAL (TRoomsDlg) ---------------- */
const RoomsModal: React.FC<{
  project: Project;
  activeWeek: number;
  isEn: boolean;
  onUpdate: (p: Promise<any>, msg?: string) => Promise<any>;
  onClose: () => void;
}> = ({ project, activeWeek, isEn, onUpdate, onClose }) => {
  const [week, setWeek] = useState(activeWeek);
  const weekSlots = project.slots.filter((s) => s.week === week);
  const [selectedSlotId, setSelectedSlotId] = useState<string>(weekSlots[0]?.id ?? '');
  const [availRooms, setAvailRooms] = useState<Resource[]>([]);

  const selectedSlot: SlotAssignment | undefined = weekSlots.find((s) => s.id === selectedSlotId);
  const courseMap = new Map(project.courses.map((c) => [c.id, c]));
  const resMap = new Map(project.resources.map((r) => [r.id, r]));

  useEffect(() => {
    if (!selectedSlot) {
      setAvailRooms([]);
      return;
    }
    api
      .availableRooms(selectedSlot.week, selectedSlot.day, selectedSlot.period, selectedSlot.id)
      .then((res) => setAvailRooms(res.rooms))
      .catch(() => setAvailRooms([]));
  }, [selectedSlot?.id, project.slots]);

  return (
    <>
      <div className="modal-header">
        <h3>
          <DoorOpen size={17} color="#2563eb" />
          {isEn ? 'Select Rooms Manually' : 'Asignar Salas Manualmente'}
        </h3>
        <button className="btn btn-icon" onClick={onClose}>
          <X size={15} />
        </button>
      </div>
      <div className="modal-body">
        <div className="view-subbar" style={{ marginBottom: 12 }}>
          <div className="control-group">
            <label style={{ fontWeight: 600 }}>{isEn ? 'Week' : 'Semana'}:</label>
            <select className="select" value={week} onChange={(e) => setWeek(Number(e.target.value))}>
              {project.time.weeks.map((w) => (
                <option key={w.index} value={w.index}>
                  {w.index} ({w.name})
                </option>
              ))}
            </select>
          </div>
          <div className="control-group">
            <button
              className="btn btn-sm btn-primary"
              onClick={() =>
                onUpdate(
                  api.roomAction({ action: 'imitate', fromWeek: 1, toWeek: project.time.maxWeeks }),
                  isEn ? 'Rooms imitated automatically' : 'Salas asignadas por imitación'
                )
              }
            >
              <Wand2 size={13} /> {isEn ? 'Imitate Rooms' : 'Imitar salas automáticamente'}
            </button>
            <button
              className="btn btn-sm"
              onClick={() =>
                onUpdate(
                  api.roomAction({ action: 'clear_week', fromWeek: week }),
                  isEn ? 'Cleared manual rooms in week' : 'Salas manuales limpiadas en esta semana'
                )
              }
            >
              {isEn ? 'Clear week' : 'Limpiar semana'}
            </button>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1.3fr 1fr', gap: 12, minHeight: 340 }}>
          <div className="panel">
            <div className="panel-header">
              <span>{isEn ? 'Scheduled slots in week' : 'Bloques programados en la semana'} ({weekSlots.length})</span>
            </div>
            <div className="panel-body" style={{ maxHeight: 360 }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{isEn ? 'Day / Period' : 'Día / Bloque'}</th>
                    <th>{isEn ? 'Event' : 'Curso'}</th>
                    <th>{isEn ? 'Room' : 'Sala'}</th>
                  </tr>
                </thead>
                <tbody>
                  {weekSlots.map((s) => {
                    const c = courseMap.get(s.courseId);
                    const manRoom = s.roomId ? resMap.get(s.roomId) : null;
                    const defRoom = c?.resources
                      .map((rid) => resMap.get(rid))
                      .find((r) => r?.category === project.roomsCategory);
                    return (
                      <tr
                        key={s.id}
                        className={selectedSlotId === s.id ? 'selected' : ''}
                        style={{ cursor: 'pointer' }}
                        onClick={() => setSelectedSlotId(s.id)}
                      >
                        <td>
                          {project.time.days[s.day - 1]?.slice(0, 3)} B{s.period}
                        </td>
                        <td>
                          <span className="code-badge course">{c?.code}</span>
                        </td>
                        <td>
                          {manRoom ? (
                            <span className="code-badge resource">{manRoom.code} ★</span>
                          ) : defRoom ? (
                            <span style={{ color: 'var(--text-muted)' }}>{defRoom.code}</span>
                          ) : (
                            <span style={{ color: '#dc2626', fontWeight: 600 }}>
                              {isEn ? 'No room' : 'Sin sala'}
                            </span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>

          <div className="panel">
            <div className="panel-header">
              <span>{isEn ? 'Available rooms for slot' : 'Salas libres para este bloque'}</span>
            </div>
            <div className="panel-body" style={{ padding: 10 }}>
              {selectedSlot ? (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 4 }}>
                    <strong>{courseMap.get(selectedSlot.courseId)?.code}</strong> ·{' '}
                    {project.time.days[selectedSlot.day - 1]} B{selectedSlot.period}
                  </div>
                  {availRooms.map((r) => (
                    <div
                      key={r.id}
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        padding: '7px 9px',
                        border:
                          selectedSlot.roomId === r.id
                            ? '1.5px solid var(--accent)'
                            : '1px solid var(--border)',
                        borderRadius: 6,
                        background: selectedSlot.roomId === r.id ? '#eff6ff' : '#fff',
                      }}
                    >
                      <div>
                        <span className="code-badge resource">{r.code}</span>{' '}
                        <span style={{ fontSize: 12 }}>{r.name}</span>
                      </div>
                      <div style={{ display: 'flex', gap: 4 }}>
                        <button
                          className="btn btn-sm btn-primary"
                          onClick={() =>
                            onUpdate(
                              api.setSlotRoom(selectedSlot.id, r.id),
                              isEn ? `Assigned room ${r.code}` : `Sala ${r.code} asignada`
                            )
                          }
                        >
                          {isEn ? 'Assign' : 'Asignar'}
                        </button>
                        <button
                          className="btn btn-sm"
                          title={isEn ? 'Copy to same time in other weeks' : 'Copiar a la misma hora en otras semanas'}
                          onClick={() =>
                            onUpdate(
                              api.roomAction({
                                action: 'copy_same_time',
                                slotId: selectedSlot.id,
                                roomId: r.id,
                              }),
                              isEn
                                ? `Copied room ${r.code} to other weeks`
                                : `Sala ${r.code} copiada a otras semanas`
                            )
                          }
                        >
                          <Copy size={12} />
                        </button>
                      </div>
                    </div>
                  ))}
                  {availRooms.length === 0 && (
                    <div style={{ color: 'var(--err-text)', fontSize: 12 }}>
                      {isEn ? 'No free rooms in this period.' : 'No hay salas disponibles en este bloque.'}
                    </div>
                  )}
                </div>
              ) : (
                <div style={{ color: 'var(--text-muted)' }}>
                  {isEn ? 'Select a slot on the left.' : 'Selecciona un bloque en la tabla izquierda.'}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
      <div className="modal-footer">
        <button className="btn" onClick={onClose}>
          {isEn ? 'Close' : 'Cerrar'}
        </button>
      </div>
    </>
  );
};

/* ---------------- 3. OPTIONS MODAL (TTabNotebook) ---------------- */
const OptionsModal: React.FC<{
  project: Project;
  isEn: boolean;
  onUpdate: (p: Promise<any>, msg?: string) => Promise<any>;
  onClose: () => void;
}> = ({ project, isEn, onUpdate, onClose }) => {
  const [tab, setTab] = useState<'limits' | 'bells' | 'categories'>('limits');
  const [name, setName] = useState(project.name);
  const [institution, setInstitution] = useState(project.institution);
  const [department, setDepartment] = useState(project.department);
  const [time, setTime] = useState(project.time);
  const [resCats, setResCats] = useState(project.resourceCategories);
  const [roomsCat, setRoomsCat] = useState(project.roomsCategory);

  const save = async () => {
    await onUpdate(
      api.updateSettings({
        name,
        institution,
        department,
        comment: project.comment,
        language: project.language,
        roomsCategory: roomsCat,
        time,
        courseCategories: project.courseCategories,
        resourceCategories: resCats,
        bookingTypes: project.bookingTypes,
      }),
      isEn ? 'Options saved' : 'Opciones actualizadas'
    );
    onClose();
  };

  return (
    <>
      <div className="modal-header">
        <h3>
          <Settings size={17} color="#2563eb" />
          {isEn ? 'Options & Limits' : 'Opciones y Límites'}
        </h3>
        <button className="btn btn-icon" onClick={onClose}>
          <X size={15} />
        </button>
      </div>
      <div className="modal-body">
        <div className="tab-bar">
          <button className={`tab-btn ${tab === 'limits' ? 'active' : ''}`} onClick={() => setTab('limits')}>
            {isEn ? 'Limits & General' : 'Límites y General'}
          </button>
          <button className={`tab-btn ${tab === 'bells' ? 'active' : ''}`} onClick={() => setTab('bells')}>
            {isEn ? 'Days & Bells' : 'Días y Bloques'}
          </button>
          <button
            className={`tab-btn ${tab === 'categories' ? 'active' : ''}`}
            onClick={() => setTab('categories')}
          >
            {isEn ? 'Categories & Rules' : 'Categorías y Reglas'}
          </button>
        </div>

        {tab === 'limits' && (
          <div className="form-grid">
            <div className="field">
              <label>{isEn ? 'Schedule name' : 'Nombre del horario'}</label>
              <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="field">
              <label>{isEn ? 'Institution / University' : 'Institución / Universidad'}</label>
              <input
                className="input"
                value={institution}
                onChange={(e) => setInstitution(e.target.value)}
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Department / Faculty' : 'Departamento / Facultad'}</label>
              <input
                className="input"
                value={department}
                onChange={(e) => setDepartment(e.target.value)}
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Start date' : 'Fecha de inicio'}</label>
              <input
                className="input"
                type="date"
                value={time.startDate}
                onChange={(e) => setTime({ ...time, startDate: e.target.value })}
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Weeks or terms (1–255)' : 'Semanas o periodos (1–255)'}</label>
              <input
                className="input"
                type="number"
                min={1}
                max={255}
                value={time.maxWeeks}
                onChange={(e) => setTime({ ...time, maxWeeks: Math.max(1, Math.min(255, Number(e.target.value))) })}
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Days in week (1–7)' : 'Días por semana (1–7)'}</label>
              <input
                className="input"
                type="number"
                min={1}
                max={7}
                value={time.maxDays}
                onChange={(e) => setTime({ ...time, maxDays: Math.max(1, Math.min(7, Number(e.target.value))) })}
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Time periods per day (1–30)' : 'Bloques horarios por día (1–30)'}</label>
              <input
                className="input"
                type="number"
                min={1}
                max={30}
                value={time.maxPeriods}
                onChange={(e) =>
                  setTime({ ...time, maxPeriods: Math.max(1, Math.min(30, Number(e.target.value))) })
                }
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Rooms category' : 'Categoría de Salas / Aulas'}</label>
              <select className="select" value={roomsCat} onChange={(e) => setRoomsCat(Number(e.target.value))}>
                {resCats.map((c) => (
                  <option key={c.index} value={c.index}>
                    {c.shortCode}: {c.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
        )}

        {tab === 'bells' && (
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
            <div>
              <strong style={{ fontSize: 12 }}>{isEn ? 'Day names' : 'Nombres de los días'}</strong>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginTop: 8 }}>
                {time.days.slice(0, time.maxDays).map((d, i) => (
                  <input
                    key={i}
                    className="input"
                    value={d}
                    onChange={(e) => {
                      const next = [...time.days];
                      next[i] = e.target.value;
                      setTime({ ...time, days: next });
                    }}
                  />
                ))}
              </div>
            </div>
            <div>
              <strong style={{ fontSize: 12 }}>{isEn ? 'Bell times' : 'Horas de bloques'}</strong>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginTop: 8, maxHeight: 280, overflowY: 'auto' }}>
                {Array.from({ length: time.maxPeriods }, (_, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ width: 28, fontWeight: 700, fontSize: 11.5 }}>B{i + 1}</span>
                    <input
                      className="input"
                      style={{ flex: 1 }}
                      value={time.bells[i] ?? ''}
                      onChange={(e) => {
                        const next = [...time.bells];
                        next[i] = e.target.value;
                        setTime({ ...time, bells: next });
                      }}
                    />
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {tab === 'categories' && (
          <table className="data-table">
            <thead>
              <tr>
                <th>#</th>
                <th>{isEn ? 'Code' : 'Letra'}</th>
                <th>{isEn ? 'Category Name' : 'Categoría de Recurso'}</th>
                <th>{isEn ? 'Allow Overlap' : 'Permitir Choques'}</th>
                <th>{isEn ? 'Ignore Gaps' : 'Ignorar Huecos'}</th>
                <th>{isEn ? 'Weekly Limit' : 'Límite Semanal'}</th>
              </tr>
            </thead>
            <tbody>
              {resCats.map((c, idx) => (
                <tr key={c.index}>
                  <td>{c.index}</td>
                  <td>
                    <input
                      className="input"
                      style={{ width: 42 }}
                      value={c.shortCode}
                      onChange={(e) => {
                        const next = [...resCats];
                        next[idx] = { ...c, shortCode: e.target.value };
                        setResCats(next);
                      }}
                    />
                  </td>
                  <td>
                    <input
                      className="input"
                      value={c.name}
                      onChange={(e) => {
                        const next = [...resCats];
                        next[idx] = { ...c, name: e.target.value };
                        setResCats(next);
                      }}
                    />
                  </td>
                  <td style={{ textAlign: 'center' }}>
                    <input
                      type="checkbox"
                      checked={c.noTimetables}
                      onChange={(e) => {
                        const next = [...resCats];
                        next[idx] = { ...c, noTimetables: e.target.checked };
                        setResCats(next);
                      }}
                    />
                  </td>
                  <td style={{ textAlign: 'center' }}>
                    <input
                      type="checkbox"
                      checked={c.noGaps}
                      onChange={(e) => {
                        const next = [...resCats];
                        next[idx] = { ...c, noGaps: e.target.checked };
                        setResCats(next);
                      }}
                    />
                  </td>
                  <td>
                    <input
                      className="input"
                      type="number"
                      min={0}
                      max={200}
                      style={{ width: 70 }}
                      value={c.weeklyLimit}
                      onChange={(e) => {
                        const next = [...resCats];
                        next[idx] = { ...c, weeklyLimit: Number(e.target.value) };
                        setResCats(next);
                      }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      <div className="modal-footer">
        <button className="btn" onClick={onClose}>
          {isEn ? 'Cancel' : 'Cancelar'}
        </button>
        <button className="btn btn-primary" onClick={save}>
          {isEn ? 'Accept changes' : 'Aceptar cambios'}
        </button>
      </div>
    </>
  );
};

/* ---------------- 4. STATISTICS & HEALTH CHECK MODAL (TStatNoteBookDlg) ---------------- */
const StatsModal: React.FC<{
  project: Project;
  stats: ProjectStats;
  isEn: boolean;
  onClose: () => void;
}> = ({ project, stats, isEn, onClose }) => {
  return (
    <>
      <div className="modal-header">
        <h3>
          <BarChart3 size={17} color="#2563eb" />
          {isEn ? 'Statistics & Health Check' : 'Estadísticas y Diagnóstico'}
        </h3>
        <button className="btn btn-icon" onClick={onClose}>
          <X size={15} />
        </button>
      </div>
      <div className="modal-body">
        <div className="kpi-grid">
          <div className="kpi-card">
            <div className="kpi-val">{stats.totalCodes}</div>
            <div className="kpi-lbl">
              {isEn ? 'Total codes' : 'Códigos totales'}
            </div>
          </div>
          <div className="kpi-card">
            <div className="kpi-val">{Math.round(stats.overallCompletion)}%</div>
            <div className="kpi-lbl">
              {stats.totalScheduled}/{stats.totalAllocated} {isEn ? 'scheduled' : 'programados'}
            </div>
          </div>
          <div className="kpi-card">
            <div className="kpi-val" style={{ color: stats.conflictCount > 0 ? '#dc2626' : '#15803d' }}>
              {stats.conflictCount}
            </div>
            <div className="kpi-lbl">{isEn ? 'Conflicts detected' : 'Conflictos detectados'}</div>
          </div>
          <div className="kpi-card">
            <div className="kpi-val">{stats.totalGaps}</div>
            <div className="kpi-lbl">{isEn ? 'Total idle gaps' : 'Huecos totales (ventanas)'}</div>
          </div>
        </div>

        <div style={{ marginBottom: 14 }}>
          <strong style={{ fontSize: 12.5 }}>
            {isEn ? 'Health Check / Problems' : 'Revisión de Conflictos y Problemas'}
          </strong>
          {stats.conflicts.length === 0 ? (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: 10,
                marginTop: 6,
                borderRadius: 8,
                background: 'var(--ok-bg)',
                color: 'var(--ok-text)',
                fontWeight: 600,
              }}
            >
              <CheckCircle2 size={16} />
              {isEn
                ? 'No schedule conflicts or booking violations found!'
                : '¡Sin choques de horarios ni violaciones de reservas!'}
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 5, marginTop: 6, maxHeight: 160, overflowY: 'auto' }}>
              {stats.conflicts.map((c, i) => (
                <div
                  key={i}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '6px 9px',
                    borderRadius: 6,
                    background: 'var(--err-bg)',
                    color: 'var(--err-text)',
                    fontSize: 12,
                  }}
                >
                  <AlertTriangle size={14} /> {c.message}
                </div>
              ))}
            </div>
          )}
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <div className="panel">
            <div className="panel-header">
              <span>{isEn ? 'Highest Weekly Load' : 'Mayor carga semanal'}</span>
            </div>
            <div className="panel-body" style={{ maxHeight: 180 }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{isEn ? 'Resource' : 'Recurso'}</th>
                    <th>{isEn ? 'Week' : 'Sem'}</th>
                    <th className="num-cell">{isEn ? 'Load' : 'Horas'}</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.maxWeeklyLoad?.map((e, i) => (
                    <tr key={i}>
                      <td>
                        <span className="code-badge resource">{e.resourceCode}</span>
                      </td>
                      <td>S{e.week}</td>
                      <td className="num-cell">{e.load}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="panel">
            <div className="panel-header">
              <span>{isEn ? 'Peak Timetable Slots' : 'Bloques de mayor ocupación'}</span>
            </div>
            <div className="panel-body" style={{ maxHeight: 180 }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{isEn ? 'Week' : 'Sem'}</th>
                    <th>{isEn ? 'Day / Period' : 'Día / Bloque'}</th>
                    <th className="num-cell">{isEn ? 'Events' : 'Cursos'}</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.maxSlotLoad?.map((e, i) => (
                    <tr key={i}>
                      <td>S{e.week}</td>
                      <td>
                        {project.time.days[e.day - 1]?.slice(0, 3)} B{e.period}
                      </td>
                      <td className="num-cell">{e.count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
      <div className="modal-footer">
        <button className="btn" onClick={onClose}>
          {isEn ? 'Close' : 'Cerrar'}
        </button>
      </div>
    </>
  );
};

/* ---------------- 5. REPORTS & EXPORT MODAL (TPrintMenu) ---------------- */
const ReportsModal: React.FC<{
  project: Project;
  activeWeek: number;
  isEn: boolean;
  onClose: () => void;
}> = ({ project, activeWeek, isEn, onClose }) => {
  const [reportType, setReportType] = useState(3);
  const [week, setWeek] = useState(activeWeek);
  const [previewHtml, setPreviewHtml] = useState('');

  useEffect(() => {
    api
      .exportFile('html', reportType, week)
      .then((res) => setPreviewHtml(res.content))
      .catch(() => {});
  }, [reportType, week]);

  const handleExport = async (fmt: string) => {
    const res = await api.exportFile(fmt, reportType, week);
    const mime =
      fmt === 'html'
        ? 'text/html;charset=utf-8'
        : fmt === 'csv'
        ? 'text/csv;charset=utf-8'
        : fmt === 'ics'
        ? 'text/calendar;charset=utf-8'
        : 'text/plain;charset=utf-8';
    downloadText(res.fileName, res.content, mime);
  };

  const reportNames = isEn
    ? [
        [1, '1. List of events'],
        [2, '2. List of resources'],
        [3, '3. Individual timetables'],
        [4, '4. Events by resources'],
        [5, '5. Resources by events'],
        [6, '6. Compact schedules'],
      ]
    : [
        [1, '1. Lista de cursos'],
        [2, '2. Lista de recursos'],
        [3, '3. Horarios individuales'],
        [4, '4. Cursos por recursos'],
        [5, '5. Recursos por cursos'],
        [6, '6. Calendario compacto'],
      ];

  return (
    <>
      <div className="modal-header">
        <h3>
          <FileText size={17} color="#2563eb" />
          {isEn ? 'Print, Publish Web & Export' : 'Informes, Publicación Web y Exportación'}
        </h3>
        <button className="btn btn-icon" onClick={onClose}>
          <X size={15} />
        </button>
      </div>
      <div className="modal-body">
        <div className="view-subbar" style={{ marginBottom: 12 }}>
          <div className="control-group">
            <select
              className="select"
              value={reportType}
              onChange={(e) => setReportType(Number(e.target.value))}
            >
              {reportNames.map(([id, lbl]) => (
                <option key={id as number} value={id as number}>
                  {lbl}
                </option>
              ))}
            </select>
            <select className="select" value={week} onChange={(e) => setWeek(Number(e.target.value))}>
              {project.time.weeks.map((w) => (
                <option key={w.index} value={w.index}>
                  {isEn ? 'Week' : 'Semana'} {w.index} ({w.name})
                </option>
              ))}
            </select>
          </div>
          <div className="control-group">
            <button className="btn btn-sm btn-primary" onClick={() => handleExport('html')}>
              <Download size={13} /> HTML Web
            </button>
            <button className="btn btn-sm" onClick={() => handleExport('mxt')}>
              <Download size={13} /> Mimosa .mxt
            </button>
            <button className="btn btn-sm" onClick={() => handleExport('csv')}>
              <Download size={13} /> Excel .csv
            </button>
            <button className="btn btn-sm" onClick={() => handleExport('ics')}>
              <Download size={13} /> iCalendar .ics
            </button>
            <button className="btn btn-sm" onClick={() => handleExport('omimo')}>
              <Download size={13} /> .omimo
            </button>
          </div>
        </div>

        <iframe
          title="Report Preview"
          srcDoc={previewHtml}
          style={{
            width: '100%',
            height: 410,
            border: '1px solid var(--border-strong)',
            borderRadius: 8,
            background: '#fff',
          }}
        />
      </div>
      <div className="modal-footer">
        <button className="btn" onClick={onClose}>
          {isEn ? 'Close' : 'Cerrar'}
        </button>
      </div>
    </>
  );
};
