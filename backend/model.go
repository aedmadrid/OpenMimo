package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Category struct {
	Index        int    `json:"index"`        // 1..7
	ShortCode    string `json:"shortCode"`    // e.g. "C", "P", "S"
	Name         string `json:"name"`         // e.g. "Clases", "Profesores", "Salas"
	NoTimetables bool   `json:"noTimetables"` // true = conflicts ignored (e.g. Subjects)
	NoGaps       bool   `json:"noGaps"`       // true = gaps not penalized (e.g. Rooms, Equipment)
	WeeklyLimit  int    `json:"weeklyLimit"`  // 0 = unlimited; >0 warns in Weeks view
	GapWeight    int    `json:"gapWeight"`    // 1..10 weight in gap optimization
}

type WeekDef struct {
	Index  int    `json:"index"`  // 1..maxWeeks
	Name   string `json:"name"`   // e.g. "S1", "[1]", "A1"
	Length int    `json:"length"` // multiplier (default 1)
}

type TimeSettings struct {
	MaxWeeks      int       `json:"maxWeeks"`      // 1..255
	MaxDays       int       `json:"maxDays"`       // 1..7
	MaxPeriods    int       `json:"maxPeriods"`    // 1..30
	FirstWeek     int       `json:"firstWeek"`     // default 1
	ProtectedWeek int       `json:"protectedWeek"` // 0 = none; weeks <= protectedWeek are protected
	Days          []string  `json:"days"`          // 7 day names
	Bells         []string  `json:"bells"`         // up to 30 bell labels e.g. "08:00-08:45"
	Weeks         []WeekDef `json:"weeks"`
	StartDate     string    `json:"startDate"` // YYYY-MM-DD
}

type Course struct {
	ID             string   `json:"id"`
	Code           string   `json:"code"`
	Name           string   `json:"name"`
	Category       int      `json:"category"` // 1..7
	Planned        int      `json:"planned"`  // MAX periods
	MinConsecutive int      `json:"minConsecutive,omitempty"` // min consecutive periods per block (0 = default)
	MaxConsecutive int      `json:"maxConsecutive,omitempty"` // max consecutive periods per block (0 = default)
	Color          string   `json:"color"`    // optional hex override
	Locked         bool     `json:"locked"`
	Memo           string   `json:"memo"`
	McpNotes       string   `json:"mcpNotes,omitempty"` // instructions/preferences for AI Agent / MCP
	GuideID        string   `json:"guideId,omitempty"`  // optional link to a shared TeachingGuide (for parallel groups)
	Resources      []string `json:"resources"`          // list of Resource IDs (Collection)
	// Derived (populated on response):
	Allocated int `json:"allocated"`
	Scheduled int `json:"scheduled"`
	Left      int `json:"left"`
	Links     int `json:"links"`
}

type GuideGroupAssignment struct {
	Group   string `json:"group"`   // e.g. "2º Gráfico A"
	Teacher string `json:"teacher"` // e.g. "Dra. Elena Torres"
	Shift   string `json:"shift"`   // "Mañana" | "Tarde" | "Paralelo"
	Notes   string `json:"notes,omitempty"`
}

type TeachingGuide struct {
	ID                 string                 `json:"id"`
	Slug               string                 `json:"slug"`
	Title              string                 `json:"title"`              // Nombre asignatura
	Degree             string                 `json:"degree"`             // Titulación
	DegreeLevel        string                 `json:"degreeLevel"`        // Nivel de título (Grado / Máster)
	CourseType         string                 `json:"courseType"`         // Formación básica / Obligatoria / Optativa / TFG
	Character          string                 `json:"character"`          // Teórico-práctica / Práctica / Teórica
	Specialty          string                 `json:"specialty"`          // Especialidad / Itinerario (Gráfico, Interiores, Moda, Producto...)
	SubjectArea        string                 `json:"subjectArea"`        // Materia
	AcademicYear       string                 `json:"academicYear"`       // Curso (1º, 2º, 3º, 4º)
	Semester           string                 `json:"semester"`           // Semestre (1º, 2º, Anual)
	CreditsECTS        float64                `json:"creditsEcts"`        // Número de créditos
	TotalHours         int                    `json:"totalHours"`         // Número de horas totales
	InPersonHours      int                    `json:"inPersonHours"`      // Número de horas presenciales
	Department         string                 `json:"department"`         // Departamento
	Prerequisites      string                 `json:"prerequisites"`      // Prelación / requisitos previos
	Languages          string                 `json:"languages"`          // Idioma/s
	Description        string                 `json:"description"`        // Descripción de la asignatura
	Groups             []GuideGroupAssignment `json:"groups"`             // Profesores y grupos paralelos (mañana, tarde, etc.)
	Competencies       string                 `json:"competencies"`       // Competencias específicas
	LearningOutcomes   string                 `json:"learningOutcomes"`   // Resultados del aprendizaje
	Contents           string                 `json:"contents"`           // Contenidos
	TheoryActivities   string                 `json:"theoryActivities"`   // Actividades teóricas
	PracticeActivities string                 `json:"practiceActivities"` // Actividades prácticas
	OtherActivities    string                 `json:"otherActivities"`    // Otras actividades obligatorias (jornadas, seminarios)
	StudentWorkHours   string                 `json:"studentWorkHours"`   // Horas de trabajo del estudiante / preparación prácticas / pruebas
	EvalAttitudePct    int                    `json:"evalAttitudePct"`    // Actitud y participación (%)
	EvalWrittenPct     int                    `json:"evalWrittenPct"`     // Pruebas escritas (%)
	EvalPracticalPct   int                    `json:"evalPracticalPct"`   // Pruebas prácticas (%)
	EvalCriteria       string                 `json:"evalCriteria"`       // Criterios de evaluación
	Bibliography       string                 `json:"bibliography"`       // Bibliografía y recursos
	UpdatedAt          string                 `json:"updatedAt"`
}

type CollabInfo struct {
	Shared    bool   `json:"shared"`
	ShareCode string `json:"shareCode,omitempty"`
	IPFSCid   string `json:"ipfsCid,omitempty"`
	IPFSMode  string `json:"ipfsMode,omitempty"` // "kubo" | "cid-local"
	IsRemote  bool   `json:"isRemote"`
	HostURL   string `json:"hostUrl,omitempty"`
	HostOpen  bool   `json:"hostOpen"`
	Revision  int64  `json:"revision"`
}

type Resource struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Category int    `json:"category"` // 1..7
	Planned  int    `json:"planned"`  // MAX capacity/load
	Color    string `json:"color"`
	Locked   bool   `json:"locked"`
	Memo     string `json:"memo"`
	McpNotes string `json:"mcpNotes,omitempty"` // instructions/constraints for AI Agent / MCP
	// Derived:
	Allocated int `json:"allocated"`
	Scheduled int `json:"scheduled"`
	Left      int `json:"left"`
	Links     int `json:"links"`
}

type WeekAllocation struct {
	CourseID string `json:"courseId"`
	Week     int    `json:"week"`    // 1..maxWeeks
	Periods  int    `json:"periods"` // allocated periods in this week
}

type SlotAssignment struct {
	ID       string `json:"id"`
	CourseID string `json:"courseId"`
	Week     int    `json:"week"`   // 1..maxWeeks
	Day      int    `json:"day"`    // 1..maxDays
	Period   int    `json:"period"` // 1..maxPeriods
	RoomID   string `json:"roomId,omitempty"` // optional manual room override
	Comment  string `json:"comment,omitempty"`
}

type Booking struct {
	ID         string `json:"id"`
	TargetType string `json:"targetType"` // "course" | "resource"
	TargetID   string `json:"targetId"`
	Scope      string `json:"scope"`      // "timetable" | "week"
	Week       int    `json:"week"`       // 1..maxWeeks
	Day        int    `json:"day"`        // 1..maxDays (0 for week scope)
	Period     int    `json:"period"`     // 1..maxPeriods (0 for week scope)
	Kind       int    `json:"kind"`       // 1="/////", 2="Libre/Holiday", 3="Reunión/Meeting", 4="Otros/Other"
	Label      string `json:"label"`
}

type UserScript struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author,omitempty"`
	Code        string `json:"code"`
	UpdatedAt   string `json:"updatedAt"`
}

type Project struct {
	Format             string           `json:"format,omitempty"` // "openmimo/1" (.omimo native format)
	ID                 string           `json:"id"`
	Name               string           `json:"name"`
	Institution        string           `json:"institution"`
	Department         string           `json:"department"`
	Comment            string           `json:"comment"`
	McpNotes           string           `json:"mcpNotes,omitempty"` // global rules/context for MCP / AI Agent
	Language           string           `json:"language"` // "es" | "en"
	RoomsCategory      int              `json:"roomsCategory"` // default 3
	Time               TimeSettings     `json:"time"`
	CourseCategories   []Category       `json:"courseCategories"`
	ResourceCategories []Category       `json:"resourceCategories"`
	BookingTypes       []string         `json:"bookingTypes"`
	Courses            []Course         `json:"courses"`
	Resources          []Resource       `json:"resources"`
	WeekAllocations    []WeekAllocation `json:"weekAllocations"`
	Slots              []SlotAssignment `json:"slots"`
	Bookings           []Booking        `json:"bookings"`
	Guides             []TeachingGuide  `json:"guides"`
	Scripts            []UserScript     `json:"scripts"`
	Collab             CollabInfo       `json:"collab"`
	UpdatedAt          string           `json:"updatedAt"`
}

func DefaultCategoriesES() ([]Category, []Category, []string) {
	courseCats := []Category{
		{Index: 1, ShortCode: "c", Name: "Clases regulares"},
		{Index: 2, ShortCode: "g", Name: "Grupos grandes / Magistral"},
		{Index: 3, ShortCode: "l", Name: "Laboratorio / Práctica"},
		{Index: 4, ShortCode: "o", Name: "Obligatorio"},
		{Index: 5, ShortCode: "v", Name: "Optativo / Electivo"},
		{Index: 6, ShortCode: "e", Name: "Exámenes"},
		{Index: 7, ShortCode: "s", Name: "Eventos especiales"},
	}
	resCats := []Category{
		{Index: 1, ShortCode: "C", Name: "Grupos / Clases", NoTimetables: false, NoGaps: false, GapWeight: 3},
		{Index: 2, ShortCode: "P", Name: "Profesores", NoTimetables: false, NoGaps: false, GapWeight: 2},
		{Index: 3, ShortCode: "S", Name: "Salas / Aulas", NoTimetables: false, NoGaps: true, GapWeight: 1},
		{Index: 4, ShortCode: "D", Name: "Departamentos / Áreas", NoTimetables: true, NoGaps: true, GapWeight: 1},
		{Index: 5, ShortCode: "A", Name: "Alumnos", NoTimetables: false, NoGaps: false, GapWeight: 1},
		{Index: 6, ShortCode: "E", Name: "Equipos", NoTimetables: false, NoGaps: true, GapWeight: 1},
		{Index: 7, ShortCode: "O", Name: "Otros", NoTimetables: false, NoGaps: true, GapWeight: 1},
	}
	bookings := []string{"/////", "Libre / Feriado", "Reunión", "Otros"}
	return courseCats, resCats, bookings
}

func DefaultCategoriesEN() ([]Category, []Category, []string) {
	courseCats := []Category{
		{Index: 1, ShortCode: "b", Name: "Basic course"},
		{Index: 2, ShortCode: "m", Name: "Middle course"},
		{Index: 3, ShortCode: "a", Name: "Advanced course"},
		{Index: 4, ShortCode: "f", Name: "Free choice"},
		{Index: 5, ShortCode: "o", Name: "Meetings"},
		{Index: 6, ShortCode: "e", Name: "Exams"},
		{Index: 7, ShortCode: "s", Name: "Special events"},
	}
	resCats := []Category{
		{Index: 1, ShortCode: "G", Name: "Groups", NoTimetables: false, NoGaps: false, GapWeight: 3},
		{Index: 2, ShortCode: "T", Name: "Teachers", NoTimetables: false, NoGaps: false, GapWeight: 2},
		{Index: 3, ShortCode: "R", Name: "Rooms", NoTimetables: false, NoGaps: true, GapWeight: 1},
		{Index: 4, ShortCode: "S", Name: "Subjects", NoTimetables: true, NoGaps: true, GapWeight: 1},
		{Index: 5, ShortCode: "P", Name: "Students", NoTimetables: false, NoGaps: false, GapWeight: 1},
		{Index: 6, ShortCode: "E", Name: "Equipment", NoTimetables: false, NoGaps: true, GapWeight: 1},
		{Index: 7, ShortCode: "O", Name: "Other", NoTimetables: false, NoGaps: true, GapWeight: 1},
	}
	bookings := []string{"/////", "Free", "Meeting", "Other"}
	return courseCats, resCats, bookings
}

func DefaultBells(count int) []string {
	bells := make([]string, 0, count)
	for i := 0; i < count; i++ {
		h := 8 + i
		if h >= 24 {
			h -= 24
		}
		bells = append(bells, fmt.Sprintf("%02d:00-%02d:45", h, h))
	}
	return bells
}

func DefaultWeeks(count int) []WeekDef {
	weeks := make([]WeekDef, 0, count)
	for i := 1; i <= count; i++ {
		weeks = append(weeks, WeekDef{
			Index:  i,
			Name:   fmt.Sprintf("S%d", i),
			Length: 1,
		})
	}
	return weeks
}

func NewEmptyProject(lang string) *Project {
	var cc, rc []Category
	var bk []string
	var days []string
	if lang == "en" {
		cc, rc, bk = DefaultCategoriesEN()
		days = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	} else {
		lang = "es"
		cc, rc, bk = DefaultCategoriesES()
		days = []string{"Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
	}
	p := &Project{
		ID:            fmt.Sprintf("proj_%d", time.Now().UnixNano()),
		Name:          "Nuevo Horario",
		Institution:   "Universidad",
		Department:    "Departamento Académico",
		Language:      lang,
		RoomsCategory: 3,
		Time: TimeSettings{
			MaxWeeks:   4,
			MaxDays:    5,
			MaxPeriods: 10,
			FirstWeek:  1,
			Days:       days,
			Bells:      DefaultBells(12),
			Weeks:      DefaultWeeks(4),
			StartDate:  time.Now().Format("2006-01-02"),
		},
		CourseCategories:   cc,
		ResourceCategories: rc,
		BookingTypes:       bk,
		Courses:            []Course{},
		Resources:          []Resource{},
		WeekAllocations:    []WeekAllocation{},
		Slots:              []SlotAssignment{},
		Bookings:           []Booking{},
		Guides:             []TeachingGuide{},
		UpdatedAt:          time.Now().Format(time.RFC3339),
	}
	if lang == "en" {
		p.Name = "New Schedule"
		p.Institution = "University"
		p.Department = "Timetabling Office"
	}
	return p
}

// RecomputeDerived updates Allocated, Scheduled, Left, Links on all courses and resources.
// Mimosa rule: if MaxWeeks == 1, every course's week-1 allocation equals its Planned periods automatically.
func (p *Project) RecomputeDerived() {
	if p.Courses == nil {
		p.Courses = []Course{}
	}
	if p.Resources == nil {
		p.Resources = []Resource{}
	}
	if p.WeekAllocations == nil {
		p.WeekAllocations = []WeekAllocation{}
	}
	if p.Slots == nil {
		p.Slots = []SlotAssignment{}
	}
	if p.Bookings == nil {
		p.Bookings = []Booking{}
	}
	if p.Guides == nil {
		p.Guides = []TeachingGuide{}
	}
	if p.Scripts == nil {
		p.Scripts = DefaultStarterScripts()
	}
	if p.Format == "" {
		p.Format = "openmimo/1"
	}
	for i := range p.Guides {
		if p.Guides[i].Groups == nil {
			p.Guides[i].Groups = []GuideGroupAssignment{}
		}
	}
	if p.Time.MaxWeeks < 1 {
		p.Time.MaxWeeks = 1
	}
	if p.Time.MaxWeeks > 255 {
		p.Time.MaxWeeks = 255
	}
	if p.Time.MaxDays < 1 {
		p.Time.MaxDays = 1
	}
	if p.Time.MaxDays > 7 {
		p.Time.MaxDays = 7
	}
	if p.Time.MaxPeriods < 1 {
		p.Time.MaxPeriods = 1
	}
	if p.Time.MaxPeriods > 30 {
		p.Time.MaxPeriods = 30
	}
	// Ensure weeks slice matches MaxWeeks
	for len(p.Time.Weeks) < p.Time.MaxWeeks {
		idx := len(p.Time.Weeks) + 1
		p.Time.Weeks = append(p.Time.Weeks, WeekDef{Index: idx, Name: fmt.Sprintf("S%d", idx), Length: 1})
	}
	if len(p.Time.Weeks) > p.Time.MaxWeeks {
		p.Time.Weeks = p.Time.Weeks[:p.Time.MaxWeeks]
	}
	for len(p.Time.Bells) < p.Time.MaxPeriods {
		i := len(p.Time.Bells)
		h := (8 + i) % 24
		p.Time.Bells = append(p.Time.Bells, fmt.Sprintf("%02d:00-%02d:45", h, h))
	}
	for len(p.Time.Days) < 7 {
		defaults := []string{"Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
		p.Time.Days = append(p.Time.Days, defaults[len(p.Time.Days)])
	}

	weekLen := make(map[int]int, len(p.Time.Weeks))
	for _, w := range p.Time.Weeks {
		l := w.Length
		if l < 1 {
			l = 1
		}
		weekLen[w.Index] = l
	}

	// If single-week mode, ensure week 1 allocation matches Planned for each course
	if p.Time.MaxWeeks == 1 {
		allocByCourse := make(map[string]int, len(p.Courses))
		for _, c := range p.Courses {
			allocByCourse[c.ID] = c.Planned
		}
		newAllocs := make([]WeekAllocation, 0, len(p.Courses))
		for _, c := range p.Courses {
			if c.Planned > 0 {
				newAllocs = append(newAllocs, WeekAllocation{CourseID: c.ID, Week: 1, Periods: c.Planned})
			}
		}
		p.WeekAllocations = newAllocs
	}

	courseAlloc := make(map[string]int)
	for _, wa := range p.WeekAllocations {
		if wa.Week >= 1 && wa.Week <= p.Time.MaxWeeks && wa.Periods > 0 {
			courseAlloc[wa.CourseID] += wa.Periods * weekLen[wa.Week]
		}
	}

	courseSched := make(map[string]int)
	for _, s := range p.Slots {
		if s.Week >= 1 && s.Week <= p.Time.MaxWeeks && s.Day >= 1 && s.Day <= p.Time.MaxDays && s.Period >= 1 && s.Period <= p.Time.MaxPeriods {
			courseSched[s.CourseID] += weekLen[s.Week]
		}
	}

	resLinks := make(map[string]int)
	resAlloc := make(map[string]int)
	resSched := make(map[string]int)
	resSet := make(map[string]bool, len(p.Resources))
	for _, r := range p.Resources {
		resSet[r.ID] = true
	}

	for i := range p.Courses {
		c := &p.Courses[i]
		// Clean dangling resource links
		cleaned := make([]string, 0, len(c.Resources))
		seen := make(map[string]bool)
		for _, rid := range c.Resources {
			if resSet[rid] && !seen[rid] {
				cleaned = append(cleaned, rid)
				seen[rid] = true
			}
		}
		c.Resources = cleaned
		c.Allocated = courseAlloc[c.ID]
		c.Scheduled = courseSched[c.ID]
		c.Left = c.Allocated - c.Scheduled
		c.Links = len(c.Resources)

		for _, rid := range c.Resources {
			resLinks[rid]++
			resAlloc[rid] += c.Allocated
			resSched[rid] += c.Scheduled
		}
	}

	// Also count manual room assignments in resource scheduled
	for _, s := range p.Slots {
		if s.RoomID != "" && resSet[s.RoomID] {
			// Avoid double counting if RoomID is already in course's Resources
			inCourse := false
			for _, c := range p.Courses {
				if c.ID == s.CourseID {
					for _, rid := range c.Resources {
						if rid == s.RoomID {
							inCourse = true
							break
						}
					}
					break
				}
			}
			if !inCourse {
				resSched[s.RoomID] += weekLen[s.Week]
			}
		}
	}

	for i := range p.Resources {
		r := &p.Resources[i]
		r.Allocated = resAlloc[r.ID]
		r.Scheduled = resSched[r.ID]
		r.Links = resLinks[r.ID]
		if r.Planned > 0 {
			r.Left = r.Planned - r.Scheduled
		} else {
			r.Left = r.Allocated - r.Scheduled
		}
	}
}

func (p *Project) CourseByID(id string) *Course {
	for i := range p.Courses {
		if p.Courses[i].ID == id {
			return &p.Courses[i]
		}
	}
	return nil
}

func (p *Project) ResourceByID(id string) *Resource {
	for i := range p.Resources {
		if p.Resources[i].ID == id {
			return &p.Resources[i]
		}
	}
	return nil
}

func (p *Project) ResourceCategory(catIdx int) Category {
	for _, c := range p.ResourceCategories {
		if c.Index == catIdx {
			return c
		}
	}
	return Category{Index: catIdx, GapWeight: 1}
}

func (p *Project) WeekAlloc(courseID string, week int) int {
	for _, wa := range p.WeekAllocations {
		if wa.CourseID == courseID && wa.Week == week {
			return wa.Periods
		}
	}
	return 0
}

func (p *Project) SetWeekAlloc(courseID string, week int, periods int) {
	if periods < 0 {
		periods = 0
	}
	for i := range p.WeekAllocations {
		if p.WeekAllocations[i].CourseID == courseID && p.WeekAllocations[i].Week == week {
			if periods == 0 {
				p.WeekAllocations = append(p.WeekAllocations[:i], p.WeekAllocations[i+1:]...)
			} else {
				p.WeekAllocations[i].Periods = periods
			}
			return
		}
	}
	if periods > 0 {
		p.WeekAllocations = append(p.WeekAllocations, WeekAllocation{
			CourseID: courseID,
			Week:     week,
			Periods:  periods,
		})
	}
}

func (p *Project) Clone() *Project {
	cp := *p
	cp.Time.Days = append([]string(nil), p.Time.Days...)
	cp.Time.Bells = append([]string(nil), p.Time.Bells...)
	cp.Time.Weeks = append([]WeekDef(nil), p.Time.Weeks...)
	cp.CourseCategories = append([]Category(nil), p.CourseCategories...)
	cp.ResourceCategories = append([]Category(nil), p.ResourceCategories...)
	cp.BookingTypes = append([]string(nil), p.BookingTypes...)
	cp.Courses = make([]Course, len(p.Courses))
	for i, c := range p.Courses {
		c.Resources = append([]string(nil), c.Resources...)
		cp.Courses[i] = c
	}
	cp.Resources = append([]Resource(nil), p.Resources...)
	cp.WeekAllocations = append([]WeekAllocation(nil), p.WeekAllocations...)
	cp.Slots = append([]SlotAssignment(nil), p.Slots...)
	cp.Bookings = append([]Booking(nil), p.Bookings...)
	cp.Guides = make([]TeachingGuide, len(p.Guides))
	for i, g := range p.Guides {
		g.Groups = append([]GuideGroupAssignment(nil), g.Groups...)
		cp.Guides[i] = g
	}
	cp.Scripts = append([]UserScript(nil), p.Scripts...)
	return &cp
}

func DefaultStarterScripts() []UserScript {
	now := time.Now().Format(time.RFC3339)
	return []UserScript{
		{
			ID:          "scr_fri_pm",
			Name:        "Bloquear viernes tarde a todos los profesores",
			Description: "Marca como no disponibles los 3 últimos bloques del viernes para todos los docentes.",
			Author:      "OpenMimo",
			UpdatedAt:   now,
			Code: `// Bloquea los 3 últimos bloques del viernes (día 5) a cada profesor (cat. 2)
const maxP = project.time.maxPeriods;
const startP = Math.max(1, maxP - 2);
let added = 0;
for (const r of project.resources.filter(x => x.category === 2)) {
  for (let w = 1; w <= project.time.maxWeeks; w++) {
    for (let h = startP; h <= maxP; h++) {
      const exists = project.bookings.some(b => b.targetId === r.id && b.week === w && b.day === 5 && b.period === h);
      if (!exists) {
        project.bookings.push({
          id: 'b_scr_' + r.id + '_' + w + '_' + h,
          targetType: 'resource', targetId: r.id, scope: 'timetable',
          week: w, day: 5, period: h, kind: 1, label: 'Viernes tarde libre'
        });
        added++;
      }
    }
  }
}
log('Bloqueos añadidos:', added);
return true; // guardar cambios`,
		},
		{
			ID:          "scr_overload",
			Name:        "Informe de docentes con sobrecarga (>18h)",
			Description: "Inspecciona la carga programada de cada profesor y avisa si supera 18 horas.",
			Author:      "OpenMimo",
			UpdatedAt:   now,
			Code: `// Solo lectura: lista profesores con más de 18 bloques programados
const teachers = project.resources.filter(r => r.category === 2);
let flagged = 0;
for (const t of teachers) {
  if (t.scheduled > 18) {
    log('ALERTA:', t.code, t.name, '->', t.scheduled, 'horas programadas');
    flagged++;
  }
}
if (flagged === 0) log('Todos los docentes están por debajo de 18h.');
return false;`,
		},
		{
			ID:          "scr_sync_guides",
			Name:        "Sincronizar profesores en Guías Docentes",
			Description: "Actualiza la tabla de grupos/profesores de cada Guía con sus sesiones contenidas.",
			Author:      "OpenMimo",
			UpdatedAt:   now,
			Code: `// Sincroniza grupos y docentes desde las sesiones vinculadas a cada guía
const resMap = Object.fromEntries(project.resources.map(r => [r.id, r]));
let updated = 0;
for (const g of project.guides) {
  const linked = project.courses.filter(c => c.guideId === g.id);
  for (const c of linked) {
    const grp = c.resources.map(id => resMap[id]).filter(r => r && r.category === 1).map(r => r.name).join(', ') || c.code;
    const prof = c.resources.map(id => resMap[id]).filter(r => r && r.category === 2).map(r => r.name).join(', ') || 'Sin docente';
    if (!g.groups.some(x => x.notes === c.code)) {
      g.groups.push({ group: grp, teacher: prof, shift: 'Paralelo', notes: c.code });
      updated++;
    }
  }
}
log('Entradas sincronizadas en guías:', updated);
return updated > 0;`,
		},
	}
}

func (p *Project) SortCourses(by string, byCategory bool, ascending bool) {
	sort.SliceStable(p.Courses, func(i, j int) bool {
		a, b := p.Courses[i], p.Courses[j]
		if byCategory && a.Category != b.Category {
			if ascending {
				return a.Category < b.Category
			}
			return a.Category > b.Category
		}
		var less bool
		switch strings.ToLower(by) {
		case "name":
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case "planned", "max":
			less = a.Planned < b.Planned
		case "allocated", "sum":
			less = a.Allocated < b.Allocated
		case "scheduled", "done":
			less = a.Scheduled < b.Scheduled
		case "links":
			less = a.Links < b.Links
		case "left":
			less = a.Left < b.Left
		default:
			less = strings.ToLower(a.Code) < strings.ToLower(b.Code)
		}
		if !ascending {
			return !less
		}
		return less
	})
}

func (p *Project) SortResources(by string, byCategory bool, ascending bool) {
	sort.SliceStable(p.Resources, func(i, j int) bool {
		a, b := p.Resources[i], p.Resources[j]
		if byCategory && a.Category != b.Category {
			if ascending {
				return a.Category < b.Category
			}
			return a.Category > b.Category
		}
		var less bool
		switch strings.ToLower(by) {
		case "name":
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case "planned", "max":
			less = a.Planned < b.Planned
		case "allocated", "sum":
			less = a.Allocated < b.Allocated
		case "scheduled", "done":
			less = a.Scheduled < b.Scheduled
		case "links":
			less = a.Links < b.Links
		case "left":
			less = a.Left < b.Left
		default:
			less = strings.ToLower(a.Code) < strings.ToLower(b.Code)
		}
		if !ascending {
			return !less
		}
		return less
	})
}
