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
	"strings"
	"time"
)

// ComputeIPFSCid computes a real IPFS CID via a local Kubo daemon (127.0.0.1:5001) when available,
// or falls back to a deterministic RFC-compliant CIDv1 (base32 raw+sha256 `bafk...`) so IPFS sharing
// works seamlessly both with and without an active local Kubo node.
func PublishToIPFS(payload []byte) (cid string, mode string) {
	// Try local Kubo IPFS API at http://127.0.0.1:5001/api/v0/add?pin=true&cid-version=1
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if fw, err := mw.CreateFormFile("file", "openmimo-project.json"); err == nil {
		_, _ = fw.Write(payload)
		_ = mw.Close()
		client := &http.Client{Timeout: 1200 * time.Millisecond}
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

func LocalLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

type ShareTokenPayload struct {
	HostURL   string `json:"h"`
	ProjectID string `json:"p"`
	IPFSCid   string `json:"c"`
}

func EncodeShareCode(hostURL, projectID, ipfsCid string) string {
	b, _ := json.Marshal(ShareTokenPayload{HostURL: hostURL, ProjectID: projectID, IPFSCid: ipfsCid})
	return "OMIPFS-" + base64.RawURLEncoding.EncodeToString(b)
}

func DecodeShareCode(code string) (ShareTokenPayload, error) {
	trimmed := strings.TrimSpace(code)
	trimmed = strings.TrimPrefix(trimmed, "OMIPFS-")
	raw, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return ShareTokenPayload{}, fmt.Errorf("código compartido inválido")
	}
	var tok ShareTokenPayload
	if err := json.Unmarshal(raw, &tok); err != nil || tok.HostURL == "" || tok.ProjectID == "" {
		return ShareTokenPayload{}, fmt.Errorf("código compartido incompleto")
	}
	return tok, nil
}

// ProbeRemoteHost queries the original PC to check whether the project is currently open in a tab.
// Per user requirement: a remote peer may ONLY edit the shared project while the host PC has it open.
func ProbeRemoteHost(hostURL, projectID string) (online bool, hostOpen bool, proj *Project) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(strings.TrimRight(hostURL, "/") + "/api/collab/status?projectId=" + projectID)
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
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&out); err != nil {
		return true, false, nil
	}
	return true, out.Open, out.Project
}

// PushToRemoteHost sends an updated project revision back to the host PC (only accepted if host has it open).
func PushToRemoteHost(hostURL, projectID string, p *Project) (*Project, error) {
	b, _ := json.Marshal(map[string]any{"projectId": projectID, "project": p})
	client := &http.Client{Timeout: 3000 * time.Millisecond}
	resp, err := client.Post(strings.TrimRight(hostURL, "/")+"/api/collab/push", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("el equipo anfitrión no está disponible; solo lectura")
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
