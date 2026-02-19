package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"embed"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	qrcode "github.com/skip2/go-qrcode"
)

// tlsErrorFilter is an io.Writer that suppresses TLS handshake errors.
// These are expected with self-signed certificates and just clutter the log.
type tlsErrorFilter struct{}

func (t *tlsErrorFilter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if strings.Contains(msg, "TLS handshake error") {
		return len(p), nil // silently discard
	}
	// Pass through any non-TLS errors
	return fmt.Fprint(log.Writer(), msg)
}

//go:embed web/*
var webFS embed.FS

const pairSessionTTL = 24 * time.Hour

// Message represents a WebSocket message from the phone.
type Message struct {
	Type string `json:"type"` // "text", "command"
	Text string `json:"text"`
	Mode string `json:"mode,omitempty"` // "raw", "tidy", "formal", "translate"
}

// StatusResponse represents the server status.
type StatusResponse struct {
	Connected     bool   `json:"connected"`
	ClientAddr    string `json:"clientAddr,omitempty"`
	ServerAddr    string `json:"serverAddr"`
	StartedAt     string `json:"startedAt"`
	AIAvailable   bool   `json:"aiAvailable"`
	Paired        bool   `json:"paired"`
	PairRequired  bool   `json:"pairRequired"`
	PairExpiresAt string `json:"pairExpiresAt,omitempty"`
}

// Server holds the HTTP/WebSocket server state.
type Server struct {
	mu             sync.RWMutex
	conn           *websocket.Conn
	clientAddr     string
	pairedDeviceID string
	pairedUntil    time.Time
	startedAt      time.Time
	addr           string
	lanIPOverride  string
	authToken      string
	pairCode       string
	upgrader       websocket.Upgrader
	ai             *AIProcessor
	hasSentText    bool // track if we've sent text to PC, for auto-newline
}

// NewServer creates a new Server instance.
func NewServer(addr string) *Server {
	ai := NewAIProcessor()
	cfg := LoadConfig()
	if ai.IsAvailable() {
		log.Printf("AI processing enabled (model: %s)", ai.model)
	} else {
		log.Printf("AI processing disabled (set DEEPSEEK_API_KEY to enable)")
	}

	authToken, err := generateAuthToken()
	if err != nil {
		log.Printf("failed to generate session token, falling back to timestamp token: %v", err)
		authToken = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	pairCode, err := generatePairCode()
	if err != nil {
		log.Printf("failed to generate pair code, falling back to 0000: %v", err)
		pairCode = "0000"
	}
	lanIPOverride := strings.TrimSpace(os.Getenv("GTALK_LAN_IP"))
	if lanIPOverride == "" {
		lanIPOverride = strings.TrimSpace(cfg.LanIP)
	}
	if lanIPOverride != "" {
		ip := net.ParseIP(lanIPOverride)
		if ip == nil || ip.To4() == nil {
			log.Printf("invalid LAN IP override: %s, falling back to auto-detect", lanIPOverride)
			lanIPOverride = ""
		} else {
			log.Printf("using configured LAN IP: %s", lanIPOverride)
		}
	}

	return &Server{
		addr:          addr,
		startedAt:     time.Now(),
		lanIPOverride: lanIPOverride,
		authToken:     authToken,
		pairCode:      pairCode,
		ai:            ai,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for LAN usage
			},
		},
	}
}

func (s *Server) LanIP() string {
	if s.lanIPOverride != "" {
		return s.lanIPOverride
	}
	return getLanIP()
}

// SetLanIPOverride sets or clears the LAN IP override.
func (s *Server) SetLanIPOverride(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lanIPOverride = ip
}

// GetLanIPOverride returns the current LAN IP override value.
func (s *Server) GetLanIPOverride() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lanIPOverride
}

// Start launches the HTTPS server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Serve embedded PWA files
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(webContent)))

	// API endpoints
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/qrcode", s.handleQRCode)
	mux.HandleFunc("/api/pair", s.handlePair)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/config", s.handleConfig)

	// Generate self-signed TLS cert for HTTPS (required for Web Speech API)
	tlsCert, err := generateSelfSignedCert(s.LanIP())
	if err != nil {
		return fmt.Errorf("failed to generate TLS cert: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	server := &http.Server{
		Addr:      s.addr,
		Handler:   mux,
		TLSConfig: tlsConfig,
		ErrorLog:  log.New(&tlsErrorFilter{}, "", 0),
	}

	log.Printf("Ginkgo Talk server starting on https://%s", s.addr)
	log.Printf("Scan the QR code to connect your phone")
	log.Printf("Pair code: %s", s.pairCode)

	// ListenAndServeTLS with empty filenames uses the TLS config certs
	return server.ListenAndServeTLS("", "")
}

// handleWebSocket handles WebSocket connections from the phone.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.isTokenAuthorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		log.Printf("Unauthorized WebSocket attempt from %s", r.RemoteAddr)
		return
	}
	if !s.isClientPaired(r) {
		http.Error(w, "pair required", http.StatusForbidden)
		log.Printf("WebSocket pair required for %s", r.RemoteAddr)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Extract base IP (without port) to detect same-client reconnects
	clientIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		clientIP = host
	}

	// Store connection
	s.mu.Lock()
	if s.conn != nil {
		s.conn.Close() // Close previous connection
	}
	s.conn = conn
	s.clientAddr = r.RemoteAddr
	s.mu.Unlock()

	log.Printf("Phone connected from %s", r.RemoteAddr)
	_ = clientIP

	defer func() {
		s.mu.Lock()
		if s.conn == conn {
			s.conn = nil
			s.clientAddr = ""
		}
		s.mu.Unlock()
		conn.Close()
		log.Printf("Phone disconnected from %s", r.RemoteAddr)
	}()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("Invalid message: %v", err)
			continue
		}

		switch msg.Type {
		case "text":
			if msg.Text != "" {
				outputText := msg.Text
				mode := AIMode(msg.Mode)
				if mode == "" {
					mode = ModeRaw
				}

				// AI processing
				if mode != ModeRaw && s.ai.IsAvailable() {
					log.Printf("AI processing [%s]: %s", mode, msg.Text)
					conn.WriteJSON(map[string]string{
						"type":   "processing",
						"text":   msg.Text,
						"status": "ai_processing",
					})
					processed, err := s.ai.Process(msg.Text, mode)
					if err != nil {
						log.Printf("AI error: %v", err)
						conn.WriteJSON(map[string]string{
							"type":  "ai_error",
							"error": err.Error(),
						})
					} else {
						log.Printf("AI result: %s", processed)
						// Return to client for preview, don't type yet
						conn.WriteJSON(map[string]interface{}{
							"type":     "ai_preview",
							"text":     processed,
							"original": msg.Text,
							"mode":     string(mode),
						})
					}
				} else {
					// Raw mode: type then submit (equivalent to pressing Enter on PC).
					log.Printf("Typing and sending: %s", outputText)

					if err := TypeText(outputText); err != nil {
						log.Printf("SendInput error: %v", err)
						conn.WriteJSON(map[string]string{
							"type":  "error",
							"error": err.Error(),
						})
					} else if err := PressEnter(); err != nil {
						log.Printf("PressEnter error: %v", err)
						conn.WriteJSON(map[string]string{
							"type":  "error",
							"error": err.Error(),
						})
					} else {
						s.hasSentText = false
						conn.WriteJSON(map[string]interface{}{
							"type":     "ack",
							"text":     outputText,
							"original": msg.Text,
							"mode":     string(mode),
							"status":   "sent",
						})
					}
				}
			}
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		case "command":
			switch msg.Text {
			case "clear":
				log.Printf("Clear PC input field")
				if err := SelectAllAndDelete(); err != nil {
					log.Printf("Clear error: %v", err)
					conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
				} else {
					s.hasSentText = false
					conn.WriteJSON(map[string]string{"type": "ack", "status": "cleared"})
				}
			case "enter":
				log.Printf("Enter")
				if err := PressEnter(); err != nil {
					conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
				} else {
					s.hasSentText = false
					conn.WriteJSON(map[string]string{"type": "ack", "status": "enter"})
				}
			case "shift_enter":
				log.Printf("Shift+Enter")
				if err := PressShiftEnter(); err != nil {
					conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
				} else {
					conn.WriteJSON(map[string]string{"type": "ack", "status": "shift_enter"})
				}
			case "ctrl_z":
				log.Printf("Ctrl+Z (undo)")
				if err := PressCtrlZ(); err != nil {
					conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
				} else {
					conn.WriteJSON(map[string]string{"type": "ack", "status": "ctrl_z"})
				}
			case "ctrl_v":
				log.Printf("Ctrl+V (paste)")
				if err := PressCtrlV(); err != nil {
					conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
				} else {
					conn.WriteJSON(map[string]string{"type": "ack", "status": "ctrl_v"})
				}
			case "tab":
				log.Printf("Tab")
				if err := PressTab(); err != nil {
					conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
				} else {
					conn.WriteJSON(map[string]string{"type": "ack", "status": "tab"})
				}
			case "escape":
				log.Printf("Escape")
				if err := PressEscape(); err != nil {
					conn.WriteJSON(map[string]string{"type": "error", "error": err.Error()})
				} else {
					conn.WriteJSON(map[string]string{"type": "ack", "status": "escape"})
				}
			default:
				log.Printf("Unknown command: %s", msg.Text)
			}
		}
	}
}

// handleQRCode generates and serves a QR code PNG image.
func (s *Server) handleQRCode(w http.ResponseWriter, r *http.Request) {
	lanIP := s.LanIP()
	url := fmt.Sprintf("https://%s%s?token=%s", lanIP, s.addr, s.authToken)

	png, err := qrcode.Encode(url, qrcode.Medium, 512)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(png)
}

// handleStatus returns the current server status.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !s.isTokenAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	s.mu.RLock()
	connected := s.conn != nil
	clientAddr := s.clientAddr
	s.mu.RUnlock()
	deviceID := deviceIDFromRequest(r)
	paired, pairExpiresAt := s.pairingState(deviceID)

	lanIP := s.LanIP()
	resp := StatusResponse{
		Connected:     connected,
		ClientAddr:    clientAddr,
		ServerAddr:    fmt.Sprintf("https://%s%s", lanIP, s.addr),
		StartedAt:     s.startedAt.Format(time.RFC3339),
		AIAvailable:   s.ai.IsAvailable(),
		Paired:        paired,
		PairRequired:  !paired,
		PairExpiresAt: pairExpiresAt.Format(time.RFC3339),
	}
	if pairExpiresAt.IsZero() {
		resp.PairExpiresAt = ""
	}

	json.NewEncoder(w).Encode(resp)
}

// handleConfig handles API key configuration.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !s.isTokenAuthorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	if !s.isClientPaired(r) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "pair required"})
		return
	}

	if r.Method == http.MethodPost {
		var body struct {
			APIKey  string `json:"apiKey"`
			BaseURL string `json:"baseUrl"`
			Model   string `json:"model"`
			LanIP   string `json:"lanIp"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}
		if body.APIKey != "" {
			s.ai.SetAPIKey(body.APIKey)
			log.Printf("API key updated from phone UI")
		}
		if body.BaseURL != "" {
			s.ai.baseURL = strings.TrimRight(body.BaseURL, "/")
			log.Printf("API base URL: %s", s.ai.baseURL)
		}
		if body.Model != "" {
			s.ai.model = body.Model
			log.Printf("Model: %s", s.ai.model)
		}

		if body.LanIP != "" {
			lanIP := strings.TrimSpace(body.LanIP)
			if strings.EqualFold(lanIP, "auto") {
				s.lanIPOverride = ""
				log.Printf("LAN IP override cleared, back to auto-detect")
			} else {
				ip := net.ParseIP(lanIP)
				if ip == nil || ip.To4() == nil {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "invalid lanIp"})
					return
				}
				s.lanIPOverride = lanIP
				log.Printf("LAN IP override updated: %s", s.lanIPOverride)
			}
		}

		// Persist config to disk
		SaveConfig(Config{
			APIKey:  s.ai.apiKey,
			BaseURL: s.ai.baseURL,
			Model:   s.ai.model,
			LanIP:   s.lanIPOverride,
		})

		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":          true,
			"aiAvailable": s.ai.IsAvailable(),
			"model":       s.ai.model,
			"baseUrl":     s.ai.baseURL,
			"lanIp":       s.lanIPOverride,
		})
		return
	}

	// GET: return current config (mask key)
	maskedKey := ""
	if s.ai.apiKey != "" {
		k := s.ai.apiKey
		if len(k) > 8 {
			maskedKey = k[:4] + "****" + k[len(k)-4:]
		} else {
			maskedKey = "****"
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"apiKey":      maskedKey,
		"baseUrl":     s.ai.baseURL,
		"model":       s.ai.model,
		"lanIp":       s.lanIPOverride,
		"aiAvailable": s.ai.IsAvailable(),
	})
}

func generateAuthToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func generatePairCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	authorized := s.isTokenAuthorized(r)

	deviceID := deviceIDFromRequest(r)
	if r.Method == http.MethodGet {
		if !authorized {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		paired, pairExpiresAt := s.pairingState(deviceID)
		pairExpiresText := ""
		if !pairExpiresAt.IsZero() {
			pairExpiresText = pairExpiresAt.Format(time.RFC3339)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"paired":        paired,
			"pairRequired":  !paired,
			"pairExpiresAt": pairExpiresText,
		})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	var body struct {
		Code     string `json:"code"`
		DeviceID string `json:"deviceId,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}
	if deviceID == "" {
		deviceID = strings.TrimSpace(body.DeviceID)
	}
	if deviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing device id"})
		return
	}

	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(body.Code)), []byte(s.pairCode)) != 1 {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid pair code"})
		return
	}

	s.mu.Lock()
	s.pairedDeviceID = deviceID
	s.pairedUntil = time.Now().Add(pairSessionTTL)
	pairedUntil := s.pairedUntil
	s.mu.Unlock()
	log.Printf("Paired device: %s (expires: %s)", deviceID, pairedUntil.Format(time.RFC3339))

	pairExpiresAt := pairedUntil.Format(time.RFC3339)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":            true,
		"paired":        true,
		"pairRequired":  false,
		"pairExpiresAt": pairExpiresAt,
		"token":         s.authToken,
	})
}

func (s *Server) isTokenAuthorized(r *http.Request) bool {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-GTalk-Token"))
	}
	if token == "" || s.authToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.authToken)) == 1
}

func (s *Server) isClientPaired(r *http.Request) bool {
	deviceID := deviceIDFromRequest(r)
	paired, _ := s.pairingState(deviceID)
	return paired
}

func (s *Server) pairingState(deviceID string) (bool, time.Time) {
	if deviceID == "" {
		return false, time.Time{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pairedDeviceID == "" || s.pairedUntil.IsZero() {
		return false, time.Time{}
	}
	if time.Now().After(s.pairedUntil) {
		s.pairedDeviceID = ""
		s.pairedUntil = time.Time{}
		return false, time.Time{}
	}
	if subtle.ConstantTimeCompare([]byte(deviceID), []byte(s.pairedDeviceID)) != 1 {
		return false, time.Time{}
	}
	return true, s.pairedUntil
}

func deviceIDFromRequest(r *http.Request) string {
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.Header.Get("X-GTalk-Device"))
	}
	return deviceID
}

// getLanIP returns the best non-loopback LAN IPv4 address.
// It uses interface names to prioritize real physical adapters (WiFi/Ethernet)
// over virtual adapters (VMware, VPN, Docker, Hyper-V, etc).
func getLanIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "localhost"
	}

	type candidate struct {
		ip       string
		priority int // lower is better
	}
	var candidates []candidate

	for _, iface := range ifaces {
		// Skip down, loopback, or non-multicast interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		name := strings.ToLower(iface.Name)
		prio := classifyInterface(name)

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil || ipNet.IP.IsLoopback() {
				continue
			}
			ip := ipNet.IP.To4()
			// Skip link-local 169.254.x.x
			if ip[0] == 169 && ip[1] == 254 {
				continue
			}
			candidates = append(candidates, candidate{ip: ip.String(), priority: prio})
		}
	}

	if len(candidates) == 0 {
		return "localhost"
	}

	// Pick the candidate with the best (lowest) priority
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.priority < best.priority {
			best = c
		}
	}
	return best.ip
}

// classifyInterface assigns a priority to a network interface by name.
// Lower number = higher priority (more likely to be the real LAN adapter).
func classifyInterface(name string) int {
	// Virtual / VPN adapters — deprioritize
	virtualKeywords := []string{
		"vmware", "vmnet", "virtualbox", "vbox",
		"hyper-v", "vethernet",
		"docker", "veth", "br-",
		"vpn", "starvpn", "tap", "tun", "wireguard", "wg",
		"蓝牙", "bluetooth",
		"loopback",
		"nodebabylink",
	}
	for _, kw := range virtualKeywords {
		if strings.Contains(name, kw) {
			return 90
		}
	}

	// Real physical adapters — high priority
	physicalKeywords := []string{
		"wlan", "wi-fi", "wifi", "无线",
		"以太网", "ethernet", "eth",
		"en0", "en1",
	}
	for _, kw := range physicalKeywords {
		if strings.Contains(name, kw) {
			return 10
		}
	}

	// Unknown interface — middle priority
	return 50
}

// generateSelfSignedCert loads an existing TLS certificate from disk,
// or generates a new one and saves it for reuse across restarts.
func generateSelfSignedCert(lanIP string) (tls.Certificate, error) {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	// Try loading existing cert
	if cert, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
		// Verify the cert is still valid and covers the current LAN IP
		if x509Cert, err := x509.ParseCertificate(cert.Certificate[0]); err == nil {
			ipMatch := false
			for _, ip := range x509Cert.IPAddresses {
				if ip.String() == lanIP {
					ipMatch = true
					break
				}
			}
			if ipMatch && time.Now().Before(x509Cert.NotAfter) {
				log.Printf("Loaded existing TLS certificate (valid until %s)", x509Cert.NotAfter.Format("2006-01-02"))
				return cert, nil
			}
			log.Printf("Certificate expired or IP changed, regenerating...")
		}
	}

	// Generate new cert
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Ginkgo Talk"},
			CommonName:   "Ginkgo Talk Local Server",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add all local IPs as SANs
	template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	if lanIP != "localhost" {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(lanIP))
	}
	template.DNSNames = []string{"localhost"}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Save cert to PEM files for reuse
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		log.Printf("Could not save cert.pem: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		log.Printf("Could not save key.pem: %v", err)
	}
	log.Printf("Generated new TLS certificate, saved to %s", dir)

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}, nil
}
