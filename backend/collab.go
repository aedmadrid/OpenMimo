package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultCollabPort is preferred so saved shared links survive app restarts and firewall rules stay stable.
const DefaultCollabPort = 18787

// PublishToIPFS computes a real IPFS CID via a local Kubo daemon (127.0.0.1:5001) when available,
// or falls back to a deterministic RFC-compliant CIDv1 (base32 raw+sha256 `b...`) so IPFS sharing
// works seamlessly both with and without an active local Kubo node.
func PublishToIPFS(payload []byte) (cid string, mode string) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if fw, err := mw.CreateFormFile("file", "openmimo-project.omimo"); err == nil {
		_, _ = fw.Write(payload)
		_ = mw.Close()
		client := &http.Client{Timeout: 800 * time.Millisecond}
		req, _ := http.NewRequest("POST", "http://127.0.0.1:5001/api/v0/add?pin=true&cid-version=1", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		if resp, err := client.Do(req); err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				var out struct {
					Hash string `json:"Hash"`
				}
				if json.NewDecoder(resp.Body).Decode(&out) == nil && out.Hash != "" {
					return out.Hash, "kubo"
				}
			}
		}
	}

	// Deterministic CIDv1 (version=0x01, codec=raw 0x55, multihash=sha2-256 0x12 len=0x20 + 32-byte digest)
	sum := sha256.Sum256(payload)
	rawCid := make([]byte, 0, 36)
	rawCid = append(rawCid, 0x01, 0x55, 0x12, 0x20)
	rawCid = append(rawCid, sum[:]...)
	enc := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawCid))
	return "b" + enc, "cid-local"
}

func isVirtualOrTunnelIface(name string) bool {
	low := strings.ToLower(name)
	for _, pref := range []string{"utun", "tun", "tap", "wg", "docker", "br-", "veth", "vmnet", "vbox", "awdl", "llw", "bridge", "lo"} {
		if strings.HasPrefix(low, pref) {
			return true
		}
	}
	return false
}

// LocalLANIPs returns all reachable IPv4 addresses of this machine ordered with
// real physical LAN interfaces (Wi-Fi/Ethernet, e.g. en0/wlan0/eth0) first,
// followed by VPN/tunnel interfaces (e.g. Tailscale/WireGuard), and 127.0.0.1 last.
func LocalLANIPs() []string {
	var lanIPs []string
	var vpnIPs []string
	seen := make(map[string]bool)

	addUnique := func(dst *[]string, ip string) {
		if ip == "" || seen[ip] {
			return
		}
		seen[ip] = true
		*dst = append(*dst, ip)
	}

	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			virt := isVirtualOrTunnelIface(iface.Name)
			for _, a := range addrs {
				ipnet, ok := a.(*net.IPNet)
				if !ok {
					continue
				}
				ip4 := ipnet.IP.To4()
				if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() || ip4.IsLinkLocalMulticast() {
					continue
				}
				if virt {
					addUnique(&vpnIPs, ip4.String())
				} else {
					addUnique(&lanIPs, ip4.String())
				}
			}
		}
	}

	// Also query OS routing table for primary outbound IPv4
	if conn, err := net.DialTimeout("udp4", "8.8.8.8:80", 400*time.Millisecond); err == nil {
		if ua, ok := conn.LocalAddr().(*net.UDPAddr); ok && ua.IP != nil {
			if ip4 := ua.IP.To4(); ip4 != nil && !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() {
				addUnique(&vpnIPs, ip4.String())
			}
		}
		_ = conn.Close()
	}

	out := make([]string, 0, len(lanIPs)+len(vpnIPs)+1)
	out = append(out, lanIPs...)
	out = append(out, vpnIPs...)
	addUnique(&out, "127.0.0.1")
	return out
}

func LocalLANIP() string {
	ips := LocalLANIPs()
	if len(ips) > 0 {
		return ips[0]
	}
	return "127.0.0.1"
}

func BuildHostURLs(port int) (primary string, alts []string) {
	ips := LocalLANIPs()
	urls := make([]string, 0, len(ips))
	for _, ip := range ips {
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip, port))
	}
	if len(urls) == 0 {
		return fmt.Sprintf("http://127.0.0.1:%d", port), nil
	}
	return urls[0], urls[1:]
}

type ShareTokenPayload struct {
	HostURL   string   `json:"h"`
	AltURLs   []string `json:"a,omitempty"`
	ProjectID string   `json:"p"`
	IPFSCid   string   `json:"c"`
}

func EncodeShareCode(hostURL string, altURLs []string, projectID, ipfsCid string) string {
	b, _ := json.Marshal(ShareTokenPayload{
		HostURL:   hostURL,
		AltURLs:   altURLs,
		ProjectID: projectID,
		IPFSCid:   ipfsCid,
	})
	return "OMIPFS-" + base64.RawURLEncoding.EncodeToString(b)
}

func DecodeShareCode(code string) (ShareTokenPayload, error) {
	trimmed := strings.TrimSpace(code)
	// Strip whitespace/newlines that often sneak in when copying from chat apps
	trimmed = strings.Join(strings.Fields(trimmed), "")
	trimmed = strings.TrimPrefix(trimmed, "OMIPFS-")
	raw, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		// Also tolerate standard base64 padding if pasted from another tool
		raw, err = base64.URLEncoding.DecodeString(trimmed)
		if err != nil {
			return ShareTokenPayload{}, fmt.Errorf("código compartido inválido")
		}
	}
	var tok ShareTokenPayload
	if err := json.Unmarshal(raw, &tok); err != nil || tok.HostURL == "" || tok.ProjectID == "" {
		return ShareTokenPayload{}, fmt.Errorf("código compartido incompleto")
	}
	return tok, nil
}

// expandCandidateURLs deduplicates all known URLs and also adds the predictable DefaultCollabPort
// fallback for each host IP in case the host restarted OpenMimo since the code was copied.
func expandCandidateURLs(primary string, alts []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(alts)+4)
	add := func(u string) {
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	add(primary)
	for _, a := range alts {
		add(a)
	}
	// Also try DefaultCollabPort on the same hostnames/IPs if the code had an older ephemeral port
	baseList := append([]string{}, out...)
	for _, raw := range baseList {
		if parsed, err := url.Parse(raw); err == nil && parsed.Hostname() != "" {
			add(fmt.Sprintf("http://%s:%d", parsed.Hostname(), DefaultCollabPort))
		}
	}
	return out
}

func probeSingleHost(hostURL, projectID string) (online bool, hostOpen bool, proj *Project) {
	client := &http.Client{Timeout: 3500 * time.Millisecond}
	endpoint := strings.TrimRight(hostURL, "/") + "/api/collab/status?projectId=" + url.QueryEscape(projectID)
	resp, err := client.Get(endpoint)
	if err != nil {
		return false, false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return true, false, nil
	}
	var out struct {
		Open    bool     `json:"open"`
		Project *Project `json:"project"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&out); err != nil {
		return true, false, nil
	}
	return true, out.Open, out.Project
}

// ProbeRemoteHost queries all candidate addresses of the host PC in parallel to check whether
// the project is currently open in a tab. Returns the working URL along with status & project.
func ProbeRemoteHost(hostURL string, altURLs []string, projectID string) (workingURL string, online bool, hostOpen bool, proj *Project) {
	candidates := expandCandidateURLs(hostURL, altURLs)
	if len(candidates) == 0 {
		return hostURL, false, false, nil
	}

	type res struct {
		idx      int
		url      string
		online   bool
		hostOpen bool
		proj     *Project
	}
	ch := make(chan res, len(candidates))
	var wg sync.WaitGroup
	for i, u := range candidates {
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()
			on, op, p := probeSingleHost(target, projectID)
			ch <- res{idx: idx, url: target, online: on, hostOpen: op, proj: p}
		}(i, u)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	results := make([]res, 0, len(candidates))
	for r := range ch {
		if r.online && r.hostOpen && r.proj != nil {
			return r.url, true, true, r.proj
		}
		results = append(results, r)
	}

	// If none returned open==true, check if ANY candidate was reachable (host online, project genuinely closed)
	for _, r := range results {
		if r.online {
			return r.url, true, false, nil
		}
	}
	return candidates[0], false, false, nil
}

// PushToRemoteHost sends an updated project revision back to the host PC (only accepted if host has it open).
func PushToRemoteHost(hostURL, projectID string, p *Project) (*Project, error) {
	b, _ := json.Marshal(map[string]any{"projectId": projectID, "project": p})
	client := &http.Client{Timeout: 4500 * time.Millisecond}
	resp, err := client.Post(strings.TrimRight(hostURL, "/")+"/api/collab/push", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar con el equipo anfitrión (%s); solo lectura", hostURL)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("el equipo original cerró este archivo (solo lectura): %s", strings.TrimSpace(string(body)))
	}
	var out struct {
		Project *Project `json:"project"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Project == nil {
		return p, nil
	}
	return out.Project, nil
}
