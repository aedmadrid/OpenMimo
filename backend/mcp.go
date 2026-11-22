package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MCP (Model Context Protocol) JSON-RPC 2.0 server — compatible with any AI Agent
// over BOTH stdio (`openmimo-backend -mcp`) and live HTTP (`POST /mcp`).

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func mcpToolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name":        "get_project_summary",
			"description": "Return current OpenMimo timetable metadata, counts, completion percentages, gaps, and conflicts.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "list_courses",
			"description": "List all courses/events with codes, names, linked resources, planned, allocated, and scheduled hours.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "list_resources",
			"description": "List all resources (groups, teachers, rooms, subjects, equipment) with category and load stats.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{"type": "integer", "description": "Optional category index (1..7)"},
				},
			},
		},
		{
			"name":        "create_course",
			"description": "Create a new course/event and optionally link resource IDs.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"code", "name"},
				"properties": map[string]any{
					"code":           map[string]any{"type": "string"},
					"name":           map[string]any{"type": "string"},
					"category":       map[string]any{"type": "integer"},
					"planned":        map[string]any{"type": "integer"},
					"minConsecutive": map[string]any{"type": "integer", "description": "Minimum consecutive hours per block (0 = default)"},
					"maxConsecutive": map[string]any{"type": "integer", "description": "Maximum consecutive hours per block (0 = default)"},
					"resources":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
			},
		},
		{
			"name":        "create_resource",
			"description": "Create a new resource (group, teacher, room, subject, student, equipment).",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"code", "name"},
				"properties": map[string]any{
					"code":     map[string]any{"type": "string"},
					"name":     map[string]any{"type": "string"},
					"category": map[string]any{"type": "integer", "description": "1=Groups, 2=Teachers, 3=Rooms, 4=Subjects, 5=Students, 6=Equipment, 7=Other"},
					"planned":  map[string]any{"type": "integer"},
				},
			},
		},
		{
			"name":        "set_week_allocation",
			"description": "Set the number of periods allocated to a course in a specific week.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"courseId", "week", "periods"},
				"properties": map[string]any{
					"courseId": map[string]any{"type": "string"},
					"week":     map[string]any{"type": "integer"},
					"periods":  map[string]any{"type": "integer"},
				},
			},
		},
		{
			"name":        "check_availability",
			"description": "Return What-If slot availability and conflict reasons for a course in a given week.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"courseId", "week"},
				"properties": map[string]any{
					"courseId": map[string]any{"type": "string"},
					"week":     map[string]any{"type": "integer"},
				},
			},
		},
		{
			"name":        "place_slot",
			"description": "Schedule a course at (week, day, period) with automatic conflict checking.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"courseId", "week", "day", "period"},
				"properties": map[string]any{
					"courseId": map[string]any{"type": "string"},
					"week":     map[string]any{"type": "integer"},
					"day":      map[string]any{"type": "integer"},
					"period":   map[string]any{"type": "integer"},
					"roomId":   map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "optimize_timetable",
			"description": "Run OpenMimo scheduler/optimizer: action='initial' (place unscheduled), 'optimise' (minimize gaps), 'auto_improve' (multi-pass self-improvement), 'copy' (copy across weeks), or 'remove'.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"action"},
				"properties": map[string]any{
					"action":    map[string]any{"type": "string", "enum": []string{"initial", "optimise", "auto_improve", "copy", "remove"}},
					"passes":    map[string]any{"type": "integer", "description": "Number of self-improvement passes (2..15)"},
					"fromWeek":  map[string]any{"type": "integer"},
					"toWeek":    map[string]any{"type": "integer"},
					"minDay":    map[string]any{"type": "integer"},
					"maxDay":    map[string]any{"type": "integer"},
					"minPeriod": map[string]any{"type": "integer"},
					"maxPeriod": map[string]any{"type": "integer"},
				},
			},
		},
		{
			"name":        "set_teacher_unavailability",
			"description": "Mark a teacher/resource as unavailable (or available) on a specific day and period across one or all weeks.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"resourceId", "day", "period", "unavailable"},
				"properties": map[string]any{
					"resourceId":  map[string]any{"type": "string"},
					"week":        map[string]any{"type": "integer", "description": "0 or omitted = apply to all weeks"},
					"day":         map[string]any{"type": "integer", "description": "1..7 (1=Monday)"},
					"period":      map[string]any{"type": "integer", "description": "1..30"},
					"unavailable": map[string]any{"type": "boolean"},
					"label":       map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "set_mcp_notes",
			"description": "Attach or update MCP scheduling notes/instructions on a teacher/resource, course, or the entire project.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"targetType", "notes"},
				"properties": map[string]any{
					"targetType": map[string]any{"type": "string", "enum": []string{"project", "resource", "course"}},
					"targetId":   map[string]any{"type": "string"},
					"notes":      map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "detect_conflicts",
			"description": "Run full conflict, booking violation, and weekly-limit diagnostics.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "imitate_rooms",
			"description": "Automatically assign free rooms to scheduled slots by imitating previous room choices.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fromWeek": map[string]any{"type": "integer"},
					"toWeek":   map[string]any{"type": "integer"},
				},
			},
		},
		{
			"name":        "import_file",
			"description": "Import a native .omimo file, legacy Mimosa .mfw binary file, .mxt text file, or .json file from disk.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"path"},
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "export_timetable",
			"description": "Export the current schedule as omimo, mxt, csv, ics, html, or json (optionally saving to outputPath).",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"format"},
				"properties": map[string]any{
					"format":     map[string]any{"type": "string", "enum": []string{"omimo", "mxt", "csv", "ics", "html", "json"}},
					"reportType": map[string]any{"type": "integer"},
					"week":       map[string]any{"type": "integer"},
					"outputPath": map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "list_scripts",
			"description": "List all saved JavaScript automation scripts in the OpenMimo Script Library.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "save_script",
			"description": "Create or update a JavaScript automation script in the OpenMimo Script Library.",
			"inputSchema": map[string]any{
				"type":     "object",
				"required": []string{"name", "code"},
				"properties": map[string]any{
					"id":          map[string]any{"type": "string"},
					"name":        map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"author":      map[string]any{"type": "string"},
					"code":        map[string]any{"type": "string"},
				},
			},
		},
	}
}

func (s *AppState) handleMCPRequest(req mcpRequest) *mcpResponse {
	// Notifications without ID return no response
	if len(req.ID) == 0 && strings.HasPrefix(req.Method, "notifications/") {
		return nil
	}

	okResp := func(res any) *mcpResponse {
		return &mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: res}
	}
	errResp := func(code int, msg string) *mcpResponse {
		return &mcpResponse{JSONRPC: "2.0", ID: req.ID, Error: &mcpError{Code: code, Message: msg}}
	}

	switch req.Method {
	case "initialize":
		return okResp(map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools":     map[string]any{"listChanged": false},
				"resources": map[string]any{"subscribe": false, "listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "openmimo",
				"version": "1.0.0",
			},
		})
	case "ping":
		return okResp(map[string]any{})
	case "tools/list":
		return okResp(map[string]any{"tools": mcpToolDefinitions()})
	case "resources/list":
		return okResp(map[string]any{
			"resources": []map[string]any{
				{
					"uri":         "openmimo://project/current",
					"name":        "Current OpenMimo Project",
					"description": "Full JSON representation of the active timetable project and live statistics.",
					"mimeType":    "application/json",
				},
			},
		})
	case "resources/read":
		s.mu.Lock()
		payload := s.responsePayload()
		s.mu.Unlock()
		b, _ := json.MarshalIndent(payload, "", "  ")
		return okResp(map[string]any{
			"contents": []map[string]any{
				{
					"uri":      "openmimo://project/current",
					"mimeType": "application/json",
					"text":     string(b),
				},
			},
		})
	case "tools/call":
		var call struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &call); err != nil {
			return errResp(-32602, "invalid tool call params: "+err.Error())
		}
		result, isErr := s.executeMCPTool(call.Name, call.Arguments)
		text, _ := json.MarshalIndent(result, "", "  ")
		return okResp(map[string]any{
			"isError": isErr,
			"content": []map[string]any{
				{"type": "text", "text": string(text)},
			},
		})
	default:
		return errResp(-32601, "method not found: "+req.Method)
	}
}

func (s *AppState) executeMCPTool(name string, rawArgs json.RawMessage) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.project

	switch name {
	case "get_project_summary":
		return map[string]any{
			"name":        p.Name,
			"institution": p.Institution,
			"department":  p.Department,
			"language":    p.Language,
			"mcpNotes":    p.McpNotes,
			"time":        p.Time,
			"stats":       p.ComputeStats(),
		}, false

	case "list_courses":
		p.RecomputeDerived()
		return p.Courses, false

	case "list_resources":
		var args struct{ Category int `json:"category"` }
		_ = json.Unmarshal(rawArgs, &args)
		p.RecomputeDerived()
		if args.Category <= 0 {
			return p.Resources, false
		}
		out := []Resource{}
		for _, r := range p.Resources {
			if r.Category == args.Category {
				out = append(out, r)
			}
		}
		return out, false

	case "create_course":
		var args Course
		if err := json.Unmarshal(rawArgs, &args); err != nil || strings.TrimSpace(args.Code) == "" {
			return map[string]any{"error": "code is required"}, true
		}
		s.pushUndo()
		if args.ID == "" {
			args.ID = fmt.Sprintf("c_mcp_%d", len(p.Courses)+1)
		}
		if args.Category < 1 || args.Category > 7 {
			args.Category = 1
		}
		if args.Resources == nil {
			args.Resources = []string{}
		}
		p.Courses = append(p.Courses, args)
		p.RecomputeDerived()
		return args, false

	case "create_resource":
		var args Resource
		if err := json.Unmarshal(rawArgs, &args); err != nil || strings.TrimSpace(args.Code) == "" {
			return map[string]any{"error": "code is required"}, true
		}
		s.pushUndo()
		if args.ID == "" {
			args.ID = fmt.Sprintf("r_mcp_%d", len(p.Resources)+1)
		}
		if args.Category < 1 || args.Category > 7 {
			args.Category = 1
		}
		p.Resources = append(p.Resources, args)
		p.RecomputeDerived()
		return args, false

	case "set_week_allocation":
		var args struct {
			CourseID string `json:"courseId"`
			Week     int    `json:"week"`
			Periods  int    `json:"periods"`
		}
		if err := json.Unmarshal(rawArgs, &args); err != nil || args.CourseID == "" {
			return map[string]any{"error": "courseId, week, periods required"}, true
		}
		s.pushUndo()
		p.SetWeekAlloc(args.CourseID, args.Week, args.Periods)
		p.RecomputeDerived()
		return map[string]any{"ok": true, "courseId": args.CourseID, "week": args.Week, "periods": args.Periods}, false

	case "check_availability":
		var args struct {
			CourseID string `json:"courseId"`
			Week     int    `json:"week"`
		}
		if err := json.Unmarshal(rawArgs, &args); err != nil || args.CourseID == "" {
			return map[string]any{"error": "courseId and week required"}, true
		}
		if args.Week < 1 {
			args.Week = 1
		}
		return p.WhatIfGrid(args.CourseID, args.Week), false

	case "place_slot":
		var args struct {
			CourseID string `json:"courseId"`
			Week     int    `json:"week"`
			Day      int    `json:"day"`
			Period   int    `json:"period"`
			RoomID   string `json:"roomId"`
		}
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return map[string]any{"error": err.Error()}, true
		}
		if ok, reasons := p.CheckSlotAvailability(args.CourseID, args.Week, args.Day, args.Period, args.RoomID, ""); !ok {
			return map[string]any{"error": "conflict", "reasons": reasons}, true
		}
		s.pushUndo()
		slot := SlotAssignment{
			ID:       fmt.Sprintf("s_mcp_%d", len(p.Slots)+1),
			CourseID: args.CourseID,
			Week:     args.Week,
			Day:      args.Day,
			Period:   args.Period,
			RoomID:   args.RoomID,
		}
		p.Slots = append(p.Slots, slot)
		p.RecomputeDerived()
		return slot, false

	case "optimize_timetable":
		var opts OptimizeOptions
		_ = json.Unmarshal(rawArgs, &opts)
		if opts.Action == "" {
			opts.Action = "optimise"
		}
		s.pushUndo()
		res := p.RunOptimizer(opts)
		return res, false

	case "set_teacher_unavailability":
		var args struct {
			ResourceID  string `json:"resourceId"`
			Week        int    `json:"week"`
			Day         int    `json:"day"`
			Period      int    `json:"period"`
			Unavailable bool   `json:"unavailable"`
			Label       string `json:"label"`
		}
		if err := json.Unmarshal(rawArgs, &args); err != nil || args.ResourceID == "" {
			return map[string]any{"error": "resourceId, day, period required"}, true
		}
		if args.Label == "" {
			args.Label = "No disponible"
		}
		s.pushUndo()
		wStart, wEnd := args.Week, args.Week
		if args.Week <= 0 {
			wStart, wEnd = 1, p.Time.MaxWeeks
		}
		for wk := wStart; wk <= wEnd; wk++ {
			for i := len(p.Bookings) - 1; i >= 0; i-- {
				b := p.Bookings[i]
				if b.TargetType == "resource" && b.TargetID == args.ResourceID && b.Scope == "timetable" &&
					b.Week == wk && b.Day == args.Day && b.Period == args.Period {
					p.Bookings = append(p.Bookings[:i], p.Bookings[i+1:]...)
				}
			}
			if args.Unavailable {
				p.Bookings = append(p.Bookings, Booking{
					ID: fmt.Sprintf("b_mcp_%d_%d_%d", wk, args.Day, args.Period),
					TargetType: "resource", TargetID: args.ResourceID,
					Scope: "timetable", Week: wk, Day: args.Day, Period: args.Period, Kind: 1, Label: args.Label,
				})
			}
		}
		s.afterMutateLocked()
		return map[string]any{"ok": true, "resourceId": args.ResourceID, "unavailable": args.Unavailable}, false

	case "set_mcp_notes":
		var args struct {
			TargetType string `json:"targetType"`
			TargetID   string `json:"targetId"`
			Notes      string `json:"notes"`
		}
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return map[string]any{"error": err.Error()}, true
		}
		s.pushUndo()
		switch args.TargetType {
		case "project":
			p.McpNotes = args.Notes
		case "resource":
			if r := p.ResourceByID(args.TargetID); r != nil {
				r.McpNotes = args.Notes
			}
		case "course":
			if c := p.CourseByID(args.TargetID); c != nil {
				c.McpNotes = args.Notes
			}
		}
		s.afterMutateLocked()
		return map[string]any{"ok": true}, false

	case "detect_conflicts":
		return map[string]any{"conflicts": p.DetectConflicts()}, false

	case "imitate_rooms":
		var args struct {
			FromWeek int `json:"fromWeek"`
			ToWeek   int `json:"toWeek"`
		}
		_ = json.Unmarshal(rawArgs, &args)
		s.pushUndo()
		assigned := p.ImitateRooms(args.FromWeek, args.ToWeek)
		return map[string]any{"assigned": assigned}, false

	case "import_file":
		var args struct{ Path string `json:"path"` }
		if err := json.Unmarshal(rawArgs, &args); err != nil || args.Path == "" {
			return map[string]any{"error": "path is required"}, true
		}
		raw, err := os.ReadFile(args.Path)
		if err != nil {
			return map[string]any{"error": err.Error()}, true
		}
		ext := strings.ToLower(filepath.Ext(args.Path))
		var proj *Project
		switch ext {
		case ".mfw":
			proj, err = ImportMFW(raw, filepath.Base(args.Path))
		case ".mxt":
			proj, err = ImportMXT(string(raw), filepath.Base(args.Path))
		case ".omimo", ".json":
			var tmp Project
			err = json.Unmarshal(raw, &tmp)
			proj = &tmp
		default:
			return map[string]any{"error": "unsupported format (use .omimo, .mfw, .mxt, or .json)"}, true
		}
		if err != nil {
			return map[string]any{"error": err.Error()}, true
		}
		s.pushUndo()
		s.project = proj
		return map[string]any{"imported": filepath.Base(args.Path), "stats": proj.ComputeStats()}, false

	case "export_timetable":
		var args struct {
			Format     string `json:"format"`
			ReportType int    `json:"reportType"`
			Week       int    `json:"week"`
			OutputPath string `json:"outputPath"`
		}
		_ = json.Unmarshal(rawArgs, &args)
		var out string
		switch strings.ToLower(args.Format) {
		case "mxt":
			out = ExportMXT(p)
		case "csv":
			out = ExportCSV(p)
		case "ics":
			out = ExportICS(p, "")
		case "html":
			rt := args.ReportType
			if rt < 1 || rt > 6 {
				rt = 3
			}
			out = GenerateHTMLReport(p, rt, args.Week)
		default:
			b, _ := json.MarshalIndent(p, "", "  ")
			out = string(b)
		}
		if args.OutputPath != "" {
			if err := os.WriteFile(args.OutputPath, []byte(out), 0644); err != nil {
				return map[string]any{"error": err.Error()}, true
			}
			return map[string]any{"written": args.OutputPath, "bytes": len(out)}, false
		}
		return map[string]any{"format": args.Format, "content": out}, false

	case "list_scripts":
		if p.Scripts == nil {
			p.Scripts = []UserScript{}
		}
		return p.Scripts, false

	case "save_script":
		var sc UserScript
		if err := json.Unmarshal(rawArgs, &sc); err != nil || strings.TrimSpace(sc.Name) == "" {
			return map[string]any{"error": "name and code are required"}, true
		}
		s.pushUndo()
		if sc.ID == "" {
			sc.ID = fmt.Sprintf("scr_mcp_%d", time.Now().UnixNano())
		}
		if sc.Author == "" {
			sc.Author = "MCP / IA"
		}
		sc.UpdatedAt = time.Now().Format(time.RFC3339)
		updated := false
		for i := range p.Scripts {
			if p.Scripts[i].ID == sc.ID {
				p.Scripts[i] = sc
				updated = true
				break
			}
		}
		if !updated {
			p.Scripts = append(p.Scripts, sc)
		}
		s.afterMutateLocked()
		return sc, false

	default:
		return map[string]any{"error": "unknown tool: " + name}, true
	}
}

// RunMCPStdio serves MCP JSON-RPC 2.0 over stdin/stdout (standard transport for AI Agents).
func RunMCPStdio(state *AppState) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req mcpRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = enc.Encode(mcpResponse{
				JSONRPC: "2.0",
				Error:   &mcpError{Code: -32700, Message: "parse error: " + err.Error()},
			})
			continue
		}
		if resp := state.handleMCPRequest(req); resp != nil {
			_ = enc.Encode(resp)
		}
	}
}

// HandleMCPHTTP serves MCP JSON-RPC 2.0 over HTTP POST /mcp (live connection to running OpenMimo app).
func (s *AppState) HandleMCPHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 200, map[string]any{
			"name":     "openmimo",
			"protocol": "mcp",
			"version":  "2024-11-05",
			"endpoint": "/mcp",
		})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 16<<20))
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	var req mcpRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, 400, mcpResponse{
			JSONRPC: "2.0",
			Error:   &mcpError{Code: -32700, Message: "parse error: " + err.Error()},
		})
		return
	}
	resp := s.handleMCPRequest(req)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, 200, resp)
}
