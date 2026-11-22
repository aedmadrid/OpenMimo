package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// AppDataDir returns the OS-standard hidden application data directory:
// - macOS: ~/Library/Application Support/openmimo
// - Linux: $XDG_DATA_HOME/openmimo or ~/.local/share/openmimo
// - Windows: %APPDATA%\openmimo
func AppDataDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "openmimo")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "openmimo")
		}
		return filepath.Join(home, "AppData", "Roaming", "openmimo")
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "openmimo")
		}
		return filepath.Join(home, ".local", "share", "openmimo")
	}
}

func ProjectsDir() string {
	dir := filepath.Join(AppDataDir(), "projects")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func SharedRegistryPath() string {
	dir := AppDataDir()
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "shared.json")
}

// AISettings stores local-only OpenAI-compatible API settings (never exported inside .omimo files).
type AISettings struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
}

func AISettingsPath() string {
	dir := AppDataDir()
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "ai_settings.json")
}

func LoadAISettings() AISettings {
	cfg := AISettings{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o-mini",
	}
	b, err := os.ReadFile(AISettingsPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(b, &cfg)
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "gpt-4o-mini"
	}
	return cfg
}

func SaveAISettings(cfg AISettings) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "gpt-4o-mini"
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(AISettingsPath(), b, 0o600)
}

type ProjectSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Institution   string `json:"institution"`
	Department    string `json:"department"`
	CoursesCount  int    `json:"coursesCount"`
	ResourcesCount int   `json:"resourcesCount"`
	GuidesCount   int    `json:"guidesCount"`
	UpdatedAt     string `json:"updatedAt"`
	IsOpen        bool   `json:"isOpen"`
	Shared        bool   `json:"shared"`
	ShareCode     string `json:"shareCode,omitempty"`
	IPFSCid       string `json:"ipfsCid,omitempty"`
}

type SharedRemoteEntry struct {
	ProjectID   string `json:"projectId"`
	Name        string `json:"name"`
	Institution string `json:"institution"`
	ShareCode   string `json:"shareCode"`
	IPFSCid     string `json:"ipfsCid,omitempty"`
	HostURL     string `json:"hostUrl"`
	HostOnline  bool   `json:"hostOnline"`
	HostOpen    bool   `json:"hostOpen"`
	LastSeen    string `json:"lastSeen"`
}

type OpenTab struct {
	Project *Project
	Undo    []*Project
	Redo    []*Project
}

type ProjectTabInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsRemote  bool   `json:"isRemote"`
	HostOpen  bool   `json:"hostOpen"`
	ShareCode string `json:"shareCode,omitempty"`
}

func SaveProjectToDisk(p *Project) error {
	if p == nil || p.ID == "" {
		return nil
	}
	if p.Collab.IsRemote {
		return nil
	}
	p.Format = "openmimo/1"
	p.UpdatedAt = time.Now().Format(time.RFC3339)
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	// Native OpenMimo format: .omimo (JSON)
	target := filepath.Join(ProjectsDir(), safeFile(p.ID)+".omimo")
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	_ = os.Rename(tmp, target)
	// Clean legacy .json duplicate if present so recent list has no duplicates
	_ = os.Remove(filepath.Join(ProjectsDir(), safeFile(p.ID)+".json"))
	return nil
}

func LoadProjectFromDisk(id string) (*Project, error) {
	// Prefer native .omimo, fall back to legacy .json
	target := filepath.Join(ProjectsDir(), safeFile(id)+".omimo")
	b, err := os.ReadFile(target)
	if err != nil {
		b, err = os.ReadFile(filepath.Join(ProjectsDir(), safeFile(id)+".json"))
		if err != nil {
			return nil, err
		}
	}
	var p Project
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	p.RecomputeDerived()
	return &p, nil
}

func ListLocalProjects(openIDs map[string]bool) []ProjectSummary {
	entries, err := os.ReadDir(ProjectsDir())
	if err != nil {
		return []ProjectSummary{}
	}
	out := make([]ProjectSummary, 0, len(entries))
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() || (!strings.HasSuffix(e.Name(), ".omimo") && !strings.HasSuffix(e.Name(), ".json")) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(ProjectsDir(), e.Name()))
		if err != nil {
			continue
		}
		var p Project
		if err := json.Unmarshal(b, &p); err != nil || p.ID == "" || seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		out = append(out, ProjectSummary{
			ID:             p.ID,
			Name:           p.Name,
			Institution:    p.Institution,
			Department:     p.Department,
			CoursesCount:   len(p.Courses),
			ResourcesCount: len(p.Resources),
			GuidesCount:    len(p.Guides),
			UpdatedAt:      p.UpdatedAt,
			IsOpen:         openIDs[p.ID],
			Shared:         p.Collab.Shared,
			ShareCode:      p.Collab.ShareCode,
			IPFSCid:        p.Collab.IPFSCid,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

func LoadSharedRegistry() []SharedRemoteEntry {
	b, err := os.ReadFile(SharedRegistryPath())
	if err != nil {
		return []SharedRemoteEntry{}
	}
	var list []SharedRemoteEntry
	if err := json.Unmarshal(b, &list); err != nil {
		return []SharedRemoteEntry{}
	}
	return list
}

func SaveSharedRegistry(list []SharedRemoteEntry) {
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(SharedRegistryPath(), b, 0o644)
}

func UpsertSharedRegistry(entry SharedRemoteEntry) {
	list := LoadSharedRegistry()
	found := false
	for i := range list {
		if list[i].ShareCode == entry.ShareCode || (entry.ProjectID != "" && list[i].ProjectID == entry.ProjectID && list[i].HostURL == entry.HostURL) {
			list[i] = entry
			found = true
			break
		}
	}
	if !found {
		list = append([]SharedRemoteEntry{entry}, list...)
	}
	SaveSharedRegistry(list)
}

func DetectSystemLang() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(os.Getenv(k))
		if strings.HasPrefix(v, "en") {
			return "en"
		}
		if strings.HasPrefix(v, "es") {
			return "es"
		}
	}
	return "es"
}

// CreateProjectFromWizard builds a customized project from the New Schedule Setup Wizard.
type WizardConfig struct {
	Name        string `json:"name"`
	Institution string `json:"institution"`
	Department  string `json:"department"`
	Language    string `json:"language"`
	MaxWeeks    int    `json:"maxWeeks"`
	MaxDays     int    `json:"maxDays"`
	MaxPeriods  int    `json:"maxPeriods"`
	StartTime   string `json:"startTime"`   // "HH:MM", default "08:30"
	StartHour   int    `json:"startHour"`   // legacy fallback
	SlotMinutes int    `json:"slotMinutes"`
	BreakMins   int    `json:"breakMins"`
	Template    string `json:"template"` // "empty" | "university" | "demo"
}

func CreateProjectFromWizard(cfg WizardConfig) *Project {
	lang := cfg.Language
	if lang != "en" && lang != "es" {
		lang = DetectSystemLang()
	}
	if cfg.Template == "demo" {
		p := NewUniversityDemoProject()
		p.ID = fmt.Sprintf("proj_%d", time.Now().UnixNano())
		if strings.TrimSpace(cfg.Name) != "" {
			p.Name = strings.TrimSpace(cfg.Name)
		}
		if strings.TrimSpace(cfg.Institution) != "" {
			p.Institution = strings.TrimSpace(cfg.Institution)
		}
		if strings.TrimSpace(cfg.Department) != "" {
			p.Department = strings.TrimSpace(cfg.Department)
		}
		p.Language = lang
		return p
	}

	p := NewEmptyProject(lang)
	if strings.TrimSpace(cfg.Name) != "" {
		p.Name = strings.TrimSpace(cfg.Name)
	}
	if strings.TrimSpace(cfg.Institution) != "" {
		p.Institution = strings.TrimSpace(cfg.Institution)
	}
	if strings.TrimSpace(cfg.Department) != "" {
		p.Department = strings.TrimSpace(cfg.Department)
	}
	if cfg.MaxWeeks >= 1 && cfg.MaxWeeks <= 255 {
		p.Time.MaxWeeks = cfg.MaxWeeks
		p.Time.Weeks = DefaultWeeks(cfg.MaxWeeks)
	}
	if cfg.MaxDays >= 1 && cfg.MaxDays <= 7 {
		p.Time.MaxDays = cfg.MaxDays
	}
	if cfg.MaxPeriods >= 1 && cfg.MaxPeriods <= 30 {
		p.Time.MaxPeriods = cfg.MaxPeriods
	}

	// Parse StartTime ("08:30" default)
	curMin := 8*60 + 30
	if strings.Contains(cfg.StartTime, ":") {
		parts := strings.Split(strings.TrimSpace(cfg.StartTime), ":")
		if len(parts) == 2 {
			var hh, mm int
			_, _ = fmt.Sscanf(parts[0], "%d", &hh)
			_, _ = fmt.Sscanf(parts[1], "%d", &mm)
			if hh >= 0 && hh <= 23 && mm >= 0 && mm <= 59 {
				curMin = hh*60 + mm
			}
		}
	} else if cfg.StartHour >= 6 && cfg.StartHour <= 20 {
		curMin = cfg.StartHour * 60
	}
	slotM := cfg.SlotMinutes
	if slotM < 15 || slotM > 180 {
		slotM = 60
	}
	brkM := cfg.BreakMins
	if brkM < 0 || brkM > 60 {
		brkM = 0
	}

	bells := make([]string, 0, p.Time.MaxPeriods)
	for i := 0; i < p.Time.MaxPeriods; i++ {
		endMin := curMin + slotM
		bells = append(bells, fmt.Sprintf("%02d:%02d-%02d:%02d", (curMin/60)%24, curMin%60, (endMin/60)%24, endMin%60))
		curMin = endMin + brkM
	}
	p.Time.Bells = bells

	if cfg.Template == "university" {
		// Starter scaffold with morning/afternoon parallel groups & sample guide
		p.Resources = []Resource{
			{ID: "r_g1", Code: "1A-M", Name: "1º Grupo A (Mañana)", Category: 1, Planned: 30},
			{ID: "r_g2", Code: "1B-T", Name: "1º Grupo B (Tarde)", Category: 1, Planned: 30},
			{ID: "r_p1", Code: "PROF-1", Name: "Docente Mañana", Category: 2, Planned: 20},
			{ID: "r_p2", Code: "PROF-2", Name: "Docente Tarde", Category: 2, Planned: 20},
			{ID: "r_s1", Code: "AULA-01", Name: "Aula Taller 01", Category: 3, Planned: 40},
		}
		p.Guides = []TeachingGuide{
			{
				ID: "g_1", Slug: "proyectos-basicos", Title: "Proyectos Básicos",
				Degree: "Grado en Diseño", DegreeLevel: "Grado", CourseType: "Formación básica",
				Character: "Teórico-práctica", Specialty: "Común", SubjectArea: "Proyectos",
				AcademicYear: "1º", Semester: "1º Semestre", CreditsECTS: 6, TotalHours: 150, InPersonHours: 60,
				Department: p.Department, Languages: "Español",
				Description: "Introducción metodológica al diseño y desarrollo de proyectos.",
				Groups: []GuideGroupAssignment{
					{Group: "1º Grupo A", Teacher: "Docente Mañana", Shift: "Mañana"},
					{Group: "1º Grupo B", Teacher: "Docente Tarde", Shift: "Tarde"},
				},
				EvalAttitudePct: 20, EvalWrittenPct: 20, EvalPracticalPct: 60,
				UpdatedAt: time.Now().Format(time.RFC3339),
			},
		}
		p.Courses = []Course{
			{ID: "c_1", Code: "PROY1-A", Name: "Proyectos Básicos (Mañana)", Category: 1, Planned: 4 * p.Time.MaxWeeks, Color: "#3b82f6", GuideID: "g_1", Resources: []string{"r_g1", "r_p1", "r_s1"}},
			{ID: "c_2", Code: "PROY1-B", Name: "Proyectos Básicos (Tarde)", Category: 1, Planned: 4 * p.Time.MaxWeeks, Color: "#f59e0b", GuideID: "g_1", Resources: []string{"r_g2", "r_p2", "r_s1"}},
		}
		for _, c := range p.Courses {
			for w := 1; w <= p.Time.MaxWeeks; w++ {
				p.WeekAllocations = append(p.WeekAllocations, WeekAllocation{CourseID: c.ID, Week: w, Periods: 4})
			}
		}
	}
	p.RecomputeDerived()
	return p
}
