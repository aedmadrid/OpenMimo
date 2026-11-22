package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

type Conflict struct {
	Type        string `json:"type"` // "resource_overlap" | "booking_conflict" | "over_allocated" | "weekly_limit"
	Week        int    `json:"week"`
	Day         int    `json:"day"`
	Period      int    `json:"period"`
	ResourceID  string `json:"resourceId,omitempty"`
	Resource    string `json:"resource,omitempty"`
	CourseIDs   []string `json:"courseIds,omitempty"`
	CourseCodes []string `json:"courseCodes,omitempty"`
	Message     string `json:"message"`
}

type SlotAvailability struct {
	Day       int      `json:"day"`
	Period    int      `json:"period"`
	Available bool     `json:"available"`
	Reasons   []string `json:"reasons,omitempty"`
}

type OptimizeOptions struct {
	Action           string         `json:"action"` // "initial" | "optimise" | "auto_improve" | "copy" | "remove"
	FromWeek         int            `json:"fromWeek"`
	ToWeek           int            `json:"toWeek"`
	MinDay           int            `json:"minDay"`
	MaxDay           int            `json:"maxDay"`
	MinPeriod        int            `json:"minPeriod"`
	MaxPeriod        int            `json:"maxPeriod"`
	MinBlock         int            `json:"minBlock"`
	MaxBlock         int            `json:"maxBlock"`
	Passes           int            `json:"passes,omitempty"` // iterations for auto_improve (default 5)
	IncludeLocked    bool           `json:"includeLocked"`
	BookingsWithGaps bool           `json:"bookingsWithGaps"`
	GapWeights       map[int]int    `json:"gapWeights"`
	CourseIDs        []string       `json:"courseIds,omitempty"` // empty = all
	RandSeed         int64          `json:"randSeed"`
}

type OptimizeResult struct {
	Action         string   `json:"action"`
	Placed         int      `json:"placed"`
	Removed        int      `json:"removed"`
	Unscheduled    int      `json:"unscheduled"`
	GapsBefore     int      `json:"gapsBefore"`
	GapsAfter      int      `json:"gapsAfter"`
	GapsChange     int      `json:"gapsChange"`
	GapsPercent    float64  `json:"gapsPercent"`
	MovesEvaluated int      `json:"movesEvaluated"`
	ChangedCourses []string `json:"changedCourses"`
}

type ProjectStats struct {
	TotalCourses        int            `json:"totalCourses"`
	TotalResources      int            `json:"totalResources"`
	TotalCodes          int            `json:"totalCodes"`
	TotalLinks          int            `json:"totalLinks"`
	TotalPlanned        int            `json:"totalPlanned"`
	TotalAllocated      int            `json:"totalAllocated"`
	TotalScheduled      int            `json:"totalScheduled"`
	TotalUnscheduled    int            `json:"totalUnscheduled"`
	TotalBookings       int            `json:"totalBookings"`
	ManualRoomsCount    int            `json:"manualRoomsCount"`
	TotalGaps           int            `json:"totalGaps"`
	WeightedGaps        int            `json:"weightedGaps"`
	GapsByCategory      map[int]int    `json:"gapsByCategory"`
	ConflictCount       int            `json:"conflictCount"`
	Conflicts           []Conflict     `json:"conflicts"`
	MaxWeeklyLoad       []LoadEntry    `json:"maxWeeklyLoad"`
	MaxSlotLoad         []SlotLoadEntry `json:"maxSlotLoad"`
	WeekCompletion      map[int]float64 `json:"weekCompletion"`
	OverallCompletion   float64        `json:"overallCompletion"`
}

type LoadEntry struct {
	ResourceID   string `json:"resourceId"`
	ResourceCode string `json:"resourceCode"`
	ResourceName string `json:"resourceName"`
	Category     int    `json:"category"`
	Week         int    `json:"week"`
	Load         int    `json:"load"`
}

type SlotLoadEntry struct {
	Week   int `json:"week"`
	Day    int `json:"day"`
	Period int `json:"period"`
	Count  int `json:"count"`
}

// effectiveResources returns the resources active for a course at a specific slot (including manual room override).
func (p *Project) effectiveResources(c *Course, manualRoomID string) []string {
	if manualRoomID == "" {
		return c.Resources
	}
	resMap := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}
	out := make([]string, 0, len(c.Resources)+1)
	for _, rid := range c.Resources {
		if r, ok := resMap[rid]; ok && r.Category == p.RoomsCategory {
			continue // manual room replaces default room on this slot
		}
		out = append(out, rid)
	}
	out = append(out, manualRoomID)
	return out
}

// CheckSlotAvailability evaluates whether courseID can be scheduled at (week, day, period) without conflict.
// If ignoreSlotID is non-empty, that existing slot is ignored (useful when moving a slot).
func (p *Project) CheckSlotAvailability(courseID string, week, day, period int, manualRoomID string, ignoreSlotID string) (bool, []string) {
	c := p.CourseByID(courseID)
	if c == nil {
		return false, []string{"Curso no encontrado"}
	}
	reasons := []string{}

	resMap := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}
	courseMap := make(map[string]Course, len(p.Courses))
	for _, cc := range p.Courses {
		courseMap[cc.ID] = cc
	}

	effRes := p.effectiveResources(c, manualRoomID)
	effSet := make(map[string]bool, len(effRes))
	for _, rid := range effRes {
		if r, ok := resMap[rid]; ok && !p.ResourceCategory(r.Category).NoTimetables {
			effSet[rid] = true
		}
	}

	// Check bookings on course or any active resource
	for _, b := range p.Bookings {
		if b.Week != week {
			continue
		}
		if b.Scope == "timetable" && (b.Day != day || b.Period != period) {
			continue
		}
		if b.TargetType == "course" && b.TargetID == courseID {
			reasons = append(reasons, fmt.Sprintf("Reserva en curso %s", c.Code))
		}
		if b.TargetType == "resource" && effSet[b.TargetID] {
			reasons = append(reasons, fmt.Sprintf("No disponible: %s", resMap[b.TargetID].Code))
		}
	}

	// Check existing slots at (week, day, period)
	for _, s := range p.Slots {
		if s.ID == ignoreSlotID || s.Week != week || s.Day != day || s.Period != period {
			continue
		}
		if s.CourseID == courseID {
			reasons = append(reasons, fmt.Sprintf("%s ya está programado en este bloque", c.Code))
			continue
		}
		other := courseMap[s.CourseID]
		otherRes := p.effectiveResources(&other, s.RoomID)
		for _, rid := range otherRes {
			if effSet[rid] {
				r := resMap[rid]
				reasons = append(reasons, fmt.Sprintf("Conflicto %s con %s", r.Code, other.Code))
			}
		}
	}

	return len(reasons) == 0, reasons
}

func (p *Project) WhatIfGrid(courseID string, week int) []SlotAvailability {
	out := make([]SlotAvailability, 0, p.Time.MaxDays*p.Time.MaxPeriods)
	for d := 1; d <= p.Time.MaxDays; d++ {
		for h := 1; h <= p.Time.MaxPeriods; h++ {
			ok, reasons := p.CheckSlotAvailability(courseID, week, d, h, "", "")
			out = append(out, SlotAvailability{Day: d, Period: h, Available: ok, Reasons: reasons})
		}
	}
	return out
}

// ComputeGaps returns raw gap count, weighted gap score, and gaps per resource category across [fromWeek..toWeek].
func (p *Project) ComputeGaps(fromWeek, toWeek int, bookingsWithGaps bool, gapWeights map[int]int) (int, int, map[int]int) {
	if fromWeek < 1 {
		fromWeek = 1
	}
	if toWeek < fromWeek || toWeek > p.Time.MaxWeeks {
		toWeek = p.Time.MaxWeeks
	}
	resMap := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}
	courseMap := make(map[string]Course, len(p.Courses))
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}

	type key struct {
		resID string
		week  int
		day   int
	}
	occ := make(map[key]uint64) // bitmask of periods 1..30

	for _, s := range p.Slots {
		if s.Week < fromWeek || s.Week > toWeek {
			continue
		}
		c, ok := courseMap[s.CourseID]
		if !ok {
			continue
		}
		bit := uint64(1) << uint(s.Period)
		for _, rid := range p.effectiveResources(&c, s.RoomID) {
			occ[key{rid, s.Week, s.Day}] |= bit
		}
	}
	if bookingsWithGaps {
		for _, b := range p.Bookings {
			if b.Scope == "timetable" && b.TargetType == "resource" && b.Week >= fromWeek && b.Week <= toWeek {
				occ[key{b.TargetID, b.Week, b.Day}] |= uint64(1) << uint(b.Period)
			}
		}
	}

	totalGaps := 0
	weightedGaps := 0
	byCat := make(map[int]int)

	for k, mask := range occ {
		r, ok := resMap[k.resID]
		if !ok {
			continue
		}
		cat := p.ResourceCategory(r.Category)
		if cat.NoGaps || cat.NoTimetables {
			continue
		}
		first, last := 0, 0
		for h := 1; h <= p.Time.MaxPeriods; h++ {
			if (mask & (uint64(1) << uint(h))) != 0 {
				if first == 0 {
					first = h
				}
				last = h
			}
		}
		if last > first+1 {
			gaps := 0
			for h := first + 1; h < last; h++ {
				if (mask & (uint64(1) << uint(h))) == 0 {
					gaps++
				}
			}
			if gaps > 0 {
				w := cat.GapWeight
				if gapWeights != nil {
					if ov, ok := gapWeights[r.Category]; ok {
						w = ov
					}
				}
				if w < 1 {
					w = 1
				}
				totalGaps += gaps
				weightedGaps += gaps * w
				byCat[r.Category] += gaps
			}
		}
	}
	return totalGaps, weightedGaps, byCat
}

// DetectConflicts checks all timetable slots, bookings, and weekly limits.
func (p *Project) DetectConflicts() []Conflict {
	conflicts := []Conflict{}
	resMap := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}
	courseMap := make(map[string]Course, len(p.Courses))
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}

	type rSlotKey struct {
		resID  string
		week   int
		day    int
		period int
	}
	occ := make(map[rSlotKey][]string)
	for _, s := range p.Slots {
		c, ok := courseMap[s.CourseID]
		if !ok {
			continue
		}
		for _, rid := range p.effectiveResources(&c, s.RoomID) {
			r, ok := resMap[rid]
			if !ok || p.ResourceCategory(r.Category).NoTimetables {
				continue
			}
			k := rSlotKey{rid, s.Week, s.Day, s.Period}
			occ[k] = append(occ[k], c.ID)
		}
	}
	for k, cids := range occ {
		if len(cids) > 1 {
			r := resMap[k.resID]
			codes := make([]string, 0, len(cids))
			for _, cid := range cids {
				codes = append(codes, courseMap[cid].Code)
			}
			conflicts = append(conflicts, Conflict{
				Type:        "resource_overlap",
				Week:        k.week,
				Day:         k.day,
				Period:      k.period,
				ResourceID:  r.ID,
				Resource:    r.Code,
				CourseIDs:   cids,
				CourseCodes: codes,
				Message:     fmt.Sprintf("Choque de %s (%s) en S%d D%d B%d: %s", r.Code, r.Name, k.week, k.day, k.period, strings.Join(codes, " + ")),
			})
		}
	}

	// Booking conflicts
	for _, s := range p.Slots {
		c := courseMap[s.CourseID]
		eff := p.effectiveResources(&c, s.RoomID)
		effSet := make(map[string]bool, len(eff))
		for _, rid := range eff {
			effSet[rid] = true
		}
		for _, b := range p.Bookings {
			if b.Week != s.Week {
				continue
			}
			if b.Scope == "timetable" && (b.Day != s.Day || b.Period != s.Period) {
				continue
			}
			if (b.TargetType == "course" && b.TargetID == s.CourseID) || (b.TargetType == "resource" && effSet[b.TargetID]) {
				name := c.Code
				if b.TargetType == "resource" {
					name = resMap[b.TargetID].Code
				}
				conflicts = append(conflicts, Conflict{
					Type:        "booking_conflict",
					Week:        s.Week,
					Day:         s.Day,
					Period:      s.Period,
					CourseIDs:   []string{s.CourseID},
					CourseCodes: []string{c.Code},
					Message:     fmt.Sprintf("Curso %s programado sobre reserva de %s en S%d D%d B%d", c.Code, name, s.Week, s.Day, s.Period),
				})
			}
		}
	}

	// Weekly allocation limits warning per resource category
	for w := 1; w <= p.Time.MaxWeeks; w++ {
		resWeekLoad := make(map[string]int)
		for _, wa := range p.WeekAllocations {
			if wa.Week == w && wa.Periods > 0 {
				if c, ok := courseMap[wa.CourseID]; ok {
					for _, rid := range c.Resources {
						resWeekLoad[rid] += wa.Periods
					}
				}
			}
		}
		for rid, load := range resWeekLoad {
			r := resMap[rid]
			lim := p.ResourceCategory(r.Category).WeeklyLimit
			if lim > 0 && load > lim {
				conflicts = append(conflicts, Conflict{
					Type:       "weekly_limit",
					Week:       w,
					ResourceID: r.ID,
					Resource:   r.Code,
					Message:    fmt.Sprintf("%s supera el límite semanal (%d > %d) en S%d", r.Code, load, lim, w),
				})
			}
		}
	}

	return conflicts
}

// ConsecutiveViolationPenalty scores how many course runs violate per-course MinConsecutive / MaxConsecutive bounds.
func (p *Project) ConsecutiveViolationPenalty(fromWeek, toWeek int) int {
	type wkDay struct {
		courseID string
		week     int
		day      int
	}
	courseMap := make(map[string]Course, len(p.Courses))
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}
	runs := make(map[wkDay]uint64)
	for _, s := range p.Slots {
		if s.Week < fromWeek || s.Week > toWeek {
			continue
		}
		c, ok := courseMap[s.CourseID]
		if !ok || (c.MinConsecutive <= 0 && c.MaxConsecutive <= 0) {
			continue
		}
		runs[wkDay{s.CourseID, s.Week, s.Day}] |= uint64(1) << uint(s.Period)
	}
	penalty := 0
	for k, mask := range runs {
		c := courseMap[k.courseID]
		runLen := 0
		for h := 1; h <= p.Time.MaxPeriods+1; h++ {
			if h <= p.Time.MaxPeriods && (mask&(uint64(1)<<uint(h))) != 0 {
				runLen++
			} else if runLen > 0 {
				if c.MinConsecutive > 0 && runLen < c.MinConsecutive {
					penalty += (c.MinConsecutive - runLen) * 3
				}
				if c.MaxConsecutive > 0 && runLen > c.MaxConsecutive {
					penalty += (runLen - c.MaxConsecutive) * 4
				}
				runLen = 0
			}
		}
	}
	return penalty
}

// RunOptimizer implements OpenMimo's scheduling & self-improvement engine:
// "initial" (Initial solution), "optimise" (Gap minimization), "auto_improve" (Multi-pass self-improvement),
// "copy" (Copy across weeks), "remove" (Unschedule).
func (p *Project) RunOptimizer(opts OptimizeOptions) OptimizeResult {
	p.RecomputeDerived()
	if opts.FromWeek < 1 {
		opts.FromWeek = 1
	}
	if opts.ToWeek < opts.FromWeek || opts.ToWeek > p.Time.MaxWeeks {
		opts.ToWeek = p.Time.MaxWeeks
	}
	if opts.MinDay < 1 {
		opts.MinDay = 1
	}
	if opts.MaxDay < opts.MinDay || opts.MaxDay > p.Time.MaxDays {
		opts.MaxDay = p.Time.MaxDays
	}
	if opts.MinPeriod < 1 {
		opts.MinPeriod = 1
	}
	if opts.MaxPeriod < opts.MinPeriod || opts.MaxPeriod > p.Time.MaxPeriods {
		opts.MaxPeriod = p.Time.MaxPeriods
	}
	if opts.MinBlock < 1 {
		opts.MinBlock = 1 // MFW-FREE.EXE 0x564434 (MinSeq default = 1)
	}
	if opts.MaxBlock < opts.MinBlock {
		opts.MaxBlock = 3 // MFW-FREE.EXE 0x56445d (MaxSeq default = 3)
	}
	if opts.RandSeed == 0 {
		opts.RandSeed = 38746 // Default deterministic seed
	}

	// Self-improvement multi-pass loop ("auto_improve")
	if opts.Action == "auto_improve" {
		passes := opts.Passes
		if passes < 2 {
			passes = 5
		}
		if passes > 15 {
			passes = 15
		}
		gapsBefore, _, _ := p.ComputeGaps(opts.FromWeek, opts.ToWeek, opts.BookingsWithGaps, opts.GapWeights)
		totalPlaced := 0
		totalMoves := 0
		changedMap := make(map[string]bool)

		// Pass 0: schedule any unscheduled hours respecting per-course min/max consecutive rules
		initOpts := opts
		initOpts.Action = "initial"
		r0 := p.RunOptimizer(initOpts)
		totalPlaced += r0.Placed
		totalMoves += r0.MovesEvaluated
		for _, cc := range r0.ChangedCourses {
			changedMap[cc] = true
		}

		scoreOf := func(cand *Project) int {
			_, wg, _ := cand.ComputeGaps(opts.FromWeek, opts.ToWeek, opts.BookingsWithGaps, opts.GapWeights)
			cp := cand.ConsecutiveViolationPenalty(opts.FromWeek, opts.ToWeek)
			cf := len(cand.DetectConflicts())
			return cf*500 + wg*10 + cp*15
		}

		best := p.Clone()
		bestScore := scoreOf(best)

		for pass := 0; pass < passes; pass++ {
			cand := best.Clone()
			optPass := opts
			optPass.Action = "optimise"
			optPass.RandSeed = opts.RandSeed + int64((pass+1)*7919)
			pr := cand.RunOptimizer(optPass)
			totalMoves += pr.MovesEvaluated
			sc := scoreOf(cand)
			if sc < bestScore {
				best = cand
				bestScore = sc
				for _, cc := range pr.ChangedCourses {
					changedMap[cc] = true
				}
			}
		}

		p.Slots = best.Slots
		p.RecomputeDerived()
		gapsAfter, _, _ := p.ComputeGaps(opts.FromWeek, opts.ToWeek, opts.BookingsWithGaps, opts.GapWeights)
		changedList := make([]string, 0, len(changedMap))
		for code := range changedMap {
			changedList = append(changedList, code)
		}
		sort.Strings(changedList)
		pct := 0.0
		if gapsBefore > 0 {
			pct = float64(gapsBefore-gapsAfter) * 100.0 / float64(gapsBefore)
		}
		unsched := 0
		for _, c := range p.Courses {
			for w := opts.FromWeek; w <= opts.ToWeek; w++ {
				alloc := p.WeekAlloc(c.ID, w)
				sc := 0
				for _, s := range p.Slots {
					if s.CourseID == c.ID && s.Week == w {
						sc++
					}
				}
				if alloc > sc {
					unsched += alloc - sc
				}
			}
		}
		return OptimizeResult{
			Action:         "auto_improve",
			Placed:         totalPlaced,
			Unscheduled:    unsched,
			GapsBefore:     gapsBefore,
			GapsAfter:      gapsAfter,
			GapsChange:     gapsBefore - gapsAfter,
			GapsPercent:    pct,
			MovesEvaluated: totalMoves,
			ChangedCourses: changedList,
		}
	}

	rng := rand.New(rand.NewSource(opts.RandSeed))

	selectedCourses := make(map[string]bool)
	if len(opts.CourseIDs) > 0 {
		for _, id := range opts.CourseIDs {
			selectedCourses[id] = true
		}
	} else {
		for _, c := range p.Courses {
			selectedCourses[c.ID] = true
		}
	}
	isEligible := func(c Course) bool {
		if !selectedCourses[c.ID] {
			return false
		}
		if c.Locked && !opts.IncludeLocked {
			return false
		}
		return true
	}

	gapsBefore, weightedBefore, _ := p.ComputeGaps(opts.FromWeek, opts.ToWeek, opts.BookingsWithGaps, opts.GapWeights)
	res := OptimizeResult{
		Action:         opts.Action,
		GapsBefore:     gapsBefore,
		ChangedCourses: []string{},
	}
	changedSet := make(map[string]bool)

	switch opts.Action {
	case "remove":
		kept := make([]SlotAssignment, 0, len(p.Slots))
		for _, s := range p.Slots {
			c := p.CourseByID(s.CourseID)
			if c != nil && isEligible(*c) && s.Week >= opts.FromWeek && s.Week <= opts.ToWeek &&
				s.Day >= opts.MinDay && s.Day <= opts.MaxDay && s.Period >= opts.MinPeriod && s.Period <= opts.MaxPeriod {
				res.Removed++
				changedSet[c.Code] = true
				continue
			}
			kept = append(kept, s)
		}
		p.Slots = kept

	case "copy":
		// Copy schedule from FromWeek to FromWeek+1..ToWeek where conflicts allow & allocations permit
		srcWeek := opts.FromWeek
		srcSlots := []SlotAssignment{}
		for _, s := range p.Slots {
			c := p.CourseByID(s.CourseID)
			if s.Week == srcWeek && c != nil && isEligible(*c) {
				srcSlots = append(srcSlots, s)
			}
		}
		seq := len(p.Slots) + 1
		for w := srcWeek + 1; w <= opts.ToWeek; w++ {
			if w <= p.Time.ProtectedWeek {
				continue
			}
			for _, src := range srcSlots {
				// Check remaining allocation in target week
				alloc := p.WeekAlloc(src.CourseID, w)
				sched := 0
				for _, ex := range p.Slots {
					if ex.CourseID == src.CourseID && ex.Week == w {
						sched++
					}
				}
				if sched >= alloc {
					continue
				}
				ok, _ := p.CheckSlotAvailability(src.CourseID, w, src.Day, src.Period, src.RoomID, "")
				if ok {
					seq++
					p.Slots = append(p.Slots, SlotAssignment{
						ID:       fmt.Sprintf("s_cp_%d", seq),
						CourseID: src.CourseID,
						Week:     w,
						Day:      src.Day,
						Period:   src.Period,
						RoomID:   src.RoomID,
					})
					res.Placed++
					if c := p.CourseByID(src.CourseID); c != nil {
						changedSet[c.Code] = true
					}
				}
			}
		}

	case "initial":
		// Place all unscheduled periods using Most-Constrained-First + Gap-Aware Best Slot placement!
		// Sort eligible courses descending by (Links, Allocated) so hardest courses schedule first.
		courses := make([]Course, 0, len(p.Courses))
		for _, c := range p.Courses {
			if isEligible(c) {
				courses = append(courses, c)
			}
		}
		sort.SliceStable(courses, func(i, j int) bool {
			if courses[i].Links != courses[j].Links {
				return courses[i].Links > courses[j].Links
			}
			return courses[i].Planned > courses[j].Planned
		})

		slotSeq := len(p.Slots) + 100
		for w := opts.FromWeek; w <= opts.ToWeek; w++ {
			if w <= p.Time.ProtectedWeek {
				continue
			}
			for _, c := range courses {
				alloc := p.WeekAlloc(c.ID, w)
				if alloc <= 0 {
					continue
				}
				sched := 0
				dayUsed := make(map[int]int)
				for _, s := range p.Slots {
					if s.CourseID == c.ID && s.Week == w {
						sched++
						dayUsed[s.Day]++
					}
				}
				rem := alloc - sched
				if rem <= 0 {
					continue
				}

				// Per-course consecutive hours rules (fallback to global MinBlock / MaxBlock)
				cMin := opts.MinBlock
				cMax := opts.MaxBlock
				if c.MinConsecutive > 0 {
					cMin = c.MinConsecutive
				}
				if c.MaxConsecutive > 0 {
					cMax = c.MaxConsecutive
				}
				if cMax < cMin {
					cMax = cMin
				}
				prefBlock := 2
				if prefBlock < cMin {
					prefBlock = cMin
				}
				if prefBlock > cMax {
					prefBlock = cMax
				}

				for rem > 0 {
					blockLen := prefBlock
					if blockLen > rem {
						blockLen = rem
					}
					placedBlock := false
					// Try blockLen down to cMin first, then fallback down to 1 only if impossible
					for bl := blockLen; bl >= 1 && !placedBlock; bl-- {
						bestScore := 1 << 30
						bestD, bestH := -1, -1

						// Evaluate all valid (day, startPeriod) placements
						for d := opts.MinDay; d <= opts.MaxDay; d++ {
							// Respect per-course maxConsecutive if already present on day d
							if cMax > 0 && dayUsed[d]+bl > cMax && rem >= bl {
								// allow soft penalty instead of hard deadlock only if no other day fits
							}
							for h := opts.MinPeriod; h+bl-1 <= opts.MaxPeriod; h++ {
								res.MovesEvaluated++
								allOk := true
								for k := 0; k < bl; k++ {
									ok, _ := p.CheckSlotAvailability(c.ID, w, d, h+k, "", "")
									if !ok {
										allOk = false
										break
									}
								}
								if !allOk {
									continue
								}
								// Score: prefer spreading across days + obey per-course min/max consecutive + minimize gaps
								score := dayUsed[d]*25 + h*2 + rng.Intn(3)
								if bl < cMin {
									score += (cMin - bl) * 40
								}
								if dayUsed[d]+bl > cMax {
									score += (dayUsed[d] + bl - cMax) * 60
								}
								// Check adjacency with existing slots of course's resources to avoid creating gaps!
								adjBonus := 0
								for _, s := range p.Slots {
									if s.Week == w && s.Day == d {
										if s.Period == h-1 || s.Period == h+bl {
											// shared resource?
											other := p.CourseByID(s.CourseID)
											if other != nil && sharesResource(c.Resources, other.Resources) {
												adjBonus += 6
											}
										}
									}
								}
								score -= adjBonus
								if score < bestScore {
									bestScore = score
									bestD, bestH = d, h
								}
							}
						}

						if bestD > 0 {
							for k := 0; k < bl; k++ {
								slotSeq++
								p.Slots = append(p.Slots, SlotAssignment{
									ID:       fmt.Sprintf("s_opt_%d", slotSeq),
									CourseID: c.ID,
									Week:     w,
									Day:      bestD,
									Period:   bestH + k,
								})
								res.Placed++
							}
							dayUsed[bestD] += bl
							rem -= bl
							changedSet[c.Code] = true
							placedBlock = true
						}
					}
					if !placedBlock {
						res.Unscheduled += rem
						break
					}
				}
			}
		}

	case "optimise":
		// Iterative Gap & Consecutive-Block Minimization (up to 10,000 candidate evaluations matching MFW-FREE.EXE 0x57fe9f)
		curObj := weightedBefore + p.ConsecutiveViolationPenalty(opts.FromWeek, opts.ToWeek)*4
		maxMoves := 1000
		if len(p.Slots) > 0 {
			for iter := 0; iter < maxMoves && res.MovesEvaluated < 10000 && curObj > 0; iter++ {
				// Pick an eligible slot index
				idx := rng.Intn(len(p.Slots))
				s := p.Slots[idx]
				if s.Week < opts.FromWeek || s.Week > opts.ToWeek || s.Week <= p.Time.ProtectedWeek {
					continue
				}
				c := p.CourseByID(s.CourseID)
				if c == nil || !isEligible(*c) {
					continue
				}

				origD, origH := s.Day, s.Period
				_, oldLocalW, _ := p.ComputeGaps(s.Week, s.Week, opts.BookingsWithGaps, opts.GapWeights)
				oldLocalPen := p.ConsecutiveViolationPenalty(s.Week, s.Week) * 4
				oldLocal := oldLocalW + oldLocalPen

				bestD, bestH := origD, origH
				bestObj := curObj

				for tries := 0; tries < 10; tries++ {
					candD := opts.MinDay + rng.Intn(opts.MaxDay-opts.MinDay+1)
					candH := opts.MinPeriod + rng.Intn(opts.MaxPeriod-opts.MinPeriod+1)
					if candD == origD && candH == origH {
						continue
					}
					res.MovesEvaluated++
					ok, _ := p.CheckSlotAvailability(s.CourseID, s.Week, candD, candH, s.RoomID, s.ID)
					if !ok {
						continue
					}
					p.Slots[idx].Day = candD
					p.Slots[idx].Period = candH
					_, newW, _ := p.ComputeGaps(s.Week, s.Week, opts.BookingsWithGaps, opts.GapWeights)
					newPen := p.ConsecutiveViolationPenalty(s.Week, s.Week) * 4
					p.Slots[idx].Day = origD
					p.Slots[idx].Period = origH
					delta := (newW + newPen) - oldLocal
					if curObj+delta < bestObj {
						bestObj = curObj + delta
						bestD, bestH = candD, candH
					}
				}

				if bestD != origD || bestH != origH {
					p.Slots[idx].Day = bestD
					p.Slots[idx].Period = bestH
					curObj = bestObj
					changedSet[c.Code] = true
				}
			}
		}
	}

	p.RecomputeDerived()
	gapsAfter, _, _ := p.ComputeGaps(opts.FromWeek, opts.ToWeek, opts.BookingsWithGaps, opts.GapWeights)
	res.GapsAfter = gapsAfter
	res.GapsChange = gapsBefore - gapsAfter
	if gapsBefore > 0 {
		res.GapsPercent = float64(res.GapsChange) * 100.0 / float64(gapsBefore)
	}
	for code := range changedSet {
		res.ChangedCourses = append(res.ChangedCourses, code)
	}
	sort.Strings(res.ChangedCourses)
	// Count remaining unscheduled across range
	unsched := 0
	for _, c := range p.Courses {
		for w := opts.FromWeek; w <= opts.ToWeek; w++ {
			alloc := p.WeekAlloc(c.ID, w)
			sc := 0
			for _, s := range p.Slots {
				if s.CourseID == c.ID && s.Week == w {
					sc++
				}
			}
			if alloc > sc {
				unsched += alloc - sc
			}
		}
	}
	res.Unscheduled = unsched
	return res
}

func sharesResource(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

// AvailableRooms returns rooms (category == RoomsCategory) that have NO conflict or booking at (week, day, period).
func (p *Project) AvailableRooms(week, day, period int, ignoreSlotID string) []Resource {
	busy := make(map[string]bool)
	courseMap := make(map[string]Course, len(p.Courses))
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}
	for _, s := range p.Slots {
		if s.ID == ignoreSlotID || s.Week != week || s.Day != day || s.Period != period {
			continue
		}
		if s.RoomID != "" {
			busy[s.RoomID] = true
		} else if c, ok := courseMap[s.CourseID]; ok {
			for _, rid := range c.Resources {
				busy[rid] = true
			}
		}
	}
	for _, b := range p.Bookings {
		if b.TargetType == "resource" && b.Week == week {
			if b.Scope == "week" || (b.Day == day && b.Period == period) {
				busy[b.TargetID] = true
			}
		}
	}

	out := []Resource{}
	for _, r := range p.Resources {
		if r.Category == p.RoomsCategory && !busy[r.ID] {
			out = append(out, r)
		}
	}
	return out
}

// ImitateRooms assigns rooms automatically to slots without manual rooms, preferring rooms already used by the same course.
func (p *Project) ImitateRooms(fromWeek, toWeek int) int {
	if fromWeek < 1 {
		fromWeek = 1
	}
	if toWeek < fromWeek || toWeek > p.Time.MaxWeeks {
		toWeek = p.Time.MaxWeeks
	}
	// Collect preferred rooms per course from existing manual rooms or default collection rooms
	pref := make(map[string][]string)
	resMap := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}
	for _, s := range p.Slots {
		if s.RoomID != "" {
			pref[s.CourseID] = append(pref[s.CourseID], s.RoomID)
		}
	}
	for _, c := range p.Courses {
		for _, rid := range c.Resources {
			if r, ok := resMap[rid]; ok && r.Category == p.RoomsCategory {
				pref[c.ID] = append(pref[c.ID], rid)
			}
		}
	}

	assigned := 0
	for i := range p.Slots {
		s := &p.Slots[i]
		if s.Week < fromWeek || s.Week > toWeek || s.RoomID != "" {
			continue
		}
		avail := p.AvailableRooms(s.Week, s.Day, s.Period, s.ID)
		if len(avail) == 0 {
			continue
		}
		availSet := make(map[string]bool, len(avail))
		for _, r := range avail {
			availSet[r.ID] = true
		}
		chosen := ""
		for _, candidate := range pref[s.CourseID] {
			if availSet[candidate] {
				chosen = candidate
				break
			}
		}
		if chosen == "" {
			chosen = avail[0].ID
		}
		s.RoomID = chosen
		pref[s.CourseID] = append([]string{chosen}, pref[s.CourseID]...)
		assigned++
	}
	p.RecomputeDerived()
	return assigned
}

func (p *Project) ComputeStats() ProjectStats {
	p.RecomputeDerived()
	conflicts := p.DetectConflicts()
	totalGaps, weightedGaps, gapsByCat := p.ComputeGaps(1, p.Time.MaxWeeks, false, nil)

	st := ProjectStats{
		TotalCourses:   len(p.Courses),
		TotalResources: len(p.Resources),
		TotalCodes:     len(p.Courses) + len(p.Resources),
		TotalBookings:  len(p.Bookings),
		TotalGaps:      totalGaps,
		WeightedGaps:   weightedGaps,
		GapsByCategory: gapsByCat,
		ConflictCount:  len(conflicts),
		Conflicts:      conflicts,
		WeekCompletion: make(map[int]float64),
	}

	for _, c := range p.Courses {
		st.TotalLinks += c.Links
		st.TotalPlanned += c.Planned
		st.TotalAllocated += c.Allocated
		st.TotalScheduled += c.Scheduled
		if c.Left > 0 {
			st.TotalUnscheduled += c.Left
		}
	}
	for _, s := range p.Slots {
		if s.RoomID != "" {
			st.ManualRoomsCount++
		}
	}
	if st.TotalAllocated > 0 {
		st.OverallCompletion = float64(st.TotalScheduled) * 100.0 / float64(st.TotalAllocated)
	}

	// Per-week completion
	for w := 1; w <= p.Time.MaxWeeks; w++ {
		wAlloc, wSched := 0, 0
		for _, wa := range p.WeekAllocations {
			if wa.Week == w {
				wAlloc += wa.Periods
			}
		}
		for _, s := range p.Slots {
			if s.Week == w {
				wSched++
			}
		}
		if wAlloc > 0 {
			st.WeekCompletion[w] = float64(wSched) * 100.0 / float64(wAlloc)
		} else {
			st.WeekCompletion[w] = 100.0
		}
	}

	// Max Weekly Load (top 10)
	courseMap := make(map[string]Course, len(p.Courses))
	for _, c := range p.Courses {
		courseMap[c.ID] = c
	}
	type rwKey struct {
		rid  string
		week int
	}
	rwLoad := make(map[rwKey]int)
	for _, wa := range p.WeekAllocations {
		if c, ok := courseMap[wa.CourseID]; ok {
			for _, rid := range c.Resources {
				rwLoad[rwKey{rid, wa.Week}] += wa.Periods
			}
		}
	}
	resMap := make(map[string]Resource, len(p.Resources))
	for _, r := range p.Resources {
		resMap[r.ID] = r
	}
	for k, load := range rwLoad {
		if r, ok := resMap[k.rid]; ok && !p.ResourceCategory(r.Category).NoTimetables {
			st.MaxWeeklyLoad = append(st.MaxWeeklyLoad, LoadEntry{
				ResourceID: r.ID, ResourceCode: r.Code, ResourceName: r.Name, Category: r.Category, Week: k.week, Load: load,
			})
		}
	}
	sort.Slice(st.MaxWeeklyLoad, func(i, j int) bool { return st.MaxWeeklyLoad[i].Load > st.MaxWeeklyLoad[j].Load })
	if len(st.MaxWeeklyLoad) > 10 {
		st.MaxWeeklyLoad = st.MaxWeeklyLoad[:10]
	}

	// Max Slot Load (top 10)
	type slotKey struct{ w, d, h int }
	slLoad := make(map[slotKey]int)
	for _, s := range p.Slots {
		slLoad[slotKey{s.Week, s.Day, s.Period}]++
	}
	for k, cnt := range slLoad {
		st.MaxSlotLoad = append(st.MaxSlotLoad, SlotLoadEntry{Week: k.w, Day: k.d, Period: k.h, Count: cnt})
	}
	sort.Slice(st.MaxSlotLoad, func(i, j int) bool { return st.MaxSlotLoad[i].Count > st.MaxSlotLoad[j].Count })
	if len(st.MaxSlotLoad) > 10 {
		st.MaxSlotLoad = st.MaxSlotLoad[:10]
	}

	return st
}
