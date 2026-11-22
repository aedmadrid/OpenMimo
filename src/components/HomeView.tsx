import React, { useState } from 'react';
import {
  Plus,
  FolderOpen,
  Share2,
  Clock,
  Globe,
  Lock,
  CheckCircle2,
  Trash2,
  FlaskConical,
  BookOpen,
  Download,
  X,
} from 'lucide-react';
import type { ProjectSummary, WorkspaceResponse } from '../types';
import { AppLogo } from './AppLogo';

interface Props {
  workspace: WorkspaceResponse | null;
  lang: 'es' | 'en';
  onOpenLocal: (id: string) => void;
  onDeleteLocal: (id: string) => void;
  onExportAndDeleteLocal: (id: string) => Promise<void>;
  onOpenSample: (key: string) => void;
  onOpenWizard: () => void;
  onImportFile: () => void;
  onJoinShareCode: (code: string) => void;
  onJoinRemoteEntry: (hostUrl: string, projectId: string, shareCode: string) => void;
  onRefresh: () => void;
}

type HomeTab = 'recent' | 'shared' | 'samples' | 'guide';

const GUIDE_STEPS = {
  es: [
    ['1. Nuevo horario', 'Configura hora de inicio (ej. 08:30), días y bloques desde el asistente.'],
    ['2. Recursos y profesores', 'Importa docentes por CSV, añade notas MCP y marca horas no disponibles.'],
    ['3. Asignaturas globales y sesiones', 'Agrupa sesiones (ej. DNOP1, DNOP2) bajo una Guía Docente y exporta a Astro.'],
    ['4. Semanas y horarios', 'Reparte cargas por semana, coloca bloques o usa el optimizador automático.'],
    ['5. Publicar y compartir', 'Exporta la web de horarios con barra lateral o comparte en tiempo real por IPFS.'],
  ],
  en: [
    ['1. New schedule', 'Set start time (e.g. 08:30), days, and periods from the setup wizard.'],
    ['2. Resources & teachers', 'Import teachers via CSV, add MCP notes, and mark unavailable hours.'],
    ['3. Global subjects & sessions', 'Group sessions (e.g. DNOP1, DNOP2) under one Teaching Guide and export to Astro.'],
    ['4. Weeks & timetables', 'Allocate weekly hours, place blocks manually, or run the gap optimiser.'],
    ['5. Publish & share', 'Export sidebar web timetables or collaborate live via IPFS share codes.'],
  ],
};

export const HomeView: React.FC<Props> = ({
  workspace,
  lang,
  onOpenLocal,
  onDeleteLocal,
  onExportAndDeleteLocal,
  onOpenSample,
  onOpenWizard,
  onImportFile,
  onJoinShareCode,
  onJoinRemoteEntry,
}) => {
  const isEn = lang === 'en';
  const [tab, setTab] = useState<HomeTab>('recent');
  const [joinCode, setJoinCode] = useState('');
  const [pendingDelete, setPendingDelete] = useState<ProjectSummary | null>(null);
  const [deleting, setDeleting] = useState(false);

  const recent = workspace?.recent ?? [];
  const shared = workspace?.shared ?? [];
  const samples = workspace?.samples ?? [];
  const guideSteps = GUIDE_STEPS[isEn ? 'en' : 'es'];

  return (
    <div className="home-layout">
      {/* Minimalist Left Rail */}
      <aside className="home-rail">
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 6px 14px' }}>
          <AppLogo size={34} />
          <div style={{ fontWeight: 800, fontSize: 16, letterSpacing: '-0.3px' }}>OpenMimo</div>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginBottom: 16 }}>
          <button
            className="btn btn-primary"
            style={{ justifyContent: 'center', padding: '8px 12px' }}
            onClick={onOpenWizard}
          >
            <Plus size={14} /> {isEn ? 'New schedule' : 'Nuevo horario'}
          </button>
          <button
            className="btn"
            style={{ justifyContent: 'center', padding: '7px 12px' }}
            onClick={onImportFile}
          >
            <FolderOpen size={14} /> {isEn ? 'Open file' : 'Abrir archivo'}
          </button>
        </div>

        <nav className="home-nav">
          <button className={`home-nav-item ${tab === 'recent' ? 'active' : ''}`} onClick={() => setTab('recent')}>
            <Clock size={14} />
            <span>{isEn ? 'Recent' : 'Recientes'}</span>
            <span className="home-count">{recent.length}</span>
          </button>
          <button className={`home-nav-item ${tab === 'shared' ? 'active' : ''}`} onClick={() => setTab('shared')}>
            <Globe size={14} />
            <span>{isEn ? 'Shared' : 'Compartidos'}</span>
            <span className="home-count">{shared.length}</span>
          </button>
          <button className={`home-nav-item ${tab === 'samples' ? 'active' : ''}`} onClick={() => setTab('samples')}>
            <FlaskConical size={14} />
            <span>{isEn ? 'Examples' : 'Ejemplos'}</span>
            <span className="home-count">{samples.length}</span>
          </button>
          <button className={`home-nav-item ${tab === 'guide' ? 'active' : ''}`} onClick={() => setTab('guide')}>
            <BookOpen size={14} />
            <span>{isEn ? 'Quick guide' : 'Guía rápida'}</span>
          </button>
        </nav>
      </aside>

      {/* Calm Right Content Stage */}
      <section className="home-stage">
        {tab === 'recent' && (
          <div className="home-stage-inner">
            <h2 className="home-heading">{isEn ? 'Recent projects' : 'Proyectos recientes'}</h2>
            {recent.length === 0 ? (
              <div className="home-empty">
                {isEn ? 'No saved schedules yet.' : 'Aún no hay horarios guardados.'}
              </div>
            ) : (
              <div className="home-list">
                {recent.map((p) => (
                  <div key={p.id} className="home-row" onClick={() => onOpenLocal(p.id)}>
                    <div style={{ minWidth: 0, flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <strong style={{ fontSize: 14 }}>{p.name}</strong>
                        {p.isOpen && <span className="stat-chip ok">{isEn ? 'Open' : 'Abierto'}</span>}
                        {p.shared && <span className="stat-chip warn">IPFS</span>}
                      </div>
                      <div style={{ fontSize: 12, color: 'var(--text-faint)', marginTop: 2 }}>
                        {p.institution} · {p.coursesCount} {isEn ? 'sessions' : 'sesiones'} · {p.resourcesCount}{' '}
                        {isEn ? 'resources' : 'recursos'}
                      </div>
                    </div>
                    <button
                      className="btn btn-icon"
                      title={isEn ? 'Delete' : 'Borrar'}
                      onClick={(e) => {
                        e.stopPropagation();
                        setPendingDelete(p);
                      }}
                    >
                      <Trash2 size={14} color="#dc2626" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'shared' && (
          <div className="home-stage-inner">
            <h2 className="home-heading">{isEn ? 'Shared (IPFS)' : 'Compartidos (IPFS)'}</h2>
            <form
              style={{ display: 'flex', gap: 8, marginBottom: 16 }}
              onSubmit={(e) => {
                e.preventDefault();
                if (joinCode.trim()) {
                  onJoinShareCode(joinCode.trim());
                  setJoinCode('');
                }
              }}
            >
              <input
                className="input"
                style={{ flex: 1 }}
                placeholder={isEn ? 'Paste IPFS share code (OMIPFS-...)' : 'Pega un código compartido (OMIPFS-...)'}
                value={joinCode}
                onChange={(e) => setJoinCode(e.target.value)}
              />
              <button className="btn btn-primary" type="submit" disabled={!joinCode.trim()}>
                <Share2 size={13} /> {isEn ? 'Join' : 'Unirse'}
              </button>
            </form>

            {shared.length === 0 ? (
              <div className="home-empty">
                {isEn
                  ? 'No shared schedules yet. Remote files are editable only while the host PC has them open.'
                  : 'Sin horarios compartidos. Los archivos remotos solo se editan si el equipo original los tiene abiertos.'}
              </div>
            ) : (
              <div className="home-list">
                {shared.map((s) => (
                  <div
                    key={s.shareCode || s.projectId}
                    className={`home-row ${!s.hostOpen ? 'offline' : ''}`}
                    onClick={() => {
                      if (s.hostOpen) onJoinRemoteEntry(s.hostUrl, s.projectId, s.shareCode);
                    }}
                  >
                    <div style={{ minWidth: 0, flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <strong style={{ fontSize: 14 }}>{s.name}</strong>
                        {s.hostOpen ? (
                          <span className="stat-chip ok" style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}>
                            <CheckCircle2 size={10} /> {isEn ? 'Online' : 'En línea'}
                          </span>
                        ) : (
                          <span className="stat-chip err" style={{ display: 'inline-flex', alignItems: 'center', gap: 3 }}>
                            <Lock size={10} /> {isEn ? 'Host closed' : 'Anfitrión cerrado'}
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: 12, color: 'var(--text-faint)', marginTop: 2 }}>
                        {s.institution || s.hostUrl}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'samples' && (
          <div className="home-stage-inner">
            <h2 className="home-heading">{isEn ? 'Example schedules' : 'Horarios de ejemplo'}</h2>
            <div className="home-cards-grid">
              {samples.map((sm) => (
                <div key={sm.key} className="home-sample-card" onClick={() => onOpenSample(sm.key)}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8 }}>
                    <strong style={{ fontSize: 13.5 }}>{isEn ? sm.titleEn : sm.titleEs}</strong>
                    <span className="stat-chip ok" style={{ fontSize: 10, flexShrink: 0 }}>{sm.badge}</span>
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 6, lineHeight: 1.4 }}>
                    {isEn ? sm.descEn : sm.descEs}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {tab === 'guide' && (
          <div className="home-stage-inner">
            <h2 className="home-heading">{isEn ? 'Quick guide' : 'Guía rápida'}</h2>
            <div className="home-list">
              {guideSteps.map(([title, desc], i) => (
                <div key={i} className="home-row" style={{ cursor: 'default' }}>
                  <div>
                    <strong style={{ fontSize: 13.5 }}>{title}</strong>
                    <div style={{ fontSize: 12.5, color: 'var(--text-muted)', marginTop: 2 }}>{desc}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </section>

      {/* Confirm Delete Before Export Dialog */}
      {pendingDelete && (
        <div className="modal-overlay" onClick={() => !deleting && setPendingDelete(null)}>
          <div className="modal-card" style={{ maxWidth: 440 }} onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>{isEn ? 'Delete project?' : '¿Borrar proyecto?'}</h3>
              <button className="btn btn-icon" disabled={deleting} onClick={() => setPendingDelete(null)}>
                <X size={15} />
              </button>
            </div>
            <div className="modal-body" style={{ fontSize: 13, color: 'var(--text-muted)', lineHeight: 1.5 }}>
              {isEn ? (
                <>
                  Do you want to delete <strong>{pendingDelete.name}</strong> without exporting a backup first?
                </>
              ) : (
                <>
                  ¿Seguro que quieres borrar <strong>{pendingDelete.name}</strong> sin exportarlo antes?
                </>
              )}
            </div>
            <div className="modal-footer" style={{ justifyContent: 'space-between' }}>
              <button className="btn" disabled={deleting} onClick={() => setPendingDelete(null)}>
                {isEn ? 'Cancel' : 'Cancelar'}
              </button>
              <div style={{ display: 'flex', gap: 8 }}>
                <button
                  className="btn btn-primary"
                  disabled={deleting}
                  onClick={async () => {
                    setDeleting(true);
                    try {
                      await onExportAndDeleteLocal(pendingDelete.id);
                      setPendingDelete(null);
                    } finally {
                      setDeleting(false);
                    }
                  }}
                >
                  <Download size={13} /> {isEn ? 'Export & delete' : 'Exportar y borrar'}
                </button>
                <button
                  className="btn btn-danger"
                  disabled={deleting}
                  onClick={() => {
                    onDeleteLocal(pendingDelete.id);
                    setPendingDelete(null);
                  }}
                >
                  {isEn ? 'Delete anyway' : 'Borrar sin exportar'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
