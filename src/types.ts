export interface Category {
  index: number;
  shortCode: string;
  name: string;
  noTimetables: boolean;
  noGaps: boolean;
  weeklyLimit: number;
  gapWeight: number;
}

export interface WeekDef {
  index: number;
  name: string;
  length: number;
}

export interface TimeSettings {
  maxWeeks: number;
  maxDays: number;
  maxPeriods: number;
  firstWeek: number;
  protectedWeek: number;
  days: string[];
  bells: string[];
  weeks: WeekDef[];
  startDate: string;
}

export interface Course {
  id: string;
  code: string;
  name: string;
  category: number;
  planned: number;
  minConsecutive?: number;
  maxConsecutive?: number;
  color: string;
  locked: boolean;
  memo: string;
  mcpNotes?: string;
  guideId?: string;
  resources: string[];
  allocated: number;
  scheduled: number;
  left: number;
  links: number;
}

export interface Resource {
  id: string;
  code: string;
  name: string;
  category: number;
  planned: number;
  color: string;
  locked: boolean;
  memo: string;
  mcpNotes?: string;
  allocated: number;
  scheduled: number;
  left: number;
  links: number;
}

export interface WeekAllocation {
  courseId: string;
  week: number;
  periods: number;
}

export interface SlotAssignment {
  id: string;
  courseId: string;
  week: number;
  day: number;
  period: number;
  roomId?: string;
  comment?: string;
}

export interface Booking {
  id: string;
  targetType: 'course' | 'resource';
  targetId: string;
  scope: 'timetable' | 'week';
  week: number;
  day: number;
  period: number;
  kind: number;
  label: string;
}

export interface GuideGroupAssignment {
  group: string;
  teacher: string;
  shift: string;
  notes?: string;
}

export interface TeachingGuide {
  id: string;
  slug: string;
  title: string;
  degree: string;
  degreeLevel: string;
  courseType: string;
  character: string;
  specialty: string;
  subjectArea: string;
  academicYear: string;
  semester: string;
  creditsEcts: number;
  totalHours: number;
  inPersonHours: number;
  department: string;
  prerequisites: string;
  languages: string;
  description: string;
  groups: GuideGroupAssignment[];
  competencies: string;
  learningOutcomes: string;
  contents: string;
  theoryActivities: string;
  practiceActivities: string;
  otherActivities: string;
  studentWorkHours: string;
  evalAttitudePct: number;
  evalWrittenPct: number;
  evalPracticalPct: number;
  evalCriteria: string;
  bibliography: string;
  updatedAt: string;
}

export interface CollabInfo {
  shared: boolean;
  shareCode?: string;
  ipfsCid?: string;
  ipfsMode?: string;
  isRemote: boolean;
  hostUrl?: string;
  hostOpen: boolean;
  revision: number;
}

export interface UserScript {
  id: string;
  name: string;
  description: string;
  author: string;
  code: string;
  updatedAt: string;
}

export interface UpdateCheckResult {
  currentVersion: string;
  latestVersion?: string;
  releaseName?: string;
  hasUpdate: boolean;
  checked: boolean;
  releaseUrl: string;
}

export interface Project {
  format?: string;
  id: string;
  name: string;
  institution: string;
  department: string;
  comment: string;
  mcpNotes?: string;
  language: 'es' | 'en';
  roomsCategory: number;
  time: TimeSettings;
  courseCategories: Category[];
  resourceCategories: Category[];
  bookingTypes: string[];
  courses: Course[];
  resources: Resource[];
  weekAllocations: WeekAllocation[];
  slots: SlotAssignment[];
  bookings: Booking[];
  guides: TeachingGuide[];
  scripts: UserScript[];
  collab: CollabInfo;
  updatedAt: string;
}

export interface Conflict {
  type: string;
  week: number;
  day: number;
  period: number;
  resourceId?: string;
  resource?: string;
  courseIds?: string[];
  courseCodes?: string[];
  message: string;
}

export interface LoadEntry {
  resourceId: string;
  resourceCode: string;
  resourceName: string;
  category: number;
  week: number;
  load: number;
}

export interface SlotLoadEntry {
  week: number;
  day: number;
  period: number;
  count: number;
}

export interface ProjectStats {
  totalCourses: number;
  totalResources: number;
  totalCodes: number;
  totalLinks: number;
  totalPlanned: number;
  totalAllocated: number;
  totalScheduled: number;
  totalUnscheduled: number;
  totalBookings: number;
  manualRoomsCount: number;
  totalGaps: number;
  weightedGaps: number;
  gapsByCategory: Record<number, number>;
  conflictCount: number;
  conflicts: Conflict[];
  maxWeeklyLoad: LoadEntry[];
  maxSlotLoad: SlotLoadEntry[];
  weekCompletion: Record<number, number>;
  overallCompletion: number;
}

export interface SlotAvailability {
  day: number;
  period: number;
  available: boolean;
  reasons?: string[];
}

export interface OptimizeOptions {
  action: 'initial' | 'optimise' | 'auto_improve' | 'copy' | 'remove';
  fromWeek: number;
  toWeek: number;
  minDay: number;
  maxDay: number;
  minPeriod: number;
  maxPeriod: number;
  minBlock: number;
  maxBlock: number;
  passes?: number;
  includeLocked: boolean;
  bookingsWithGaps: boolean;
  gapWeights: Record<number, number>;
  courseIds?: string[];
  randSeed: number;
}

export interface OptimizeResult {
  action: string;
  placed: number;
  removed: number;
  unscheduled: number;
  gapsBefore: number;
  gapsAfter: number;
  gapsChange: number;
  gapsPercent: number;
  movesEvaluated: number;
  changedCourses: string[];
}

export interface ProjectTabInfo {
  id: string;
  name: string;
  isRemote: boolean;
  hostOpen: boolean;
  shareCode?: string;
}

export interface ProjectSummary {
  id: string;
  name: string;
  institution: string;
  department: string;
  coursesCount: number;
  resourcesCount: number;
  guidesCount: number;
  updatedAt: string;
  isOpen: boolean;
  shared: boolean;
  shareCode?: string;
  ipfsCid?: string;
}

export interface SharedRemoteEntry {
  projectId: string;
  name: string;
  institution: string;
  shareCode: string;
  ipfsCid?: string;
  hostUrl: string;
  hostOnline: boolean;
  hostOpen: boolean;
  lastSeen: string;
}

export interface SampleDescriptor {
  key: string;
  titleEs: string;
  titleEn: string;
  descEs: string;
  descEn: string;
  badge: string;
}

export interface WorkspaceResponse {
  recent: ProjectSummary[];
  shared: SharedRemoteEntry[];
  samples?: SampleDescriptor[];
  tabs: ProjectTabInfo[];
  activeTab: string;
  dataDir: string;
  mcpEnabled: boolean;
  mcpHttpUrl: string;
  systemLang: 'es' | 'en';
}

export interface WizardConfig {
  name: string;
  institution: string;
  department: string;
  language: 'es' | 'en';
  maxWeeks: number;
  maxDays: number;
  maxPeriods: number;
  startTime: string;
  slotMinutes: number;
  breakMins: number;
  template: 'empty' | 'university' | 'demo';
}

export interface ApiPayload {
  project: Project | null;
  stats: ProjectStats;
  canUndo: boolean;
  canRedo: boolean;
  tabs?: ProjectTabInfo[];
  activeTab?: string;
  mcpEnabled?: boolean;
  mcpHttpUrl?: string;
  dataDir?: string;
  systemLang?: 'es' | 'en';
  result?: OptimizeResult;
  assigned?: number;
  guide?: TeachingGuide;
}

export type MainView = 'timetables' | 'weeks' | 'courses' | 'resources' | 'guides' | 'scripts';
export type ModalKind =
  | null
  | 'optimize'
  | 'rooms'
  | 'options'
  | 'stats'
  | 'reports'
  | 'wizard'
  | 'collab'
  | 'mcp';
