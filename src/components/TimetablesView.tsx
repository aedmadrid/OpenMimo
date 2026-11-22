import React, { useEffect, useMemo, useState } from 'react';
import {
  ChevronLeft,
  ChevronRight,
  Lock,
  Sparkles,
  Copy,
  Ban,
  Eye,
  X,
  AlertTriangle,
} from 'lucide-react';
import type { Project, ProjectStats, SlotAvailability } from '../types';
import { api } from '../api';
import { I18N } from '../i18n';

interface Props {
  project: Project;
  stats: ProjectStats;
  activeWeek: number;
  setActiveWeek: (w: number) => void;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
  onError: (msg: string) => void;
}

export const TimetablesView: React.FC<Props> = ({
  project,
  stats,
  activeWeek,
  setActiveWeek,
  onUpdate,
  onError,
}) => {
  const t = I18N[project.language === 'en' ? 'en' : 'es'];

  const timetableResources = useMemo(
    () =>
      project.resources.filter((r) => {
        const cat = project.resourceCategories.find((c) => c.index === r.category);
        return !cat?.noTimetables;
      }),
    [project.resources, project.resourceCategories]
  );

  const [viewMode, setViewMode] = useState<'resource' | 'course'>('resource');
  const [selectedTargetId, setSelectedTargetId] = useState<string>('');
  const [selectedCourseId, setSelectedCourseId] = useState<string>('');
  const [bookingMode, setBookingMode] = useState(false);
  const [whatIfEnabled, setWhatIfEnabled] = useState(true);
  const [whatIfMap, setWhatIfMap] = useState<Record<string, SlotAvailability>>({});

  // Ensure valid selected target
  useEffect(() => {
    if (viewMode === 'resource') {
      if (!timetableResources.some((r) => r.id === selectedTargetId)) {
        setSelectedTargetId(timetableResources[0]?.id ?? '');
      }
    } else {
      if (!project.courses.some((c) => c.id === selectedTargetId)) {
        setSelectedTargetId(project.courses[0]?.id ?? '');
      }
    }
  }, [viewMode, timetableResources, project.courses, selectedTargetId]);

  // Courses relevant to the current timetable target
  const relevantCourses = useMemo(() => {
    if (viewMode === 'course') {
      const c = project.courses.find((x) => x.id === selectedTargetId);
      return c ? [c] : project.courses;
    }
    if (!selectedTargetId) return project.courses;
    return project.courses.filter(
      (c) =>
        c.resources.includes(selectedTargetId) ||
        project.slots.some(
          (s) => s.courseId === c.id && s.week === activeWeek && s.roomId === selectedTargetId
        )
    );
  }, [viewMode, selectedTargetId, project.courses, project.slots, activeWeek]);

  // Auto-select a course (prefer one with unscheduled hours in active week)
  useEffect(() => {
    if (relevantCourses.length === 0) {
      setSelectedCourseId('');
      return;
    }
    if (!relevantCourses.some((c) => c.id === selectedCourseId)) {
      const withLeft = relevantCourses.find((c) => {
        const alloc =
          project.weekAllocations.find((w) => w.courseId === c.id && w.week === activeWeek)
            ?.periods ?? 0;
        const sched = project.slots.filter(
          (s) => s.courseId === c.id && s.week === activeWeek
        ).length;
        return alloc > sched;
      });
      setSelectedCourseId(withLeft?.id ?? relevantCourses[0].id);
    }
  }, [relevantCourses, selectedCourseId, project.weekAllocations, project.slots, activeWeek]);

  // Load What-If availability for selectedCourseId
  useEffect(() => {
    if (!whatIfEnabled || !selectedCourseId || bookingMode) {
      setWhatIfMap({});
      return;
    }
    let cancelled = false;
    api
      .whatIf(selectedCourseId, activeWeek)
      .then((res) => {
        if (cancelled) return;
        const m: Record<string, SlotAvailability> = {};
        for (const s of res.slots) {
          m[`${s.day}_${s.period}`] = s;
        }
        setWhatIfMap(m);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [whatIfEnabled, selectedCourseId, activeWeek, bookingMode, project.slots, project.bookings]);

  const resMap = useMemo(
    () => new Map(project.resources.map((r) => [r.id, r])),
    [project.resources]
  );
  const courseMap = useMemo(
    () => new Map(project.courses.map((c) => [c.id, c])),
    [project.courses]
  );

  // Slots visible on current grid
  const visibleSlots = useMemo(() => {
    return project.slots.filter((s) => {
      if (s.week !== activeWeek) return false;
      if (viewMode === 'course') {
        return !selectedTargetId || s.courseId === selectedTargetId;
      }
      if (!selectedTargetId) return true;
      const c = courseMap.get(s.courseId);
      if (!c) return false;
      if (s.roomId) {
        if (s.roomId === selectedTargetId) return true;
        // If target is another room in course's default resources, hide because manual room overrode it
        const targetRes = resMap.get(selectedTargetId);
        if (targetRes && targetRes.category === project.roomsCategory) {
          return false;
        }
      }
      return c.resources.includes(selectedTargetId);
    });
  }, [project.slots, activeWeek, viewMode, selectedTargetId, courseMap, resMap, project.roomsCategory]);

  // Bookings visible on current grid
  const visibleBookings = useMemo(() => {
    return project.bookings.filter(
      (b) =>
        b.scope === 'timetable' &&
        b.week === activeWeek &&
        b.targetType === viewMode &&
        b.targetId === selectedTargetId
    );
  }, [project.bookings, activeWeek, viewMode, selectedTargetId]);

  const conflictCellKeys = useMemo(() => {
    const s = new Set<string>();
    for (const c of stats.conflicts) {
      if (c.week === activeWeek && c.day > 0 && c.period > 0) {
        s.add(`${c.day}_${c.period}`);
      }
    }
    return s;
  }, [stats.conflicts, activeWeek]);

  const selectedCourse = courseMap.get(selectedCourseId);
  const linkedResources = useMemo(() => {
    if (!selectedCourse) return [];
    return selectedCourse.resources
      .map((rid) => resMap.get(rid))
      .filter((r): r is NonNullable<typeof r> => Boolean(r))
      .filter((r) => {
        const cat = project.resourceCategories.find((c) => c.index === r.category);
        return !cat?.noTimetables;
      });
  }, [selectedCourse, resMap, project.resourceCategories]);

  const weekPct = Math.round(stats.weekCompletion[activeWeek] ?? 0);
  const allPct = Math.round(stats.overallCompletion ?? 0);

  const cycleTarget = (delta: number) => {
    const list = viewMode === 'resource' ? timetableResources : project.courses;
    if (list.length === 0) return;
    const idx = list.findIndex((x) => x.id === selectedTargetId);
    const next = (idx + delta + list.length) % list.length;
    setSelectedTargetId(list[next].id);
  };

  const handleCellClick = async (day: number, period: number) => {
    if (bookingMode) {
      if (!selectedTargetId) return;
      await onUpdate(
        api.toggleBooking({
          targetType: viewMode,
          targetId: selectedTargetId,
          scope: 'timetable',
          week: activeWeek,
          day,
          period,
          kind: 1,
          label: '/////',
        })
      );
      return;
    }

    if (!selectedCourseId) return;
    const avail = whatIfMap[`${day}_${period}`];
    if (avail && !avail.available) {
      onError(
        avail.reasons?.join(' · ') ||
          (project.language === 'en' ? 'Conflicting time slot' : 'Bloque con conflicto')
      );
      return;
    }
    await onUpdate(api.placeSlot(selectedCourseId, activeWeek, day, period));
  };

  const handlePlaceBest = async () => {
    if (!selectedCourseId) return;
    await onUpdate(
      api.optimize({
        action: 'initial',
        fromWeek: activeWeek,
        toWeek: activeWeek,
        minDay: 1,
        maxDay: project.time.maxDays,
        minPeriod: 1,
        maxPeriod: project.time.maxPeriods,
        minBlock: 1,
        maxBlock: 2,
        includeLocked: false,
        bookingsWithGaps: false,
        gapWeights: {},
        courseIds: [selectedCourseId],
        randSeed: 38746,
      }),
      project.language === 'en' ? 'Scheduled in best available slot' : 'Programado en el mejor bloque disponible'
    );
  };

  const handleCopyNextWeek = async () => {
    if (activeWeek >= project.time.maxWeeks) {
      onError(project.language === 'en' ? 'Already on the last week' : 'Ya estás en la última semana');
      return;
    }
    await onUpdate(
      api.optimize({
        action: 'copy',
        fromWeek: activeWeek,
        toWeek: activeWeek + 1,
        minDay: 1,
        maxDay: project.time.maxDays,
        minPeriod: 1,
        maxPeriod: project.time.maxPeriods,
        minBlock: 1,
        maxBlock: 10,
        includeLocked: false,
        bookingsWithGaps: false,
        gapWeights: {},
        randSeed: 38746,
      }),
      project.language === 'en'
        ? `Copied Week ${activeWeek} → Week ${activeWeek + 1}`
        : `Horario copiado de Semana ${activeWeek} → Semana ${activeWeek + 1}`
    );
  };

  return (
    <>
      {/* Sub-toolbar matching Mimosa Timetables navigation & gauges */}
      <div className="view-subbar">
        <div className="control-group">
          <select
            className="select"
            value={viewMode}
            onChange={(e) => setViewMode(e.target.value as 'resource' | 'course')}
          >
            <option value="resource">{t.timetables.allResources}</option>
            <option value="course">{t.timetables.byCourse}</option>
          </select>

          <button
            className="btn btn-icon"
            onClick={() => cycleTarget(-1)}
            title={project.language === 'en' ? 'Previous' : 'Anterior'}
          >
            <ChevronLeft size={14} />
          </button>
          <select
            className="select"
            style={{ minWidth: 210, fontWeight: 600 }}
            value={selectedTargetId}
            onChange={(e) => setSelectedTargetId(e.target.value)}
          >
            {viewMode === 'resource'
              ? timetableResources.map((r) => {
                  const cat = project.resourceCategories.find((c) => c.index === r.category);
                  return (
                    <option key={r.id} value={r.id}>
                      [{cat?.shortCode ?? r.category}] {r.code} — {r.name}
                    </option>
                  );
                })
              : project.courses.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.code} — {c.name}
                  </option>
                ))}
          </select>
          <button
            className="btn btn-icon"
            onClick={() => cycleTarget(1)}
            title={project.language === 'en' ? 'Next' : 'Siguiente'}
          >
            <ChevronRight size={14} />
          </button>

          <div className="divider-v" />

          <button
            className="btn btn-icon"
            disabled={activeWeek <= 1}
            onClick={() => setActiveWeek(Math.max(1, activeWeek - 1))}
          >
            <ChevronLeft size={14} />
          </button>
          <select
            className="select"
            value={activeWeek}
            onChange={(e) => setActiveWeek(Number(e.target.value))}
          >
            {project.time.weeks.map((w) => (
              <option key={w.index} value={w.index}>
                {t.timetables.week} {w.index} ({w.name})
              </option>
            ))}
          </select>
          <button
            className="btn btn-icon"
            disabled={activeWeek >= project.time.maxWeeks}
            onClick={() => setActiveWeek(Math.min(project.time.maxWeeks, activeWeek + 1))}
          >
            <ChevronRight size={14} />
          </button>
        </div>

        <div className="control-group">
          <button
            className={`btn btn-sm ${whatIfEnabled && !bookingMode ? 'btn-ok' : ''}`}
            onClick={() => {
              setBookingMode(false);
              setWhatIfEnabled((v) => !v);
            }}
          >
            <Eye size={13} /> {t.timetables.whatIfMode}
          </button>
          <button
            className={`btn btn-sm ${bookingMode ? 'btn-amber' : ''}`}
            onClick={() => setBookingMode((v) => !v)}
          >
            <Ban size={13} /> {t.timetables.bookingMode}
          </button>
          <button
            className="btn btn-sm btn-primary"
            disabled={!selectedCourseId}
            onClick={handlePlaceBest}
          >
            <Sparkles size={13} /> {t.timetables.placeBest}
          </button>
          {project.time.maxWeeks > 1 && (
            <button className="btn btn-sm" onClick={handleCopyNextWeek}>
              <Copy size={13} /> {t.timetables.copyNextWeek}
            </button>
          )}
        </div>

        <div className="gauge-box">
          <div className="gauge-item">
            <div className="gauge-label">
              <span>{t.timetables.thisWeek}</span>
              <span>{weekPct}%</span>
            </div>
            <div className="gauge-track">
              <div className="gauge-fill" style={{ width: `${Math.min(100, weekPct)}%` }} />
            </div>
          </div>
          <div className="gauge-item">
            <div className="gauge-label">
              <span>{t.timetables.allWeeks}</span>
              <span>{allPct}%</span>
            </div>
            <div className="gauge-track">
              <div
                className="gauge-fill"
                style={{
                  width: `${Math.min(100, allPct)}%`,
                  background: 'linear-gradient(90deg, #3b82f6, #0ea5e9)',
                }}
              />
            </div>
          </div>
        </div>
      </div>

      {/* Main Split: Left Course List + Right Timetable Grid & Linked Mini-Grids */}
      <div className="split-main">
        <div className="panel">
          <div className="panel-header">
            <span>{t.timetables.eventsInTable}</span>
            <span className="stat-chip ok">{relevantCourses.length}</span>
          </div>
          <div className="panel-body">
            {relevantCourses.map((c) => {
              const alloc =
                project.weekAllocations.find((w) => w.courseId === c.id && w.week === activeWeek)
                  ?.periods ?? 0;
              const sched = project.slots.filter(
                (s) => s.courseId === c.id && s.week === activeWeek
              ).length;
              const left = alloc - sched;
              const chipCls = left === 0 ? 'ok' : left > 0 ? 'warn' : 'err';
              return (
                <div
                  key={c.id}
                  className={`event-row ${selectedCourseId === c.id ? 'selected' : ''}`}
                  onClick={() => {
                    setSelectedCourseId(c.id);
                    setBookingMode(false);
                  }}
                >
                  <div style={{ minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                      <span
                        className="code-badge course"
                        style={
                          c.color
                            ? {
                                background: `${c.color}20`,
                                borderColor: c.color,
                                color: '#0f172a',
                              }
                            : undefined
                        }
                      >
                        {c.code}
                      </span>
                      {c.locked && <Lock size={11} color="#64748b" />}
                    </div>
                    <div
                      style={{
                        fontSize: 11.5,
                        color: 'var(--text-muted)',
                        marginTop: 2,
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                      }}
                    >
                      {c.name}
                    </div>
                  </div>
                  <div style={{ textAlign: 'right', flexShrink: 0 }}>
                    <span className={`stat-chip ${chipCls}`}>
                      {sched}/{alloc}
                    </span>
                    {left > 0 && (
                      <div style={{ fontSize: 10, color: '#b45309', fontWeight: 700, marginTop: 2 }}>
                        +{left} {t.cols.left.toLowerCase()}
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
          <div
            style={{
              padding: '7px 10px',
              borderTop: '1px solid var(--border)',
              background: '#f8fafc',
              fontSize: 11,
              color: 'var(--text-faint)',
            }}
          >
            {t.timetables.clickToPlace}
          </div>
        </div>

        <div className="tt-wrapper">
          <div className="tt-grid-scroll">
            <table className="tt-table">
              <thead>
                <tr>
                  <th className="time-col">{project.language === 'en' ? 'Period' : 'Bloque'}</th>
                  {Array.from({ length: project.time.maxDays }, (_, di) => (
                    <th key={di}>{project.time.days[di] ?? `D${di + 1}`}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {Array.from({ length: project.time.maxPeriods }, (_, pi) => {
                  const period = pi + 1;
                  const bell = project.time.bells[pi] ?? `P${period}`;
                  return (
                    <tr key={period}>
                      <td className="time-cell">
                        <div style={{ fontWeight: 700, color: 'var(--text-main)' }}>B{period}</div>
                        <div style={{ fontSize: 10, color: 'var(--text-faint)' }}>{bell}</div>
                      </td>
                      {Array.from({ length: project.time.maxDays }, (_, di) => {
                        const day = di + 1;
                        const cellSlots = visibleSlots.filter(
                          (s) => s.day === day && s.period === period
                        );
                        const cellBooking = visibleBookings.find(
                          (b) => b.day === day && b.period === period
                        );
                        const wi = whatIfMap[`${day}_${period}`];
                        const isAvail =
                          whatIfEnabled &&
                          !bookingMode &&
                          selectedCourseId &&
                          wi?.available &&
                          cellSlots.length === 0;
                        const isConflictHint =
                          whatIfEnabled &&
                          !bookingMode &&
                          selectedCourseId &&
                          wi &&
                          !wi.available &&
                          cellSlots.length === 0;
                        const hasRealConflict = conflictCellKeys.has(`${day}_${period}`);

                        let tdClass = 'tt-cell-clickable';
                        if (cellBooking) tdClass += ' tt-cell-booking';
                        else if (isAvail) tdClass += ' tt-cell-avail';
                        else if (isConflictHint) tdClass += ' tt-cell-conflict';

                        const tooltip = cellBooking
                          ? `${t.timetables.booking}: ${cellBooking.label || '/////'}`
                          : wi && !wi.available
                          ? wi.reasons?.join(' | ')
                          : isAvail
                          ? `${t.timetables.available} — ${selectedCourse?.code}`
                          : '';

                        return (
                          <td
                            key={day}
                            className={tdClass}
                            title={tooltip}
                            onClick={() => handleCellClick(day, period)}
                          >
                            {cellBooking && (
                              <div
                                style={{
                                  fontSize: 10,
                                  fontWeight: 700,
                                  color: '#475569',
                                  padding: '2px 4px',
                                }}
                              >
                                {cellBooking.label || '/////'}
                              </div>
                            )}
                            {cellSlots.map((s) => {
                              const c = courseMap.get(s.courseId);
                              if (!c) return null;
                              const accent = c.color || '#2563eb';
                              const otherCodes = c.resources
                                .filter((rid) => rid !== selectedTargetId)
                                .map((rid) => resMap.get(rid))
                                .filter((r): r is NonNullable<typeof r> => Boolean(r))
                                .filter((r) => !s.roomId || r.category !== project.roomsCategory)
                                .map((r) => r.code);
                              if (s.roomId && s.roomId !== selectedTargetId) {
                                const rm = resMap.get(s.roomId);
                                if (rm) otherCodes.push(`[${rm.code}]`);
                              }
                              return (
                                <div
                                  key={s.id}
                                  className={`slot-card ${
                                    selectedCourseId === c.id ? 'highlighted' : ''
                                  } ${hasRealConflict ? 'has-conflict' : ''}`}
                                  style={{
                                    borderLeftColor: accent,
                                    background: `${accent}18`,
                                  }}
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    setSelectedCourseId(c.id);
                                  }}
                                >
                                  <div className="slot-card-head">
                                    <span>
                                      {hasRealConflict && (
                                        <AlertTriangle
                                          size={10}
                                          color="#dc2626"
                                          style={{ marginRight: 3 }}
                                        />
                                      )}
                                      {c.code}
                                    </span>
                                    <button
                                      className="slot-del-btn"
                                      title={
                                        project.language === 'en'
                                          ? 'Unschedule slot'
                                          : 'Desprogramar bloque'
                                      }
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        onUpdate(api.deleteSlot(s.id));
                                      }}
                                    >
                                      <X size={11} />
                                    </button>
                                  </div>
                                  <div className="slot-card-sub">{otherCodes.join(' · ')}</div>
                                </div>
                              );
                            })}
                          </td>
                        );
                      })}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Linked Resources Live Mini-Timetables ("Other Grids") */}
          <div className="linked-strip">
            <div className="panel-header" style={{ padding: '5px 10px', fontSize: 11.5 }}>
              <span>
                {t.timetables.linkedGrids}{' '}
                {selectedCourse && (
                  <strong style={{ color: 'var(--accent)', marginLeft: 4 }}>
                    — {selectedCourse.code} ({selectedCourse.name})
                  </strong>
                )}
              </span>
              <span style={{ fontSize: 10.5, color: 'var(--text-faint)' }}>
                {project.language === 'en' ? 'Click any mini-grid to open' : 'Clic en cualquier mini-horario para abrir'}
              </span>
            </div>
            <div className="linked-cards">
              {linkedResources.map((r) => {
                const cat = project.resourceCategories.find((c) => c.index === r.category);
                return (
                  <div
                    key={r.id}
                    className="mini-tt-card"
                    onClick={() => {
                      setViewMode('resource');
                      setSelectedTargetId(r.id);
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        fontSize: 11,
                      }}
                    >
                      <span className="code-badge resource" style={{ padding: '1px 5px', fontSize: 10 }}>
                        {r.code}
                      </span>
                      <span style={{ fontSize: 10, color: 'var(--text-faint)' }}>
                        {cat?.name ?? ''}
                      </span>
                    </div>
                    <div
                      className="mini-grid"
                      style={{
                        gridTemplateColumns: `repeat(${project.time.maxDays}, 1fr)`,
                        gridTemplateRows: `repeat(${Math.min(10, project.time.maxPeriods)}, 1fr)`,
                      }}
                    >
                      {Array.from({ length: Math.min(10, project.time.maxPeriods) }, (_, pi) =>
                        Array.from({ length: project.time.maxDays }, (_, di) => {
                          const d = di + 1;
                          const h = pi + 1;
                          const hitSlot = project.slots.find((s) => {
                            if (s.week !== activeWeek || s.day !== d || s.period !== h) return false;
                            if (s.roomId === r.id) return true;
                            const cc = courseMap.get(s.courseId);
                            return cc?.resources.includes(r.id);
                          });
                          const hitBook = project.bookings.some(
                            (b) =>
                              b.scope === 'timetable' &&
                              b.targetType === 'resource' &&
                              b.targetId === r.id &&
                              b.week === activeWeek &&
                              b.day === d &&
                              b.period === h
                          );
                          let cls = 'mini-cell';
                          if (hitSlot) {
                            cls += hitSlot.courseId === selectedCourseId ? ' selected-course' : ' busy';
                          } else if (hitBook) {
                            cls += ' booking';
                          }
                          return <div key={`${d}_${h}`} className={cls} />;
                        })
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </>
  );
};
