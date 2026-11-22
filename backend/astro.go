package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
)

var nonSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

func makeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	repl := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n", "ü", "u")
	s = repl.Replace(s)
	s = nonSlugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "asignatura"
	}
	return s
}

// GenerateAstroZip packages all TeachingGuides (and synced parallel groups/teachers from Courses)
// into a ready-to-run Astro 5 static website project (.zip).
func GenerateAstroZip(p *Project) ([]byte, error) {
	p.RecomputeDerived()

	// Build enriched guides list (including auto-linked parallel courses/teachers)
	guides := make([]TeachingGuide, 0, len(p.Guides))
	seenSlugs := make(map[string]int)
	resMap := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}

	for _, g := range p.Guides {
		cp := g
		baseSlug := makeSlug(cp.Slug)
		if cp.Slug == "" {
			baseSlug = makeSlug(cp.Title)
		}
		if cnt := seenSlugs[baseSlug]; cnt > 0 {
			cp.Slug = fmt.Sprintf("%s-%d", baseSlug, cnt+1)
		} else {
			cp.Slug = baseSlug
		}
		seenSlugs[baseSlug]++

		// Also merge any linked courses into Groups if not already present
		existingKeys := make(map[string]bool)
		for _, gr := range cp.Groups {
			existingKeys[strings.ToLower(gr.Group+"|"+gr.Teacher)] = true
		}
		for _, c := range p.Courses {
			if c.GuideID != cp.ID {
				continue
			}
			var grpNames, profNames []string
			for _, rid := range c.Resources {
				if r, ok := resMap[rid]; ok {
					if r.Category == 1 {
						grpNames = append(grpNames, r.Name)
					} else if r.Category == 2 {
						profNames = append(profNames, r.Name)
					}
				}
			}
			gName := strings.Join(grpNames, ", ")
			if gName == "" {
				gName = c.Code
			}
			pName := strings.Join(profNames, ", ")
			if pName == "" {
				pName = "Por asignar"
			}
			k := strings.ToLower(gName + "|" + pName)
			if !existingKeys[k] {
				cp.Groups = append(cp.Groups, GuideGroupAssignment{
					Group:   gName,
					Teacher: pName,
					Shift:   "Paralelo",
					Notes:   c.Code,
				})
				existingKeys[k] = true
			}
		}
		guides = append(guides, cp)
	}

	// If no explicit guides yet, generate starter entries from courses so the Astro export is never empty
	if len(guides) == 0 {
		for _, c := range p.Courses {
			var grpNames, profNames []string
			for _, rid := range c.Resources {
				if r, ok := resMap[rid]; ok {
					if r.Category == 1 {
						grpNames = append(grpNames, r.Name)
					} else if r.Category == 2 {
						profNames = append(profNames, r.Name)
					}
				}
			}
			guides = append(guides, TeachingGuide{
				ID:            "auto_" + c.ID,
				Slug:          makeSlug(c.Code + "-" + c.Name),
				Title:         c.Name,
				Degree:        p.Institution,
				DegreeLevel:   "Grado",
				CourseType:    "Obligatoria",
				Character:     "Teórico-práctica",
				Department:    p.Department,
				CreditsECTS:   6,
				TotalHours:    150,
				InPersonHours: c.Planned,
				Languages:     "Español",
				Groups: []GuideGroupAssignment{
					{Group: strings.Join(grpNames, ", "), Teacher: strings.Join(profNames, ", "), Shift: "Regular"},
				},
				EvalAttitudePct:  20,
				EvalWrittenPct:   30,
				EvalPracticalPct: 50,
			})
		}
	}

	guidesJSON, _ := json.MarshalIndent(guides, "", "  ")
	siteMetaJSON, _ := json.MarshalIndent(map[string]any{
		"name":        p.Name,
		"institution": p.Institution,
		"department":  p.Department,
		"language":    p.Language,
	}, "", "  ")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	addFile := func(name, content string) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(content))
		return err
	}

	_ = addFile("package.json", `{
  "name": "openmimo-guias-docentes",
  "type": "module",
  "version": "1.0.0",
  "scripts": {
    "dev": "astro dev",
    "build": "astro build",
    "preview": "astro preview"
  },
  "dependencies": {
    "astro": "^5.5.0"
  }
}
`)

	_ = addFile("astro.config.mjs", `import { defineConfig } from 'astro/config';

export default defineConfig({
  output: 'static',
});
`)

	_ = addFile("README.md", fmt.Sprintf(`# Guías Docentes — %s

Web estática generada con **openmimo** en **Astro 5**.

## Compilar y previsualizar

`+"```bash"+`
bun install   # o npm install
bun run build # genera la web estática en dist/
bun run preview
`+"```"+`
`, p.Institution))

	_ = addFile("src/data/guides.json", string(guidesJSON))
	_ = addFile("src/data/site.json", string(siteMetaJSON))

	_ = addFile("src/layouts/Layout.astro", `---
import site from '../data/site.json';
interface Props { title: string; }
const { title } = Astro.props;
---
<!doctype html>
<html lang={site.language || 'es'}>
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>{title} | {site.institution}</title>
    <style is:global>
      :root { --bg:#f8fafc; --card:#fff; --ink:#0f172a; --muted:#475569; --line:#e2e8f0; --accent:#2563eb; }
      * { box-sizing:border-box; }
      body { margin:0; font-family:-apple-system,BlinkMacSystemFont,'Inter',Segoe UI,sans-serif; background:var(--bg); color:var(--ink); line-height:1.55; }
      header.site-head { background:#fff; border-bottom:1px solid var(--line); padding:18px 24px; }
      header.site-head .inner { max-width:1120px; margin:0 auto; display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:12px; }
      header.site-head a { color:var(--ink); text-decoration:none; font-weight:800; font-size:18px; }
      header.site-head .sub { font-size:13px; color:var(--muted); }
      main { max-width:1120px; margin:24px auto; padding:0 20px 48px; }
      .card { background:var(--card); border:1px solid var(--line); border-radius:10px; padding:18px 20px; margin-bottom:16px; }
      .pill { display:inline-block; padding:2px 9px; border-radius:999px; font-size:12px; font-weight:600; background:#eff6ff; color:#1d4ed8; border:1px solid #bfdbfe; }
      table { width:100%; border-collapse:collapse; font-size:14px; }
      th, td { border:1px solid var(--line); padding:8px 10px; text-align:left; vertical-align:top; }
      th { background:#f1f5f9; font-weight:600; }
    </style>
  </head>
  <body>
    <header class="site-head">
      <div class="inner">
        <div>
          <a href="/">Guías Docentes · {site.institution}</a>
          <div class="sub">{site.department} · {site.name}</div>
        </div>
        <a href="/" style="font-size:13px; color:var(--accent);">← Todas las guías</a>
      </div>
    </header>
    <main>
      <slot />
    </main>
  </body>
</html>
`)

	_ = addFile("src/pages/index.astro", `---
import Layout from '../layouts/Layout.astro';
import guides from '../data/guides.json';

const departments = [...new Set(guides.map((g) => g.department).filter(Boolean))].sort();
const types = [...new Set(guides.map((g) => g.courseType).filter(Boolean))].sort();
const specialties = [...new Set(guides.map((g) => g.specialty).filter(Boolean))].sort();
---
<Layout title="Guías docentes">
  <div class="card">
    <h1 style="margin:0 0 12px; font-size:22px;">Catálogo de Guías Docentes</h1>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(190px, 1fr)); gap:10px;">
      <input id="q" placeholder="Buscar por título o profesor..." style="padding:8px 10px; border:1px solid #cbd5e1; border-radius:6px;" />
      <select id="dept" style="padding:8px 10px; border:1px solid #cbd5e1; border-radius:6px;">
        <option value="">Todos los departamentos</option>
        {departments.map((d) => <option value={d}>{d}</option>)}
      </select>
      <select id="type" style="padding:8px 10px; border:1px solid #cbd5e1; border-radius:6px;">
        <option value="">Todos los tipos</option>
        {types.map((t) => <option value={t}>{t}</option>)}
      </select>
      <select id="spec" style="padding:8px 10px; border:1px solid #cbd5e1; border-radius:6px;">
        <option value="">Todas las especialidades</option>
        {specialties.map((s) => <option value={s}>{s}</option>)}
      </select>
    </div>
  </div>

  <div id="grid" style="display:grid; grid-template-columns:repeat(auto-fill, minmax(310px, 1fr)); gap:14px;">
    {guides.map((g) => (
      <a
        href={'/guia/' + g.slug}
        class="card guide-item"
        data-title={(g.title + ' ' + (g.groups || []).map((x) => x.teacher + ' ' + x.group).join(' ')).toLowerCase()}
        data-dept={g.department || ''}
        data-type={g.courseType || ''}
        data-spec={g.specialty || ''}
        style="text-decoration:none; color:inherit; display:block; margin:0;"
      >
        <div style="display:flex; justify-content:space-between; gap:8px; align-items:flex-start;">
          <strong style="font-size:16px; color:#0f172a;">{g.title}</strong>
          <span class="pill">{g.creditsEcts} ECTS</span>
        </div>
        <div style="font-size:12.5px; color:#475569; margin-top:6px;">
          {[g.courseType, g.specialty, g.academicYear, g.semester].filter(Boolean).join(' · ')}
        </div>
        <div style="font-size:12px; color:#64748b; margin-top:8px;">
          {g.department} {g.groups?.length ? ('· ' + g.groups.length + ' grupo(s)') : ''}
        </div>
      </a>
    ))}
  </div>

  <script>
    const q = document.getElementById('q') as HTMLInputElement;
    const dept = document.getElementById('dept') as HTMLSelectElement;
    const type = document.getElementById('type') as HTMLSelectElement;
    const spec = document.getElementById('spec') as HTMLSelectElement;
    const items = Array.from(document.querySelectorAll('.guide-item')) as HTMLElement[];
    function filter() {
      const term = q.value.trim().toLowerCase();
      for (const el of items) {
        const ok =
          (!term || (el.dataset.title || '').includes(term)) &&
          (!dept.value || el.dataset.dept === dept.value) &&
          (!type.value || el.dataset.type === type.value) &&
          (!spec.value || el.dataset.spec === spec.value);
        el.style.display = ok ? 'block' : 'none';
      }
    }
    q.addEventListener('input', filter);
    dept.addEventListener('change', filter);
    type.addEventListener('change', filter);
    spec.addEventListener('change', filter);
  </script>
</Layout>
`)

	_ = addFile("src/pages/guia/[slug].astro", `---
import Layout from '../../layouts/Layout.astro';
import guides from '../../data/guides.json';

export function getStaticPaths() {
  return guides.map((g) => ({
    params: { slug: g.slug },
    props: { guide: g },
  }));
}

const { guide } = Astro.props;
---
<Layout title={guide.title}>
  <div class="card">
    <div style="display:flex; justify-content:space-between; align-items:flex-start; flex-wrap:wrap; gap:10px;">
      <div>
        <h1 style="margin:0 0 6px; font-size:26px;">{guide.title}</h1>
        <div style="color:#475569; font-size:14px;">{guide.degree} · {guide.degreeLevel} · {guide.department}</div>
      </div>
      <div style="display:flex; gap:6px; flex-wrap:wrap;">
        <span class="pill">{guide.creditsEcts} ECTS</span>
        {guide.courseType && <span class="pill">{guide.courseType}</span>}
        {guide.academicYear && <span class="pill">{guide.academicYear} · {guide.semester}</span>}
      </div>
    </div>
    {guide.description && <p style="margin-top:14px;">{guide.description}</p>}
  </div>

  <div class="card">
    <h2 style="margin-top:0; font-size:17px;">1. Identificación de la asignatura</h2>
    <table>
      <tbody>
        <tr><th style="width:28%;">Titulación / Nivel</th><td>{guide.degree} ({guide.degreeLevel})</td></tr>
        <tr><th>Especialidad / Itinerario</th><td>{guide.specialty || '—'}</td></tr>
        <tr><th>Materia</th><td>{guide.subjectArea || '—'}</td></tr>
        <tr><th>Tipo y Carácter</th><td>{guide.courseType} · {guide.character}</td></tr>
        <tr><th>Periodo de impartición</th><td>{guide.academicYear} — {guide.semester}</td></tr>
        <tr><th>Créditos / Horas</th><td>{guide.creditsEcts} ECTS · {guide.totalHours} h totales ({guide.inPersonHours} h presenciales)</td></tr>
        <tr><th>Prelación / Requisitos previos</th><td>{guide.prerequisites || 'Ninguno'}</td></tr>
        <tr><th>Idioma(s)</th><td>{guide.languages || 'Español'}</td></tr>
      </tbody>
    </table>
  </div>

  <div class="card">
    <h2 style="margin-top:0; font-size:17px;">2. Profesorado y Grupos (Turnos / Paralelos)</h2>
    <table>
      <thead>
        <tr><th>Grupo</th><th>Profesor/a</th><th>Turno</th><th>Observaciones</th></tr>
      </thead>
      <tbody>
        {(guide.groups || []).map((gr) => (
          <tr>
            <td><strong>{gr.group}</strong></td>
            <td>{gr.teacher}</td>
            <td>{gr.shift}</td>
            <td>{gr.notes || '—'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>

  <div class="card">
    <h2 style="margin-top:0; font-size:17px;">3. Competencias, Resultados y Contenidos</h2>
    {guide.competencies && <><h3>Competencias específicas</h3><p style="white-space:pre-line;">{guide.competencies}</p></>}
    {guide.learningOutcomes && <><h3>Resultados del aprendizaje</h3><p style="white-space:pre-line;">{guide.learningOutcomes}</p></>}
    {guide.contents && <><h3>Contenidos</h3><p style="white-space:pre-line;">{guide.contents}</p></>}
  </div>

  <div class="card">
    <h2 style="margin-top:0; font-size:17px;">4. Actividades Formativas y Evaluación</h2>
    {guide.theoryActivities && <p><strong>Actividades teóricas:</strong> {guide.theoryActivities}</p>}
    {guide.practiceActivities && <p><strong>Actividades prácticas:</strong> {guide.practiceActivities}</p>}
    {guide.otherActivities && <p><strong>Otras actividades obligatorias:</strong> {guide.otherActivities}</p>}
    {guide.studentWorkHours && <p><strong>Trabajo del estudiante:</strong> {guide.studentWorkHours}</p>}
    <table style="margin-top:10px;">
      <thead>
        <tr><th>Actitud y participación</th><th>Pruebas escritas</th><th>Pruebas prácticas</th></tr>
      </thead>
      <tbody>
        <tr><td>{guide.evalAttitudePct}%</td><td>{guide.evalWrittenPct}%</td><td>{guide.evalPracticalPct}%</td></tr>
      </tbody>
    </table>
    {guide.evalCriteria && <p style="margin-top:10px; white-space:pre-line;"><strong>Criterios:</strong> {guide.evalCriteria}</p>}
    {guide.bibliography && <p style="margin-top:10px; white-space:pre-line;"><strong>Bibliografía:</strong> {guide.bibliography}</p>}
  </div>
</Layout>
`)

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateSidebarWebTimetables generates a Mimosa-inspired interactive HTML timetable site
// with a collapsible accordion sidebar grouped by category.
func GenerateSidebarWebTimetables(p *Project, week int) string {
	p.RecomputeDerived()
	if week < 1 || week > p.Time.MaxWeeks {
		week = 1
	}
	isEn := strings.ToLower(p.Language) == "en"

	resMap := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}
	courseMap := make(map[string]Course, len(p.Courses))
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}

	type tableEntry struct {
		ID      string
		Code    string
		Name    string
		CatIdx  int
		CatName string
		HTML    string
	}

	// Build individual HTML grid per resource
	var entries []tableEntry
	catGroups := make(map[int][]tableEntry)

	for _, r := range p.Resources {
		cat := p.ResourceCategory(r.Category)
		if cat.NoTimetables {
			continue
		}
		var tb strings.Builder
		fmt.Fprintf(&tb, "<div class=\"tt-head\"><h2>%s — %s</h2><span>%s · %s %d</span></div>",
			html.EscapeString(r.Code), html.EscapeString(r.Name), html.EscapeString(cat.Name),
			map[bool]string{true: "Week", false: "Semana"}[isEn], week)
		tb.WriteString("<table class=\"tt-grid\"><thead><tr><th class=\"tcol\">")
		if isEn {
			tb.WriteString("Period")
		} else {
			tb.WriteString("Bloque")
		}
		tb.WriteString("</th>")
		for d := 1; d <= p.Time.MaxDays; d++ {
			fmt.Fprintf(&tb, "<th>%s</th>", html.EscapeString(p.Time.Days[d-1]))
		}
		tb.WriteString("</tr></thead><tbody>")

		for h := 1; h <= p.Time.MaxPeriods; h++ {
			bell := fmt.Sprintf("P%d", h)
			if h-1 < len(p.Time.Bells) {
				bell = p.Time.Bells[h-1]
			}
			fmt.Fprintf(&tb, "<tr><td class=\"tcol\"><strong>B%d</strong><br><small>%s</small></td>", h, html.EscapeString(bell))
			for d := 1; d <= p.Time.MaxDays; d++ {
				tb.WriteString("<td>")
				for _, s := range p.Slots {
					if s.Week != week || s.Day != d || s.Period != h {
						continue
					}
					c := courseMap[s.CourseID]
					hit := s.RoomID == r.ID
					if !hit {
						for _, rid := range c.Resources {
							if rid == r.ID {
								// If manual room overrode another room, skip
								if s.RoomID != "" && r.Category == p.RoomsCategory {
									hit = false
								} else {
									hit = true
								}
							}
						}
					}
					if hit {
						others := []string{}
						for _, rid := range c.Resources {
							if rid != r.ID {
								if rr, ok := resMap[rid]; ok {
									if s.RoomID != "" && rr.Category == p.RoomsCategory {
										continue
									}
									others = append(others, rr.Code)
								}
							}
						}
						if s.RoomID != "" && s.RoomID != r.ID {
							if rr, ok := resMap[s.RoomID]; ok {
								others = append(others, "["+rr.Code+"]")
							}
						}
						col := c.Color
						if col == "" {
							col = "#2563eb"
						}
						fmt.Fprintf(&tb, "<div class=\"card-slot\" style=\"border-left-color:%s;background:%s18\"><div class=\"sc-code\">%s</div><div class=\"sc-name\">%s</div><div class=\"sc-meta\">%s</div></div>",
							html.EscapeString(col), html.EscapeString(col), html.EscapeString(c.Code), html.EscapeString(c.Name), html.EscapeString(strings.Join(others, " · ")))
					}
				}
				tb.WriteString("</td>")
			}
			tb.WriteString("</tr>")
		}
		tb.WriteString("</tbody></table>")

		te := tableEntry{ID: r.ID, Code: r.Code, Name: r.Name, CatIdx: r.Category, CatName: cat.Name, HTML: tb.String()}
		entries = append(entries, te)
		catGroups[r.Category] = append(catGroups[r.Category], te)
	}

	catOrder := make([]int, 0, len(catGroups))
	for k := range catGroups {
		catOrder = append(catOrder, k)
	}
	sort.Ints(catOrder)

	tablesMap := make(map[string]string, len(entries))
	firstID := ""
	for _, e := range entries {
		if firstID == "" {
			firstID = e.ID
		}
		tablesMap[e.ID] = e.HTML
	}
	tablesJSON, _ := json.Marshal(tablesMap)

	var navSB strings.Builder
	for idx, catIdx := range catOrder {
		items := catGroups[catIdx]
		catName := items[0].CatName
		openStyle := "none"
		if idx == 0 {
			openStyle = "block"
		}
		fmt.Fprintf(&navSB, `<div class="nav-cat"><button class="cat-btn" onclick="toggleCat('cat_%d')">%s <span>(%d)</span></button><ul id="cat_%d" style="display:%s">`,
			catIdx, html.EscapeString(strings.ToUpper(catName)), len(items), catIdx, openStyle)
		for _, it := range items {
			fmt.Fprintf(&navSB, `<li><a href="#" data-id="%s" data-search="%s" onclick="showTT('%s');return false;"><strong>%s</strong> · %s</a></li>`,
				html.EscapeString(it.ID), html.EscapeString(strings.ToLower(it.Code+" "+it.Name)), html.EscapeString(it.ID),
				html.EscapeString(it.Code), html.EscapeString(it.Name))
		}
		navSB.WriteString("</ul></div>")
	}

	title := fmt.Sprintf("%s — %s", p.Name, p.Institution)
	return fmt.Sprintf(`<!doctype html>
<html lang="%s">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width,initial-scale=1" />
<title>%s</title>
<style>
*{box-sizing:border-box}
html,body{margin:0;height:100%%;font-family:-apple-system,BlinkMacSystemFont,'Inter',Segoe UI,sans-serif;background:#f8fafc;color:#0f172a;overflow:hidden}
.shell{display:flex;height:100vh;width:100vw;position:relative}
nav#sidebar{width:270px;background:#fff;border-right:1px solid #cbd5e1;box-shadow:2px 0 14px rgba(15,23,42,.08);display:flex;flex-direction:column;transition:margin-left .28s ease;z-index:20;flex-shrink:0}
nav#sidebar.collapsed{margin-left:-270px}
.sb-head{padding:14px 14px 10px;border-bottom:1px solid #e2e8f0;background:#f1f5f9}
.sb-title{font-weight:800;font-size:15px}
.sb-sub{font-size:11.5px;color:#475569;margin-top:2px}
.sb-tools{padding:8px 12px;border-bottom:1px solid #e2e8f0;display:flex;gap:6px}
.sb-tools input{flex:1;padding:6px 8px;border:1px solid #cbd5e1;border-radius:6px;font-size:12px}
.sb-list{flex:1;overflow-y:auto;padding:8px 10px}
.nav-cat{margin-bottom:6px}
.cat-btn{width:100%%;text-align:left;background:#eff6ff;color:#1e3a8a;border:1px solid #bfdbfe;border-radius:6px;padding:7px 10px;font-weight:700;font-size:12px;cursor:pointer;display:flex;justify-content:space-between}
.nav-cat ul{list-style:none;margin:4px 0 6px;padding:0}
.nav-cat li a{display:block;padding:6px 9px;border-radius:5px;text-decoration:none;color:#1e293b;font-size:12.5px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.nav-cat li a:hover,.nav-cat li a.active{background:#2563eb;color:#fff}
.viewer{flex:1;display:flex;flex-direction:column;min-width:0;overflow:hidden}
.vbar{display:flex;align-items:center;justify-content:space-between;padding:8px 14px;background:#fff;border-bottom:1px solid #e2e8f0;gap:10px}
.vbtn{padding:6px 10px;border:1px solid #cbd5e1;background:#fff;border-radius:6px;font-size:12px;font-weight:600;cursor:pointer}
.vbtn:hover{background:#f1f5f9}
.stage{flex:1;overflow:auto;padding:16px 20px}
.tt-head{display:flex;justify-content:space-between;align-items:baseline;margin-bottom:12px;flex-wrap:wrap;gap:8px}
.tt-head h2{margin:0;font-size:19px}
.tt-head span{color:#64748b;font-size:13px;font-weight:600}
.tt-grid{width:100%%;border-collapse:collapse;background:#fff;border:1px solid #cbd5e1;table-layout:fixed}
.tt-grid th{background:#f1f5f9;border:1px solid #cbd5e1;padding:8px;font-size:12.5px}
.tt-grid td{border:1px solid #e2e8f0;padding:4px;vertical-align:top;height:56px}
.tt-grid th.tcol,.tt-grid td.tcol{width:92px;background:#fafbfd;text-align:center;vertical-align:middle;font-size:11.5px}
.card-slot{border-left:3.5px solid #2563eb;border-radius:4px;padding:4px 6px;margin-bottom:3px;font-size:11.5px}
.sc-code{font-weight:700}
.sc-name{font-size:10.5px;color:#334155}
.sc-meta{font-size:10px;color:#64748b}
@media print{nav#sidebar,.vbar{display:none!important}.stage{padding:0;overflow:visible}}
</style>
</head>
<body>
<div class="shell">
  <nav id="sidebar">
    <div class="sb-head">
      <div class="sb-title">%s</div>
      <div class="sb-sub">%s · %s</div>
    </div>
    <div class="sb-tools">
      <input id="q" placeholder="%s" oninput="filterNav(this.value)" />
    </div>
    <div class="sb-list">%s</div>
  </nav>
  <div class="viewer">
    <div class="vbar">
      <div style="display:flex;gap:8px;align-items:center">
        <button class="vbtn" onclick="document.getElementById('sidebar').classList.toggle('collapsed')">☰ %s</button>
        <strong>%s</strong>
      </div>
      <div style="display:flex;gap:8px">
        <button class="vbtn" onclick="toggleFull()">⛶ %s</button>
        <button class="vbtn" onclick="window.print()">🖨 %s</button>
      </div>
    </div>
    <div class="stage" id="stage"></div>
  </div>
</div>
<script>
const TABLES = %s;
function toggleCat(id){const el=document.getElementById(id);el.style.display=el.style.display==='none'?'block':'none';}
function showTT(id){
  document.getElementById('stage').innerHTML = TABLES[id] || '';
  document.querySelectorAll('#sidebar a').forEach(a => a.classList.toggle('active', a.dataset.id === id));
}
function filterNav(v){
  const q=v.trim().toLowerCase();
  document.querySelectorAll('#sidebar li').forEach(li=>{
    const a=li.querySelector('a');
    li.style.display=!q || (a.dataset.search||'').includes(q)?'block':'none';
  });
  if(q) document.querySelectorAll('.nav-cat ul').forEach(u=>u.style.display='block');
}
function toggleFull(){if(!document.fullscreenElement){document.documentElement.requestFullscreen?.();}else{document.exitFullscreen?.();}}
showTT(%q);
</script>
</body>
</html>`,
		map[bool]string{true: "en", false: "es"}[isEn],
		html.EscapeString(title),
		html.EscapeString(p.Name),
		html.EscapeString(p.Institution),
		html.EscapeString(p.Department),
		map[bool]string{true: "Filter groups, teachers, rooms...", false: "Filtrar grupos, profesores, aulas..."}[isEn],
		navSB.String(),
		map[bool]string{true: "Menu", false: "Menú"}[isEn],
		html.EscapeString(p.Institution),
		map[bool]string{true: "Fullscreen", false: "Pantalla completa"}[isEn],
		map[bool]string{true: "Print", false: "Imprimir"}[isEn],
		string(tablesJSON),
		firstID,
	)
}
