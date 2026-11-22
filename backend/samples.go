package main

import (
	"fmt"
	"strings"
	"time"
)

type SampleDescriptor struct {
	Key         string `json:"key"`
	TitleES     string `json:"titleEs"`
	TitleEN     string `json:"titleEn"`
	DescES      string `json:"descEs"`
	DescEN      string `json:"descEn"`
	Badge       string `json:"badge"`
}

func ListBuiltInSamples() []SampleDescriptor {
	return []SampleDescriptor{
		{
			Key:     "design_school",
			TitleES: "Facultad de Artes y Medios (Especialidades y Guías)",
			TitleEN: "Faculty of Arts & Media (Specialties & Guides)",
			DescES:  "Comunicación Visual, Arquitectura Efímera, Estilismo y Prototipado con turnos mañana/tarde, bloqueos y 3 guías docentes listas para Astro.",
			DescEN:  "Visual Communication, Spatial Media, Textiles & Prototyping with morning/afternoon parallel groups, teacher blocks, and 3 Astro-ready guides.",
			Badge:   "Artes · 2 sem",
		},
		{
			Key:     "secondary_es",
			TitleES: "Instituto / Secundaria (Semana Completa)",
			TitleEN: "Secondary School (Full Weekly Timetable)",
			DescES:  "6 grupos, 9 docentes, 7 aulas y 24 materias en una semana de 5 días × 8 bloques con restricciones de profesorado.",
			DescEN:  "6 classes, 9 teachers, 7 rooms, and 24 courses in a compact 5-day × 8-period weekly schedule.",
			Badge:   "Secundaria · 1 sem",
		},
		{
			Key:     "multi_intake",
			TitleES: "Convocatorias Intensivas (4 Semanas)",
			TitleEN: "Multi-Intake Intensive Schedule (4 Weeks)",
			DescES:  "Varios grupos rotando por laboratorios y aulas en bloques intensivos a lo largo de 4 semanas.",
			DescEN:  "Multiple cohorts rotating across labs and classrooms over 4 consecutive weeks.",
			Badge:   "Intensivo · 4 sem",
		},
		{
			Key:     "tiny_tutorial",
			TitleES: "Mini Horario Tutorial (Empezar en 2 min)",
			TitleEN: "Mini Tutorial Schedule (2-min Starter)",
			DescES:  "3 grupos, 3 profesores, 3 aulas y 6 asignaturas parcialmente programadas para practicar paso a paso.",
			DescEN:  "3 groups, 3 teachers, 3 rooms, and 6 courses partially scheduled so you can practice step by step.",
			Badge:   "Tutorial · 1 sem",
		},
	}
}

func BuildSampleProject(key string, lang string) *Project {
	if lang != "en" {
		lang = "es"
	}
	switch key {
	case "design_school":
		return buildDesignSchoolSample(lang)
	case "secondary_es":
		return buildSecondarySample(lang)
	case "multi_intake":
		return buildMultiIntakeSample(lang)
	case "tiny_tutorial":
		return buildTinyTutorialSample(lang)
	default:
		p := NewUniversityDemoProject()
		p.ID = fmt.Sprintf("proj_%d", time.Now().UnixNano())
		p.Language = lang
		return p
	}
}

func buildDesignSchoolSample(lang string) *Project {
	p := NewEmptyProject(lang)
	p.ID = fmt.Sprintf("proj_design_%d", time.Now().UnixNano())
	p.Name = "Horarios 1er Semestre — Facultad de Artes y Medios"
	p.Institution = "Instituto Norte de Artes y Tecnología (Demo)"
	p.Department = "Jefatura de Estudios"
	p.McpNotes = "Priorizar mañanas para grupos A y tardes para grupos B. Evitar huecos de más de 1 bloque para docentes asociados."
	p.Time.MaxWeeks = 2
	p.Time.MaxDays = 5
	p.Time.MaxPeriods = 10
	p.Time.Weeks = []WeekDef{{Index: 1, Name: "S1", Length: 1}, {Index: 2, Name: "S2", Length: 1}}
	p.Time.Bells = []string{
		"09:00-10:00", "10:00-11:00", "11:00-12:00", "12:00-13:00", "13:00-14:00",
		"15:00-16:00", "16:00-17:00", "17:00-18:00", "18:00-19:00", "19:00-20:00",
	}

	p.Resources = []Resource{
		{ID: "r_g1", Code: "2GA", Name: "2º Gráfico (A · Mañana)", Category: 1, Planned: 40},
		{ID: "r_g2", Code: "2GB", Name: "2º Gráfico (B · Tarde)", Category: 1, Planned: 40},
		{ID: "r_g3", Code: "2IA", Name: "2º Interiores (A · Mañana)", Category: 1, Planned: 40},
		{ID: "r_g4", Code: "2MA", Name: "2º Moda (A · Mañana)", Category: 1, Planned: 40},
		{ID: "r_g5", Code: "2PA", Name: "2º Producto (A · Tarde)", Category: 1, Planned: 40},
		{ID: "r_p1", Code: "P-LARA", Name: "Prof. Carmen Lara", Category: 2, Planned: 32, McpNotes: "No puede los lunes a primera hora; prefiere bloques dobles de taller."},
		{ID: "r_p2", Code: "P-RUIZ", Name: "Prof. Javier Ruiz", Category: 2, Planned: 32, McpNotes: "Disponible solo turno de mañana (B1-B5)."},
		{ID: "r_p3", Code: "P-ALBA", Name: "Prof. Marta Alba", Category: 2, Planned: 32, McpNotes: "Imparte Tipografía en mañana y tarde; necesita 1h para comer."},
		{ID: "r_p4", Code: "P-SOLER", Name: "Prof. Diego Soler", Category: 2, Planned: 32, McpNotes: "Reunión de departamento los miércoles B4-B5."},
		{ID: "r_p5", Code: "P-CANO", Name: "Prof. Nuria Cano", Category: 2, Planned: 28, McpNotes: "Prefiere talleres en Aula Taller Moda."},
		{ID: "r_s1", Code: "TALL-G1", Name: "Taller Gráfico 101", Category: 3, Planned: 60},
		{ID: "r_s2", Code: "TALL-I2", Name: "Taller Interiores 202", Category: 3, Planned: 60},
		{ID: "r_s3", Code: "TALL-M1", Name: "Taller Patronaje y Moda", Category: 3, Planned: 60},
		{ID: "r_s4", Code: "LAB-CAD", Name: "Laboratorio 3D / CAD", Category: 3, Planned: 60},
		{ID: "r_s5", Code: "AULA-TEO", Name: "Aula Teórica Magna", Category: 3, Planned: 60},
	}

	p.Guides = []TeachingGuide{
		{
			ID: "g_tipo", Slug: "tipografia-aplicada", Title: "Tipografía Aplicada",
			Degree: "Grado en Diseño", DegreeLevel: "Grado", CourseType: "Obligatoria",
			Character: "Teórico-práctica", Specialty: "Diseño Gráfico", SubjectArea: "Tipografía",
			AcademicYear: "2º", Semester: "1º Semestre", CreditsECTS: 6, TotalHours: 150, InPersonHours: 60,
			Department: "Departamento de Diseño Gráfico", Prerequisites: "Fundamentos de Tipografía",
			Languages: "Español",
			Description: "Sistemas tipográficos editoriales, diseño de retículas complejas y tipografía en pantalla.",
			Groups: []GuideGroupAssignment{
				{Group: "2º Gráfico (A)", Teacher: "Prof. Marta Alba", Shift: "Mañana", Notes: "TALL-G1"},
				{Group: "2º Gráfico (B)", Teacher: "Prof. Carmen Lara", Shift: "Tarde", Notes: "TALL-G1"},
			},
			Competencies: "Proyectar sistemas editoriales y microtipografía con rigor técnico y expresivo.",
			LearningOutcomes: "Diseñar publicaciones multiformato y familias tipográficas experimentales.",
			Contents: "1. Arquitectura editorial.\n2. Retículas modulares.\n3. Tipografía variable y web.",
			TheoryActivities: "Clases magistrales y análisis de casos (24 h).",
			PracticeActivities: "Taller práctico y correcciones colectivas (36 h).",
			EvalAttitudePct: 20, EvalWrittenPct: 10, EvalPracticalPct: 70,
			EvalCriteria: "Entrega de 3 proyectos prácticos y memoria tipográfica.",
			Bibliography: "Bringhurst, R. Los elementos del estilo tipográfico.",
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
		{
			ID: "g_const", Slug: "analisis-elementos-construccion", Title: "Análisis de Elementos de Construcción",
			Degree: "Grado en Diseño", DegreeLevel: "Grado", CourseType: "Obligatoria",
			Character: "Teórico-práctica", Specialty: "Diseño de Interiores", SubjectArea: "Tecnología",
			AcademicYear: "2º", Semester: "1º Semestre", CreditsECTS: 6, TotalHours: 150, InPersonHours: 60,
			Department: "Departamento de Tecnología y Espacios", Languages: "Español",
			Description: "Estudio de sistemas constructivos, materiales y soluciones técnicas en arquitectura interior.",
			Groups: []GuideGroupAssignment{
				{Group: "2º Interiores (A)", Teacher: "Prof. Diego Soler", Shift: "Mañana", Notes: "TALL-I2"},
			},
			Competencies: "Seleccionar soluciones constructivas viables, sostenibles y conformes al CTE.",
			LearningOutcomes: "Elaborar detalles constructivos y mediciones de obra interior.",
			Contents: "1. Particiones y revestimientos.\n2. Instalaciones básicas.\n3. Confort acústico y térmico.",
			EvalAttitudePct: 15, EvalWrittenPct: 35, EvalPracticalPct: 50,
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
		{
			ID: "g_mod3d", Slug: "modelado-prototipado-3d", Title: "Modelado y Prototipado 3D",
			Degree: "Grado en Diseño", DegreeLevel: "Grado", CourseType: "Obligatoria",
			Character: "Práctica", Specialty: "Diseño de Producto", SubjectArea: "Representación Digital",
			AcademicYear: "2º", Semester: "1º Semestre", CreditsECTS: 4, TotalHours: 100, InPersonHours: 48,
			Department: "Departamento de Producto", Languages: "Español / Inglés",
			Description: "Modelado paramétrico NURBS, renderizado fotorrealista e impresión 3D de prototipos.",
			Groups: []GuideGroupAssignment{
				{Group: "2º Producto (A)", Teacher: "Prof. Carmen Lara", Shift: "Tarde", Notes: "LAB-CAD"},
				{Group: "2º Interiores (A)", Teacher: "Prof. Javier Ruiz", Shift: "Mañana", Notes: "LAB-CAD"},
			},
			EvalAttitudePct: 10, EvalWrittenPct: 15, EvalPracticalPct: 75,
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}

	p.Courses = []Course{
		{ID: "c_1", Code: "TIPO-2GA", Name: "Tipografía Aplicada (Mañana)", Category: 1, Planned: 8, Color: "#2563eb", GuideID: "g_tipo", Resources: []string{"r_g1", "r_p3", "r_s1"}},
		{ID: "c_2", Code: "TIPO-2GB", Name: "Tipografía Aplicada (Tarde)", Category: 1, Planned: 8, Color: "#2563eb", GuideID: "g_tipo", Resources: []string{"r_g2", "r_p1", "r_s1"}},
		{ID: "c_3", Code: "EDIT-2GA", Name: "Diseño Editorial (Mañana)", Category: 3, Planned: 8, Color: "#0ea5e9", Resources: []string{"r_g1", "r_p2", "r_s1"}},
		{ID: "c_4", Code: "EDIT-2GB", Name: "Diseño Editorial (Tarde)", Category: 3, Planned: 8, Color: "#0ea5e9", Resources: []string{"r_g2", "r_p3", "r_s1"}},
		{ID: "c_5", Code: "CONST-2IA", Name: "Elementos de Construcción", Category: 1, Planned: 8, Color: "#f59e0b", GuideID: "g_const", Resources: []string{"r_g3", "r_p4", "r_s2"}},
		{ID: "c_6", Code: "CAD-2IA", Name: "Modelado 3D Interiores", Category: 3, Planned: 6, Color: "#8b5cf6", GuideID: "g_mod3d", Resources: []string{"r_g3", "r_p2", "r_s4"}},
		{ID: "c_7", Code: "CAD-2PA", Name: "Modelado y Prototipado 3D", Category: 3, Planned: 8, Color: "#8b5cf6", GuideID: "g_mod3d", Resources: []string{"r_g5", "r_p1", "r_s4"}},
		{ID: "c_8", Code: "PATR-2MA", Name: "Patronaje Experimental", Category: 3, Planned: 8, Color: "#ec4899", Resources: []string{"r_g4", "r_p5", "r_s3"}},
		{ID: "c_9", Code: "HIST-2ALL", Name: "Historia del Diseño (Magistral)", Category: 2, Planned: 4, Color: "#10b981", Resources: []string{"r_g1", "r_g3", "r_g4", "r_p4", "r_s5"}},
		{ID: "c_10", Code: "ERG-2PA", Name: "Ergonomía y Materiales", Category: 1, Planned: 8, Color: "#14b8a6", Resources: []string{"r_g5", "r_p4", "r_s2"}},
	}

	for _, c := range p.Courses {
		perW := c.Planned / 2
		p.WeekAllocations = append(p.WeekAllocations,
			WeekAllocation{CourseID: c.ID, Week: 1, Periods: perW},
			WeekAllocation{CourseID: c.ID, Week: 2, Periods: c.Planned - perW},
		)
	}
	// Teacher unavailability bookings
	p.Bookings = []Booking{
		{ID: "bk_1", TargetType: "resource", TargetID: "r_p1", Scope: "timetable", Week: 1, Day: 1, Period: 1, Kind: 1, Label: "No disponible"},
		{ID: "bk_2", TargetType: "resource", TargetID: "r_p4", Scope: "timetable", Week: 1, Day: 3, Period: 4, Kind: 3, Label: "Reunión Dept."},
		{ID: "bk_3", TargetType: "resource", TargetID: "r_p4", Scope: "timetable", Week: 1, Day: 3, Period: 5, Kind: 3, Label: "Reunión Dept."},
	}
	// Block afternoons for Prof Ruiz (only morning)
	for d := 1; d <= 5; d++ {
		for h := 6; h <= 10; h++ {
			p.Bookings = append(p.Bookings, Booking{
				ID: fmt.Sprintf("bk_r_%d_%d", d, h), TargetType: "resource", TargetID: "r_p2",
				Scope: "timetable", Week: 1, Day: d, Period: h, Kind: 1, Label: "Solo mañanas",
			})
		}
	}

	p.RecomputeDerived()
	p.RunOptimizer(OptimizeOptions{Action: "initial", FromWeek: 1, ToWeek: 2, MinDay: 1, MaxDay: 5, MinPeriod: 1, MaxPeriod: 10, MinBlock: 2, MaxBlock: 2, RandSeed: 38746})
	p.RunOptimizer(OptimizeOptions{Action: "optimise", FromWeek: 1, ToWeek: 1, MinDay: 1, MaxDay: 5, MinPeriod: 1, MaxPeriod: 10, RandSeed: 38746})
	p.RecomputeDerived()
	return p
}

func buildSecondarySample(lang string) *Project {
	p := NewEmptyProject(lang)
	p.ID = fmt.Sprintf("proj_sec_%d", time.Now().UnixNano())
	p.Name = "Instituto Secundaria — Semana Tipo"
	p.Institution = "IES Modelo (Demo)"
	p.Department = "Jefatura de Estudios"
	p.Time.MaxWeeks = 1
	p.Time.MaxDays = 5
	p.Time.MaxPeriods = 8
	p.Time.Weeks = []WeekDef{{Index: 1, Name: "Sem", Length: 1}}
	p.Time.Bells = DefaultBells(8)

	groups := []string{"1ESO-A", "1ESO-B", "2ESO-A", "2ESO-B", "1BACH-A", "2BACH-A"}
	teachers := []struct{ code, name, note string }{
		{"PR-MAT1", "Ana Morales (Matemáticas)", "Prefiere primeras horas."},
		{"PR-MAT2", "Luis Herrera (Matemáticas)", ""},
		{"PR-LEN1", "Clara Gómez (Lengua)", "No disponible viernes B7-B8."},
		{"PR-ING1", "David Brooks (Inglés)", ""},
		{"PR-HIS1", "Sofía Navarro (Historia)", ""},
		{"PR-BIO1", "Pedro Castillo (Biología)", "Usa Laboratorio Ciencias."},
		{"PR-FIS1", "Irene Molina (Física y Química)", ""},
		{"PR-EF1", "Raúl Ortega (Educación Física)", "Usa Gimnasio."},
	}
	rooms := []string{"A-01", "A-02", "A-03", "A-04", "A-05", "LAB-C", "GIM"}

	for i, g := range groups {
		p.Resources = append(p.Resources, Resource{ID: fmt.Sprintf("rg_%d", i+1), Code: g, Name: "Grupo " + g, Category: 1, Planned: 30})
	}
	for i, tr := range teachers {
		p.Resources = append(p.Resources, Resource{ID: fmt.Sprintf("rp_%d", i+1), Code: tr.code, Name: tr.name, Category: 2, Planned: 20, McpNotes: tr.note})
	}
	for i, rm := range rooms {
		p.Resources = append(p.Resources, Resource{ID: fmt.Sprintf("rs_%d", i+1), Code: rm, Name: "Aula " + rm, Category: 3, Planned: 40})
	}

	subjects := []struct {
		code, name, color string
		profIdx, roomIdx  int
		hrs               int
	}{
		{"MAT", "Matemáticas", "#f59e0b", 1, 1, 4},
		{"LEN", "Lengua y Literatura", "#3b82f6", 3, 2, 4},
		{"ING", "Inglés", "#8b5cf6", 4, 3, 3},
		{"HIS", "Historia", "#10b981", 5, 4, 3},
	}
	cid := 0
	for gi, g := range groups {
		for _, sub := range subjects {
			cid++
			pi := sub.profIdx
			if sub.code == "MAT" && gi >= 3 {
				pi = 2
			}
			ri := (gi % 5) + 1
			p.Courses = append(p.Courses, Course{
				ID:        fmt.Sprintf("cs_%d", cid),
				Code:      sub.code + "-" + strings.ReplaceAll(g, "ESO", ""),
				Name:      sub.name + " (" + g + ")",
				Category:  1,
				Planned:   sub.hrs,
				Color:     sub.color,
				Resources: []string{fmt.Sprintf("rg_%d", gi+1), fmt.Sprintf("rp_%d", pi), fmt.Sprintf("rs_%d", ri)},
			})
		}
	}
	p.Bookings = []Booking{
		{ID: "bsec_1", TargetType: "resource", TargetID: "rp_3", Scope: "timetable", Week: 1, Day: 5, Period: 7, Kind: 1, Label: "No disponible"},
		{ID: "bsec_2", TargetType: "resource", TargetID: "rp_3", Scope: "timetable", Week: 1, Day: 5, Period: 8, Kind: 1, Label: "No disponible"},
	}
	p.RecomputeDerived()
	p.RunOptimizer(OptimizeOptions{Action: "initial", FromWeek: 1, ToWeek: 1, MinDay: 1, MaxDay: 5, MinPeriod: 1, MaxPeriod: 8, MinBlock: 1, MaxBlock: 2, RandSeed: 38746})
	p.RunOptimizer(OptimizeOptions{Action: "optimise", FromWeek: 1, ToWeek: 1, MinDay: 1, MaxDay: 5, MinPeriod: 1, MaxPeriod: 8, RandSeed: 38746})
	p.RecomputeDerived()
	return p
}

func buildMultiIntakeSample(lang string) *Project {
	p := NewEmptyProject(lang)
	p.ID = fmt.Sprintf("proj_intake_%d", time.Now().UnixNano())
	p.Name = "Convocatorias Intensivas (4 Semanas)"
	p.Institution = "Centro de Estudios Superiores (Demo)"
	p.Department = "Formación Continua"
	p.Time.MaxWeeks = 4
	p.Time.MaxDays = 5
	p.Time.MaxPeriods = 8
	p.Time.Weeks = DefaultWeeks(4)
	p.Time.Bells = DefaultBells(8)

	p.Resources = []Resource{
		{ID: "ri_g1", Code: "COH-1", Name: "Convocatoria 1 (A)", Category: 1, Planned: 60},
		{ID: "ri_g2", Code: "COH-2", Name: "Convocatoria 2 (B)", Category: 1, Planned: 60},
		{ID: "ri_g3", Code: "COH-3", Name: "Convocatoria 3 (C)", Category: 1, Planned: 60},
		{ID: "ri_p1", Code: "DOC-01", Name: "Dr. Sergio Vidal", Category: 2, Planned: 48},
		{ID: "ri_p2", Code: "DOC-02", Name: "Dra. Paula Romero", Category: 2, Planned: 48},
		{ID: "ri_p3", Code: "DOC-03", Name: "Ing. Marcos Gil", Category: 2, Planned: 48},
		{ID: "ri_s1", Code: "LAB-A", Name: "Laboratorio Multimedia A", Category: 3, Planned: 80},
		{ID: "ri_s2", Code: "LAB-B", Name: "Laboratorio Multimedia B", Category: 3, Planned: 80},
	}
	p.Courses = []Course{
		{ID: "ci_1", Code: "UX-C1", Name: "Diseño UX Intensivo", Category: 1, Planned: 16, Color: "#3b82f6", Resources: []string{"ri_g1", "ri_p1", "ri_s1"}},
		{ID: "ci_2", Code: "DAT-C1", Name: "Bases de Datos SQL", Category: 3, Planned: 16, Color: "#f59e0b", Resources: []string{"ri_g1", "ri_p2", "ri_s2"}},
		{ID: "ci_3", Code: "WEB-C2", Name: "Desarrollo Web Astro", Category: 3, Planned: 16, Color: "#10b981", Resources: []string{"ri_g2", "ri_p3", "ri_s1"}},
		{ID: "ci_4", Code: "UX-C2", Name: "Diseño UX Intensivo", Category: 1, Planned: 16, Color: "#3b82f6", Resources: []string{"ri_g2", "ri_p1", "ri_s2"}},
		{ID: "ci_5", Code: "IA-C3", Name: "Agentes e IA Aplicada", Category: 3, Planned: 16, Color: "#8b5cf6", Resources: []string{"ri_g3", "ri_p2", "ri_s1"}},
		{ID: "ci_6", Code: "WEB-C3", Name: "Desarrollo Web Astro", Category: 3, Planned: 16, Color: "#10b981", Resources: []string{"ri_g3", "ri_p3", "ri_s2"}},
	}
	// Staggered week allocations (weeks 1-2 vs weeks 3-4)
	allocs := []WeekAllocation{
		{CourseID: "ci_1", Week: 1, Periods: 8}, {CourseID: "ci_1", Week: 2, Periods: 8},
		{CourseID: "ci_2", Week: 3, Periods: 8}, {CourseID: "ci_2", Week: 4, Periods: 8},
		{CourseID: "ci_3", Week: 1, Periods: 8}, {CourseID: "ci_3", Week: 2, Periods: 8},
		{CourseID: "ci_4", Week: 3, Periods: 8}, {CourseID: "ci_4", Week: 4, Periods: 8},
		{CourseID: "ci_5", Week: 2, Periods: 8}, {CourseID: "ci_5", Week: 3, Periods: 8},
		{CourseID: "ci_6", Week: 1, Periods: 8}, {CourseID: "ci_6", Week: 4, Periods: 8},
	}
	p.WeekAllocations = allocs
	p.RecomputeDerived()
	p.RunOptimizer(OptimizeOptions{Action: "initial", FromWeek: 1, ToWeek: 4, MinDay: 1, MaxDay: 5, MinPeriod: 1, MaxPeriod: 8, MinBlock: 2, MaxBlock: 2, RandSeed: 38746})
	p.RecomputeDerived()
	return p
}

func buildTinyTutorialSample(lang string) *Project {
	p := NewEmptyProject(lang)
	p.ID = fmt.Sprintf("proj_tiny_%d", time.Now().UnixNano())
	p.Name = "Tutorial Rápido (3 Grupos)"
	p.Institution = "Academia Tutorial"
	p.Department = "Guía de Inicio"
	p.Time.MaxWeeks = 1
	p.Time.MaxDays = 5
	p.Time.MaxPeriods = 6
	p.Time.Weeks = []WeekDef{{Index: 1, Name: "S1", Length: 1}}
	p.Time.Bells = DefaultBells(6)

	p.Resources = []Resource{
		{ID: "rt_g1", Code: "GRP-A", Name: "Grupo A", Category: 1, Planned: 12},
		{ID: "rt_g2", Code: "GRP-B", Name: "Grupo B", Category: 1, Planned: 12},
		{ID: "rt_g3", Code: "GRP-C", Name: "Grupo C", Category: 1, Planned: 12},
		{ID: "rt_p1", Code: "P-ANA", Name: "Prof. Ana Pérez", Category: 2, Planned: 12, McpNotes: "No puede miércoles a última hora."},
		{ID: "rt_p2", Code: "P-LUIS", Name: "Prof. Luis Martín", Category: 2, Planned: 12},
		{ID: "rt_p3", Code: "P-EVA", Name: "Prof. Eva Sanz", Category: 2, Planned: 12},
		{ID: "rt_s1", Code: "AULA-1", Name: "Aula 1", Category: 3, Planned: 24},
		{ID: "rt_s2", Code: "AULA-2", Name: "Aula 2", Category: 3, Planned: 24},
		{ID: "rt_s3", Code: "LAB-1", Name: "Laboratorio 1", Category: 3, Planned: 24},
	}
	p.Courses = []Course{
		{ID: "ct_1", Code: "MAT-A", Name: "Matemáticas A", Category: 1, Planned: 4, Color: "#f59e0b", Resources: []string{"rt_g1", "rt_p1", "rt_s1"}},
		{ID: "ct_2", Code: "MAT-B", Name: "Matemáticas B", Category: 1, Planned: 4, Color: "#f59e0b", Resources: []string{"rt_g2", "rt_p1", "rt_s2"}},
		{ID: "ct_3", Code: "LAB-A", Name: "Práctica Lab A", Category: 3, Planned: 4, Color: "#3b82f6", Resources: []string{"rt_g1", "rt_p2", "rt_s3"}},
		{ID: "ct_4", Code: "LAB-C", Name: "Práctica Lab C", Category: 3, Planned: 4, Color: "#3b82f6", Resources: []string{"rt_g3", "rt_p2", "rt_s3"}},
		{ID: "ct_5", Code: "DIS-B", Name: "Diseño B", Category: 1, Planned: 4, Color: "#10b981", Resources: []string{"rt_g2", "rt_p3", "rt_s1"}},
		{ID: "ct_6", Code: "DIS-C", Name: "Diseño C", Category: 1, Planned: 4, Color: "#10b981", Resources: []string{"rt_g3", "rt_p3", "rt_s2"}},
	}
	p.Bookings = []Booking{
		{ID: "bt_1", TargetType: "resource", TargetID: "rt_p1", Scope: "timetable", Week: 1, Day: 3, Period: 6, Kind: 1, Label: "No disponible"},
	}
	// Pre-schedule half so user can practice placing the rest or clicking Optimise!
	p.Slots = []SlotAssignment{
		{ID: "st_1", CourseID: "ct_1", Week: 1, Day: 1, Period: 1},
		{ID: "st_2", CourseID: "ct_1", Week: 1, Day: 1, Period: 2},
		{ID: "st_3", CourseID: "ct_3", Week: 1, Day: 2, Period: 1},
		{ID: "st_4", CourseID: "ct_3", Week: 1, Day: 2, Period: 2},
	}
	p.RecomputeDerived()
	return p
}
