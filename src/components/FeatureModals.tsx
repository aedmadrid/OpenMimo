import React, { useEffect, useState } from 'react';
import {
  X,
  Share2,
  Copy,
  Check,
  Radio,
  Cpu,
  Wand2,
} from 'lucide-react';
import type {
  ModalKind,
  Project,
  WizardConfig,
} from '../types';
import { api } from '../api';

interface Props {
  modal: ModalKind;
  onClose: () => void;
  project: Project | null;
  lang: 'es' | 'en';
  mcpEnabled: boolean;
  mcpHttpUrl: string;
  onUpdate: (promise: Promise<any>, toastMsg?: string) => Promise<any>;
  onWizardDone: (payload: any) => void;
  onMcpToggle: (enabled: boolean) => Promise<void>;
}

export const FeatureModals: React.FC<Props> = ({
  modal,
  onClose,
  project,
  lang,
  mcpEnabled,
  mcpHttpUrl,
  onUpdate,
  onWizardDone,
  onMcpToggle,
}) => {
  if (modal !== 'wizard' && modal !== 'collab' && modal !== 'mcp') {
    return null;
  }
  const isEn = lang === 'en';

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal-card"
        onClick={(e) => e.stopPropagation()}
      >
        {modal === 'wizard' && (
          <WizardModal lang={lang} isEn={isEn} onClose={onClose} onCreated={onWizardDone} />
        )}
        {modal === 'collab' && project && (
          <CollabModal project={project} isEn={isEn} onUpdate={onUpdate} onClose={onClose} />
        )}
        {modal === 'mcp' && (
          <McpModal
            isEn={isEn}
            enabled={mcpEnabled}
            httpUrl={mcpHttpUrl}
            onToggle={onMcpToggle}
            onClose={onClose}
          />
        )}
      </div>
    </div>
  );
};

/* ---------------- 1. NEW SCHEDULE SETUP WIZARD ---------------- */
const WizardModal: React.FC<{
  lang: 'es' | 'en';
  isEn: boolean;
  onClose: () => void;
  onCreated: (payload: any) => void;
}> = ({ lang, isEn, onClose, onCreated }) => {
  const [cfg, setCfg] = useState<WizardConfig>({
    name: isEn ? 'Academic Year 2026-27' : 'Curso 2026-27 — 1er Semestre',
    institution: isEn ? 'Northridge Arts & Sciences Institute' : 'Instituto Norte de Ciencias y Artes',
    department: isEn ? 'Academic Coordination' : 'Jefatura de Estudios',
    language: lang,
    maxWeeks: 4,
    maxDays: 5,
    maxPeriods: 10,
    startTime: '08:30',
    slotMinutes: 60,
    breakMins: 0,
    template: 'university',
  });
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      const payload = await api.createProject(cfg);
      onCreated(payload);
      onClose();
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="modal-header">
        <h3>
          <Wand2 size={17} color="#2563eb" />
          {isEn ? 'New Schedule — Quick Setup' : 'Nuevo Horario — Configuración Inicial'}
        </h3>
        <button className="btn btn-icon" onClick={onClose}>
          <X size={15} />
        </button>
      </div>
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
        <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div className="form-grid">
            <div className="field">
              <label>{isEn ? 'Schedule name' : 'Nombre del horario'}</label>
              <input
                className="input"
                required
                value={cfg.name}
                onChange={(e) => setCfg({ ...cfg, name: e.target.value })}
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Institution / Center' : 'Centro / Universidad'}</label>
              <input
                className="input"
                value={cfg.institution}
                onChange={(e) => setCfg({ ...cfg, institution: e.target.value })}
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Department / Office' : 'Departamento / Jefatura'}</label>
              <input
                className="input"
                value={cfg.department}
                onChange={(e) => setCfg({ ...cfg, department: e.target.value })}
              />
            </div>
            <div className="field">
              <label>{isEn ? 'Starting template' : 'Plantilla inicial'}</label>
              <select
                className="select"
                value={cfg.template}
                onChange={(e) => setCfg({ ...cfg, template: e.target.value as WizardConfig['template'] })}
              >
                <option value="university">
                  {isEn
                    ? 'University starter (Morning/Afternoon parallel groups + guide)'
                    : 'Base universitaria (Grupos mañana/tarde paralelos + guía docente)'}
                </option>
                <option value="empty">{isEn ? 'Empty timetable' : 'Horario en blanco'}</option>
                <option value="demo">{isEn ? 'Full sample schedule' : 'Horario de ejemplo completo'}</option>
              </select>
            </div>
          </div>

          <div
            style={{
              background: '#f8fafc',
              border: '1px solid var(--border)',
              borderRadius: 8,
              padding: 12,
            }}
          >
            <strong style={{ fontSize: 12 }}>
              {isEn ? 'Basic Time Structure' : 'Estructura básica de tiempo'}
            </strong>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 10, marginTop: 10 }}>
              <div className="field">
                <label>{isEn ? 'Weeks / Terms (1–255)' : 'Semanas / Periodos (1–255)'}</label>
                <input
                  className="input"
                  type="number"
                  min={1}
                  max={255}
                  value={cfg.maxWeeks}
                  onChange={(e) => setCfg({ ...cfg, maxWeeks: Number(e.target.value) })}
                />
              </div>
              <div className="field">
                <label>{isEn ? 'Days per week (1–7)' : 'Días por semana (1–7)'}</label>
                <input
                  className="input"
                  type="number"
                  min={1}
                  max={7}
                  value={cfg.maxDays}
                  onChange={(e) => setCfg({ ...cfg, maxDays: Number(e.target.value) })}
                />
              </div>
              <div className="field">
                <label>{isEn ? 'Periods per day (1–30)' : 'Bloques por día (1–30)'}</label>
                <input
                  className="input"
                  type="number"
                  min={1}
                  max={30}
                  value={cfg.maxPeriods}
                  onChange={(e) => setCfg({ ...cfg, maxPeriods: Number(e.target.value) })}
                />
              </div>
              <div className="field">
                <label>{isEn ? 'Start time (default 08:30)' : 'Hora de inicio (ej. 08:30)'}</label>
                <input
                  className="input"
                  type="time"
                  value={cfg.startTime}
                  onChange={(e) => setCfg({ ...cfg, startTime: e.target.value })}
                />
              </div>
              <div className="field">
                <label>{isEn ? 'Period length (min)' : 'Duración de bloque (min)'}</label>
                <input
                  className="input"
                  type="number"
                  min={15}
                  max={180}
                  value={cfg.slotMinutes}
                  onChange={(e) => setCfg({ ...cfg, slotMinutes: Number(e.target.value) })}
                />
              </div>
              <div className="field">
                <label>{isEn ? 'Break between periods (min)' : 'Descanso entre bloques (min)'}</label>
                <input
                  className="input"
                  type="number"
                  min={0}
                  max={60}
                  value={cfg.breakMins}
                  onChange={(e) => setCfg({ ...cfg, breakMins: Number(e.target.value) })}
                />
              </div>
            </div>
          </div>
        </div>
        <div className="modal-footer">
          <button type="button" className="btn" onClick={onClose}>
            {isEn ? 'Cancel' : 'Cancelar'}
          </button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {isEn ? 'Create schedule' : 'Crear horario'}
          </button>
        </div>
      </form>
    </>
  );
};

/* ---------------- 3. REAL-TIME & IPFS COLLABORATION MODAL ---------------- */
const CollabModal: React.FC<{
  project: Project;
  isEn: boolean;
  onUpdate: (p: Promise<any>, msg?: string) => Promise<any>;
  onClose: () => void;
}> = ({ project, isEn, onUpdate, onClose }) => {
  const [joinCode, setJoinCode] = useState('');
  const [copied, setCopied] = useState(false);

  const copyCode = (txt: string) => {
    navigator.clipboard?.writeText(txt);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  return (
    <>
      <div className="modal-header">
        <h3>
          <Share2 size={17} color="#2563eb" />
          {isEn ? 'Real-Time & IPFS Sharing' : 'Compartir en Tiempo Real e IPFS'}
        </h3>
        <button className="btn btn-icon" onClick={onClose}>
          <X size={15} />
        </button>
      </div>
      <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <div
          style={{
            padding: 14,
            borderRadius: 8,
            background: project.collab?.shared ? '#f0fdf4' : '#f8fafc',
            border: `1px solid ${project.collab?.shared ? '#86efac' : 'var(--border)'}`,
            display: 'flex',
            flexDirection: 'column',
            gap: 10,
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Radio size={16} color={project.collab?.shared ? '#16a34a' : '#64748b'} />
              <strong>
                {project.collab?.shared
                  ? isEn
                    ? 'Live sharing active (IPFS + Real-time)'
                    : 'Compartición activa (IPFS + Tiempo real)'
                  : isEn
                  ? 'Share this schedule with a code'
                  : 'Compartir este horario con un código'}
              </strong>
            </div>
            {!project.collab?.isRemote && (
              <button
                className={`btn ${project.collab?.shared ? 'btn-danger' : 'btn-primary'}`}
                onClick={() =>
                  onUpdate(
                    api.shareCollab(Boolean(project.collab?.shared)),
                    project.collab?.shared
                      ? isEn
                        ? 'Stopped sharing'
                        : 'Compartición detenida'
                      : isEn
                      ? 'IPFS share code generated'
                      : 'Código IPFS generado'
                  )
                }
              >
                {project.collab?.shared
                  ? isEn
                    ? 'Stop sharing'
                    : 'Dejar de compartir'
                  : isEn
                  ? 'Enable IPFS sharing'
                  : 'Activar compartición IPFS'}
              </button>
            )}
          </div>

          {project.collab?.shared && project.collab.shareCode && (
            <>
              <div className="field">
                <label>{isEn ? 'Share code (send to collaborators):' : 'Código para compartir (envíalo a otros editores):'}</label>
                <div style={{ display: 'flex', gap: 6 }}>
                  <input className="input" readOnly style={{ flex: 1, fontFamily: 'monospace', fontSize: 11.5 }} value={project.collab.shareCode} />
                  <button className="btn btn-primary" onClick={() => copyCode(project.collab.shareCode || '')}>
                    {copied ? <Check size={13} /> : <Copy size={13} />} {copied ? 'OK' : isEn ? 'Copy' : 'Copiar'}
                  </button>
                </div>
              </div>
              <div style={{ fontSize: 11.5, color: 'var(--text-muted)', fontFamily: 'monospace' }}>
                {isEn ? 'Host LAN URL' : 'Dirección de red local'}: {project.collab.hostUrl}
              </div>
              <div style={{ fontSize: 11.5, color: 'var(--text-muted)', fontFamily: 'monospace' }}>
                IPFS CID ({project.collab.ipfsMode === 'kubo' ? 'Kubo node' : 'CIDv1'}): {project.collab.ipfsCid}
              </div>
              <div style={{ fontSize: 11.5, color: '#15803d' }}>
                {isEn
                  ? 'Collaborators can edit this schedule in real time only while this tab stays open on your computer.'
                  : 'Otros usuarios podrán editar en tiempo real solo mientras mantengas esta pestaña abierta en tu equipo.'}
              </div>
            </>
          )}
        </div>

        <form
          style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
          onSubmit={(e) => {
            e.preventDefault();
            if (!joinCode.trim()) return;
            onUpdate(
              api.joinCollab({ shareCode: joinCode.trim() }),
              isEn ? 'Connected to shared schedule' : 'Conectado al horario compartido'
            ).then((res) => {
              if (res) onClose();
            });
          }}
        >
          <strong style={{ fontSize: 12.5 }}>
            {isEn ? 'Join another user’s shared schedule' : 'Unirse al horario compartido de otro equipo'}
          </strong>
          <div style={{ display: 'flex', gap: 6 }}>
            <input
              className="input"
              style={{ flex: 1 }}
              placeholder="OMIPFS-..."
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value)}
            />
            <button type="submit" className="btn btn-primary" disabled={!joinCode.trim()}>
              {isEn ? 'Join live' : 'Unirse'}
            </button>
          </div>
        </form>
      </div>
      <div className="modal-footer">
        <button className="btn" onClick={onClose}>
          {isEn ? 'Close' : 'Cerrar'}
        </button>
      </div>
    </>
  );
};

/* ---------------- 4. MCP SERVER CONFIG MODAL (AI AGENT) ---------------- */
const McpModal: React.FC<{
  isEn: boolean;
  enabled: boolean;
  httpUrl: string;
  onToggle: (en: boolean) => Promise<void>;
  onClose: () => void;
}> = ({ isEn, enabled, httpUrl, onToggle, onClose }) => {
  const [cfg, setCfg] = useState<{ stdioCommand: string; toolsCount: number; httpUrl: string }>({
    stdioCommand: 'openmimo-backend',
    toolsCount: 17,
    httpUrl,
  });
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    api
      .getMcpConfig()
      .then((res) => setCfg({ stdioCommand: res.stdioCommand, toolsCount: res.toolsCount, httpUrl: res.httpUrl }))
      .catch(() => {});
  }, [enabled]);

  const agentSnippet = `mcp_servers:\n  openmimo:\n    command: "${cfg.stdioCommand}"\n    args: ["-mcp"]\n  # ${isEn ? 'Or live HTTP session:' : 'O por HTTP en vivo:'}\n  # url: "${cfg.httpUrl}"`;

  return (
    <>
      <div className="modal-header">
        <h3>
          <Cpu size={17} color="#2563eb" />
          {isEn ? 'MCP Server Configuration (AI Agent)' : 'Configuración del Servidor MCP (Agente)'}
        </h3>
        <button className="btn btn-icon" onClick={onClose}>
          <X size={15} />
        </button>
      </div>
      <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: 12,
            borderRadius: 8,
            background: enabled ? '#f0fdf4' : '#fef2f2',
            border: `1px solid ${enabled ? '#86efac' : '#fecaca'}`,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <span className={`mcp-dot ${enabled ? 'on' : 'off'}`} style={{ width: 10, height: 10 }} />
            <div>
              <strong>
                {enabled
                  ? isEn
                    ? 'MCP Server Active'
                    : 'Servidor MCP activo'
                  : isEn
                  ? 'MCP Server Inactive'
                  : 'Servidor MCP desactivado'}
              </strong>
              <div style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>
                {cfg.toolsCount}{' '}
                {isEn
                  ? 'timetabling tools exposed (JSON-RPC 2.0)'
                  : 'herramientas de horarios disponibles (JSON-RPC 2.0)'}
              </div>
            </div>
          </div>
          <button
            className={`btn ${enabled ? 'btn-danger' : 'btn-ok'}`}
            onClick={() => onToggle(!enabled)}
          >
            {enabled ? (isEn ? 'Disable MCP' : 'Desactivar') : isEn ? 'Enable MCP' : 'Activar'}
          </button>
        </div>

        <div className="field">
          <label>{isEn ? 'Live HTTP MCP Endpoint' : 'Endpoint MCP HTTP en vivo'}</label>
          <input className="input" readOnly value={cfg.httpUrl} style={{ fontFamily: 'monospace' }} />
        </div>

        <div className="field">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <label>{isEn ? 'Agent Configuration (MCP config)' : 'Configuración para Agente (MCP)'}</label>
            <button
              className="btn btn-sm"
              onClick={() => {
                navigator.clipboard?.writeText(agentSnippet);
                setCopied(true);
                setTimeout(() => setCopied(false), 1800);
              }}
            >
              {copied ? <Check size={12} /> : <Copy size={12} />} {copied ? 'OK' : isEn ? 'Copy' : 'Copiar'}
            </button>
          </div>
          <textarea
            className="input"
            readOnly
            rows={5}
            style={{ fontFamily: 'monospace', fontSize: 11.5, background: '#f8fafc' }}
            value={agentSnippet}
          />
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
