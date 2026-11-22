import React, { useEffect, useState } from 'react';
import {
  Plus,
  Trash2,
  BookOpenCheck,
  Download,
  Users,
  Radio,
  Share2,
  Save,
} from 'lucide-react';
import type { GuideGroupAssignment, Project, TeachingGuide } from '../types';
import { api } from '../api';

interface Props {
  project: Project;
  initialGuideId?: string;
  linkCourseId?: string;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
  onOpenCollab: () => void;
}

function downloadBase64Zip(fileName: string, base64: string) {
  const bin = atob(base64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  const blob = new Blob([bytes], { type: 'application/zip' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = fileName;
  a.click();
  URL.revokeObjectURL(url);
}

function emptyGuide(project: Project, seedTitle = ''): TeachingGuide {
  return {
    id: '',
    slug: '',
    title: seedTitle,
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
    groups: [{ group: '1º A', teacher: '', shift: 'Mañana', notes: '' }],
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
    updatedAt: '',
  };
}

export const GuidesView: React.FC<Props> = ({
  project,
  initialGuideId,
  linkCourseId,
  onUpdate,
  onOpenCollab,
}) => {
  const isEn = project.language === 'en';
  const guides = project.guides || [];
  const [selectedId, setSelectedId] = useState<string>(initialGuideId || guides[0]?.id || '__new__');
  const [section, setSection] = useState<'id' | 'groups' | 'contents' | 'eval'>('id');
  const [form, setForm] = useState<TeachingGuide>(() => {
    const existing = guides.find((g) => g.id === (initialGuideId || guides[0]?.id));
    if (existing) return { ...existing, groups: [...(existing.groups || [])] };
    const linkedCourse = project.courses.find((c) => c.id === linkCourseId);
    return emptyGuide(project, linkedCourse?.name || '');
  });
  const [containedIds, setContainedIds] = useState<string[]>(() => {
    const gid = initialGuideId || guides[0]?.id;
    if (gid) return project.courses.filter((c) => c.guideId === gid).map((c) => c.id);
    return linkCourseId ? [linkCourseId] : [];
  });
  const [exporting, setExporting] = useState(false);

  // Sync if user jumped from a course or external real-time collaborator edited the selected guide
  useEffect(() => {
    if (initialGuideId && initialGuideId !== selectedId) {
      const target = guides.find((g) => g.id === initialGuideId);
      if (target) {
        setSelectedId(target.id);
        setForm({ ...target, groups: [...(target.groups || [])] });
        setContainedIds(project.courses.filter((c) => c.guideId === target.id).map((c) => c.id));
      }
    } else if (!initialGuideId && linkCourseId && selectedId !== '__new__') {
      const linked = project.courses.find((c) => c.id === linkCourseId);
      if (linked && !linked.guideId) {
        setSelectedId('__new__');
        setForm(emptyGuide(project, linked.name));
        setContainedIds([linked.id]);
      }
    }
  }, [initialGuideId, linkCourseId]);

  const selectGuide = (g: TeachingGuide) => {
    setSelectedId(g.id);
    setForm({ ...g, groups: [...(g.groups || [])] });
    setContainedIds(project.courses.filter((c) => c.guideId === g.id).map((c) => c.id));
  };

  const startNew = () => {
    setSelectedId('__new__');
    setForm(emptyGuide(project));
    setContainedIds([]);
  };

  const toggleContained = (cid: string) => {
    setContainedIds((prev) => (prev.includes(cid) ? prev.filter((x) => x !== cid) : [...prev, cid]));
  };

  const addGroupRow = () => {
    setForm({
      ...form,
      groups: [...form.groups, { group: '', teacher: '', shift: 'Tarde', notes: '' }],
    });
  };

  const updateGroupRow = (idx: number, patch: Partial<GuideGroupAssignment>) => {
    const next = form.groups.map((row, i) => (i === idx ? { ...row, ...patch } : row));
    setForm({ ...form, groups: next });
  };

  const removeGroupRow = (idx: number) => {
    setForm({ ...form, groups: form.groups.filter((_, i) => i !== idx) });
  };

  const handleSave = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!form.title.trim()) return;
    const res = await onUpdate(
      api.saveGuide(
        { ...form, containedCourseIds: containedIds, syncContained: true },
        selectedId === '__new__' ? linkCourseId : ''
      ),
      isEn ? 'Teaching guide saved' : 'Guía docente guardada'
    );
    if (res?.guide?.id) {
      setSelectedId(res.guide.id);
      setForm(res.guide);
    }
  };

  const handleExportAstro = async () => {
    setExporting(true);
    try {
      const res = await api.exportAstroZip();
      downloadBase64Zip(res.fileName, res.base64Zip);
    } finally {
      setExporting(false);
    }
  };

  const linkedCourses = project.courses.filter((c) => c.guideId && c.guideId === form.id);
  const isCollabActive = Boolean(project.collab?.shared || project.collab?.isRemote);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, gap: 10 }}>
      {/* Studio Subbar */}
      <div className="view-subbar">
        <div className="control-group">
          <BookOpenCheck size={16} color="#6366f1" />
          <strong>{isEn ? 'Teaching Guides' : 'Guías Docentes'}</strong>
          <span className="stat-chip ok">{guides.length}</span>
          {isCollabActive ? (
            <button
              className="btn btn-sm btn-ok"
              onClick={onOpenCollab}
              title={isEn ? 'Live collaboration active for teachers' : 'Edición colaborativa en vivo activa'}
            >
              <Radio size={12} /> {isEn ? 'Live collaborative' : 'Colaborativo en vivo'}
            </button>
          ) : (
            <button className="btn btn-sm" onClick={onOpenCollab}>
              <Share2 size={12} /> {isEn ? 'Collaborate with teachers (IPFS)' : 'Colaborar con profesores (IPFS)'}
            </button>
          )}
        </div>
        <div className="control-group">
          <button className="btn btn-sm" onClick={startNew}>
            <Plus size={13} /> {isEn ? 'New guide' : 'Nueva guía'}
          </button>
          <button
            className="btn btn-sm btn-primary"
            disabled={exporting}
            onClick={handleExportAstro}
          >
            <Download size={13} />{' '}
            {exporting
              ? isEn
                ? 'Exporting...'
                : 'Exportando...'
              : isEn
              ? 'Export Astro Static Site (.zip)'
              : 'Exportar web Astro (.zip)'}
          </button>
        </div>
      </div>

      {/* Main Split Studio */}
      <div className="split-main">
        {/* Left Guide List */}
        <div className="panel">
          <div className="panel-header">
            <span>{isEn ? 'Global subjects' : 'Asignaturas globales'} ({guides.length})</span>
            <button className="btn btn-sm btn-primary" onClick={startNew}>
              <Plus size={12} />
            </button>
          </div>
          <div className="panel-body">
            {guides.map((g) => {
              const countContained = project.courses.filter((c) => c.guideId === g.id).length;
              return (
                <div
                  key={g.id}
                  data-ctx="guide"
                  data-ctx-id={g.id}
                  className={`event-row ${selectedId === g.id ? 'selected' : ''}`}
                  onClick={() => selectGuide(g)}
                >
                  <div style={{ minWidth: 0 }}>
                    <div style={{ fontWeight: 700, fontSize: 12.5 }}>{g.title}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-faint)' }}>
                      {g.creditsEcts} ECTS · {countContained}{' '}
                      {isEn ? 'session(s)' : 'sesión(es)'} · {(g.groups || []).length}{' '}
                      {isEn ? 'group(s)' : 'grupo(s)'}
                    </div>
                  </div>
                </div>
              );
            })}
            {guides.length === 0 && (
              <div style={{ padding: 16, fontSize: 12, color: 'var(--text-faint)', textAlign: 'center' }}>
                {isEn ? 'Create your first teaching guide.' : 'Crea tu primera guía docente.'}
              </div>
            )}
          </div>
        </div>

        {/* Right Full Editor */}
        <form onSubmit={handleSave} className="panel" style={{ minWidth: 0 }}>
          <div className="panel-header" style={{ padding: '6px 12px', gap: 10 }}>
            <div className="tab-bar" style={{ marginBottom: 0, borderBottom: 'none' }}>
              <button type="button" className={`tab-btn ${section === 'id' ? 'active' : ''}`} onClick={() => setSection('id')}>
                {isEn ? '1. Identification' : '1. Identificación'}
              </button>
              <button type="button" className={`tab-btn ${section === 'groups' ? 'active' : ''}`} onClick={() => setSection('groups')}>
                <Users size={12} style={{ marginRight: 4, verticalAlign: -2 }} />
                {isEn ? '2. Sessions & Teachers' : '2. Sesiones y Profesorado'} ({containedIds.length})
              </button>
              <button type="button" className={`tab-btn ${section === 'contents' ? 'active' : ''}`} onClick={() => setSection('contents')}>
                {isEn ? '3. Competencies & Contents' : '3. Competencias y Contenidos'}
              </button>
              <button type="button" className={`tab-btn ${section === 'eval' ? 'active' : ''}`} onClick={() => setSection('eval')}>
                {isEn ? '4. Activities & Evaluation' : '4. Actividades y Evaluación'}
              </button>
            </div>
            <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
              {form.id && (
                <button
                  type="button"
                  className="btn btn-sm btn-danger"
                  onClick={() => {
                    onUpdate(api.deleteGuide(form.id), isEn ? 'Guide deleted' : 'Guía eliminada');
                    startNew();
                  }}
                >
                  <Trash2 size={12} />
                </button>
              )}
              <button type="submit" className="btn btn-sm btn-primary">
                <Save size={12} /> {isEn ? 'Save' : 'Guardar'}
              </button>
            </div>
          </div>

          <div className="panel-body" style={{ padding: 14, display: 'flex', flexDirection: 'column', gap: 12 }}>
            {section === 'id' && (
              <>
                <div className="form-grid">
                  <div className="field">
                    <label>{isEn ? 'Global subject title' : 'Título de la asignatura global'} *</label>
                    <input className="input" required value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Degree' : 'Titulación'}</label>
                    <input className="input" value={form.degree} onChange={(e) => setForm({ ...form, degree: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Degree level' : 'Nivel de título'}</label>
                    <input className="input" value={form.degreeLevel} onChange={(e) => setForm({ ...form, degreeLevel: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Department' : 'Departamento'}</label>
                    <input className="input" value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Type' : 'Tipo'}</label>
                    <select className="select" value={form.courseType} onChange={(e) => setForm({ ...form, courseType: e.target.value })}>
                      <option value="Formación básica">Formación básica</option>
                      <option value="Obligatoria">Obligatoria</option>
                      <option value="Optativa">Optativa</option>
                      <option value="TFG / TFM">TFG / TFM</option>
                    </select>
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Character' : 'Carácter'}</label>
                    <input className="input" value={form.character} onChange={(e) => setForm({ ...form, character: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Specialty / Itinerary' : 'Especialidad / Itinerario'}</label>
                    <input className="input" placeholder="Ciencias, Humanidades, Ingeniería, Artes..." value={form.specialty} onChange={(e) => setForm({ ...form, specialty: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Subject area' : 'Materia'}</label>
                    <input className="input" value={form.subjectArea} onChange={(e) => setForm({ ...form, subjectArea: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Academic year & Semester' : 'Curso y Semestre'}</label>
                    <div style={{ display: 'flex', gap: 6 }}>
                      <input className="input" style={{ width: '40%' }} value={form.academicYear} onChange={(e) => setForm({ ...form, academicYear: e.target.value })} />
                      <input className="input" style={{ width: '60%' }} value={form.semester} onChange={(e) => setForm({ ...form, semester: e.target.value })} />
                    </div>
                  </div>
                  <div className="field">
                    <label>{isEn ? 'ECTS / Total h / In-person h' : 'ECTS / Horas totales / Presenciales'}</label>
                    <div style={{ display: 'flex', gap: 6 }}>
                      <input className="input" type="number" step="0.5" style={{ width: '33%' }} value={form.creditsEcts} onChange={(e) => setForm({ ...form, creditsEcts: Number(e.target.value) })} />
                      <input className="input" type="number" style={{ width: '33%' }} value={form.totalHours} onChange={(e) => setForm({ ...form, totalHours: Number(e.target.value) })} />
                      <input className="input" type="number" style={{ width: '34%' }} value={form.inPersonHours} onChange={(e) => setForm({ ...form, inPersonHours: Number(e.target.value) })} />
                    </div>
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Prerequisites' : 'Prelación / Requisitos previos'}</label>
                    <input className="input" value={form.prerequisites} onChange={(e) => setForm({ ...form, prerequisites: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Teaching language(s)' : 'Idioma(s) de impartición'}</label>
                    <input className="input" value={form.languages} onChange={(e) => setForm({ ...form, languages: e.target.value })} />
                  </div>
                </div>
                <div className="field">
                  <label>{isEn ? 'Subject description' : 'Descripción de la asignatura'}</label>
                  <textarea className="input" rows={4} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
                </div>
              </>
            )}

            {section === 'groups' && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <div style={{ background: '#f8fafc', border: '1px solid var(--border)', borderRadius: 8, padding: 12 }}>
                  <div style={{ fontSize: 12, fontWeight: 700, marginBottom: 6 }}>
                    {isEn
                      ? 'Contained sessions (e.g. DNOP1, DNOP2, DNOP3)'
                      : 'Sesiones contenidas en esta asignatura global (ej. DNOP1, DNOP2, DNOP3)'}
                  </div>
                  <div className="pill-list">
                    {project.courses.map((c) => {
                      const active = containedIds.includes(c.id);
                      return (
                        <button
                          type="button"
                          key={c.id}
                          className={`btn btn-sm ${active ? 'btn-primary' : ''}`}
                          onClick={() => toggleContained(c.id)}
                          title={c.name}
                        >
                          {c.code}
                        </button>
                      );
                    })}
                  </div>
                </div>

                {linkedCourses.length > 0 && (
                  <div style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>
                    {linkedCourses.map((c) => `${c.code} (${c.name})`).join(' · ')}
                  </div>
                )}

                <strong style={{ fontSize: 12 }}>
                  {isEn ? 'Parallel groups, teachers & shifts' : 'Grupos paralelos, profesorado y turnos'}
                </strong>
                {form.groups.map((row, idx) => (
                  <div key={idx} style={{ display: 'grid', gridTemplateColumns: '1fr 1.2fr 110px 1fr 32px', gap: 6, alignItems: 'center' }}>
                    <input
                      className="input"
                      placeholder={isEn ? 'Group (e.g. 2º A)' : 'Grupo (ej. 2º Gráfico A)'}
                      value={row.group}
                      onChange={(e) => updateGroupRow(idx, { group: e.target.value })}
                    />
                    <input
                      className="input"
                      placeholder={isEn ? 'Teacher' : 'Profesor/a'}
                      value={row.teacher}
                      onChange={(e) => updateGroupRow(idx, { teacher: e.target.value })}
                    />
                    <select className="select" value={row.shift} onChange={(e) => updateGroupRow(idx, { shift: e.target.value })}>
                      <option value="Mañana">{isEn ? 'Morning' : 'Mañana'}</option>
                      <option value="Tarde">{isEn ? 'Afternoon' : 'Tarde'}</option>
                      <option value="Paralelo">{isEn ? 'Parallel' : 'Paralelo'}</option>
                    </select>
                    <input
                      className="input"
                      placeholder={isEn ? 'Notes / Room' : 'Notas / Aula'}
                      value={row.notes || ''}
                      onChange={(e) => updateGroupRow(idx, { notes: e.target.value })}
                    />
                    <button type="button" className="btn btn-icon" onClick={() => removeGroupRow(idx)}>
                      <Trash2 size={13} color="#dc2626" />
                    </button>
                  </div>
                ))}
                <div>
                  <button type="button" className="btn btn-sm" onClick={addGroupRow}>
                    <Plus size={13} /> {isEn ? 'Add group / teacher' : 'Añadir grupo / profesor'}
                  </button>
                </div>
              </div>
            )}

            {section === 'contents' && (
              <>
                <div className="field">
                  <label>{isEn ? 'Specific competencies' : 'Competencias específicas'}</label>
                  <textarea className="input" rows={4} value={form.competencies} onChange={(e) => setForm({ ...form, competencies: e.target.value })} />
                </div>
                <div className="field">
                  <label>{isEn ? 'Learning outcomes' : 'Resultados del aprendizaje'}</label>
                  <textarea className="input" rows={4} value={form.learningOutcomes} onChange={(e) => setForm({ ...form, learningOutcomes: e.target.value })} />
                </div>
                <div className="field">
                  <label>{isEn ? 'Contents / Syllabus' : 'Contenidos'}</label>
                  <textarea className="input" rows={6} value={form.contents} onChange={(e) => setForm({ ...form, contents: e.target.value })} />
                </div>
              </>
            )}

            {section === 'eval' && (
              <>
                <div className="form-grid">
                  <div className="field">
                    <label>{isEn ? 'Theoretical activities' : 'Actividades teóricas'}</label>
                    <textarea className="input" rows={3} value={form.theoryActivities} onChange={(e) => setForm({ ...form, theoryActivities: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Practical activities' : 'Actividades prácticas'}</label>
                    <textarea className="input" rows={3} value={form.practiceActivities} onChange={(e) => setForm({ ...form, practiceActivities: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Other mandatory activities' : 'Otras actividades obligatorias'}</label>
                    <textarea className="input" rows={3} value={form.otherActivities} onChange={(e) => setForm({ ...form, otherActivities: e.target.value })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Student autonomous work' : 'Trabajo autónomo del estudiante'}</label>
                    <textarea className="input" rows={3} value={form.studentWorkHours} onChange={(e) => setForm({ ...form, studentWorkHours: e.target.value })} />
                  </div>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
                  <div className="field">
                    <label>{isEn ? 'Attitude & participation (%)' : 'Actitud y participación (%)'}</label>
                    <input className="input" type="number" min={0} max={100} value={form.evalAttitudePct} onChange={(e) => setForm({ ...form, evalAttitudePct: Number(e.target.value) })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Written exams (%)' : 'Pruebas escritas (%)'}</label>
                    <input className="input" type="number" min={0} max={100} value={form.evalWrittenPct} onChange={(e) => setForm({ ...form, evalWrittenPct: Number(e.target.value) })} />
                  </div>
                  <div className="field">
                    <label>{isEn ? 'Practical projects (%)' : 'Pruebas prácticas (%)'}</label>
                    <input className="input" type="number" min={0} max={100} value={form.evalPracticalPct} onChange={(e) => setForm({ ...form, evalPracticalPct: Number(e.target.value) })} />
                  </div>
                </div>
                <div className="field">
                  <label>{isEn ? 'Evaluation criteria' : 'Criterios y sistema de evaluación'}</label>
                  <textarea className="input" rows={3} value={form.evalCriteria} onChange={(e) => setForm({ ...form, evalCriteria: e.target.value })} />
                </div>
                <div className="field">
                  <label>{isEn ? 'Bibliography' : 'Bibliografía y recursos'}</label>
                  <textarea className="input" rows={3} value={form.bibliography} onChange={(e) => setForm({ ...form, bibliography: e.target.value })} />
                </div>
              </>
            )}
          </div>
        </form>
      </div>
    </div>
  );
};
