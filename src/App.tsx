import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Users,
  BookOpen,
  CalendarRange,
  Grid3X3,
  Download,
  Sparkles,
  DoorOpen,
  Undo2,
  Redo2,
  CheckCircle2,
  AlertTriangle,
  Home,
  Plus,
  X,
  BookOpenCheck,
  Share2,
  Radio,
  Code2,
  ArrowUpCircle,
  Bug,
} from 'lucide-react';
import type {
  ApiPayload,
  MainView,
  ModalKind,
  UpdateCheckResult,
  WorkspaceResponse,
} from './types';
import { api, getApiBase } from './api';
import { I18N } from './i18n';
import { TimetablesView } from './components/TimetablesView';
import { WeeksView } from './components/WeeksView';
import { CoursesView } from './components/CoursesView';
import { ResourcesView } from './components/ResourcesView';
import { GuidesView } from './components/GuidesView';
import { ScriptsView } from './components/ScriptsView';
import { HomeView } from './components/HomeView';
import { Modals } from './components/Modals';
import { FeatureModals } from './components/FeatureModals';
import { ContextMenu, type ContextMenuItem } from './components/ContextMenu';
import { AppLogo } from './components/AppLogo';

declare global {
  interface Window {
    __OPENMIMO_INIT_VIEW__?: MainView | 'home' | null;
    __OPENMIMO_INIT_MODAL__?: ModalKind;
    __OPENMIMO_INIT_LANG__?: 'es' | 'en';
    __OPENMIMO_NATIVE_MENU__?: boolean;
    __OPENMIMO_MAC_CSD__?: boolean;
    __OPENMIMO_PLATFORM__?: string;
    __openmimoMenuAction?: (actionId: string) => void;
  }
}

function detectInitialLang(): 'es' | 'en' {
  if (window.__OPENMIMO_INIT_LANG__ === 'en' || window.__OPENMIMO_INIT_LANG__ === 'es') {
    return window.__OPENMIMO_INIT_LANG__;
  }
  const nav = (navigator.language || 'es').toLowerCase();
  return nav.startsWith('en') ? 'en' : 'es';
}

export const App: React.FC = () => {
  const [data, setData] = useState<ApiPayload | null>(null);
  const [workspace, setWorkspace] = useState<WorkspaceResponse | null>(null);
  const [showHome, setShowHome] = useState<boolean>(
    !window.__OPENMIMO_INIT_VIEW__ && !window.__OPENMIMO_INIT_MODAL__
  );
  const [view, setView] = useState<MainView>(
    window.__OPENMIMO_INIT_VIEW__ && window.__OPENMIMO_INIT_VIEW__ !== 'home'
      ? window.__OPENMIMO_INIT_VIEW__
      : 'timetables'
  );
  const [lang, setLang] = useState<'es' | 'en'>(detectInitialLang);
  const [openMenu, setOpenMenu] = useState<string | null>(null);
  const [activeWeek, setActiveWeek] = useState(1);
  const [modal, setModal] = useState<ModalKind>(window.__OPENMIMO_INIT_MODAL__ || null);
  const [optionsTab, setOptionsTab] = useState<'limits' | 'bells' | 'categories' | 'ai'>('limits');
  const [guideCtx, setGuideCtx] = useState<{ guideId?: string; linkCourseId?: string }>({});
  const [mcpEnabled, setMcpEnabled] = useState<boolean>(true);
  const [mcpHttpUrl, setMcpHttpUrl] = useState<string>('');
  const [toast, setToast] = useState<{ msg: string; error?: boolean } | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [debugError, setDebugError] = useState<{ msg: string; stack?: string } | null>(null);
  const [updateInfo, setUpdateInfo] = useState<UpdateCheckResult | null>(null);
  const [ctxMenu, setCtxMenu] = useState<{ x: number; y: number; items: ContextMenuItem[] } | null>(null);
  const [renamingTabId, setRenamingTabId] = useState<string | null>(null);
  const [renameDraft, setRenameDraft] = useState<string>('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const hasNativeMenu = Boolean(window.__OPENMIMO_NATIVE_MENU__);
  const isMacCSD = Boolean(window.__OPENMIMO_MAC_CSD__);

  const showToast = useCallback((msg: string, error = false) => {
    setToast({ msg, error });
    setTimeout(() => setToast((cur) => (cur?.msg === msg ? null : cur)), 3200);
  }, []);

  // Capture any unexpected runtime error so user always sees a debug panel instead of blank screen
  useEffect(() => {
    const onErr = (ev: ErrorEvent) => {
      setDebugError({
        msg: ev.message || 'Runtime Error',
        stack: ev.error?.stack || `${ev.filename || ''}:${ev.lineno || ''}:${ev.colno || ''}`,
      });
    };
    const onRej = (ev: PromiseRejectionEvent) => {
      const reason = ev.reason;
      setDebugError({
        msg: reason?.message || String(reason || 'Unhandled Promise Rejection'),
        stack: reason?.stack,
      });
    };
    window.addEventListener('error', onErr);
    window.addEventListener('unhandledrejection', onRej);
    return () => {
      window.removeEventListener('error', onErr);
      window.removeEventListener('unhandledrejection', onRej);
    };
  }, []);

  const syncNativeLanguage = useCallback((nextLang: 'es' | 'en') => {
    setLang(nextLang);
    document.documentElement.lang = nextLang;
    if (window.__TAURI_INTERNALS__?.invoke) {
      window.__TAURI_INTERNALS__.invoke('set_app_language', { lang: nextLang }).catch(() => {});
    }
  }, []);

  const refreshWorkspace = useCallback(async () => {
    try {
      const ws = await api.getWorkspace();
      setWorkspace(ws);
      setMcpEnabled(ws.mcpEnabled);
      setMcpHttpUrl(ws.mcpHttpUrl);
    } catch {
      // ignore transient error
    }
  }, []);

  const runUpdateCheck = useCallback(
    async (notifyIfUpToDate = false) => {
      try {
        const res = await api.checkUpdates();
        setUpdateInfo(res);
        if (notifyIfUpToDate) {
          if (res.hasUpdate && res.latestVersion) {
            showToast(
              lang === 'en'
                ? `OpenMimo v${res.latestVersion} available!`
                : `¡Nueva versión OpenMimo v${res.latestVersion} disponible!`
            );
          } else {
            showToast(
              lang === 'en'
                ? `OpenMimo v${res.currentVersion} is up to date`
                : `OpenMimo v${res.currentVersion} está actualizado`
            );
          }
        }
      } catch {
        // silent offline fallback
      }
    },
    [lang, showToast]
  );

  const loadInitial = useCallback(async () => {
    for (let attempt = 0; attempt < 25; attempt++) {
      try {
        const [projRes, ws] = await Promise.all([api.getProject(), api.getWorkspace()]);
        const effectiveLang: 'es' | 'en' =
          window.__OPENMIMO_INIT_LANG__ ||
          (ws.systemLang === 'en' ? 'en' : detectInitialLang());
        setData(projRes);
        setWorkspace(ws);
        setMcpEnabled(ws.mcpEnabled);
        setMcpHttpUrl(ws.mcpHttpUrl);
        syncNativeLanguage(effectiveLang);
        setLoadError(null);
        // Non-blocking startup update check against https://github.com/aedmadrid/openmimo/releases
        if (!window.__OPENMIMO_SCREENSHOT__) {
          api.checkUpdates().then(setUpdateInfo).catch(() => {});
        }
        return;
      } catch (e: any) {
        if (attempt === 24) {
          setLoadError(e?.message || 'No se pudo conectar con el motor de OpenMimo');
        }
        await new Promise((r) => setTimeout(r, 180));
      }
    }
  }, [syncNativeLanguage]);

  useEffect(() => {
    loadInitial();
  }, [loadInitial]);

  const onUpdate = useCallback(
    async (promise: Promise<ApiPayload>, toastMsg?: string) => {
      try {
        const res = await promise;
        setData(res);
        if (typeof res.mcpEnabled === 'boolean') setMcpEnabled(res.mcpEnabled);
        if (res.mcpHttpUrl) setMcpHttpUrl(res.mcpHttpUrl);
        refreshWorkspace();
        if (toastMsg) showToast(toastMsg);
        return res;
      } catch (e: any) {
        showToast(e?.message || 'Error', true);
      }
    },
    [showToast, refreshWorkspace]
  );

  const handleQuickExportOmimo = useCallback(async () => {
    try {
      const exp = await api.exportFile('omimo');
      const blob = new Blob([exp.content], { type: 'application/json;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = exp.fileName;
      a.click();
      URL.revokeObjectURL(url);
      showToast(lang === 'en' ? `Saved ${exp.fileName}` : `Guardado ${exp.fileName}`);
    } catch (e: any) {
      showToast(e?.message || 'Error', true);
    }
  }, [lang, showToast]);

  // Real-time SSE listener + fallback sync when active project is shared or remote (Timetables + Collaborative Guides!)
  useEffect(() => {
    const proj = data?.project;
    if (!proj || (!proj.collab?.shared && !proj.collab?.isRemote)) return;
    let es: EventSource | null = null;
    let pollTimer: number | null = null;
    let cancelled = false;
    const syncLatest = () => {
      if (proj.collab?.isRemote) {
        api.pullCollab().then(setData).catch(() => {});
      } else {
        api.getProject().then(setData).catch(() => {});
      }
    };
    (async () => {
      const base = proj.collab.isRemote && proj.collab.hostUrl ? proj.collab.hostUrl : await getApiBase();
      if (cancelled) return;
      es = new EventSource(`${base}/api/collab/events?projectId=${encodeURIComponent(proj.id)}`);
      es.addEventListener('updated', syncLatest);
      es.addEventListener('closed', () => {
        showToast(
          lang === 'en'
            ? 'Host closed the shared file (read-only)'
            : 'El equipo anfitrión cerró el archivo (solo lectura)',
          true
        );
        syncLatest();
        refreshWorkspace();
      });
      if (proj.collab.isRemote) {
        pollTimer = window.setInterval(syncLatest, 5000);
      }
    })();
    return () => {
      cancelled = true;
      if (pollTimer !== null) window.clearInterval(pollTimer);
      es?.close();
    };
  }, [data?.project?.id, data?.project?.collab?.shared, data?.project?.collab?.isRemote, data?.project?.collab?.hostUrl, lang, showToast, refreshWorkspace]);

  const toggleLanguage = useCallback(() => {
    const nextLang: 'es' | 'en' = lang === 'en' ? 'es' : 'en';
    syncNativeLanguage(nextLang);
    if (data?.project) {
      onUpdate(
        api.updateSettings({
          ...data.project,
          language: nextLang,
        }),
        nextLang === 'en' ? 'Language switched to English' : 'Idioma cambiado a Español'
      );
    } else {
      showToast(nextLang === 'en' ? 'Language switched to English' : 'Idioma cambiado a Español');
    }
  }, [lang, data?.project, onUpdate, syncNativeLanguage, showToast]);

  const handleCloseTab = useCallback(
    async (id: string) => {
      const res = await onUpdate(api.closeTab(id));
      if (!res?.project) {
        setShowHome(true);
      }
    },
    [onUpdate]
  );

  // Full cross-platform keyboard shortcuts
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (ctxMenu) setCtxMenu(null);
        else if (modal) setModal(null);
        return;
      }
      const mod = e.metaKey || e.ctrlKey;
      if (!mod) return;
      const k = e.key.toLowerCase();
      const tag = (e.target as HTMLElement)?.tagName?.toLowerCase();
      const inEditable = tag === 'input' || tag === 'textarea' || tag === 'select';

      if (k === 'z' && !inEditable) {
        e.preventDefault();
        if (e.shiftKey) onUpdate(api.redo());
        else onUpdate(api.undo());
      } else if (k === 'y' && !inEditable) {
        e.preventDefault();
        onUpdate(api.redo());
      } else if (k === 'n') {
        e.preventDefault();
        setModal('wizard');
      } else if (k === 'o' && e.shiftKey) {
        e.preventDefault();
        if (data?.project) setModal('optimize');
      } else if (k === 'o') {
        e.preventDefault();
        fileInputRef.current?.click();
      } else if (k === 's') {
        e.preventDefault();
        if (data?.project) handleQuickExportOmimo();
      } else if (k === 'p') {
        e.preventDefault();
        if (data?.project) setModal('reports');
      } else if (k === 'i') {
        e.preventDefault();
        if (data?.project) setModal('stats');
      } else if (k === 'g') {
        e.preventDefault();
        if (data?.project) {
          setShowHome(false);
          setView('guides');
        }
      } else if (k === ',') {
        e.preventDefault();
        if (data?.project) setModal('options');
      } else if (k === 'h') {
        e.preventDefault();
        setShowHome(true);
        refreshWorkspace();
      } else if (k === 'w' && data?.project && !showHome) {
        e.preventDefault();
        handleCloseTab(data.project.id);
      } else if (k === '1') {
        e.preventDefault();
        setShowHome(false);
        setView('resources');
      } else if (k === '2') {
        e.preventDefault();
        setShowHome(false);
        setView('courses');
      } else if (k === '3') {
        e.preventDefault();
        setShowHome(false);
        setView('weeks');
      } else if (k === '4') {
        e.preventDefault();
        setShowHome(false);
        setView('timetables');
      } else if (k === '5') {
        e.preventDefault();
        setShowHome(false);
        setView('guides');
      } else if (k === '6') {
        e.preventDefault();
        setShowHome(false);
        setView('scripts');
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [modal, ctxMenu, data?.project, showHome, onUpdate, handleCloseTab, refreshWorkspace, handleQuickExportOmimo]);

  // Native OS menu bar bridge (macOS system menu & Linux KDE Global Menu)
  useEffect(() => {
    window.__openmimoMenuAction = (id: string) => {
      switch (id) {
        case 'app:check_updates':
          runUpdateCheck(true);
          break;
        case 'file:home':
          setShowHome(true);
          refreshWorkspace();
          break;
        case 'file:new':
          setModal('wizard');
          break;
        case 'file:open':
          fileInputRef.current?.click();
          break;
        case 'file:export':
        case 'reports:open':
          if (data?.project) setModal('reports');
          break;
        case 'file:close_tab':
          if (data?.project) handleCloseTab(data.project.id);
          break;
        case 'sample:design_school':
        case 'sample:tiny_tutorial': {
          const key = id.replace('sample:', '');
          onUpdate(
            api.openSample(key, lang),
            lang === 'en' ? 'Sample opened' : 'Ejemplo abierto'
          ).then((res) => {
            if (res?.project) {
              setShowHome(false);
              setActiveWeek(1);
            }
          });
          break;
        }
        case 'view:reload':
          window.location.reload();
          break;
        case 'edit:undo':
          onUpdate(api.undo());
          break;
        case 'edit:redo':
          onUpdate(api.redo());
          break;
        case 'edit:auto_alloc':
          if (data?.project) {
            onUpdate(
              api.courseAction({ action: 'auto_allocate', maxWeek: data.project.time.maxWeeks }),
              lang === 'en' ? 'Allocated remaining periods' : 'Periodos restantes asignados'
            );
          }
          break;
        case 'view:resources':
          setShowHome(false);
          setView('resources');
          break;
        case 'view:courses':
          setShowHome(false);
          setView('courses');
          break;
        case 'view:weeks':
          setShowHome(false);
          setView('weeks');
          break;
        case 'view:timetables':
          setShowHome(false);
          setView('timetables');
          break;
        case 'view:guides':
        case 'tools:guides':
          if (data?.project) {
            setShowHome(false);
            setView('guides');
          }
          break;
        case 'view:scripts':
          if (data?.project) {
            setShowHome(false);
            setView('scripts');
          }
          break;
        case 'tools:optimize':
          if (data?.project) setModal('optimize');
          break;
        case 'tools:rooms':
          if (data?.project) setModal('rooms');
          break;
        case 'tools:collab':
          if (data?.project) setModal('collab');
          break;
        case 'tools:stats':
          if (data?.project) setModal('stats');
          break;
        case 'tools:mcp':
          setModal('mcp');
          break;
        case 'options:settings':
          if (data?.project) setModal('options');
          break;
        case 'options:lang':
          toggleLanguage();
          break;
      }
    };
    return () => {
      delete window.__openmimoMenuAction;
    };
  }, [data?.project, lang, onUpdate, toggleLanguage, handleCloseTab, refreshWorkspace, runUpdateCheck]);

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = '';
    const ext = file.name.split('.').pop()?.toLowerCase() || '';
    if (ext === 'mfw') {
      const buf = await file.arrayBuffer();
      const bytes = new Uint8Array(buf);
      let bin = '';
      for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
      const res = await onUpdate(api.importFile(file.name, 'mfw', btoa(bin)), `${file.name}`);
      if (res) {
        setShowHome(false);
        setActiveWeek(1);
      }
    } else {
      const txt = await file.text();
      const res = await onUpdate(api.importFile(file.name, ext || 'omimo', txt), `${file.name}`);
      if (res) {
        setShowHome(false);
        setActiveWeek(1);
      }
    }
  };

  // Custom native-feeling right-click context menu (replaces browser "Reload")
  const handleGlobalContextMenu = (e: React.MouseEvent) => {
    const target = e.target as HTMLElement | null;
    const tag = target?.tagName?.toLowerCase() || '';
    if (tag === 'input' || tag === 'textarea' || target?.isContentEditable) {
      return; // allow native copy/paste inside text editors
    }
    e.preventDefault();
    const isEn = lang === 'en';
    const proj = data?.project;

    const ctxEl = target?.closest('[data-ctx]') as HTMLElement | null;
    const ctxType = ctxEl?.getAttribute('data-ctx');
    const ctxId = ctxEl?.getAttribute('data-ctx-id') || '';

    const items: ContextMenuItem[] = [];

    if (ctxType === 'guide' && ctxId && proj) {
      items.push({
        label: isEn ? 'Delete teaching guide' : 'Eliminar guía docente',
        danger: true,
        onClick: () => onUpdate(api.deleteGuide(ctxId), isEn ? 'Guide deleted' : 'Guía eliminada'),
      });
      items.push({ divider: true, label: '' });
    } else if (ctxType === 'script' && ctxId && proj) {
      items.push({
        label: isEn ? 'Delete script' : 'Eliminar script',
        danger: true,
        onClick: () => onUpdate(api.deleteScript(ctxId), isEn ? 'Script deleted' : 'Script eliminado'),
      });
      items.push({ divider: true, label: '' });
    }

    if (proj && !showHome) {
      items.push(
        {
          label: isEn ? 'Undo' : 'Deshacer',
          shortcut: '⌘Z',
          disabled: !data?.canUndo,
          onClick: () => onUpdate(api.undo()),
        },
        {
          label: isEn ? 'Redo' : 'Rehacer',
          shortcut: '⇧⌘Z',
          disabled: !data?.canRedo,
          onClick: () => onUpdate(api.redo()),
        },
        { divider: true, label: '' },
        {
          label: isEn ? 'Optimise timetables...' : 'Optimizar horarios...',
          shortcut: '⇧⌘O',
          onClick: () => setModal('optimize'),
        },
        {
          label: isEn ? 'Assign rooms...' : 'Asignar salas...',
          onClick: () => setModal('rooms'),
        },
        {
          label: isEn ? 'Open Teaching Guides' : 'Abrir Guías Docentes',
          shortcut: '⌘5',
          onClick: () => setView('guides'),
        },
        {
          label: isEn ? 'Open Script Library' : 'Abrir Librería de Scripts',
          shortcut: '⌘6',
          onClick: () => setView('scripts'),
        },
        { divider: true, label: '' },
        {
          label: isEn ? 'Save as .omimo' : 'Guardar como .omimo',
          shortcut: '⌘S',
          onClick: handleQuickExportOmimo,
        },
        {
          label: isEn ? 'Share live (IPFS)...' : 'Compartir en vivo (IPFS)...',
          onClick: () => setModal('collab'),
        },
        {
          label: isEn ? 'Publish Web / Export...' : 'Publicar Web / Exportar...',
          onClick: () => setModal('reports'),
        },
        { divider: true, label: '' }
      );
    }

    items.push(
      {
        label: isEn ? 'New schedule...' : 'Nuevo horario...',
        shortcut: '⌘N',
        onClick: () => setModal('wizard'),
      },
      {
        label: isEn ? 'Open / Import (.omimo, .mfw)...' : 'Abrir / Importar (.omimo, .mfw)...',
        shortcut: '⌘O',
        onClick: () => fileInputRef.current?.click(),
      },
      {
        label: isEn ? 'Go to Home' : 'Ir a Inicio',
        shortcut: '⌘H',
        onClick: () => {
          setShowHome(true);
          refreshWorkspace();
        },
      },
      {
        label: isEn ? 'Check for updates...' : 'Buscar actualizaciones...',
        onClick: () => runUpdateCheck(true),
      }
    );

    setCtxMenu({ x: e.clientX, y: e.clientY, items });
  };

  if (!data && !workspace) {
    return (
      <div
        style={{
          height: '100vh',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexDirection: 'column',
          gap: 10,
          background: 'var(--bg-app)',
        }}
      >
        <AppLogo size={48} />
        <div style={{ fontWeight: 700, fontSize: 15 }}>
          {loadError ? (lang === 'en' ? 'Connection error' : 'Error de conexión') : 'OpenMimo'}
        </div>
        {loadError && (
          <>
            <div style={{ color: '#dc2626', fontSize: 12.5 }}>{loadError}</div>
            <button className="btn btn-primary" onClick={() => loadInitial()}>
              {lang === 'en' ? 'Retry connection' : 'Reintentar'}
            </button>
          </>
        )}
      </div>
    );
  }

  const project = data?.project ?? null;
  const stats = data?.stats;
  const tabs = data?.tabs ?? workspace?.tabs ?? [];
  const t = I18N[lang];
  const isEn = lang === 'en';
  const closeMenus = () => {
    setOpenMenu(null);
    setCtxMenu(null);
  };

  return (
    <div className="app-shell" onClick={closeMenus} onContextMenu={handleGlobalContextMenu}>
      <input
        ref={fileInputRef}
        type="file"
        accept=".omimo,.mfw,.mxt,.json"
        style={{ display: 'none' }}
        onChange={handleImportFile}
      />

      {/* In-window Menu Bar for Windows, Android Tablets & Web (macOS & Linux use native OS / KDE Global Menu) */}
      {!hasNativeMenu && (
        <div className="menubar" onClick={(e) => e.stopPropagation()}>
          <div className="menubar-items">
            <div className="menu-wrap">
              <button className={`menu-trigger ${openMenu === 'file' ? 'open' : ''}`} onClick={() => setOpenMenu(openMenu === 'file' ? null : 'file')}>
                {isEn ? 'File' : 'Archivo'}
              </button>
              {openMenu === 'file' && (
                <div className="menu-dropdown">
                  <button className="menu-item" onClick={() => { closeMenus(); setShowHome(true); refreshWorkspace(); }}>
                    <span>{isEn ? 'Home & Recent' : 'Inicio y recientes'}</span>
                  </button>
                  <button className="menu-item" onClick={() => { closeMenus(); setModal('wizard'); }}>
                    <span>{isEn ? 'New schedule...' : 'Nuevo horario...'}</span><span style={{ opacity: 0.6 }}>Ctrl+N</span>
                  </button>
                  <button className="menu-item" onClick={() => { closeMenus(); fileInputRef.current?.click(); }}>
                    <span>{isEn ? 'Open / Import (.omimo, .mfw)...' : 'Abrir / Importar (.omimo, .mfw)...'}</span><span style={{ opacity: 0.6 }}>Ctrl+O</span>
                  </button>
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); handleQuickExportOmimo(); }}>
                    <span>{isEn ? 'Save as .omimo' : 'Guardar como .omimo'}</span><span style={{ opacity: 0.6 }}>Ctrl+S</span>
                  </button>
                  <div className="menu-sep" />
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); setModal('reports'); }}>
                    <span>{isEn ? 'Publish Web / Export...' : 'Publicar Web / Exportar...'}</span>
                  </button>
                </div>
              )}
            </div>
            <div className="menu-wrap">
              <button className={`menu-trigger ${openMenu === 'edit' ? 'open' : ''}`} onClick={() => setOpenMenu(openMenu === 'edit' ? null : 'edit')}>
                {isEn ? 'Edit' : 'Editar'}
              </button>
              {openMenu === 'edit' && (
                <div className="menu-dropdown">
                  <button className="menu-item" disabled={!data?.canUndo} onClick={() => { closeMenus(); onUpdate(api.undo()); }}>
                    <span>{t.toolbar.undo}</span><span style={{ opacity: 0.6 }}>Ctrl+Z</span>
                  </button>
                  <button className="menu-item" disabled={!data?.canRedo} onClick={() => { closeMenus(); onUpdate(api.redo()); }}>
                    <span>{t.toolbar.redo}</span><span style={{ opacity: 0.6 }}>Ctrl+Shift+Z</span>
                  </button>
                </div>
              )}
            </div>
            <div className="menu-wrap">
              <button className={`menu-trigger ${openMenu === 'tools' ? 'open' : ''}`} onClick={() => setOpenMenu(openMenu === 'tools' ? null : 'tools')}>
                {isEn ? 'Tools' : 'Herramientas'}
              </button>
              {openMenu === 'tools' && (
                <div className="menu-dropdown">
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); setModal('optimize'); }}>
                    <span>{isEn ? 'Optimise timetables...' : 'Optimizar horarios...'}</span>
                  </button>
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); setModal('rooms'); }}>
                    <span>{isEn ? 'Select rooms manually...' : 'Asignar salas...'}</span>
                  </button>
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); setShowHome(false); setView('guides'); }}>
                    <span>{isEn ? 'Teaching Guides & Astro' : 'Guías Docentes y Astro'}</span>
                  </button>
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); setShowHome(false); setView('scripts'); }}>
                    <span>{isEn ? 'Script Library' : 'Librería de Scripts'}</span>
                  </button>
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); setModal('collab'); }}>
                    <span>{isEn ? 'Real-Time & IPFS Share...' : 'Compartir IPFS / Tiempo real...'}</span>
                  </button>
                  <div className="menu-sep" />
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); setModal('stats'); }}>
                    <span>{isEn ? 'Statistics & Health Check' : 'Estadísticas y diagnóstico'}</span>
                  </button>
                  <button className="menu-item" onClick={() => { closeMenus(); setModal('mcp'); }}>
                    <span>{isEn ? 'MCP Server Config...' : 'Configurar MCP...'}</span>
                  </button>
                </div>
              )}
            </div>
            <div className="menu-wrap">
              <button className={`menu-trigger ${openMenu === 'settings' ? 'open' : ''}`} onClick={() => setOpenMenu(openMenu === 'settings' ? null : 'settings')}>
                {isEn ? 'Settings' : 'Configuración'}
              </button>
              {openMenu === 'settings' && (
                <div className="menu-dropdown">
                  <button className="menu-item" disabled={!project} onClick={() => { closeMenus(); setModal('options'); }}>
                    <span>{isEn ? 'Limits, Bells & Categories...' : 'Límites, bloques y categorías...'}</span>
                  </button>
                  <button className="menu-item" onClick={() => { closeMenus(); toggleLanguage(); }}>
                    <span>{isEn ? 'Idioma: Español (ES)' : 'Language: English (EN)'}</span>
                  </button>
                  <div className="menu-sep" />
                  <button className="menu-item" onClick={() => { closeMenus(); runUpdateCheck(true); }}>
                    <span>{isEn ? 'Check for updates...' : 'Buscar actualizaciones...'}</span>
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Affinity-Inspired Clean Top Toolbar (Logo only, no app name text, solid-color active studio pills) */}
      <header
        className="topbar"
        data-csd={isMacCSD ? 'true' : undefined}
        data-tauri-drag-region={isMacCSD ? '' : undefined}
      >
        <button
          className="brand-logo-btn"
          title="OpenMimo — Inicio"
          onClick={() => {
            setShowHome(true);
            refreshWorkspace();
          }}
        >
          <AppLogo size={26} />
        </button>

        {/* 6 Primary Studio Views — Left-aligned right after logo, Affinity solid-color active pills */}
        {project && !showHome && (
          <nav className="view-switcher" aria-label="Main studios">
            <button
              data-studio="resources"
              className={`view-tab ${view === 'resources' ? 'active' : ''}`}
              onClick={() => setView('resources')}
            >
              <Users size={13} /> <span>{t.views.resources}</span>
            </button>
            <button
              data-studio="courses"
              className={`view-tab ${view === 'courses' ? 'active' : ''}`}
              onClick={() => setView('courses')}
            >
              <BookOpen size={13} /> <span>{t.views.courses}</span>
            </button>
            <button
              data-studio="weeks"
              className={`view-tab ${view === 'weeks' ? 'active' : ''}`}
              onClick={() => setView('weeks')}
            >
              <CalendarRange size={13} /> <span>{t.views.weeks}</span>
            </button>
            <button
              data-studio="timetables"
              className={`view-tab ${view === 'timetables' ? 'active' : ''}`}
              onClick={() => setView('timetables')}
            >
              <Grid3X3 size={13} /> <span>{t.views.timetables}</span>
            </button>
            <button
              data-studio="guides"
              className={`view-tab ${view === 'guides' ? 'active' : ''}`}
              onClick={() => setView('guides')}
            >
              <BookOpenCheck size={13} /> <span>{t.views.guides}</span>
            </button>
            <button
              data-studio="scripts"
              className={`view-tab ${view === 'scripts' ? 'active' : ''}`}
              onClick={() => setView('scripts')}
            >
              <Code2 size={13} /> <span>{t.views.scripts}</span>
            </button>
          </nav>
        )}

        <div className="topbar-spacer" data-tauri-drag-region={isMacCSD ? '' : undefined} />

        {/* Compact Responsive Action Toolbar */}
        <div className="toolbar-actions">
          {project && !showHome && (
            <>
              <button className="btn btn-icon" title={`${t.toolbar.undo} (⌘Z)`} disabled={!data?.canUndo} onClick={() => onUpdate(api.undo())}>
                <Undo2 size={14} />
              </button>
              <button className="btn btn-icon" title={`${t.toolbar.redo} (⇧⌘Z)`} disabled={!data?.canRedo} onClick={() => onUpdate(api.redo())}>
                <Redo2 size={14} />
              </button>
              <div className="divider-v" />
              <button className="btn btn-primary" title={t.toolbar.optimize} onClick={() => setModal('optimize')}>
                <Sparkles size={13} /> <span className="btn-label">{t.toolbar.optimize}</span>
              </button>
              <button className="btn" title={t.toolbar.rooms} onClick={() => setModal('rooms')}>
                <DoorOpen size={13} /> <span className="btn-label">{t.toolbar.rooms}</span>
              </button>
              <button
                className={`btn ${project.collab?.shared ? 'btn-ok' : ''}`}
                title={isEn ? 'Share (IPFS)' : 'Compartir (IPFS)'}
                onClick={() => setModal('collab')}
              >
                {project.collab?.shared ? <Radio size={13} /> : <Share2 size={13} />}
                <span className="btn-label">{isEn ? 'Share' : 'Compartir'}</span>
              </button>
              <button className="btn" title={t.toolbar.reports} onClick={() => setModal('reports')}>
                <Download size={13} /> <span className="btn-label">{t.toolbar.reports}</span>
              </button>
              <div className="divider-v" />
            </>
          )}

          {updateInfo?.hasUpdate && (
            <a
              className="update-pill"
              href={updateInfo.releaseUrl}
              target="_blank"
              rel="noreferrer"
              title={`OpenMimo v${updateInfo.latestVersion}`}
            >
              <ArrowUpCircle size={13} />
              <span className="btn-label">v{updateInfo.latestVersion}</span>
            </a>
          )}

          {/* Compact MCP Status Indicator */}
          <button
            className="mcp-pill"
            onClick={() => setModal('mcp')}
            title={
              mcpEnabled
                ? isEn
                  ? 'MCP active — click to configure'
                  : 'MCP activo — clic para configurar'
                : isEn
                ? 'MCP inactive — click to configure'
                : 'MCP desactivado — clic para configurar'
            }
          >
            <span className={`mcp-dot ${mcpEnabled ? 'on' : 'off'}`} />
            MCP
          </button>
        </div>
      </header>

      {/* Runtime Debug Banner (shown whenever an unhandled error occurs) */}
      {debugError && (
        <div className="debug-error-bar">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 7, fontWeight: 700, fontSize: 12 }}>
              <Bug size={14} /> Debug Error: {debugError.msg}
            </div>
            <button className="btn btn-sm" onClick={() => setDebugError(null)}>
              <X size={12} />
            </button>
          </div>
          {debugError.stack && <pre className="debug-error-stack">{debugError.stack}</pre>}
        </div>
      )}

      {/* Multi-Project Tabs Bar */}
      <div className="tabstrip">
        <button
          className={`proj-tab ${showHome ? 'active' : ''}`}
          onClick={() => {
            setShowHome(true);
            refreshWorkspace();
          }}
        >
          <Home size={13} /> {isEn ? 'Home' : 'Inicio'}
        </button>

        {tabs.map((tab) => {
          const isActive = !showHome && project?.id === tab.id;
          const isRenaming = isActive && renamingTabId === tab.id;
          const commitRename = async () => {
            const nextName = renameDraft.trim();
            setRenamingTabId(null);
            if (!project || !nextName || nextName === project.name) return;
            await onUpdate(
              api.updateSettings({
                name: nextName,
                institution: project.institution,
                department: project.department,
                comment: project.comment,
                language: project.language,
                roomsCategory: project.roomsCategory,
                time: project.time,
                courseCategories: project.courseCategories,
                resourceCategories: project.resourceCategories,
                bookingTypes: project.bookingTypes,
              }),
              isEn ? 'Schedule renamed' : 'Nombre del horario actualizado'
            );
          };
          return (
            <div
              key={tab.id}
              className={`proj-tab ${isActive ? 'active' : ''}`}
              title={
                isActive
                  ? isEn
                    ? 'Double-click to rename schedule'
                    : 'Doble clic para cambiar el nombre del horario'
                  : tab.name
              }
              onClick={async () => {
                if (isRenaming) return;
                if (!isActive) {
                  await onUpdate(api.switchTab(tab.id));
                  setShowHome(false);
                  setActiveWeek(1);
                }
              }}
              onDoubleClick={(e) => {
                if (!isActive || !project) return;
                e.stopPropagation();
                setRenameDraft(project.name);
                setRenamingTabId(tab.id);
              }}
            >
              {tab.isRemote && <Radio size={11} color={tab.hostOpen ? '#16a34a' : '#dc2626'} />}
              {isRenaming ? (
                <input
                  className="proj-tab-input"
                  autoFocus
                  value={renameDraft}
                  onFocus={(e) => e.currentTarget.select()}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e) => setRenameDraft(e.target.value)}
                  onBlur={commitRename}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      commitRename();
                    } else if (e.key === 'Escape') {
                      e.preventDefault();
                      setRenamingTabId(null);
                    }
                  }}
                />
              ) : (
                <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{tab.name}</span>
              )}
              <button
                className="proj-tab-close"
                title={isEn ? 'Close tab' : 'Cerrar pestaña'}
                onClick={(e) => {
                  e.stopPropagation();
                  handleCloseTab(tab.id);
                }}
              >
                <X size={11} />
              </button>
            </div>
          );
        })}

        <button
          className="proj-tab"
          style={{ padding: '6px 8px' }}
          title={isEn ? 'New schedule (Setup Wizard)' : 'Nuevo horario (Asistente)'}
          onClick={() => setModal('wizard')}
        >
          <Plus size={13} />
        </button>
      </div>

      {/* Main Content: Home Screen or Active Project Studio */}
      {showHome || !project || !stats ? (
        <HomeView
          workspace={workspace}
          lang={lang}
          onOpenLocal={async (id) => {
            const res = await onUpdate(api.openProject(id));
            if (res?.project) {
              setShowHome(false);
              setActiveWeek(1);
            }
          }}
          onDeleteLocal={(id) => onUpdate(api.deleteProject(id), isEn ? 'Project deleted' : 'Proyecto eliminado')}
          onExportAndDeleteLocal={async (id) => {
            try {
              const exp = await api.exportProjectById(id);
              const blob = new Blob([exp.content], { type: 'application/json;charset=utf-8' });
              const url = URL.createObjectURL(blob);
              const a = document.createElement('a');
              a.href = url;
              a.download = exp.fileName;
              a.click();
              URL.revokeObjectURL(url);
              await onUpdate(
                api.deleteProject(id),
                isEn ? 'Exported and deleted' : 'Exportado y borrado'
              );
            } catch (e: any) {
              showToast(e?.message || 'Error', true);
            }
          }}
          onOpenSample={async (key) => {
            const res = await onUpdate(
              api.openSample(key, lang),
              isEn ? 'Sample schedule opened' : 'Horario de prueba abierto'
            );
            if (res?.project) {
              setShowHome(false);
              setActiveWeek(1);
            }
          }}
          onOpenWizard={() => setModal('wizard')}
          onImportFile={() => fileInputRef.current?.click()}
          onJoinShareCode={async (code) => {
            const res = await onUpdate(api.joinCollab({ shareCode: code }), isEn ? 'Joined shared schedule' : 'Conectado al horario compartido');
            if (res?.project) setShowHome(false);
          }}
          onJoinRemoteEntry={async (hostUrl, projectId, shareCode) => {
            const res = await onUpdate(api.joinCollab({ hostUrl, projectId, shareCode }));
            if (res?.project) setShowHome(false);
          }}
          onRefresh={refreshWorkspace}
        />
      ) : (
        <>
          <main className="main-workspace">
            {view === 'timetables' && (
              <TimetablesView
                project={{ ...project, language: lang }}
                stats={stats}
                activeWeek={activeWeek}
                setActiveWeek={setActiveWeek}
                onUpdate={onUpdate}
                onError={(m) => showToast(m, true)}
              />
            )}
            {view === 'weeks' && (
              <WeeksView
                project={{ ...project, language: lang }}
                activeWeek={activeWeek}
                setActiveWeek={setActiveWeek}
                onUpdate={onUpdate}
              />
            )}
            {view === 'courses' && (
              <CoursesView
                project={{ ...project, language: lang }}
                onUpdate={onUpdate}
                onOpenGuide={(guideId, linkCourseId) => {
                  setGuideCtx({ guideId, linkCourseId });
                  setView('guides');
                }}
              />
            )}
            {view === 'resources' && (
              <ResourcesView
                project={{ ...project, language: lang }}
                activeWeek={activeWeek}
                onUpdate={onUpdate}
              />
            )}
            {view === 'guides' && (
              <GuidesView
                project={{ ...project, language: lang }}
                initialGuideId={guideCtx.guideId}
                linkCourseId={guideCtx.linkCourseId}
                onUpdate={onUpdate}
                onOpenCollab={() => setModal('collab')}
              />
            )}
            {view === 'scripts' && (
              <ScriptsView
                project={{ ...project, language: lang }}
                onUpdate={onUpdate}
                onOpenAISettings={() => {
                  setOptionsTab('ai');
                  setModal('options');
                }}
              />
            )}
          </main>

          {/* Clean Bottom Status Bar */}
          <footer className="statusbar">
            <div className="status-pills">
              <span className="status-item">
                <strong>{stats.totalCodes}</strong> {t.status.codes} ({stats.totalCourses}{' '}
                {t.views.courses.toLowerCase()} · {stats.totalResources}{' '}
                {t.views.resources.toLowerCase()})
              </span>
              <span className="status-item">
                {stats.conflictCount === 0 ? (
                  <>
                    <CheckCircle2 size={13} color="#16a34a" />
                    <span style={{ color: '#15803d', fontWeight: 600 }}>0 {t.status.conflicts}</span>
                  </>
                ) : (
                  <>
                    <AlertTriangle size={13} color="#dc2626" />
                    <span
                      style={{ color: '#dc2626', fontWeight: 700, cursor: 'pointer', textDecoration: 'underline' }}
                      onClick={() => setModal('stats')}
                    >
                      {stats.conflictCount} {t.status.conflicts}
                    </span>
                  </>
                )}
              </span>
              <span className="status-item">
                {t.status.gaps}: <strong>{stats.totalGaps}</strong>
              </span>
              {project.collab?.shared && (
                <span className="status-item" style={{ color: '#15803d', fontWeight: 600 }}>
                  <Radio size={12} /> IPFS
                </span>
              )}
            </div>
            <div className="status-pills">
              <span>
                {t.timetables.week} <strong>{activeWeek}/{project.time.maxWeeks}</strong> ·{' '}
                {project.time.maxDays}d × {project.time.maxPeriods}b
              </span>
              <span>
                {t.status.scheduled}: <strong>{stats.totalScheduled}/{stats.totalAllocated}</strong> (
                {Math.round(stats.overallCompletion)}%)
              </span>
            </div>
          </footer>
        </>
      )}

      {project && stats && (
        <Modals
          modal={modal}
          onClose={() => {
            setModal(null);
            setOptionsTab('limits');
          }}
          project={{ ...project, language: lang }}
          stats={stats}
          activeWeek={activeWeek}
          initialOptionsTab={optionsTab}
          onUpdate={onUpdate}
        />
      )}

      <FeatureModals
        modal={modal}
        onClose={() => setModal(null)}
        project={project ? { ...project, language: lang } : null}
        lang={lang}
        mcpEnabled={mcpEnabled}
        mcpHttpUrl={mcpHttpUrl}
        onUpdate={onUpdate}
        onWizardDone={(payload) => {
          setData(payload);
          setShowHome(false);
          setActiveWeek(1);
          refreshWorkspace();
          showToast(isEn ? 'New schedule created' : 'Nuevo horario creado');
        }}
        onMcpToggle={async (en) => {
          const res = await api.setMcpEnabled(en);
          setMcpEnabled(res.enabled);
          setMcpHttpUrl(res.httpUrl);
          showToast(
            res.enabled
              ? isEn
                ? 'MCP server enabled'
                : 'Servidor MCP activado'
              : isEn
              ? 'MCP server disabled'
              : 'Servidor MCP desactivado'
          );
        }}
      />

      {ctxMenu && (
        <ContextMenu
          x={ctxMenu.x}
          y={ctxMenu.y}
          items={ctxMenu.items}
          onClose={() => setCtxMenu(null)}
        />
      )}

      {toast && <div className={`toast ${toast.error ? 'error' : ''}`}>{toast.msg}</div>}
    </div>
  );
};
