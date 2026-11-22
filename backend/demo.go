package main

import (
	"fmt"
	"time"
)

// NewUniversityDemoProject creates a realistic, original Spanish university timetable dataset
// so university staff can immediately explore all 4 views (Resources, Events, Weeks, Timetables),
// conflict checking, bookings, manual rooms, and gap optimization on first launch.
func NewUniversityDemoProject() *Project {
	p := NewEmptyProject("es")
	p.Name = "Semestre 2026-I — Ingeniería"
	p.Institution = "Universidad Demo — Facultad de Ingeniería"
	p.Department = "Coordinación de Horarios Académicos"
	p.Comment = "Horario demostrativo con 4 grupos, 8 docentes, 6 aulas/labs y 16 asignaturas. Sin límites de versión gratuita."
	p.Time.MaxWeeks = 4
	p.Time.MaxDays = 5
	p.Time.MaxPeriods = 10
	p.Time.Weeks = []WeekDef{
		{Index: 1, Name: "Sem 1", Length: 1},
		{Index: 2, Name: "Sem 2", Length: 1},
		{Index: 3, Name: "Sem 3", Length: 1},
		{Index: 4, Name: "Sem 4", Length: 1},
	}
	p.Time.Bells = []string{
		"08:00-08:50", "09:00-09:50", "10:00-10:50", "11:00-11:50", "12:00-12:50",
		"13:00-13:50", "14:00-14:50", "15:00-15:50", "16:00-16:50", "17:00-17:50",
	}
	p.Time.StartDate = "2026-03-02"

	// Resources: Groups (1), Teachers (2), Rooms (3), Subjects (4)
	p.Resources = []Resource{
		// Groups / Clases (cat 1)
		{ID: "r_g1", Code: "ING-1A", Name: "Ingeniería 1° Año — Sección A", Category: 1, Planned: 80},
		{ID: "r_g2", Code: "ING-1B", Name: "Ingeniería 1° Año — Sección B", Category: 1, Planned: 80},
		{ID: "r_g3", Code: "ING-2A", Name: "Ingeniería 2° Año — Sección A", Category: 1, Planned: 80},
		{ID: "r_g4", Code: "ING-3A", Name: "Ingeniería 3° Año — Sistemas", Category: 1, Planned: 72},
		// Teachers / Profesores (cat 2)
		{ID: "r_p1", Code: "PR-TORRES", Name: "Dra. Elena Torres", Category: 2, Planned: 64},
		{ID: "r_p2", Code: "PR-MENDEZ", Name: "Dr. Carlos Méndez", Category: 2, Planned: 64},
		{ID: "r_p3", Code: "PR-SILVA", Name: "Ing. Sofía Silva", Category: 2, Planned: 64},
		{ID: "r_p4", Code: "PR-ROJAS", Name: "Dr. Mateo Rojas", Category: 2, Planned: 64},
		{ID: "r_p5", Code: "PR-VEGA", Name: "MSc. Lucía Vega", Category: 2, Planned: 56},
		{ID: "r_p6", Code: "PR-DIAZ", Name: "Ing. Andrés Díaz", Category: 2, Planned: 56},
		// Rooms / Salas (cat 3)
		{ID: "r_s1", Code: "AULA-101", Name: "Aula Magna 101 (Cap. 60)", Category: 3, Planned: 120},
		{ID: "r_s2", Code: "AULA-102", Name: "Aula Teórica 102 (Cap. 45)", Category: 3, Planned: 120},
		{ID: "r_s3", Code: "AULA-204", Name: "Aula Seminario 204 (Cap. 35)", Category: 3, Planned: 120},
		{ID: "r_s4", Code: "LAB-COMP1", Name: "Laboratorio Cómputo 1 (30 PCs)", Category: 3, Planned: 100},
		{ID: "r_s5", Code: "LAB-FIS", Name: "Laboratorio Física Aplicada", Category: 3, Planned: 100},
		{ID: "r_s6", Code: "LAB-RED", Name: "Laboratorio Redes y Sistemas", Category: 3, Planned: 100},
		// Subjects / Materias (cat 4 - NoTimetables = true)
		{ID: "r_m1", Code: "MAT", Name: "Matemáticas", Category: 4, Planned: 0},
		{ID: "r_m2", Code: "PROG", Name: "Programación y Software", Category: 4, Planned: 0},
		{ID: "r_m3", Code: "FIS", Name: "Física", Category: 4, Planned: 0},
		{ID: "r_m4", Code: "SIS", Name: "Sistemas y Redes", Category: 4, Planned: 0},
	}

	p.Guides = []TeachingGuide{
		{
			ID: "g_calc1", Slug: "calculo-1", Title: "Cálculo I",
			Degree: "Grado en Ingeniería", DegreeLevel: "Grado", CourseType: "Formación básica",
			Character: "Teórico-práctica", Specialty: "Común", SubjectArea: "Matemáticas",
			AcademicYear: "1º", Semester: "1º Semestre", CreditsECTS: 6, TotalHours: 150, InPersonHours: 60,
			Department: "Departamento de Matemáticas", Prerequisites: "Matemáticas de Bachillerato",
			Languages: "Español",
			Description: "Fundamentos de cálculo diferencial e integral de una variable aplicados a problemas de ingeniería.",
			Groups: []GuideGroupAssignment{
				{Group: "ING-1A (Sección A)", Teacher: "Dra. Elena Torres", Shift: "Mañana", Notes: "CALC1-A"},
				{Group: "ING-1B (Sección B)", Teacher: "Dra. Elena Torres", Shift: "Tarde", Notes: "CALC1-B"},
			},
			Competencies: "Capacidad para resolver problemas matemáticos aplicados a la ingeniería y modelado cuantitativo.",
			LearningOutcomes: "Dominar límites, derivación, optimización e integración numérica y analítica.",
			Contents: "1. Números reales y sucesiones.\n2. Límites y continuidad.\n3. Cálculo diferencial y optimización.\n4. Integral definida y aplicaciones.",
			TheoryActivities: "Sesiones magistrales con resolución guiada de problemas (36 h).",
			PracticeActivities: "Talleres de cálculo simbólico y resolución en grupo (24 h).",
			OtherActivities: "Seminario de modelado matemático aplicado.",
			StudentWorkHours: "Estudio autónomo y preparación de entregas prácticas (90 h).",
			EvalAttitudePct: 15, EvalWrittenPct: 55, EvalPracticalPct: 30,
			EvalCriteria: "Evaluación continua mediante dos parciales escritos, entregas prácticas y participación.",
			Bibliography: "Stewart, J. Cálculo de una variable. Cengage Learning.",
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
		{
			ID: "g_prog1", Slug: "programacion-1", Title: "Programación I",
			Degree: "Grado en Ingeniería", DegreeLevel: "Grado", CourseType: "Formación básica",
			Character: "Práctica", Specialty: "Sistemas y Software", SubjectArea: "Programación",
			AcademicYear: "1º", Semester: "1º Semestre", CreditsECTS: 6, TotalHours: 150, InPersonHours: 64,
			Department: "Departamento de Informática", Prerequisites: "Ninguno",
			Languages: "Español / Inglés",
			Description: "Diseño de algoritmos, programación estructurada y modular en laboratorio.",
			Groups: []GuideGroupAssignment{
				{Group: "ING-1A (Sección A)", Teacher: "Ing. Sofía Silva", Shift: "Mañana", Notes: "PROG1-A · LAB-COMP1"},
				{Group: "ING-1B (Sección B)", Teacher: "Ing. Sofía Silva", Shift: "Tarde", Notes: "PROG1-B · LAB-COMP1"},
			},
			Competencies: "Diseñar, implementar y depurar soluciones algorítmicas limpias y verificables.",
			LearningOutcomes: "Construir programas modulares, manejar estructuras de datos básicas y pruebas unitarias.",
			Contents: "1. Pensamiento algorítmico.\n2. Tipos, control de flujo y funciones.\n3. Estructuras compuestas y ficheros.\n4. Pruebas y proyecto final.",
			TheoryActivities: "Clases expositivas con demostraciones de código en vivo (24 h).",
			PracticeActivities: "Prácticas semanales en laboratorio de cómputo (40 h).",
			OtherActivities: "Hackathon de cierre de semestre.",
			StudentWorkHours: "Desarrollo de prácticas y proyecto final autónomo (86 h).",
			EvalAttitudePct: 10, EvalWrittenPct: 25, EvalPracticalPct: 65,
			EvalCriteria: "Evaluación basada en entregas de laboratorio, prueba práctica y proyecto integrador.",
			Bibliography: "Sedgewick & Wayne. Computer Science: An Interdisciplinary Approach.",
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}

	p.Courses = []Course{
		{ID: "c_1", Code: "CALC1-A", Name: "Cálculo I (Sec. A)", Category: 1, Planned: 16, Color: "#f59e0b", GuideID: "g_calc1", Resources: []string{"r_g1", "r_p1", "r_s1", "r_m1"}},
		{ID: "c_2", Code: "CALC1-B", Name: "Cálculo I (Sec. B)", Category: 1, Planned: 16, Color: "#f59e0b", GuideID: "g_calc1", Resources: []string{"r_g2", "r_p1", "r_s2", "r_m1"}},
		{ID: "c_3", Code: "PROG1-A", Name: "Programación I (Sec. A)", Category: 3, Planned: 16, Color: "#3b82f6", GuideID: "g_prog1", Resources: []string{"r_g1", "r_p3", "r_s4", "r_m2"}},
		{ID: "c_4", Code: "PROG1-B", Name: "Programación I (Sec. B)", Category: 3, Planned: 16, Color: "#3b82f6", GuideID: "g_prog1", Resources: []string{"r_g2", "r_p3", "r_s4", "r_m2"}},
		{ID: "c_5", Code: "FIS1-A", Name: "Física Mecánica (Sec. A)", Category: 3, Planned: 12, Color: "#10b981", Resources: []string{"r_g1", "r_p2", "r_s5", "r_m3"}},
		{ID: "c_6", Code: "FIS1-B", Name: "Física Mecánica (Sec. B)", Category: 3, Planned: 12, Color: "#10b981", Resources: []string{"r_g2", "r_p2", "r_s5", "r_m3"}},
		{ID: "c_7", Code: "ALG-1AB", Name: "Álgebra Lineal (Magistral)", Category: 2, Planned: 8, Color: "#8b5cf6", Resources: []string{"r_g1", "r_g2", "r_p4", "r_s1", "r_m1"}},
		{ID: "c_8", Code: "ESTDAT-2A", Name: "Estructuras de Datos", Category: 3, Planned: 16, Color: "#0ea5e9", Resources: []string{"r_g3", "r_p5", "r_s4", "r_m2"}},
		{ID: "c_9", Code: "CALC2-2A", Name: "Cálculo Vectorial", Category: 1, Planned: 16, Color: "#d97706", Resources: []string{"r_g3", "r_p4", "r_s2", "r_m1"}},
		{ID: "c_10", Code: "ELECT-2A", Name: "Electromagnetismo", Category: 3, Planned: 12, Color: "#059669", Resources: []string{"r_g3", "r_p2", "r_s5", "r_m3"}},
		{ID: "c_11", Code: "ARQ-2A", Name: "Arquitectura de Computadores", Category: 1, Planned: 12, Color: "#6366f1", Resources: []string{"r_g3", "r_p6", "r_s3", "r_m4"}},
		{ID: "c_12", Code: "BDD-3A", Name: "Bases de Datos Avanzadas", Category: 3, Planned: 16, Color: "#0284c7", Resources: []string{"r_g4", "r_p5", "r_s4", "r_m2"}},
		{ID: "c_13", Code: "REDES-3A", Name: "Redes de Computadores", Category: 3, Planned: 16, Color: "#7c3aed", Resources: []string{"r_g4", "r_p6", "r_s6", "r_m4"}},
		{ID: "c_14", Code: "SO-3A", Name: "Sistemas Operativos", Category: 1, Planned: 12, Color: "#ec4899", Resources: []string{"r_g4", "r_p3", "r_s3", "r_m4"}},
		{ID: "c_15", Code: "IA-3A", Name: "Inteligencia Artificial", Category: 5, Planned: 12, Color: "#14b8a6", Resources: []string{"r_g4", "r_p1", "r_s6", "r_m2"}},
		{ID: "c_16", Code: "ETICA-3A", Name: "Ética y Gestión de Proyectos", Category: 4, Planned: 8, Color: "#64748b", Resources: []string{"r_g4", "r_p4", "r_s3"}},
	}

	// Allocate weekly periods evenly across the 4 weeks
	for _, c := range p.Courses {
		perWeek := c.Planned / 4
		for w := 1; w <= 4; w++ {
			p.WeekAllocations = append(p.WeekAllocations, WeekAllocation{
				CourseID: c.ID,
				Week:     w,
				Periods:  perWeek,
			})
		}
	}

	// Add a few realistic Bookings (e.g. faculty meeting Friday P9-P10 for Prof Torres, Lab maintenance Wed P6)
	p.Bookings = []Booking{
		{ID: "b_1", TargetType: "resource", TargetID: "r_p1", Scope: "timetable", Week: 1, Day: 5, Period: 9, Kind: 3, Label: "Consejo de Facultad"},
		{ID: "b_2", TargetType: "resource", TargetID: "r_p1", Scope: "timetable", Week: 1, Day: 5, Period: 10, Kind: 3, Label: "Consejo de Facultad"},
		{ID: "b_3", TargetType: "resource", TargetID: "r_s4", Scope: "timetable", Week: 1, Day: 3, Period: 6, Kind: 1, Label: "Mantenimiento Lab"},
	}

	// Schedule Week 1 & Week 2 automatically with clean blocks, leaving a couple of periods in Week 1 for live demo interaction
	p.RecomputeDerived()
	p.RunOptimizer(OptimizeOptions{
		Action:    "initial",
		FromWeek:  1,
		ToWeek:    2,
		MinDay:    1,
		MaxDay:    5,
		MinPeriod: 1,
		MaxPeriod: 8,
		MinBlock:  2,
		MaxBlock:  2,
		RandSeed:  38746,
	})
	// Optimize gaps on Week 1 so the timetable looks crisp & professional on launch
	p.RunOptimizer(OptimizeOptions{
		Action:    "optimise",
		FromWeek:  1,
		ToWeek:    1,
		MinDay:    1,
		MaxDay:    5,
		MinPeriod: 1,
		MaxPeriod: 8,
		RandSeed:  38746,
	})
	// Unschedule 1 slot of CALC1-A in Week 1 so users immediately see LEFT > 0 and can click to place it!
	for i := len(p.Slots) - 1; i >= 0; i-- {
		if p.Slots[i].CourseID == "c_1" && p.Slots[i].Week == 1 {
			p.Slots = append(p.Slots[:i], p.Slots[i+1:]...)
			break
		}
	}

	p.RecomputeDerived()
	p.UpdatedAt = time.Now().Format(time.RFC3339)
	_ = fmt.Sprintf("")
	return p
}
