package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AppState struct {
	mu           sync.Mutex
	tabs         []*OpenTab
	activeID     string
	project      *Project // always mirrors active tab's project (or nil if none open)
	undo         []*Project
	redo         []*Project
	mcpEnabled   bool
	listenPort   int
	sseClients   map[string]map[chan string]bool // projectID -> set of SSE channels
	systemLang   string
}

func NewAppState() *AppState {
	sysLang := DetectSystemLang()
	s := &AppState{
		mcpEnabled: true,
		sseClients: make(map[string]map[chan string]bool),
		systemLang: sysLang,
	}

	// Load existing projects from OS App Data Dir; seed one demo project on very first run
	locals := ListLocalProjects(map[string]bool{})
	if len(locals) == 0 {
		demo := NewUniversityDemoProject()
		demo.Language = sysLang
		_ = SaveProjectToDisk(demo)
		s.openTabLocked(demo)
	} else {
		if p, err := LoadProjectFromDisk(locals[0].ID); err == nil {
			s.openTabLocked(p)
		} else {
			demo := NewUniversityDemoProject()
			demo.Language = sysLang
			_ = SaveProjectToDisk(demo)
			s.openTabLocked(demo)
		}
	}
	return s
}

func (s *AppState) activeTabLocked() *OpenTab {
	for _, t := range s.tabs {
		if t.Project.ID == s.activeID {
			return t
		}
	}
	if len(s.tabs) > 0 {
		s.activeID = s.tabs[0].Project.ID
		return s.tabs[0]
	}
	return nil
}

func (s *AppState) syncMirrorsLocked() {
	if tab := s.activeTabLocked(); tab != nil {
		s.project = tab.Project
		s.undo = tab.Undo
		s.redo = tab.Redo
	} else {
		s.project = nil
		s.undo = nil
		s.redo = nil
	}
}

func (s *AppState) refreshSharedTabLocked(p *Project) {
	if p == nil || !p.Collab.Shared || p.Collab.IsRemote || s.listenPort <= 0 {
		return
	}
	primary, alts := BuildHostURLs(s.listenPort)
	p.Collab.HostURL = primary
	p.Collab.HostOpen = true
	if p.Collab.IPFSCid == "" {
		raw, _ := json.Marshal(p)
		cid, mode := PublishToIPFS(raw)
		p.Collab.IPFSCid = cid
		p.Collab.IPFSMode = mode
	}
	p.Collab.ShareCode = EncodeShareCode(primary, alts, p.ID, p.Collab.IPFSCid)
}

func (s *AppState) refreshAllSharedLocked() {
	for _, t := range s.tabs {
		s.refreshSharedTabLocked(t.Project)
	}
}

func (s *AppState) openTabLocked(p *Project) {
	if p.ID == "" {
		p.ID = fmt.Sprintf("proj_%d", time.Now().UnixNano())
	}
	s.refreshSharedTabLocked(p)
	for _, t := range s.tabs {
		if t.Project.ID == p.ID {
			t.Project = p
			s.activeID = p.ID
			s.syncMirrorsLocked()
			return
		}
	}
	s.tabs = append(s.tabs, &OpenTab{Project: p})
	s.activeID = p.ID
	s.syncMirrorsLocked()
}

func (s *AppState) closeTabLocked(id string) {
	idx := -1
	for i, t := range s.tabs {
		if t.Project.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	s.tabs = append(s.tabs[:idx], s.tabs[idx+1:]...)
	if s.activeID == id {
		if len(s.tabs) > 0 {
			next := idx
			if next >= len(s.tabs) {
				next = len(s.tabs) - 1
			}
			s.activeID = s.tabs[next].Project.ID
		} else {
			s.activeID = ""
		}
	}
	s.syncMirrorsLocked()
	s.broadcastSSELocked(id, "closed")
}

func (s *AppState) pushUndo() {
	tab := s.activeTabLocked()
	if tab == nil {
		return
	}
	tab.Undo = append(tab.Undo, tab.Project.Clone())
	if len(tab.Undo) > 50 {
		tab.Undo = tab.Undo[len(tab.Undo)-50:]
	}
	tab.Redo = nil
	s.syncMirrorsLocked()
}

// guardEditable enforces that remote projects can ONLY be edited when the host PC has the project open!
func (s *AppState) guardEditableLocked() error {
	tab := s.activeTabLocked()
	if tab == nil {
		return fmt.Errorf("no hay proyecto activo")
	}
	p := tab.Project
	if p.Collab.IsRemote {
		var alts []string
		if tok, err := DecodeShareCode(p.Collab.ShareCode); err == nil {
			alts = tok.AltURLs
		}
		workingURL, online, open, _ := ProbeRemoteHost(p.Collab.HostURL, alts, p.ID)
		if workingURL != "" {
			p.Collab.HostURL = workingURL
		}
		p.Collab.HostOpen = online && open
		if !online {
			return fmt.Errorf("no se puede conectar con el equipo anfitrión (%s); verifica la red o el firewall", p.Collab.HostURL)
		}
		if !open {
			return fmt.Errorf("este archivo remoto es solo lectura porque el equipo original lo cerró o dejó de compartirlo")
		}
	}
	return nil
}

// afterMutateLocked persists local projects to OS AppDataDir, refreshes IPFS CID if shared,
// pushes remote changes back to host PC, and broadcasts real-time SSE updates.
func (s *AppState) afterMutateLocked() {
	tab := s.activeTabLocked()
	if tab == nil {
		return
	}
	p := tab.Project
	p.RecomputeDerived()
	p.UpdatedAt = time.Now().Format(time.RFC3339)
	p.Collab.Revision++

	if p.Collab.IsRemote {
		if synced, err := PushToRemoteHost(p.Collab.HostURL, p.ID, p); err == nil && synced != nil {
			synced.Collab.IsRemote = true
			synced.Collab.HostURL = p.Collab.HostURL
			synced.Collab.HostOpen = true
			synced.Collab.ShareCode = p.Collab.ShareCode
			tab.Project = synced
			s.syncMirrorsLocked()
		}
	} else {
		if p.Collab.Shared {
			if raw, err := json.Marshal(p); err == nil {
				cid, mode := PublishToIPFS(raw)
				p.Collab.IPFSCid = cid
				p.Collab.IPFSMode = mode
				primary, alts := BuildHostURLs(s.listenPort)
				p.Collab.HostURL = primary
				p.Collab.ShareCode = EncodeShareCode(primary, alts, p.ID, cid)
			}
			s.broadcastSSELocked(p.ID, "updated")
		}
		_ = SaveProjectToDisk(p)
	}
}

func (s *AppState) broadcastSSELocked(projectID, eventType string) {
	subs, ok := s.sseClients[projectID]
	if !ok {
		return
	}
	msg := fmt.Sprintf("event: %s\ndata: {\"projectId\":%q,\"ts\":%d}\n\n", eventType, projectID, time.Now().Unix())
	for ch := range subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *AppState) openIDSetLocked() map[string]bool {
	m := make(map[string]bool, len(s.tabs))
	for _, t := range s.tabs {
		m[t.Project.ID] = true
	}
	return m
}

func (s *AppState) tabListLocked() []ProjectTabInfo {
	out := make([]ProjectTabInfo, 0, len(s.tabs))
	for _, t := range s.tabs {
		out = append(out, ProjectTabInfo{
			ID:        t.Project.ID,
			Name:      t.Project.Name,
			IsRemote:  t.Project.Collab.IsRemote,
			HostOpen:  t.Project.Collab.HostOpen,
			ShareCode: t.Project.Collab.ShareCode,
		})
	}
	return out
}

func (s *AppState) sharedListWithLiveStatusLocked() []SharedRemoteEntry {
	reg := LoadSharedRegistry()
	for i := range reg {
		alts := reg[i].AltURLs
		if len(alts) == 0 && reg[i].ShareCode != "" {
			if tok, err := DecodeShareCode(reg[i].ShareCode); err == nil {
				alts = tok.AltURLs
			}
		}
		workingURL, online, open, _ := ProbeRemoteHost(reg[i].HostURL, alts, reg[i].ProjectID)
		if workingURL != "" && online {
			reg[i].HostURL = workingURL
		}
		reg[i].HostOnline = online
		reg[i].HostOpen = online && open
		if online {
			reg[i].LastSeen = time.Now().Format(time.RFC3339)
		}
	}
	return reg
}

func (s *AppState) responsePayload() map[string]any {
	s.syncMirrorsLocked()
	tab := s.activeTabLocked()
	var proj *Project
	var stats ProjectStats
	canUndo, canRedo := false, false
	if tab != nil {
		tab.Project.RecomputeDerived()
		proj = tab.Project
		stats = proj.ComputeStats()
		canUndo = len(tab.Undo) > 0
		canRedo = len(tab.Redo) > 0
	}
	return map[string]any{
		"project":    proj,
		"stats":      stats,
		"canUndo":    canUndo,
		"canRedo":    canRedo,
		"tabs":       s.tabListLocked(),
		"activeTab":  s.activeID,
		"mcpEnabled": s.mcpEnabled,
		"mcpHttpUrl": fmt.Sprintf("http://127.0.0.1:%d/mcp", s.listenPort),
		"dataDir":    AppDataDir(),
		"systemLang": s.systemLang,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func decodeBody(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 32<<20)).Decode(dst)
}

func main() {
	portFlag := flag.Int("port", 0, "TCP port to listen on (0 = ephemeral)")
	testImport := flag.String("test-import", "", "Test importing a .mfw or .mxt file and exit")
	mcpStdio := flag.Bool("mcp", false, "Run as an MCP (Model Context Protocol) JSON-RPC stdio server")
	flag.Parse()

	if *mcpStdio {
		state := NewAppState()
		RunMCPStdio(state)
		return
	}

	if *testImport != "" {
		raw, err := os.ReadFile(*testImport)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			os.Exit(1)
		}
		var proj *Project
		if strings.HasSuffix(strings.ToLower(*testImport), ".mfw") {
			proj, err = ImportMFW(raw, filepath.Base(*testImport))
		} else {
			proj, err = ImportMXT(string(raw), filepath.Base(*testImport))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "import error: %v\n", err)
			os.Exit(1)
		}
		st := proj.ComputeStats()
		fmt.Printf("Imported %s: %d courses, %d resources, %d links, %d weekAllocs, %d slots, %d conflicts, %d gaps\n",
			filepath.Base(*testImport), st.TotalCourses, st.TotalResources, st.TotalLinks, len(proj.WeekAllocations), len(proj.Slots), st.ConflictCount, st.TotalGaps)
		return
	}

	state := NewAppState()
	mux := http.NewServeMux()

	withCORS := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			h(w, r)
		}
	}

	mux.HandleFunc("/api/health", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		en := state.mcpEnabled
		state.mu.Unlock()
		writeJSON(w, 200, map[string]any{"status": "ok", "version": "1.0.0", "mcpEnabled": en, "mcp": "/mcp"})
	}))

	// Check GitHub Releases at https://github.com/aedmadrid/openmimo/releases
	mux.HandleFunc("/api/updates/check", withCORS(func(w http.ResponseWriter, r *http.Request) {
		const currentVersion = "1.0.0"
		const repoURL = "https://github.com/aedmadrid/openmimo/releases"
		client := &http.Client{Timeout: 3000 * time.Millisecond}
		req, _ := http.NewRequest("GET", "https://api.github.com/repos/aedmadrid/openmimo/releases/latest", nil)
		req.Header.Set("User-Agent", "OpenMimo/1.0.0")
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := client.Do(req)
		if err != nil {
			writeJSON(w, 200, map[string]any{
				"currentVersion": currentVersion, "hasUpdate": false, "checked": false, "releaseUrl": repoURL,
			})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			writeJSON(w, 200, map[string]any{
				"currentVersion": currentVersion, "latestVersion": currentVersion, "hasUpdate": false, "checked": true, "releaseUrl": repoURL,
			})
			return
		}
		var rel struct {
			TagName string `json:"tag_name"`
			HTMLURL string `json:"html_url"`
			Name    string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			writeJSON(w, 200, map[string]any{"currentVersion": currentVersion, "hasUpdate": false, "checked": false, "releaseUrl": repoURL})
			return
		}
		latest := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
		hasUpdate := latest != "" && latest != currentVersion && latest > currentVersion
		url := rel.HTMLURL
		if url == "" {
			url = repoURL
		}
		writeJSON(w, 200, map[string]any{
			"currentVersion": currentVersion,
			"latestVersion":  latest,
			"releaseName":    rel.Name,
			"hasUpdate":      hasUpdate,
			"checked":        true,
			"releaseUrl":     url,
		})
	}))

	// Local AI Settings (OpenAI-compatible API key stored locally with 0600 permissions, never in .omimo)
	mux.HandleFunc("/api/ai/config", withCORS(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			cfg := LoadAISettings()
			writeJSON(w, 200, map[string]any{
				"baseUrl":   cfg.BaseURL,
				"model":     cfg.Model,
				"hasApiKey": strings.TrimSpace(cfg.APIKey) != "",
				"apiKey":    cfg.APIKey,
			})
		case http.MethodPost, http.MethodPut:
			var req AISettings
			if err := decodeBody(r, &req); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			if err := SaveAISettings(req); err != nil {
				writeError(w, 500, err.Error())
				return
			}
			cfg := LoadAISettings()
			writeJSON(w, 200, map[string]any{
				"baseUrl":   cfg.BaseURL,
				"model":     cfg.Model,
				"hasApiKey": strings.TrimSpace(cfg.APIKey) != "",
				"apiKey":    cfg.APIKey,
			})
		}
	}))

	// Generate an OpenMimo JavaScript automation script using the configured OpenAI-compatible API Key
	mux.HandleFunc("/api/ai/generate-script", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Prompt string `json:"prompt"`
		}
		if err := decodeBody(r, &req); err != nil || strings.TrimSpace(req.Prompt) == "" {
			writeError(w, 400, "Escribe una descripción para generar el script")
			return
		}
		cfg := LoadAISettings()
		if strings.TrimSpace(cfg.APIKey) == "" && !strings.Contains(cfg.BaseURL, "127.0.0.1") && !strings.Contains(cfg.BaseURL, "localhost") {
			writeError(w, 400, "Configura tu API Key (estilo OpenAI) en Configuración → IA antes de generar scripts con IA.")
			return
		}
		endpoint := strings.TrimRight(cfg.BaseURL, "/")
		if !strings.HasSuffix(endpoint, "/chat/completions") {
			endpoint += "/chat/completions"
		}
		state.mu.Lock()
		projSummary := ""
		if state.project != nil {
			projSummary = fmt.Sprintf("Project: %s | Courses: %d | Resources: %d | Weeks: %d | Days: %d | Periods: %d | Guides: %d",
				state.project.Name, len(state.project.Courses), len(state.project.Resources),
				state.project.Time.MaxWeeks, state.project.Time.MaxDays, state.project.Time.MaxPeriods, len(state.project.Guides))
		}
		state.mu.Unlock()

		sysPrompt := "You write concise JavaScript script bodies for OpenMimo's `function(project, log)` runner. " +
			"project fields: courses (id,code,name,category,planned,minConsecutive,maxConsecutive,color,locked,guideId,resources[]), " +
			"resources (id,code,name,category 1=Group 2=Teacher 3=Room,planned,mcpNotes), bookings (id,targetType,targetId,scope,week,day,period,kind,label), " +
			"slots (id,courseId,week,day,period,roomId), guides (id,title,creditsEcts,groups[]), time (maxWeeks,maxDays,maxPeriods). " +
			"Call log(...) to report actions. Return true if project was mutated, false if read-only. Return ONLY raw JavaScript code without markdown fences. " + projSummary

		bodyObj := map[string]any{
			"model": cfg.Model,
			"messages": []map[string]string{
				{"role": "system", "content": sysPrompt},
				{"role": "user", "content": req.Prompt},
			},
			"temperature": 0.2,
		}
		payloadBytes, _ := json.Marshal(bodyObj)
		httpReq, err := http.NewRequest("POST", endpoint, strings.NewReader(string(payloadBytes)))
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if strings.TrimSpace(cfg.APIKey) != "" {
			httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
		}
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			writeError(w, 502, "Error conectando con la API de IA: "+err.Error())
			return
		}
		defer resp.Body.Close()
		rawResp, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		if resp.StatusCode >= 400 {
			writeError(w, resp.StatusCode, fmt.Sprintf("Error de la API IA (%d): %s", resp.StatusCode, strings.TrimSpace(string(rawResp))))
			return
		}
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(rawResp, &parsed); err != nil || len(parsed.Choices) == 0 {
			writeError(w, 502, "Respuesta inesperada del proveedor de IA")
			return
		}
		code := strings.TrimSpace(parsed.Choices[0].Message.Content)
		code = strings.TrimPrefix(code, "```javascript")
		code = strings.TrimPrefix(code, "```js")
		code = strings.TrimPrefix(code, "```")
		code = strings.TrimSuffix(code, "```")
		code = strings.TrimSpace(code)

		name := req.Prompt
		if len(name) > 46 {
			name = name[:46]
		}
		writeJSON(w, 200, map[string]any{
			"name":        name,
			"description": req.Prompt,
			"code":        code,
			"model":       cfg.Model,
		})
	}))

	// Script Library CRUD (Affinity-style scripting)
	mux.HandleFunc("/api/scripts", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		p := state.project
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			var sc UserScript
			if err := decodeBody(r, &sc); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			if strings.TrimSpace(sc.Name) == "" {
				writeError(w, 400, "El nombre del script es obligatorio")
				return
			}
			state.pushUndo()
			if sc.ID == "" {
				sc.ID = fmt.Sprintf("scr_%d", time.Now().UnixNano())
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
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			state.pushUndo()
			out := make([]UserScript, 0, len(p.Scripts))
			for _, sc := range p.Scripts {
				if sc.ID != id {
					out = append(out, sc)
				}
			}
			p.Scripts = out
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		}
	}))

	// Live MCP HTTP JSON-RPC endpoint (respects user's MCP enabled/disabled toggle!)
	mux.HandleFunc("/mcp", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		en := state.mcpEnabled
		state.mu.Unlock()
		if !en {
			writeError(w, 503, "MCP server is currently disabled in OpenMimo settings")
			return
		}
		state.HandleMCPHTTP(w, r)
	}))

	// MCP toggle & config endpoint
	mux.HandleFunc("/api/mcp/config", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if r.Method == http.MethodPost {
			var req struct{ Enabled bool `json:"enabled"` }
			if err := decodeBody(r, &req); err == nil {
				state.mcpEnabled = req.Enabled
			}
		}
		exePath, _ := os.Executable()
		writeJSON(w, 200, map[string]any{
			"enabled":       state.mcpEnabled,
			"httpUrl":       fmt.Sprintf("http://127.0.0.1:%d/mcp", state.listenPort),
			"stdioCommand":  exePath,
			"stdioArgs":     []string{"-mcp"},
			"toolsCount":    len(mcpToolDefinitions()),
			"payload":       state.responsePayload(),
		})
	}))

	// Workspace: Recent Local Projects + Shared Remote Projects + Open Tabs
	mux.HandleFunc("/api/workspace", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		openIDs := state.openIDSetLocked()
		tabs := state.tabListLocked()
		active := state.activeID
		mcpEn := state.mcpEnabled
		port := state.listenPort
		sysLang := state.systemLang
		state.mu.Unlock()

		locals := ListLocalProjects(openIDs)
		shared := state.sharedListWithLiveStatusLocked()

		writeJSON(w, 200, map[string]any{
			"recent":     locals,
			"shared":     shared,
			"samples":    ListBuiltInSamples(),
			"tabs":       tabs,
			"activeTab":  active,
			"dataDir":    AppDataDir(),
			"mcpEnabled": mcpEn,
			"mcpHttpUrl": fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
			"systemLang": sysLang,
		})
	}))

	// Instantiate and open one of the built-in sample/tutorial projects
	mux.HandleFunc("/api/projects/sample", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key  string `json:"key"`
			Lang string `json:"lang"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		p := BuildSampleProject(req.Key, req.Lang)
		_ = SaveProjectToDisk(p)
		state.mu.Lock()
		state.openTabLocked(p)
		resp := state.responsePayload()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	}))

	// Create new project from Setup Wizard
	mux.HandleFunc("/api/projects/create", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var cfg WizardConfig
		if err := decodeBody(r, &cfg); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		p := CreateProjectFromWizard(cfg)
		_ = SaveProjectToDisk(p)
		state.mu.Lock()
		state.openTabLocked(p)
		resp := state.responsePayload()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	}))

	// Open local project by ID into a tab
	mux.HandleFunc("/api/projects/open", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID string `json:"id"` }
		if err := decodeBody(r, &req); err != nil || req.ID == "" {
			writeError(w, 400, "id requerido")
			return
		}
		p, err := LoadProjectFromDisk(req.ID)
		if err != nil {
			writeError(w, 404, "proyecto no encontrado: "+err.Error())
			return
		}
		state.mu.Lock()
		state.openTabLocked(p)
		resp := state.responsePayload()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	}))

	// Switch active project tab
	mux.HandleFunc("/api/projects/switch", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID string `json:"id"` }
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		for _, t := range state.tabs {
			if t.Project.ID == req.ID {
				state.activeID = req.ID
				state.syncMirrorsLocked()
				writeJSON(w, 200, state.responsePayload())
				return
			}
		}
		writeError(w, 404, "pestaña no encontrada")
	}))

	// Close project tab (also notifies remote peers if this project was shared!)
	mux.HandleFunc("/api/projects/close", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID string `json:"id"` }
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		state.closeTabLocked(req.ID)
		writeJSON(w, 200, state.responsePayload())
	}))

	// Export any local project by ID (used for "Export & Delete" in Recent Projects)
	mux.HandleFunc("/api/projects/export-by-id", withCORS(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, 400, "id requerido")
			return
		}
		p, err := LoadProjectFromDisk(id)
		if err != nil {
			writeError(w, 404, "proyecto no encontrado: "+err.Error())
			return
		}
		b, _ := json.MarshalIndent(p, "", "  ")
		writeJSON(w, 200, map[string]any{
			"fileName": safeFile(p.Name) + ".omimo",
			"content":  string(b),
		})
	}))

	// Delete local project from disk
	mux.HandleFunc("/api/projects/delete", withCORS(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, 400, "id requerido")
			return
		}
		state.mu.Lock()
		state.closeTabLocked(id)
		state.mu.Unlock()
		_ = os.Remove(filepath.Join(ProjectsDir(), safeFile(id)+".omimo"))
		_ = os.Remove(filepath.Join(ProjectsDir(), safeFile(id)+".json"))
		state.mu.Lock()
		resp := state.responsePayload()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	}))

	// Full active project state
	mux.HandleFunc("/api/project", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, state.responsePayload())
		case http.MethodPut:
			if err := state.guardEditableLocked(); err != nil {
				writeError(w, 403, err.Error())
				return
			}
			var upd Project
			if err := decodeBody(r, &upd); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			state.pushUndo()
			if tab := state.activeTabLocked(); tab != nil {
				tab.Project = &upd
				state.afterMutateLocked()
			}
			writeJSON(w, 200, state.responsePayload())
		default:
			writeError(w, 405, "method not allowed")
		}
	}))

	// Legacy /api/project/new compatibility
	mux.HandleFunc("/api/project/new", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode string `json:"mode"`
			Lang string `json:"lang"`
		}
		_ = decodeBody(r, &req)
		p := CreateProjectFromWizard(WizardConfig{Template: req.Mode, Language: req.Lang, MaxWeeks: 4, MaxDays: 5, MaxPeriods: 10})
		_ = SaveProjectToDisk(p)
		state.mu.Lock()
		state.openTabLocked(p)
		resp := state.responsePayload()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	}))

	// Undo / Redo
	mux.HandleFunc("/api/undo", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		if tab := state.activeTabLocked(); tab != nil && len(tab.Undo) > 0 {
			tab.Redo = append(tab.Redo, tab.Project.Clone())
			tab.Project = tab.Undo[len(tab.Undo)-1]
			tab.Undo = tab.Undo[:len(tab.Undo)-1]
			state.afterMutateLocked()
		}
		writeJSON(w, 200, state.responsePayload())
	}))
	mux.HandleFunc("/api/redo", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		if tab := state.activeTabLocked(); tab != nil && len(tab.Redo) > 0 {
			tab.Undo = append(tab.Undo, tab.Project.Clone())
			tab.Project = tab.Redo[len(tab.Redo)-1]
			tab.Redo = tab.Redo[:len(tab.Redo)-1]
			state.afterMutateLocked()
		}
		writeJSON(w, 200, state.responsePayload())
	}))

	// Teaching Guides CRUD (Guías Docentes)
	mux.HandleFunc("/api/guides", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		p := state.project
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			var req struct {
				TeachingGuide
				ContainedCourseIDs []string `json:"containedCourseIds,omitempty"`
				SyncContained      bool     `json:"syncContained,omitempty"`
			}
			if err := decodeBody(r, &req); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			g := req.TeachingGuide
			if strings.TrimSpace(g.Title) == "" {
				writeError(w, 400, "El título de la asignatura global es obligatorio")
				return
			}
			state.pushUndo()
			if g.ID == "" {
				g.ID = fmt.Sprintf("g_%d", time.Now().UnixNano())
			}
			if g.Slug == "" {
				g.Slug = makeSlug(g.Title)
			}
			g.UpdatedAt = time.Now().Format(time.RFC3339)
			updated := false
			for i := range p.Guides {
				if p.Guides[i].ID == g.ID {
					p.Guides[i] = g
					updated = true
					break
				}
			}
			if !updated {
				p.Guides = append(p.Guides, g)
			}
			// Sync contained session codes (e.g. DNOP1, DNOP2, DNOP3 -> Global Subject)
			if req.SyncContained {
				wanted := make(map[string]bool, len(req.ContainedCourseIDs))
				for _, cid := range req.ContainedCourseIDs {
					wanted[cid] = true
				}
				for i := range p.Courses {
					if wanted[p.Courses[i].ID] {
						p.Courses[i].GuideID = g.ID
					} else if p.Courses[i].GuideID == g.ID {
						p.Courses[i].GuideID = ""
					}
				}
			}
			if cid := r.URL.Query().Get("linkCourseId"); cid != "" {
				if c := p.CourseByID(cid); c != nil {
					c.GuideID = g.ID
				}
			}
			state.afterMutateLocked()
			resp := state.responsePayload()
			resp["guide"] = g
			writeJSON(w, 200, resp)
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			state.pushUndo()
			out := make([]TeachingGuide, 0, len(p.Guides))
			for _, g := range p.Guides {
				if g.ID != id {
					out = append(out, g)
				}
			}
			p.Guides = out
			for i := range p.Courses {
				if p.Courses[i].GuideID == id {
					p.Courses[i].GuideID = ""
				}
			}
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		}
	}))

	// Export Astro 5 Static Site (.zip base64)
	mux.HandleFunc("/api/export/astro", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		p := state.project
		if p == nil {
			state.mu.Unlock()
			writeError(w, 400, "no hay proyecto activo")
			return
		}
		cp := p.Clone()
		state.mu.Unlock()

		zipBytes, err := GenerateAstroZip(cp)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{
			"fileName":  safeFile(cp.Name) + "-guias-astro.zip",
			"base64Zip": base64.StdEncoding.EncodeToString(zipBytes),
			"bytes":     len(zipBytes),
		})
	}))

	// Real-Time Collab & IPFS Sharing Endpoints
	mux.HandleFunc("/api/collab/share", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		p := state.project
		if p == nil {
			writeError(w, 400, "no hay proyecto activo")
			return
		}
		var req struct{ Stop bool `json:"stop"` }
		_ = decodeBody(r, &req)
		if req.Stop {
			p.Collab.Shared = false
			p.Collab.ShareCode = ""
			_ = SaveProjectToDisk(p)
			state.broadcastSSELocked(p.ID, "closed")
			writeJSON(w, 200, state.responsePayload())
			return
		}
		p.Collab.Shared = true
		p.Collab.IsRemote = false
		p.Collab.HostOpen = true
		raw, _ := json.Marshal(p)
		cid, mode := PublishToIPFS(raw)
		primary, alts := BuildHostURLs(state.listenPort)
		p.Collab.IPFSCid = cid
		p.Collab.IPFSMode = mode
		p.Collab.HostURL = primary
		p.Collab.ShareCode = EncodeShareCode(primary, alts, p.ID, cid)
		_ = SaveProjectToDisk(p)
		writeJSON(w, 200, state.responsePayload())
	}))

	// Join a shared project using its IPFS/Collab Share Code
	mux.HandleFunc("/api/collab/join", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ShareCode string `json:"shareCode"`
			HostURL   string `json:"hostUrl"`
			ProjectID string `json:"projectId"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		hostURL, projID, cid := req.HostURL, req.ProjectID, ""
		var altURLs []string
		if req.ShareCode != "" {
			tok, err := DecodeShareCode(req.ShareCode)
			if err != nil {
				writeError(w, 400, err.Error())
				return
			}
			hostURL, altURLs, projID, cid = tok.HostURL, tok.AltURLs, tok.ProjectID, tok.IPFSCid
		}
		workingURL, online, open, remoteProj := ProbeRemoteHost(hostURL, altURLs, projID)
		if !online || !open || remoteProj == nil {
			if req.ShareCode != "" {
				UpsertSharedRegistry(SharedRemoteEntry{
					ProjectID: projID, Name: "Proyecto remoto (" + projID + ")", ShareCode: req.ShareCode,
					IPFSCid: cid, HostURL: hostURL, AltURLs: altURLs, HostOnline: online, HostOpen: false,
					LastSeen: time.Now().Format(time.RFC3339),
				})
			}
			if !online {
				writeError(w, 502, fmt.Sprintf("No se pudo conectar con el equipo anfitrión (%s). Comprueba que ambos equipos estén en la misma red Wi-Fi/LAN, que OpenMimo esté abierto en el otro equipo y que el firewall permita la conexión.", hostURL))
				return
			}
			writeError(w, 423, "El equipo anfitrión está en línea, pero este horario está cerrado o dejó de compartirse allí.")
			return
		}
		remoteProj.Collab.IsRemote = true
		remoteProj.Collab.HostURL = workingURL
		remoteProj.Collab.HostOpen = true
		remoteProj.Collab.ShareCode = req.ShareCode
		UpsertSharedRegistry(SharedRemoteEntry{
			ProjectID: remoteProj.ID, Name: remoteProj.Name, Institution: remoteProj.Institution,
			ShareCode: remoteProj.Collab.ShareCode, IPFSCid: remoteProj.Collab.IPFSCid,
			HostURL: workingURL, AltURLs: altURLs, HostOnline: true, HostOpen: true, LastSeen: time.Now().Format(time.RFC3339),
		})
		state.mu.Lock()
		state.openTabLocked(remoteProj)
		resp := state.responsePayload()
		state.mu.Unlock()
		writeJSON(w, 200, resp)
	}))

	// Pull latest revision from host PC for the active remote tab (triggered on real-time SSE 'updated')
	mux.HandleFunc("/api/collab/pull", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		tab := state.activeTabLocked()
		if tab == nil || !tab.Project.Collab.IsRemote {
			resp := state.responsePayload()
			state.mu.Unlock()
			writeJSON(w, 200, resp)
			return
		}
		hostURL := tab.Project.Collab.HostURL
		shareCode := tab.Project.Collab.ShareCode
		pid := tab.Project.ID
		state.mu.Unlock()

		var alts []string
		if tok, err := DecodeShareCode(shareCode); err == nil {
			alts = tok.AltURLs
		}
		workingURL, online, open, remoteProj := ProbeRemoteHost(hostURL, alts, pid)
		state.mu.Lock()
		defer state.mu.Unlock()
		if tab := state.activeTabLocked(); tab != nil && tab.Project.ID == pid {
			if online && open && remoteProj != nil {
				remoteProj.Collab.IsRemote = true
				remoteProj.Collab.HostURL = workingURL
				remoteProj.Collab.HostOpen = true
				remoteProj.Collab.ShareCode = shareCode
				remoteProj.RecomputeDerived()
				tab.Project = remoteProj
				state.syncMirrorsLocked()
			} else {
				tab.Project.Collab.HostOpen = false
			}
		}
		writeJSON(w, 200, state.responsePayload())
	}))

	// Host endpoint: check if a shared project is currently open in a tab on this PC
	mux.HandleFunc("/api/collab/status", withCORS(func(w http.ResponseWriter, r *http.Request) {
		pid := r.URL.Query().Get("projectId")
		state.mu.Lock()
		defer state.mu.Unlock()
		for _, t := range state.tabs {
			if t.Project.ID == pid && t.Project.Collab.Shared {
				t.Project.RecomputeDerived()
				writeJSON(w, 200, map[string]any{"open": true, "project": t.Project})
				return
			}
		}
		writeJSON(w, 200, map[string]any{"open": false})
	}))

	// Host endpoint: accept live edits from remote peers ONLY while the project is open on this host PC
	mux.HandleFunc("/api/collab/push", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ProjectID string   `json:"projectId"`
			Project   *Project `json:"project"`
		}
		if err := decodeBody(r, &req); err != nil || req.Project == nil {
			writeError(w, 400, "payload inválido")
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		for _, t := range state.tabs {
			if t.Project.ID == req.ProjectID && t.Project.Collab.Shared {
				req.Project.Collab.Shared = true
				req.Project.Collab.IsRemote = false
				req.Project.Collab.HostOpen = true
				req.Project.Collab.HostURL = t.Project.Collab.HostURL
				req.Project.Collab.Revision = t.Project.Collab.Revision + 1
				raw, _ := json.Marshal(req.Project)
				cid, mode := PublishToIPFS(raw)
				req.Project.Collab.IPFSCid = cid
				req.Project.Collab.IPFSMode = mode
				primary, alts := BuildHostURLs(state.listenPort)
				req.Project.Collab.HostURL = primary
				req.Project.Collab.ShareCode = EncodeShareCode(primary, alts, t.Project.ID, cid)
				req.Project.RecomputeDerived()
				t.Project = req.Project
				_ = SaveProjectToDisk(t.Project)
				state.syncMirrorsLocked()
				state.broadcastSSELocked(t.Project.ID, "updated")
				writeJSON(w, 200, map[string]any{"project": t.Project})
				return
			}
		}
		writeError(w, 423, "proyecto cerrado en el equipo anfitrión")
	}))

	// Real-time Server-Sent Events stream for instant multi-user refresh
	mux.HandleFunc("/api/collab/events", withCORS(func(w http.ResponseWriter, r *http.Request) {
		pid := r.URL.Query().Get("projectId")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}
		ch := make(chan string, 8)
		state.mu.Lock()
		if state.sseClients[pid] == nil {
			state.sseClients[pid] = make(map[chan string]bool)
		}
		state.sseClients[pid][ch] = true
		state.mu.Unlock()

		defer func() {
			state.mu.Lock()
			delete(state.sseClients[pid], ch)
			state.mu.Unlock()
		}()

		fmt.Fprintf(w, "event: connected\ndata: {\"projectId\":%q}\n\n", pid)
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				_, _ = io.WriteString(w, msg)
				flusher.Flush()
			}
		}
	}))

	// Courses CRUD + Actions
	mux.HandleFunc("/api/courses", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		switch r.Method {
		case http.MethodPost:
			var c Course
			if err := decodeBody(r, &c); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			if strings.TrimSpace(c.Code) == "" {
				writeError(w, 400, "El código del curso es obligatorio")
				return
			}
			state.pushUndo()
			if c.ID == "" {
				c.ID = fmt.Sprintf("c_%d", time.Now().UnixNano())
			}
			if c.Category < 1 || c.Category > 7 {
				c.Category = 1
			}
			if c.Resources == nil {
				c.Resources = []string{}
			}
			state.project.Courses = append(state.project.Courses, c)
			if state.project.Time.MaxWeeks == 1 && c.Planned > 0 {
				state.project.SetWeekAlloc(c.ID, 1, c.Planned)
			}
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		case http.MethodPut:
			var c Course
			if err := decodeBody(r, &c); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			state.pushUndo()
			for i := range state.project.Courses {
				if state.project.Courses[i].ID == c.ID {
					state.project.Courses[i] = c
					break
				}
			}
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			state.pushUndo()
			out := make([]Course, 0, len(state.project.Courses))
			for _, c := range state.project.Courses {
				if c.ID != id {
					out = append(out, c)
				}
			}
			state.project.Courses = out
			was := make([]WeekAllocation, 0, len(state.project.WeekAllocations))
			for _, wa := range state.project.WeekAllocations {
				if wa.CourseID != id {
					was = append(was, wa)
				}
			}
			state.project.WeekAllocations = was
			sls := make([]SlotAssignment, 0, len(state.project.Slots))
			for _, sl := range state.project.Slots {
				if sl.CourseID != id {
					sls = append(sls, sl)
				}
			}
			state.project.Slots = sls
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		}
	}))

	mux.HandleFunc("/api/courses/action", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Action    string `json:"action"`
			CourseID  string `json:"courseId"`
			By        string `json:"by"`
			ByCat     bool   `json:"byCategory"`
			Ascending bool   `json:"ascending"`
			MaxWeek   int    `json:"maxWeek"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		state.pushUndo()
		p := state.project

		switch req.Action {
		case "duplicate":
			if orig := p.CourseByID(req.CourseID); orig != nil {
				cp := *orig
				cp.ID = fmt.Sprintf("c_%d", time.Now().UnixNano())
				cp.Code = orig.Code + "-2"
				cp.Name = orig.Name + " (Copia)"
				cp.Resources = append([]string(nil), orig.Resources...)
				p.Courses = append(p.Courses, cp)
			}
		case "split":
			if orig := p.CourseByID(req.CourseID); orig != nil {
				half1 := orig.Planned / 2
				half2 := orig.Planned - half1
				orig.Planned = half1
				cp := *orig
				cp.ID = fmt.Sprintf("c_%d", time.Now().UnixNano())
				cp.Code = orig.Code + "-B"
				cp.Name = orig.Name + " (Parte 2)"
				cp.Planned = half2
				cp.Resources = append([]string(nil), orig.Resources...)
				p.Courses = append(p.Courses, cp)
			}
		case "unschedule":
			kept := make([]SlotAssignment, 0, len(p.Slots))
			for _, s := range p.Slots {
				if req.CourseID != "" && s.CourseID != req.CourseID {
					kept = append(kept, s)
				}
			}
			p.Slots = kept
		case "clear_weeks":
			kept := make([]WeekAllocation, 0, len(p.WeekAllocations))
			for _, wa := range p.WeekAllocations {
				if req.CourseID != "" && wa.CourseID != req.CourseID {
					kept = append(kept, wa)
				}
			}
			p.WeekAllocations = kept
		case "auto_allocate":
			mw := req.MaxWeek
			if mw < 1 || mw > p.Time.MaxWeeks {
				mw = p.Time.MaxWeeks
			}
			targets := p.Courses
			if req.CourseID != "" {
				if c := p.CourseByID(req.CourseID); c != nil {
					targets = []Course{*c}
				}
			}
			for _, c := range targets {
				rem := c.Planned - c.Allocated
				if rem <= 0 {
					continue
				}
				base := rem / mw
				extra := rem % mw
				for w := 1; w <= mw; w++ {
					add := base
					if w <= extra {
						add++
					}
					if add > 0 {
						p.SetWeekAlloc(c.ID, w, p.WeekAlloc(c.ID, w)+add)
					}
				}
			}
		case "sort":
			p.SortCourses(req.By, req.ByCat, req.Ascending)
		}
		state.afterMutateLocked()
		writeJSON(w, 200, state.responsePayload())
	}))

	// Resources CRUD + Sort
	mux.HandleFunc("/api/resources", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		switch r.Method {
		case http.MethodPost:
			var res Resource
			if err := decodeBody(r, &res); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			if strings.TrimSpace(res.Code) == "" {
				writeError(w, 400, "El código del recurso es obligatorio")
				return
			}
			state.pushUndo()
			if res.ID == "" {
				res.ID = fmt.Sprintf("r_%d", time.Now().UnixNano())
			}
			if res.Category < 1 || res.Category > 7 {
				res.Category = 1
			}
			state.project.Resources = append(state.project.Resources, res)
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		case http.MethodPut:
			var res Resource
			if err := decodeBody(r, &res); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			state.pushUndo()
			for i := range state.project.Resources {
				if state.project.Resources[i].ID == res.ID {
					state.project.Resources[i] = res
					break
				}
			}
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			state.pushUndo()
			out := make([]Resource, 0, len(state.project.Resources))
			for _, res := range state.project.Resources {
				if res.ID != id {
					out = append(out, res)
				}
			}
			state.project.Resources = out
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		}
	}))

	// Bulk import teachers (CSV / semicolon / tab / newline list)
	mux.HandleFunc("/api/resources/import-teachers", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text     string `json:"text"`
			Category int    `json:"category"` // defaults to 2 (Teachers)
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		cat := req.Category
		if cat < 1 || cat > 7 {
			cat = 2
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		state.pushUndo()
		added := 0
		existingCodes := make(map[string]bool)
		for _, res := range state.project.Resources {
			existingCodes[strings.ToUpper(res.Code)] = true
		}

		for _, rawLine := range strings.Split(req.Text, "\n") {
			line := strings.TrimSpace(rawLine)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Detect delimiter: ; or \t or ,
			sep := ";"
			if !strings.Contains(line, ";") {
				if strings.Contains(line, "\t") {
					sep = "\t"
				} else if strings.Contains(line, ",") {
					sep = ","
				}
			}
			parts := strings.Split(line, sep)
			for i := range parts {
				parts[i] = strings.TrimSpace(strings.Trim(parts[i], `"`))
			}
			// Skip header row if detected
			low0 := strings.ToLower(parts[0])
			if low0 == "code" || low0 == "codigo" || low0 == "código" || low0 == "nombre" || low0 == "name" {
				continue
			}

			var code, name, mcpNote string
			planned := 20
			if len(parts) == 1 {
				name = parts[0]
				toks := strings.Fields(name)
				base := toks[len(toks)-1]
				if len(base) > 7 {
					base = base[:7]
				}
				code = "P-" + strings.ToUpper(makeSlug(base))
			} else {
				code = parts[0]
				name = parts[1]
				if len(parts) >= 3 {
					if n, err := strconv.Atoi(parts[2]); err == nil && n > 0 {
						planned = n
					} else {
						mcpNote = parts[2]
					}
				}
				if len(parts) >= 4 && parts[3] != "" {
					mcpNote = parts[3]
				}
			}
			if name == "" {
				name = code
			}
			if code == "" {
				code = fmt.Sprintf("P-%d", len(state.project.Resources)+1)
			}
			origCode := code
			suffix := 2
			for existingCodes[strings.ToUpper(code)] {
				code = fmt.Sprintf("%s%d", origCode, suffix)
				suffix++
			}
			existingCodes[strings.ToUpper(code)] = true
			added++
			state.project.Resources = append(state.project.Resources, Resource{
				ID:       fmt.Sprintf("r_imp_%d_%d", time.Now().UnixNano(), added),
				Code:     code,
				Name:     name,
				Category: cat,
				Planned:  planned,
				McpNotes: mcpNote,
			})
		}
		state.afterMutateLocked()
		resp := state.responsePayload()
		resp["added"] = added
		writeJSON(w, 200, resp)
	}))

	// Batch set/clear teacher unavailability (single cell, full day, afternoons, or clear all — optionally across all weeks!)
	mux.HandleFunc("/api/bookings/batch", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TargetType string `json:"targetType"` // "resource" | "course"
			TargetID   string `json:"targetId"`
			Mode       string `json:"mode"`     // "toggle_cell" | "block_day" | "block_afternoons" | "clear_all"
			Week       int    `json:"week"`
			AllWeeks   bool   `json:"allWeeks"`
			Day        int    `json:"day"`
			Period     int    `json:"period"`
			Label      string `json:"label"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		if req.TargetType == "" {
			req.TargetType = "resource"
		}
		if req.Label == "" {
			req.Label = "No disponible"
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		state.pushUndo()
		p := state.project

		weeks := []int{req.Week}
		if req.AllWeeks || req.Week < 1 {
			weeks = make([]int, 0, p.Time.MaxWeeks)
			for wk := 1; wk <= p.Time.MaxWeeks; wk++ {
				weeks = append(weeks, wk)
			}
		}

		setCell := func(wk, d, h int, present bool) {
			for i := len(p.Bookings) - 1; i >= 0; i-- {
				b := p.Bookings[i]
				if b.TargetType == req.TargetType && b.TargetID == req.TargetID && b.Scope == "timetable" &&
					b.Week == wk && b.Day == d && b.Period == h {
					p.Bookings = append(p.Bookings[:i], p.Bookings[i+1:]...)
				}
			}
			if present {
				p.Bookings = append(p.Bookings, Booking{
					ID: fmt.Sprintf("b_%d_%d_%d_%d", time.Now().UnixNano(), wk, d, h),
					TargetType: req.TargetType, TargetID: req.TargetID,
					Scope: "timetable", Week: wk, Day: d, Period: h, Kind: 1, Label: req.Label,
				})
			}
		}

		switch req.Mode {
		case "clear_all":
			kept := make([]Booking, 0, len(p.Bookings))
			weekSet := make(map[int]bool, len(weeks))
			for _, wk := range weeks {
				weekSet[wk] = true
			}
			for _, b := range p.Bookings {
				if b.TargetType == req.TargetType && b.TargetID == req.TargetID && weekSet[b.Week] {
					continue
				}
				kept = append(kept, b)
			}
			p.Bookings = kept
		case "block_day":
			for _, wk := range weeks {
				for h := 1; h <= p.Time.MaxPeriods; h++ {
					setCell(wk, req.Day, h, true)
				}
			}
		case "block_afternoons":
			startP := (p.Time.MaxPeriods / 2) + 1
			for _, wk := range weeks {
				for d := 1; d <= p.Time.MaxDays; d++ {
					for h := startP; h <= p.Time.MaxPeriods; h++ {
						setCell(wk, d, h, true)
					}
				}
			}
		default: // toggle_cell
			exists := false
			for _, b := range p.Bookings {
				if b.TargetType == req.TargetType && b.TargetID == req.TargetID && b.Scope == "timetable" &&
					b.Week == weeks[0] && b.Day == req.Day && b.Period == req.Period {
					exists = true
					break
				}
			}
			for _, wk := range weeks {
				setCell(wk, req.Day, req.Period, !exists)
			}
		}

		state.afterMutateLocked()
		writeJSON(w, 200, state.responsePayload())
	}))

	mux.HandleFunc("/api/resources/sort", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			By        string `json:"by"`
			ByCat     bool   `json:"byCategory"`
			Ascending bool   `json:"ascending"`
		}
		_ = decodeBody(r, &req)
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		state.pushUndo()
		state.project.SortResources(req.By, req.ByCat, req.Ascending)
		state.afterMutateLocked()
		writeJSON(w, 200, state.responsePayload())
	}))

	mux.HandleFunc("/api/weeks/allocation", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CourseID string `json:"courseId"`
			Week     int    `json:"week"`
			Periods  int    `json:"periods"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		state.pushUndo()
		state.project.SetWeekAlloc(req.CourseID, req.Week, req.Periods)
		state.afterMutateLocked()
		writeJSON(w, 200, state.responsePayload())
	}))

	mux.HandleFunc("/api/slots", withCORS(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		switch r.Method {
		case http.MethodPost:
			var req struct {
				CourseID string `json:"courseId"`
				Week     int    `json:"week"`
				Day      int    `json:"day"`
				Period   int    `json:"period"`
				RoomID   string `json:"roomId"`
				Force    bool   `json:"force"`
			}
			if err := decodeBody(r, &req); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			if !req.Force {
				if ok, reasons := state.project.CheckSlotAvailability(req.CourseID, req.Week, req.Day, req.Period, req.RoomID, ""); !ok {
					writeJSON(w, 409, map[string]any{"error": strings.Join(reasons, "; "), "reasons": reasons})
					return
				}
			}
			state.pushUndo()
			state.project.Slots = append(state.project.Slots, SlotAssignment{
				ID:       fmt.Sprintf("s_%d", time.Now().UnixNano()),
				CourseID: req.CourseID,
				Week:     req.Week,
				Day:      req.Day,
				Period:   req.Period,
				RoomID:   req.RoomID,
			})
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		case http.MethodPut:
			var req struct {
				ID          string `json:"id"`
				Day         int    `json:"day"`
				Period      int    `json:"period"`
				RoomID      string `json:"roomId"`
				SetRoomOnly bool   `json:"setRoomOnly"`
			}
			if err := decodeBody(r, &req); err != nil {
				writeError(w, 400, err.Error())
				return
			}
			for i := range state.project.Slots {
				if state.project.Slots[i].ID == req.ID {
					s := state.project.Slots[i]
					if req.SetRoomOnly {
						state.pushUndo()
						state.project.Slots[i].RoomID = req.RoomID
						state.afterMutateLocked()
						writeJSON(w, 200, state.responsePayload())
						return
					}
					if ok, reasons := state.project.CheckSlotAvailability(s.CourseID, s.Week, req.Day, req.Period, s.RoomID, s.ID); !ok {
						writeJSON(w, 409, map[string]any{"error": strings.Join(reasons, "; "), "reasons": reasons})
						return
					}
					state.pushUndo()
					state.project.Slots[i].Day = req.Day
					state.project.Slots[i].Period = req.Period
					state.afterMutateLocked()
					writeJSON(w, 200, state.responsePayload())
					return
				}
			}
			writeError(w, 404, "slot no encontrado")
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			state.pushUndo()
			kept := make([]SlotAssignment, 0, len(state.project.Slots))
			for _, s := range state.project.Slots {
				if s.ID != id {
					kept = append(kept, s)
				}
			}
			state.project.Slots = kept
			state.afterMutateLocked()
			writeJSON(w, 200, state.responsePayload())
		}
	}))

	mux.HandleFunc("/api/whatif", withCORS(func(w http.ResponseWriter, r *http.Request) {
		cid := r.URL.Query().Get("courseId")
		week, _ := strconv.Atoi(r.URL.Query().Get("week"))
		if week < 1 {
			week = 1
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.project == nil {
			writeJSON(w, 200, map[string]any{"slots": []SlotAvailability{}})
			return
		}
		writeJSON(w, 200, map[string]any{"slots": state.project.WhatIfGrid(cid, week)})
	}))

	mux.HandleFunc("/api/bookings/toggle", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TargetType string `json:"targetType"`
			TargetID   string `json:"targetId"`
			Scope      string `json:"scope"`
			Week       int    `json:"week"`
			Day        int    `json:"day"`
			Period     int    `json:"period"`
			Kind       int    `json:"kind"`
			Label      string `json:"label"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		if req.Scope == "" {
			req.Scope = "timetable"
		}
		if req.Kind < 1 {
			req.Kind = 1
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		state.pushUndo()
		for i, b := range state.project.Bookings {
			if b.TargetType == req.TargetType && b.TargetID == req.TargetID && b.Scope == req.Scope &&
				b.Week == req.Week && b.Day == req.Day && b.Period == req.Period {
				state.project.Bookings = append(state.project.Bookings[:i], state.project.Bookings[i+1:]...)
				state.afterMutateLocked()
				writeJSON(w, 200, state.responsePayload())
				return
			}
		}
		state.project.Bookings = append(state.project.Bookings, Booking{
			ID:         fmt.Sprintf("b_%d", time.Now().UnixNano()),
			TargetType: req.TargetType,
			TargetID:   req.TargetID,
			Scope:      req.Scope,
			Week:       req.Week,
			Day:        req.Day,
			Period:     req.Period,
			Kind:       req.Kind,
			Label:      req.Label,
		})
		state.afterMutateLocked()
		writeJSON(w, 200, state.responsePayload())
	}))

	mux.HandleFunc("/api/optimize", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var opts OptimizeOptions
		if err := decodeBody(r, &opts); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		state.pushUndo()
		result := state.project.RunOptimizer(opts)
		state.afterMutateLocked()
		resp := state.responsePayload()
		resp["result"] = result
		writeJSON(w, 200, resp)
	}))

	mux.HandleFunc("/api/rooms/available", withCORS(func(w http.ResponseWriter, r *http.Request) {
		week, _ := strconv.Atoi(r.URL.Query().Get("week"))
		day, _ := strconv.Atoi(r.URL.Query().Get("day"))
		period, _ := strconv.Atoi(r.URL.Query().Get("period"))
		ignore := r.URL.Query().Get("ignoreSlotId")
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.project == nil {
			writeJSON(w, 200, map[string]any{"rooms": []Resource{}})
			return
		}
		writeJSON(w, 200, map[string]any{"rooms": state.project.AvailableRooms(week, day, period, ignore)})
	}))

	mux.HandleFunc("/api/rooms/action", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Action   string `json:"action"`
			FromWeek int    `json:"fromWeek"`
			ToWeek   int    `json:"toWeek"`
			SlotID   string `json:"slotId"`
			RoomID   string `json:"roomId"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		state.pushUndo()
		assigned := 0
		switch req.Action {
		case "imitate":
			assigned = state.project.ImitateRooms(req.FromWeek, req.ToWeek)
		case "clear_week":
			for i := range state.project.Slots {
				if state.project.Slots[i].Week == req.FromWeek {
					state.project.Slots[i].RoomID = ""
				}
			}
		case "clear_all":
			for i := range state.project.Slots {
				state.project.Slots[i].RoomID = ""
			}
		case "copy_same_time":
			var src *SlotAssignment
			for i := range state.project.Slots {
				if state.project.Slots[i].ID == req.SlotID {
					src = &state.project.Slots[i]
					break
				}
			}
			if src != nil && req.RoomID != "" {
				for i := range state.project.Slots {
					s := &state.project.Slots[i]
					if s.CourseID == src.CourseID && s.Day == src.Day && s.Period == src.Period {
						avail := state.project.AvailableRooms(s.Week, s.Day, s.Period, s.ID)
						for _, ar := range avail {
							if ar.ID == req.RoomID {
								s.RoomID = req.RoomID
								assigned++
								break
							}
						}
					}
				}
			}
		}
		state.afterMutateLocked()
		resp := state.responsePayload()
		resp["assigned"] = assigned
		writeJSON(w, 200, resp)
	}))

	// Import .mfw, .mxt, or .json into a new auto-saved project tab
	mux.HandleFunc("/api/import", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FileName string `json:"fileName"`
			Format   string `json:"format"`
			Content  string `json:"content"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		fmtLower := strings.ToLower(req.Format)
		if fmtLower == "" {
			fmtLower = strings.TrimPrefix(strings.ToLower(filepath.Ext(req.FileName)), ".")
		}

		var proj *Project
		var err error
		switch fmtLower {
		case "mfw":
			raw, berr := base64.StdEncoding.DecodeString(req.Content)
			if berr != nil {
				writeError(w, 400, "Error decodificando archivo .mfw: "+berr.Error())
				return
			}
			proj, err = ImportMFW(raw, req.FileName)
		case "mxt":
			proj, err = ImportMXT(req.Content, req.FileName)
		case "omimo", "json":
			var p Project
			err = json.Unmarshal([]byte(req.Content), &p)
			proj = &p
		default:
			writeError(w, 400, "Formato no soportado (usa .omimo, .mfw, .mxt o .json)")
			return
		}
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		if proj.ID == "" {
			proj.ID = fmt.Sprintf("proj_%d", time.Now().UnixNano())
		}
		_ = SaveProjectToDisk(proj)

		state.mu.Lock()
		defer state.mu.Unlock()
		state.openTabLocked(proj)
		writeJSON(w, 200, state.responsePayload())
	}))

	mux.HandleFunc("/api/export", withCORS(func(w http.ResponseWriter, r *http.Request) {
		format := strings.ToLower(r.URL.Query().Get("format"))
		reportType, _ := strconv.Atoi(r.URL.Query().Get("reportType"))
		week, _ := strconv.Atoi(r.URL.Query().Get("week"))
		resID := r.URL.Query().Get("resourceId")

		state.mu.Lock()
		defer state.mu.Unlock()
		if state.project == nil {
			writeError(w, 400, "no hay proyecto activo")
			return
		}

		switch format {
		case "mxt":
			writeJSON(w, 200, map[string]any{"fileName": safeFile(state.project.Name) + ".mxt", "content": ExportMXT(state.project)})
		case "csv":
			writeJSON(w, 200, map[string]any{"fileName": safeFile(state.project.Name) + ".csv", "content": ExportCSV(state.project)})
		case "ics":
			writeJSON(w, 200, map[string]any{"fileName": safeFile(state.project.Name) + ".ics", "content": ExportICS(state.project, resID)})
		case "html":
			if reportType < 1 || reportType > 6 {
				reportType = 3
			}
			writeJSON(w, 200, map[string]any{
				"fileName": fmt.Sprintf("%s_web_%d.html", safeFile(state.project.Name), reportType),
				"content":  GenerateHTMLReport(state.project, reportType, week),
			})
		default:
			state.project.Format = "openmimo/1"
			b, _ := json.MarshalIndent(state.project, "", "  ")
			writeJSON(w, 200, map[string]any{"fileName": safeFile(state.project.Name) + ".omimo", "content": string(b)})
		}
	}))

	mux.HandleFunc("/api/settings", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name               string       `json:"name"`
			Institution        string       `json:"institution"`
			Department         string       `json:"department"`
			Comment            string       `json:"comment"`
			Language           string       `json:"language"`
			RoomsCategory      int          `json:"roomsCategory"`
			Time               TimeSettings `json:"time"`
			CourseCategories   []Category   `json:"courseCategories"`
			ResourceCategories []Category   `json:"resourceCategories"`
			BookingTypes       []string     `json:"bookingTypes"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if err := state.guardEditableLocked(); err != nil {
			writeError(w, 403, err.Error())
			return
		}
		p := state.project
		usedW, usedD, usedP := 1, 1, 1
		for _, wa := range p.WeekAllocations {
			if wa.Periods > 0 && wa.Week > usedW {
				usedW = wa.Week
			}
		}
		for _, sl := range p.Slots {
			if sl.Week > usedW {
				usedW = sl.Week
			}
			if sl.Day > usedD {
				usedD = sl.Day
			}
			if sl.Period > usedP {
				usedP = sl.Period
			}
		}
		if req.Time.MaxWeeks > 0 && req.Time.MaxWeeks < usedW {
			writeError(w, 400, fmt.Sprintf("El máximo de semanas no puede ser menor que %d (hay datos programados)", usedW))
			return
		}
		if req.Time.MaxDays > 0 && req.Time.MaxDays < usedD {
			writeError(w, 400, fmt.Sprintf("El máximo de días por semana no puede ser menor que %d (hay clases programadas)", usedD))
			return
		}
		if req.Time.MaxPeriods > 0 && req.Time.MaxPeriods < usedP {
			writeError(w, 400, fmt.Sprintf("El máximo de bloques por día no puede ser menor que %d (hay clases programadas)", usedP))
			return
		}
		state.pushUndo()
		if req.Name != "" {
			p.Name = req.Name
		}
		p.Institution = req.Institution
		p.Department = req.Department
		p.Comment = req.Comment
		if req.Language != "" {
			p.Language = req.Language
		}
		if req.RoomsCategory >= 1 && req.RoomsCategory <= 7 {
			p.RoomsCategory = req.RoomsCategory
		}
		if req.Time.MaxWeeks > 0 {
			p.Time = req.Time
		}
		if len(req.CourseCategories) > 0 {
			p.CourseCategories = req.CourseCategories
		}
		if len(req.ResourceCategories) > 0 {
			p.ResourceCategories = req.ResourceCategories
		}
		if len(req.BookingTypes) > 0 {
			p.BookingTypes = req.BookingTypes
		}
		state.afterMutateLocked()
		writeJSON(w, 200, state.responsePayload())
	}))

	// Bind on 0.0.0.0 preferring predictable DefaultCollabPort (18787..18797, then :0)
	// so saved shared links survive restarts and firewall rules stay stable across sessions.
	var ln net.Listener
	var err error
	if *portFlag == 0 {
		for tryPort := DefaultCollabPort; tryPort <= DefaultCollabPort+10; tryPort++ {
			ln, err = net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", tryPort))
			if err == nil {
				break
			}
		}
		if ln == nil {
			ln, err = net.Listen("tcp4", "0.0.0.0:0")
		}
	} else {
		ln, err = net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", *portFlag))
	}
	if err != nil {
		ln, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *portFlag))
		if err != nil {
			fmt.Fprintf(os.Stderr, "listen failed: %v\n", err)
			os.Exit(1)
		}
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port
	state.mu.Lock()
	state.listenPort = actualPort
	state.refreshAllSharedLocked()
	state.mu.Unlock()
	fmt.Printf("OPENMIMO_PORT=%d\n", actualPort)
	_ = os.Stdout.Sync()

	srv := &http.Server{Handler: mux}
	if err := srv.Serve(ln); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func safeFile(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "horario"
	}
	r := strings.NewReplacer(" ", "_", "/", "-", "\\", "-", ":", "-", "—", "-")
	return r.Replace(s)
}
