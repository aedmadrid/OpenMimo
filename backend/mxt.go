package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"
)

func parseCategoryString(raw string, isResource bool) []Category {
	parts := strings.Split(raw, "$")
	out := make([]Category, 0, 7)
	for i := 0; i < 7; i++ {
		var short, name string
		if i < len(parts) && strings.TrimSpace(parts[i]) != "" {
			tok := strings.TrimSpace(parts[i])
			if colon := strings.IndexByte(tok, ':'); colon >= 0 {
				short = strings.TrimSpace(tok[:colon])
				name = strings.TrimSpace(tok[colon+1:])
			} else {
				short = fmt.Sprintf("%d", i+1)
				name = tok
			}
		} else {
			if isResource {
				short = fmt.Sprintf("R%d", i+1)
				name = fmt.Sprintf("Categoría %d", i+1)
			} else {
				short = fmt.Sprintf("c%d", i+1)
				name = fmt.Sprintf("Tipo %d", i+1)
			}
		}
		cat := Category{
			Index:     i + 1,
			ShortCode: short,
			Name:      name,
			GapWeight: 1,
		}
		if isResource && i == 3 {
			cat.NoTimetables = true
		}
		if isResource && (i == 2 || i == 3 || i >= 5) {
			cat.NoGaps = true
		}
		out = append(out, cat)
	}
	return out
}

func ImportMXT(content string, fileName string) (*Project, error) {
	p := NewEmptyProject("es")
	if fileName != "" {
		p.Name = strings.TrimSuffix(fileName, ".mxt")
	}

	section := ""
	courseByCode := make(map[string]string)
	resByCode := make(map[string]string)
	courseOrder := 0
	resOrder := 0
	slotOrder := 0
	bookingOrder := 0

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		kl := strings.ToLower(k)

		switch section {
		case "header":
			switch kl {
			case "customer":
				if v != "" {
					p.Institution = v
				}
			case "department":
				if v != "" {
					p.Department = v
				}
			case "comment":
				p.Comment = v
			}
		case "time":
			switch kl {
			case "maxweeks":
				if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 255 {
					p.Time.MaxWeeks = n
					p.Time.Weeks = DefaultWeeks(n)
				}
			case "maxdays":
				if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 7 {
					p.Time.MaxDays = n
				}
			case "maxlectures":
				if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 30 {
					p.Time.MaxPeriods = n
				}
			case "firstweek":
				if n, err := strconv.Atoi(v); err == nil && n >= 1 {
					p.Time.FirstWeek = n
				}
			case "yyyymmdd":
				if len(v) == 8 {
					p.Time.StartDate = v[0:4] + "-" + v[4:6] + "-" + v[6:8]
				}
			case "days":
				parts := strings.Split(v, "$")
				days := make([]string, 0, 7)
				for _, d := range parts {
					if strings.TrimSpace(d) != "" {
						days = append(days, strings.TrimSpace(d))
					}
				}
				if len(days) >= 5 {
					p.Time.Days = days
				}
			}
		case "timeperiods":
			if idx, err := strconv.Atoi(k); err == nil && idx >= 1 && idx <= 30 {
				for len(p.Time.Bells) < idx {
					p.Time.Bells = append(p.Time.Bells, "")
				}
				p.Time.Bells[idx-1] = v
			}
		case "weeks":
			if idx, err := strconv.Atoi(k); err == nil && idx >= 1 && idx <= 255 {
				for len(p.Time.Weeks) < idx {
					wIdx := len(p.Time.Weeks) + 1
					p.Time.Weeks = append(p.Time.Weeks, WeekDef{Index: wIdx, Name: fmt.Sprintf("S%d", wIdx), Length: 1})
				}
				parts := strings.Split(v, ";")
				if len(parts) >= 1 && parts[0] != "" {
					p.Time.Weeks[idx-1].Name = parts[0]
				}
				if len(parts) >= 2 {
					if l, err := strconv.Atoi(parts[1]); err == nil && l >= 1 {
						p.Time.Weeks[idx-1].Length = l
					}
				}
			}
		case "courses":
			switch kl {
			case "categories":
				p.CourseCategories = parseCategoryString(v, false)
			case "bookings":
				bks := []string{}
				for _, tok := range strings.Split(v, "$") {
					if strings.TrimSpace(tok) != "" {
						bks = append(bks, strings.TrimSpace(tok))
					}
				}
				if len(bks) > 0 {
					p.BookingTypes = bks
				}
			}
		case "components":
			switch kl {
			case "categories":
				p.ResourceCategories = parseCategoryString(v, true)
			case "notimetables":
				bits := strings.Split(v, ";")
				for i := 0; i < len(bits) && i < len(p.ResourceCategories); i++ {
					p.ResourceCategories[i].NoTimetables = strings.TrimSpace(bits[i]) == "1"
				}
			case "nogaps":
				bits := strings.Split(v, ";")
				for i := 0; i < len(bits) && i < len(p.ResourceCategories); i++ {
					p.ResourceCategories[i].NoGaps = strings.TrimSpace(bits[i]) == "1"
				}
			case "rooms":
				if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 7 {
					p.RoomsCategory = n
				}
			}
		case "coursedata":
			// 1=CODE;NAME;CAT;PLANNED;...
			parts := strings.Split(v, ";")
			if len(parts) >= 4 {
				courseOrder++
				code := strings.TrimSpace(parts[0])
				name := strings.TrimSpace(parts[1])
				cat, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
				if cat < 1 || cat > 7 {
					cat = 1
				}
				planned, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
				id := fmt.Sprintf("c_%d", courseOrder)
				p.Courses = append(p.Courses, Course{
					ID:        id,
					Code:      code,
					Name:      name,
					Category:  cat,
					Planned:   planned,
					Resources: []string{},
				})
				courseByCode[strings.ToUpper(code)] = id
			}
		case "componentdata":
			// 1=CODE;NAME;CAT;PLANNED;...
			parts := strings.Split(v, ";")
			if len(parts) >= 4 {
				resOrder++
				code := strings.TrimSpace(parts[0])
				name := strings.TrimSpace(parts[1])
				cat, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
				if cat < 1 || cat > 7 {
					cat = 1
				}
				planned, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
				id := fmt.Sprintf("r_%d", resOrder)
				p.Resources = append(p.Resources, Resource{
					ID:       id,
					Code:     code,
					Name:     name,
					Category: cat,
					Planned:  planned,
				})
				resByCode[strings.ToUpper(code)] = id
			}
		case "collections":
			// 1=COURSE_CODE;RESOURCE_CODE
			parts := strings.Split(v, ";")
			if len(parts) >= 2 {
				cid := courseByCode[strings.ToUpper(strings.TrimSpace(parts[0]))]
				rid := resByCode[strings.ToUpper(strings.TrimSpace(parts[1]))]
				if cid != "" && rid != "" {
					if c := p.CourseByID(cid); c != nil {
						c.Resources = append(c.Resources, rid)
					}
				}
			}
		case "weeklectures":
			// 1=COURSE_CODE;W1;W2;PERIODS
			parts := strings.Split(v, ";")
			if len(parts) >= 4 {
				cid := courseByCode[strings.ToUpper(strings.TrimSpace(parts[0]))]
				w1, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				w2, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
				per, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
				if cid != "" && w1 >= 1 && w2 >= w1 && per > 0 {
					for w := w1; w <= w2 && w <= p.Time.MaxWeeks; w++ {
						p.SetWeekAlloc(cid, w, p.WeekAlloc(cid, w)+per)
					}
				}
			}
		case "tablelectures":
			// 1=COURSE_CODE;W1;W2;D1;D2;H1;H2
			parts := strings.Split(v, ";")
			if len(parts) >= 7 {
				cid := courseByCode[strings.ToUpper(strings.TrimSpace(parts[0]))]
				w1, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				w2, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
				d1, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
				d2, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
				h1, _ := strconv.Atoi(strings.TrimSpace(parts[5]))
				h2, _ := strconv.Atoi(strings.TrimSpace(parts[6]))
				if cid != "" && w1 >= 1 && d1 >= 1 && h1 >= 1 {
					for w := w1; w <= w2 && w <= p.Time.MaxWeeks; w++ {
						for d := d1; d <= d2 && d <= p.Time.MaxDays; d++ {
							for h := h1; h <= h2 && h <= p.Time.MaxPeriods; h++ {
								slotOrder++
								p.Slots = append(p.Slots, SlotAssignment{
									ID:       fmt.Sprintf("s_%d", slotOrder),
									CourseID: cid,
									Week:     w,
									Day:      d,
									Period:   h,
								})
							}
						}
					}
				}
			}
		case "tablebookings", "weekbookings":
			parts := strings.Split(v, ";")
			if len(parts) >= 4 {
				code := strings.ToUpper(strings.TrimSpace(parts[0]))
				targetType, targetID := "", ""
				if id, ok := courseByCode[code]; ok {
					targetType, targetID = "course", id
				} else if id, ok := resByCode[code]; ok {
					targetType, targetID = "resource", id
				}
				if targetID != "" {
					if section == "weekbookings" {
						w1, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
						w2, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
						kind, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
						if kind < 1 {
							kind = 1
						}
						for w := w1; w <= w2 && w <= p.Time.MaxWeeks; w++ {
							bookingOrder++
							p.Bookings = append(p.Bookings, Booking{
								ID: fmt.Sprintf("b_%d", bookingOrder), TargetType: targetType, TargetID: targetID,
								Scope: "week", Week: w, Kind: kind,
							})
						}
					} else if len(parts) >= 7 {
						w1, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
						w2, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
						d1, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
						d2, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
						h1, _ := strconv.Atoi(strings.TrimSpace(parts[5]))
						h2, _ := strconv.Atoi(strings.TrimSpace(parts[6]))
						kind := 1
						if len(parts) >= 8 {
							if kk, err := strconv.Atoi(strings.TrimSpace(parts[7])); err == nil && kk >= 1 {
								kind = kk
							}
						}
						for w := w1; w <= w2 && w <= p.Time.MaxWeeks; w++ {
							for d := d1; d <= d2 && d <= p.Time.MaxDays; d++ {
								for h := h1; h <= h2 && h <= p.Time.MaxPeriods; h++ {
									bookingOrder++
									p.Bookings = append(p.Bookings, Booking{
										ID: fmt.Sprintf("b_%d", bookingOrder), TargetType: targetType, TargetID: targetID,
										Scope: "timetable", Week: w, Day: d, Period: h, Kind: kind,
									})
								}
							}
						}
					}
				}
			}
		case "manualrooms":
			// 1=COURSE_CODE;W;D;H;ROOM_CODE
			parts := strings.Split(v, ";")
			if len(parts) >= 5 {
				cid := courseByCode[strings.ToUpper(strings.TrimSpace(parts[0]))]
				w, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				d, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
				h, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
				rid := resByCode[strings.ToUpper(strings.TrimSpace(parts[4]))]
				if cid != "" && rid != "" {
					for i := range p.Slots {
						if p.Slots[i].CourseID == cid && p.Slots[i].Week == w && p.Slots[i].Day == d && p.Slots[i].Period == h {
							p.Slots[i].RoomID = rid
						}
					}
				}
			}
		}
	}

	p.RecomputeDerived()
	return p, nil
}

func ExportMXT(p *Project) string {
	p.RecomputeDerived()
	var sb strings.Builder
	now := time.Now().Format("20060102 15:04:05")
	sb.WriteString("[Header]\r\n")
	sb.WriteString("Version=6.0.0-OpenMimo\r\n")
	sb.WriteString("Created=" + now + "\r\n")
	sb.WriteString("Customer=" + p.Institution + "\r\n")
	sb.WriteString("Department=" + p.Department + "\r\n")
	sb.WriteString("Comment=" + p.Comment + "\r\n")
	sb.WriteString("Description=OpenMimo FOSS Scheduling Software\r\n\r\n")

	sb.WriteString("[Time]\r\n")
	fmt.Fprintf(&sb, "MaxWeeks=%d\r\n", p.Time.MaxWeeks)
	fmt.Fprintf(&sb, "MaxDays=%d\r\n", p.Time.MaxDays)
	fmt.Fprintf(&sb, "MaxLectures=%d\r\n", p.Time.MaxPeriods)
	fmt.Fprintf(&sb, "YYYYMMDD=%s\r\n", strings.ReplaceAll(p.Time.StartDate, "-", ""))
	fmt.Fprintf(&sb, "FirstWeek=%d\r\n", p.Time.FirstWeek)
	sb.WriteString("Days=" + strings.Join(p.Time.Days, "$") + "$\r\n\r\n")

	sb.WriteString("[Timeperiods]\r\n")
	for i := 0; i < p.Time.MaxPeriods && i < len(p.Time.Bells); i++ {
		fmt.Fprintf(&sb, "%d=%s\r\n", i+1, p.Time.Bells[i])
	}
	sb.WriteString("\r\n[Weeks]\r\n")
	for _, w := range p.Time.Weeks {
		fmt.Fprintf(&sb, "%d=%s;%d\r\n", w.Index, w.Name, w.Length)
	}

	sb.WriteString("\r\n[Courses]\r\nCategories=")
	for _, c := range p.CourseCategories {
		fmt.Fprintf(&sb, "%s:%s$", c.ShortCode, c.Name)
	}
	sb.WriteString("\r\nBookings=" + strings.Join(p.BookingTypes, "$") + "$\r\nDefault=1\r\n")

	sb.WriteString("\r\n[Components]\r\nCategories=")
	for _, c := range p.ResourceCategories {
		fmt.Fprintf(&sb, "%s:%s$", c.ShortCode, c.Name)
	}
	sb.WriteString("\r\nNoTimetables=")
	for _, c := range p.ResourceCategories {
		if c.NoTimetables {
			sb.WriteString("1;")
		} else {
			sb.WriteString("0;")
		}
	}
	sb.WriteString("\r\nNoGaps=")
	for _, c := range p.ResourceCategories {
		if c.NoGaps {
			sb.WriteString("1;")
		} else {
			sb.WriteString("0;")
		}
	}
	fmt.Fprintf(&sb, "\r\nRooms=%d\r\n", p.RoomsCategory)

	courseCode := make(map[string]string)
	resCode := make(map[string]string)

	sb.WriteString("\r\n[Coursedata]\r\n")
	for i, c := range p.Courses {
		courseCode[c.ID] = c.Code
		fmt.Fprintf(&sb, "%d=%s;%s;%d;%d;0;0\r\n", i+1, c.Code, c.Name, c.Category, c.Planned)
	}

	sb.WriteString("\r\n[Componentdata]\r\n")
	for i, r := range p.Resources {
		resCode[r.ID] = r.Code
		fmt.Fprintf(&sb, "%d=%s;%s;%d;%d;0\r\n", i+1, r.Code, r.Name, r.Category, r.Planned)
	}

	sb.WriteString("\r\n[Collections]\r\n")
	idx := 1
	for _, c := range p.Courses {
		for _, rid := range c.Resources {
			if rc, ok := resCode[rid]; ok {
				fmt.Fprintf(&sb, "%d=%s;%s\r\n", idx, c.Code, rc)
				idx++
			}
		}
	}

	sb.WriteString("\r\n[Weeklectures]\r\n")
	idx = 1
	for _, wa := range p.WeekAllocations {
		if cc, ok := courseCode[wa.CourseID]; ok && wa.Periods > 0 {
			fmt.Fprintf(&sb, "%d=%s;%d;%d;%d\r\n", idx, cc, wa.Week, wa.Week, wa.Periods)
			idx++
		}
	}

	sb.WriteString("\r\n[Weekbookings]\r\n")
	idx = 1
	for _, b := range p.Bookings {
		if b.Scope == "week" {
			code := courseCode[b.TargetID]
			if b.TargetType == "resource" {
				code = resCode[b.TargetID]
			}
			if code != "" {
				fmt.Fprintf(&sb, "%d=%s;%d;%d;%d\r\n", idx, code, b.Week, b.Week, b.Kind)
				idx++
			}
		}
	}

	sb.WriteString("\r\n[Tablelectures]\r\n")
	idx = 1
	for _, s := range p.Slots {
		if cc, ok := courseCode[s.CourseID]; ok {
			fmt.Fprintf(&sb, "%d=%s;%d;%d;%d;%d;%d;%d\r\n", idx, cc, s.Week, s.Week, s.Day, s.Day, s.Period, s.Period)
			idx++
		}
	}

	sb.WriteString("\r\n[Tablebookings]\r\n")
	idx = 1
	for _, b := range p.Bookings {
		if b.Scope == "timetable" {
			code := courseCode[b.TargetID]
			if b.TargetType == "resource" {
				code = resCode[b.TargetID]
			}
			if code != "" {
				fmt.Fprintf(&sb, "%d=%s;%d;%d;%d;%d;%d;%d;%d\r\n", idx, code, b.Week, b.Week, b.Day, b.Day, b.Period, b.Period, b.Kind)
				idx++
			}
		}
	}

	sb.WriteString("\r\n[Manualrooms]\r\n")
	idx = 1
	for _, s := range p.Slots {
		if s.RoomID != "" {
			if cc, ok := courseCode[s.CourseID]; ok {
				if rc, ok2 := resCode[s.RoomID]; ok2 {
					fmt.Fprintf(&sb, "%d=%s;%d;%d;%d;%s\r\n", idx, cc, s.Week, s.Day, s.Period, rc)
					idx++
				}
			}
		}
	}
	return sb.String()
}

func ExportCSV(p *Project) string {
	p.RecomputeDerived()
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"Week", "Day", "Period", "Time", "EventCode", "EventName", "Resources", "ManualRoom"})

	courseMap := make(map[string]Course)
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}
	resMap := make(map[string]Resource)
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}

	sorted := append([]SlotAssignment(nil), p.Slots...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Week != sorted[j].Week {
			return sorted[i].Week < sorted[j].Week
		}
		if sorted[i].Day != sorted[j].Day {
			return sorted[i].Day < sorted[j].Day
		}
		return sorted[i].Period < sorted[j].Period
	})

	for _, s := range sorted {
		c := courseMap[s.CourseID]
		dayName := fmt.Sprintf("D%d", s.Day)
		if s.Day >= 1 && s.Day <= len(p.Time.Days) {
			dayName = p.Time.Days[s.Day-1]
		}
		bell := fmt.Sprintf("P%d", s.Period)
		if s.Period >= 1 && s.Period <= len(p.Time.Bells) {
			bell = p.Time.Bells[s.Period-1]
		}
		resNames := []string{}
		for _, rid := range c.Resources {
			if r, ok := resMap[rid]; ok {
				resNames = append(resNames, r.Code)
			}
		}
		room := ""
		if s.RoomID != "" {
			if r, ok := resMap[s.RoomID]; ok {
				room = r.Code
			}
		}
		_ = w.Write([]string{
			fmt.Sprintf("%d", s.Week), dayName, fmt.Sprintf("%d", s.Period), bell,
			c.Code, c.Name, strings.Join(resNames, ", "), room,
		})
	}
	w.Flush()
	return sb.String()
}

func ExportICS(p *Project, filterResourceID string) string {
	p.RecomputeDerived()
	baseDate, err := time.Parse("2006-01-02", p.Time.StartDate)
	if err != nil {
		baseDate = time.Now()
	}
	// Align baseDate to Monday
	for baseDate.Weekday() != time.Monday {
		baseDate = baseDate.AddDate(0, 0, -1)
	}
	courseMap := make(map[string]Course)
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}
	resMap := make(map[string]Resource)
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//OpenMimo//Timetabling//EN\r\n")
	for _, s := range p.Slots {
		c := courseMap[s.CourseID]
		if filterResourceID != "" {
			match := s.RoomID == filterResourceID
			for _, rid := range c.Resources {
				if rid == filterResourceID {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		d := baseDate.AddDate(0, 0, (s.Week-1)*7+(s.Day-1))
		startH, startM, endH, endM := 7+s.Period, 0, 7+s.Period, 45
		if s.Period >= 1 && s.Period <= len(p.Time.Bells) {
			parts := strings.Split(p.Time.Bells[s.Period-1], "-")
			if len(parts) == 2 {
				if hm := strings.Split(strings.TrimSpace(parts[0]), ":"); len(hm) == 2 {
					startH, _ = strconv.Atoi(hm[0])
					startM, _ = strconv.Atoi(hm[1])
				}
				if hm := strings.Split(strings.TrimSpace(parts[1]), ":"); len(hm) == 2 {
					endH, _ = strconv.Atoi(hm[0])
					endM, _ = strconv.Atoi(hm[1])
				}
			}
		}
		dtStart := time.Date(d.Year(), d.Month(), d.Day(), startH, startM, 0, 0, time.Local)
		dtEnd := time.Date(d.Year(), d.Month(), d.Day(), endH, endM, 0, 0, time.Local)
		loc := ""
		if s.RoomID != "" {
			loc = resMap[s.RoomID].Name
		} else {
			for _, rid := range c.Resources {
				if r, ok := resMap[rid]; ok && r.Category == p.RoomsCategory {
					loc = r.Name
					break
				}
			}
		}
		fmt.Fprintf(&sb, "BEGIN:VEVENT\r\nUID:%s@openmimo\r\nDTSTART:%s\r\nDTEND:%s\r\nSUMMARY:%s - %s\r\nLOCATION:%s\r\nEND:VEVENT\r\n",
			s.ID, dtStart.Format("20060102T150405"), dtEnd.Format("20060102T150405"), c.Code, c.Name, loc)
	}
	sb.WriteString("END:VCALENDAR\r\n")
	return sb.String()
}

// GenerateHTMLReport builds any of the 6 Mimosa report types as standalone styled HTML.
// Report 3 generates the interactive sidebar web timetable viewer.
func GenerateHTMLReport(p *Project, reportType int, week int) string {
	if reportType == 3 {
		return GenerateSidebarWebTimetables(p, week)
	}
	p.RecomputeDerived()
	if week < 1 || week > p.Time.MaxWeeks {
		week = 1
	}
	resMap := make(map[string]Resource)
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}
	courseMap := make(map[string]Course)
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}

	isEn := strings.ToLower(p.Language) == "en"
	titles := map[int]string{
		1: "Lista de Cursos / Eventos",
		2: "Lista de Recursos / Componentes",
		3: fmt.Sprintf("Horarios — Semana %d", week),
		4: "Cursos por Recursos",
		5: "Recursos por Cursos",
		6: fmt.Sprintf("Calendario Compacto — Semana %d", week),
	}
	if isEn {
		titles = map[int]string{
			1: "List of Events / Courses",
			2: "List of Resources / Components",
			3: fmt.Sprintf("Timetables — Week %d", week),
			4: "Events by Resources",
			5: "Resources by Events",
			6: fmt.Sprintf("Compact Schedules — Week %d", week),
		}
	}
	title := titles[reportType]
	if title == "" {
		if isEn {
			title = "OpenMimo Report"
		} else {
			title = "Reporte OpenMimo"
		}
	}

	var sb strings.Builder
	htmlLang := "es"
	if isEn {
		htmlLang = "en"
	}
	sb.WriteString(`<!DOCTYPE html><html lang="` + htmlLang + `"><head><meta charset="utf-8"><title>`)
	sb.WriteString(html.EscapeString(title + " - " + p.Institution))
	sb.WriteString(`</title><style>
body{font-family:-apple-system,BlinkMacSystemFont,'Inter',Segoe UI,sans-serif;color:#0f172a;margin:28px;background:#fff}
h1{font-size:20px;margin:0 0 4px}
.sub{color:#64748b;font-size:13px;margin-bottom:20px}
h2{font-size:15px;margin:22px 0 8px;color:#1e293b;border-bottom:2px solid #e2e8f0;padding-bottom:4px}
table{border-collapse:collapse;width:100%;margin-bottom:18px;font-size:12px}
th,td{border:1px solid #cbd5e1;padding:6px 8px;text-align:left;vertical-align:top}
th{background:#f1f5f9;font-weight:600;color:#334155}
.pill{display:inline-block;padding:2px 6px;border-radius:4px;font-size:11px;font-weight:600;margin:1px}
.ev{background:#fef3c7;color:#92400e;border:1px solid #fcd34d}
.rs{background:#dbeafe;color:#1e40af;border:1px solid #93c5fd}
.cell-card{background:#eff6ff;border-left:3px solid #2563eb;padding:4px 6px;border-radius:3px;margin-bottom:3px}
.cell-code{font-weight:700;color:#1e3a8a}
.cell-meta{font-size:10px;color:#475569}
</style></head><body>`)
	fmt.Fprintf(&sb, "<h1>%s</h1><div class=\"sub\">%s · %s · OpenMimo</div>",
		html.EscapeString(title), html.EscapeString(p.Institution), html.EscapeString(p.Department))

	switch reportType {
	case 1:
		if isEn {
			sb.WriteString("<table><thead><tr><th>#</th><th>Code</th><th>Name</th><th>Cat.</th><th>Plan</th><th>Alloc.</th><th>Sched.</th><th>Left</th><th>Resources</th></tr></thead><tbody>")
		} else {
			sb.WriteString("<table><thead><tr><th>#</th><th>Código</th><th>Nombre</th><th>Cat.</th><th>Plan</th><th>Asig.</th><th>Prog.</th><th>Faltan</th><th>Recursos</th></tr></thead><tbody>")
		}
		for i, c := range p.Courses {
			rCodes := []string{}
			for _, rid := range c.Resources {
				if r, ok := resMap[rid]; ok {
					rCodes = append(rCodes, r.Code)
				}
			}
			fmt.Fprintf(&sb, "<tr><td>%d</td><td><span class=\"pill ev\">%s</span></td><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%s</td></tr>",
				i+1, html.EscapeString(c.Code), html.EscapeString(c.Name), c.Category, c.Planned, c.Allocated, c.Scheduled, c.Left, html.EscapeString(strings.Join(rCodes, ", ")))
		}
		sb.WriteString("</tbody></table>")
	case 2:
		if isEn {
			sb.WriteString("<table><thead><tr><th>#</th><th>Code</th><th>Name</th><th>Cat.</th><th>Max</th><th>Alloc.</th><th>Sched.</th><th>Links</th></tr></thead><tbody>")
		} else {
			sb.WriteString("<table><thead><tr><th>#</th><th>Código</th><th>Nombre</th><th>Cat.</th><th>Máx</th><th>Asig.</th><th>Prog.</th><th>Enlaces</th></tr></thead><tbody>")
		}
		for i, r := range p.Resources {
			fmt.Fprintf(&sb, "<tr><td>%d</td><td><span class=\"pill rs\">%s</span></td><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td></tr>",
				i+1, html.EscapeString(r.Code), html.EscapeString(r.Name), r.Category, r.Planned, r.Allocated, r.Scheduled, r.Links)
		}
		sb.WriteString("</tbody></table>")
	case 4:
		if isEn {
			sb.WriteString("<table><thead><tr><th>Resource</th><th>Name</th><th>Linked Events</th></tr></thead><tbody>")
		} else {
			sb.WriteString("<table><thead><tr><th>Recurso</th><th>Nombre</th><th>Cursos vinculados</th></tr></thead><tbody>")
		}
		for _, r := range p.Resources {
			cs := []string{}
			for _, c := range p.Courses {
				for _, rid := range c.Resources {
					if rid == r.ID {
						cs = append(cs, fmt.Sprintf("%s (%s)", c.Code, c.Name))
					}
				}
			}
			fmt.Fprintf(&sb, "<tr><td><span class=\"pill rs\">%s</span></td><td>%s</td><td>%s</td></tr>",
				html.EscapeString(r.Code), html.EscapeString(r.Name), html.EscapeString(strings.Join(cs, " · ")))
		}
		sb.WriteString("</tbody></table>")
	case 5:
		if isEn {
			sb.WriteString("<table><thead><tr><th>Event</th><th>Name</th><th>Linked Resources</th></tr></thead><tbody>")
		} else {
			sb.WriteString("<table><thead><tr><th>Curso</th><th>Nombre</th><th>Recursos vinculados</th></tr></thead><tbody>")
		}
		for _, c := range p.Courses {
			rs := []string{}
			for _, rid := range c.Resources {
				if r, ok := resMap[rid]; ok {
					rs = append(rs, fmt.Sprintf("%s (%s)", r.Code, r.Name))
				}
			}
			fmt.Fprintf(&sb, "<tr><td><span class=\"pill ev\">%s</span></td><td>%s</td><td>%s</td></tr>",
				html.EscapeString(c.Code), html.EscapeString(c.Name), html.EscapeString(strings.Join(rs, " · ")))
		}
		sb.WriteString("</tbody></table>")
	case 6:
		// Compact schedule across all resources for the week
		resCol := "Recurso"
		if isEn {
			resCol = "Resource"
		}
		sb.WriteString("<table><thead><tr><th>" + resCol + "</th>")
		for d := 1; d <= p.Time.MaxDays; d++ {
			fmt.Fprintf(&sb, "<th>%s</th>", html.EscapeString(p.Time.Days[d-1]))
		}
		sb.WriteString("</tr></thead><tbody>")
		for _, r := range p.Resources {
			if p.ResourceCategory(r.Category).NoTimetables {
				continue
			}
			fmt.Fprintf(&sb, "<tr><td><strong>%s</strong><br><span class=\"cell-meta\">%s</span></td>", html.EscapeString(r.Code), html.EscapeString(r.Name))
			for d := 1; d <= p.Time.MaxDays; d++ {
				items := []string{}
				for h := 1; h <= p.Time.MaxPeriods; h++ {
					for _, s := range p.Slots {
						if s.Week == week && s.Day == d && s.Period == h {
							c := courseMap[s.CourseID]
							hit := s.RoomID == r.ID
							for _, rid := range c.Resources {
								if rid == r.ID {
									hit = true
								}
							}
							if hit {
								items = append(items, fmt.Sprintf("P%d: %s", h, c.Code))
							}
						}
					}
				}
				fmt.Fprintf(&sb, "<td>%s</td>", html.EscapeString(strings.Join(items, ", ")))
			}
			sb.WriteString("</tr>")
		}
		sb.WriteString("</tbody></table>")
	default:
		// Report 3: Individual timetables per resource
		for _, r := range p.Resources {
			if p.ResourceCategory(r.Category).NoTimetables {
				continue
			}
			fmt.Fprintf(&sb, "<h2>%s — %s</h2><table><thead><tr><th>Bloque</th>", html.EscapeString(r.Code), html.EscapeString(r.Name))
			for d := 1; d <= p.Time.MaxDays; d++ {
				fmt.Fprintf(&sb, "<th>%s</th>", html.EscapeString(p.Time.Days[d-1]))
			}
			sb.WriteString("</tr></thead><tbody>")
			for h := 1; h <= p.Time.MaxPeriods; h++ {
				bell := fmt.Sprintf("P%d", h)
				if h-1 < len(p.Time.Bells) {
					bell = p.Time.Bells[h-1]
				}
				fmt.Fprintf(&sb, "<tr><th>%s</th>", html.EscapeString(bell))
				for d := 1; d <= p.Time.MaxDays; d++ {
					sb.WriteString("<td>")
					for _, s := range p.Slots {
						if s.Week == week && s.Day == d && s.Period == h {
							c := courseMap[s.CourseID]
							hit := s.RoomID == r.ID
							for _, rid := range c.Resources {
								if rid == r.ID {
									hit = true
								}
							}
							if hit {
								others := []string{}
								for _, rid := range c.Resources {
									if rid != r.ID {
										if rr, ok := resMap[rid]; ok {
											others = append(others, rr.Code)
										}
									}
								}
								if s.RoomID != "" && s.RoomID != r.ID {
									if rr, ok := resMap[s.RoomID]; ok {
										others = append(others, "["+rr.Code+"]")
									}
								}
								fmt.Fprintf(&sb, "<div class=\"cell-card\"><div class=\"cell-code\">%s</div><div class=\"cell-meta\">%s</div></div>",
									html.EscapeString(c.Code), html.EscapeString(strings.Join(others, " · ")))
							}
						}
					}
					sb.WriteString("</td>")
				}
				sb.WriteString("</tr>")
			}
			sb.WriteString("</tbody></table>")
		}
	}

	sb.WriteString("</body></html>")
	return sb.String()
}
