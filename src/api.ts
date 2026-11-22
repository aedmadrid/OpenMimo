import type {
  ApiPayload,
  Course,
  OptimizeOptions,
  Project,
  Resource,
  SlotAvailability,
  TeachingGuide,
  UpdateCheckResult,
  UserScript,
  WizardConfig,
  WorkspaceResponse,
} from './types';

declare global {
  interface Window {
    __OPENMIMO_API__?: string;
    __OPENMIMO_SCREENSHOT__?: boolean;
    __TAURI_INTERNALS__?: {
      invoke?: (cmd: string, args?: Record<string, unknown>) => Promise<unknown>;
    };
  }
}

let cachedBaseUrl: string | null = null;

export async function getApiBase(): Promise<string> {
  if (cachedBaseUrl) return cachedBaseUrl;
  if (window.__OPENMIMO_API__) {
    cachedBaseUrl = window.__OPENMIMO_API__;
    return cachedBaseUrl;
  }
  if (window.__TAURI_INTERNALS__?.invoke) {
    try {
      const url = (await window.__TAURI_INTERNALS__.invoke('get_backend_url')) as string;
      if (url) {
        cachedBaseUrl = url;
        return url;
      }
    } catch {
      // fallback
    }
  }
  cachedBaseUrl = 'http://127.0.0.1:8787';
  return cachedBaseUrl;
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const base = await getApiBase();
  const res = await fetch(`${base}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });
  const json = await res.json();
  if (!res.ok) {
    throw new Error(json.error || `HTTP ${res.status}`);
  }
  return json as T;
}

export const api = {
  getWorkspace: () => req<WorkspaceResponse>('/api/workspace'),
  openSample: (key: string, lang: 'es' | 'en') =>
    req<ApiPayload>('/api/projects/sample', { method: 'POST', body: JSON.stringify({ key, lang }) }),
  createProject: (cfg: WizardConfig) =>
    req<ApiPayload>('/api/projects/create', { method: 'POST', body: JSON.stringify(cfg) }),
  openProject: (id: string) =>
    req<ApiPayload>('/api/projects/open', { method: 'POST', body: JSON.stringify({ id }) }),
  switchTab: (id: string) =>
    req<ApiPayload>('/api/projects/switch', { method: 'POST', body: JSON.stringify({ id }) }),
  closeTab: (id: string) =>
    req<ApiPayload>('/api/projects/close', { method: 'POST', body: JSON.stringify({ id }) }),
  deleteProject: (id: string) =>
    req<ApiPayload>(`/api/projects/delete?id=${encodeURIComponent(id)}`, { method: 'DELETE' }),
  exportProjectById: (id: string) =>
    req<{ fileName: string; content: string }>(
      `/api/projects/export-by-id?id=${encodeURIComponent(id)}`
    ),

  getProject: () => req<ApiPayload>('/api/project'),
  replaceProject: (proj: Project) =>
    req<ApiPayload>('/api/project', { method: 'PUT', body: JSON.stringify(proj) }),
  checkUpdates: () => req<UpdateCheckResult>('/api/updates/check'),
  saveScript: (sc: Partial<UserScript>) =>
    req<ApiPayload>('/api/scripts', { method: 'POST', body: JSON.stringify(sc) }),
  deleteScript: (id: string) =>
    req<ApiPayload>(`/api/scripts?id=${encodeURIComponent(id)}`, { method: 'DELETE' }),
  newProject: (mode: 'empty' | 'demo', lang: 'es' | 'en') =>
    req<ApiPayload>('/api/project/new', { method: 'POST', body: JSON.stringify({ mode, lang }) }),
  undo: () => req<ApiPayload>('/api/undo', { method: 'POST' }),
  redo: () => req<ApiPayload>('/api/redo', { method: 'POST' }),

  createCourse: (c: Partial<Course>) =>
    req<ApiPayload>('/api/courses', { method: 'POST', body: JSON.stringify(c) }),
  updateCourse: (c: Course) =>
    req<ApiPayload>('/api/courses', { method: 'PUT', body: JSON.stringify(c) }),
  deleteCourse: (id: string) =>
    req<ApiPayload>(`/api/courses?id=${encodeURIComponent(id)}`, { method: 'DELETE' }),
  courseAction: (body: Record<string, unknown>) =>
    req<ApiPayload>('/api/courses/action', { method: 'POST', body: JSON.stringify(body) }),

  createResource: (r: Partial<Resource>) =>
    req<ApiPayload>('/api/resources', { method: 'POST', body: JSON.stringify(r) }),
  updateResource: (r: Resource) =>
    req<ApiPayload>('/api/resources', { method: 'PUT', body: JSON.stringify(r) }),
  deleteResource: (id: string) =>
    req<ApiPayload>(`/api/resources?id=${encodeURIComponent(id)}`, { method: 'DELETE' }),
  sortResources: (by: string, byCategory: boolean, ascending: boolean) =>
    req<ApiPayload>('/api/resources/sort', { method: 'POST', body: JSON.stringify({ by, byCategory, ascending }) }),
  importTeachers: (text: string, category = 2) =>
    req<ApiPayload & { added?: number }>('/api/resources/import-teachers', {
      method: 'POST',
      body: JSON.stringify({ text, category }),
    }),
  batchBookings: (body: {
    targetType?: 'resource' | 'course';
    targetId: string;
    mode: 'toggle_cell' | 'block_day' | 'block_afternoons' | 'clear_all';
    week: number;
    allWeeks: boolean;
    day?: number;
    period?: number;
    label?: string;
  }) => req<ApiPayload>('/api/bookings/batch', { method: 'POST', body: JSON.stringify(body) }),

  setWeekAllocation: (courseId: string, week: number, periods: number) =>
    req<ApiPayload>('/api/weeks/allocation', { method: 'POST', body: JSON.stringify({ courseId, week, periods }) }),

  placeSlot: (courseId: string, week: number, day: number, period: number, roomId = '', force = false) =>
    req<ApiPayload>('/api/slots', { method: 'POST', body: JSON.stringify({ courseId, week, day, period, roomId, force }) }),
  moveSlot: (id: string, day: number, period: number) =>
    req<ApiPayload>('/api/slots', { method: 'PUT', body: JSON.stringify({ id, day, period }) }),
  setSlotRoom: (id: string, roomId: string) =>
    req<ApiPayload>('/api/slots', { method: 'PUT', body: JSON.stringify({ id, roomId, setRoomOnly: true }) }),
  deleteSlot: (id: string) =>
    req<ApiPayload>(`/api/slots?id=${encodeURIComponent(id)}`, { method: 'DELETE' }),

  whatIf: (courseId: string, week: number) =>
    req<{ slots: SlotAvailability[] }>(`/api/whatif?courseId=${encodeURIComponent(courseId)}&week=${week}`),

  toggleBooking: (body: Record<string, unknown>) =>
    req<ApiPayload>('/api/bookings/toggle', { method: 'POST', body: JSON.stringify(body) }),

  optimize: (opts: OptimizeOptions) =>
    req<ApiPayload>('/api/optimize', { method: 'POST', body: JSON.stringify(opts) }),

  availableRooms: (week: number, day: number, period: number, ignoreSlotId = '') =>
    req<{ rooms: Resource[] }>(
      `/api/rooms/available?week=${week}&day=${day}&period=${period}&ignoreSlotId=${encodeURIComponent(ignoreSlotId)}`
    ),
  roomAction: (body: Record<string, unknown>) =>
    req<ApiPayload>('/api/rooms/action', { method: 'POST', body: JSON.stringify(body) }),

  saveGuide: (
    guide: Partial<TeachingGuide> & { containedCourseIds?: string[]; syncContained?: boolean },
    linkCourseId = ''
  ) =>
    req<ApiPayload>(
      `/api/guides${linkCourseId ? `?linkCourseId=${encodeURIComponent(linkCourseId)}` : ''}`,
      { method: 'POST', body: JSON.stringify(guide) }
    ),
  deleteGuide: (id: string) =>
    req<ApiPayload>(`/api/guides?id=${encodeURIComponent(id)}`, { method: 'DELETE' }),
  exportAstroZip: () =>
    req<{ fileName: string; base64Zip: string; bytes: number }>('/api/export/astro'),

  shareCollab: (stop = false) =>
    req<ApiPayload>('/api/collab/share', { method: 'POST', body: JSON.stringify({ stop }) }),
  joinCollab: (body: { shareCode?: string; hostUrl?: string; projectId?: string }) =>
    req<ApiPayload>('/api/collab/join', { method: 'POST', body: JSON.stringify(body) }),

  getMcpConfig: () =>
    req<{ enabled: boolean; httpUrl: string; stdioCommand: string; stdioArgs: string[]; toolsCount: number; payload: ApiPayload }>(
      '/api/mcp/config'
    ),
  setMcpEnabled: (enabled: boolean) =>
    req<{ enabled: boolean; httpUrl: string; stdioCommand: string; stdioArgs: string[]; toolsCount: number; payload: ApiPayload }>(
      '/api/mcp/config',
      { method: 'POST', body: JSON.stringify({ enabled }) }
    ),

  importFile: (fileName: string, format: string, content: string) =>
    req<ApiPayload>('/api/import', { method: 'POST', body: JSON.stringify({ fileName, format, content }) }),
  exportFile: (format: string, reportType = 3, week = 1, resourceId = '') =>
    req<{ fileName: string; content: string }>(
      `/api/export?format=${format}&reportType=${reportType}&week=${week}&resourceId=${encodeURIComponent(resourceId)}`
    ),
  updateSettings: (body: Record<string, unknown>) =>
    req<ApiPayload>('/api/settings', { method: 'POST', body: JSON.stringify(body) }),
};
