package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"inventario-backend/internal/scanner"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Permitir peticiones desde el frontend
	},
}

// ScannerHandler gestiona las rutas del escáner Symbol USB y WebSocket.
type ScannerHandler struct {
	DB  *sql.DB
	Hub *scanner.Hub
	// Auth valida el token enviado como primer mensaje del WebSocket.
	Auth *AuthHandler
}

// DeviceInfo retorna información del hardware Symbol USB conectado.
func (h *ScannerHandler) DeviceInfo(w http.ResponseWriter, r *http.Request) {
	info := scanner.GetScannerInfo(h.Hub != nil && h.Hub.EvdevActive())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// ScanInput estructura de entrada para registrar un escaneo.
type ScanInput struct {
	Code       string  `json:"code"`
	DurationMs float64 `json:"duration_ms"`
	Source     string  `json:"source"`
}

// RecordScan procesa un escaneo proveniente de la web (Wedge/Cámara/Manual).
func (h *ScannerHandler) RecordScan(w http.ResponseWriter, r *http.Request) {
	var input ScanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	cleanCode := strings.TrimSpace(input.Code)
	if cleanCode == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ignored_or_empty"})
		return
	}

	duration := input.DurationMs
	if duration <= 0 {
		duration = 25.0
	}
	src := input.Source
	if src == "" {
		src = "web_wedge"
	}

	rec, err := h.Hub.ProcessScan(cleanCode, duration, src)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if rec == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ignored_or_duplicate"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": rec})
}

// ListScans retorna el historial de escaneos de la sesión.
func (h *ScannerHandler) ListScans(w http.ResponseWriter, r *http.Request) {
	history := h.Hub.GetHistory()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// ClearScans limpia el historial en memoria.
func (h *ScannerHandler) ClearScans(w http.ResponseWriter, r *http.Request) {
	h.Hub.ClearHistory()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "cleared"})
}

// Catalog retorna los productos de prueba preconfigurados.
func (h *ScannerHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scanner.SampleProducts)
}

// Lookup realiza una búsqueda universal en el inventario por cualquier código escaneado.
func (h *ScannerHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		code = r.URL.Query().Get("q")
	}

	matchedType, entity, prod := h.Hub.Lookup(code)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":           code,
		"matched_type":   matchedType,
		"matched_entity": entity,
		"product":        prod,
	})
}

// BarcodePNG genera y sirve una imagen PNG del código de barras.
func (h *ScannerHandler) BarcodePNG(w http.ResponseWriter, r *http.Request) {
	formatName := chi.URLParam(r, "format")
	code := chi.URLParam(r, "code")
	// Quitar sufijo .png si vino en la URL
	code = strings.TrimSuffix(code, ".png")

	wStr := r.URL.Query().Get("w")
	hStr := r.URL.Query().Get("h")
	width, _ := strconv.Atoi(wStr)
	height, _ := strconv.Atoi(hStr)

	pngBytes, err := scanner.GeneratePNG(formatName, code, width, height)
	if err != nil {
		http.Error(w, "Error generando código de barras: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(pngBytes)
}

// BarcodeSVG genera y sirve un vector SVG nítido del código de barras.
func (h *ScannerHandler) BarcodeSVG(w http.ResponseWriter, r *http.Request) {
	formatName := chi.URLParam(r, "format")
	code := chi.URLParam(r, "code")
	// Quitar sufijo .svg si vino en la URL
	code = strings.TrimSuffix(code, ".svg")

	svgStr, err := scanner.GenerateSVG(formatName, code)
	if err != nil {
		http.Error(w, "Error generando SVG: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write([]byte(svgStr))
}

// WebSocketEndpoint maneja la conexión WebSocket en tiempo real.
func (h *ScannerHandler) WebSocketEndpoint(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Autenticación por primer mensaje {"type":"auth","token":"..."}: el token no
	// viaja en la URL (queda fuera de logs/historial). Sin auth válida en 5s se cierra.
	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var first struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" || h.Auth == nil || !h.Auth.ValidToken(first.Token) {
		_ = conn.WriteJSON(map[string]string{"type": "error", "error": "no autenticado"})
		conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	h.Hub.RegisterClient(conn)
	defer h.Hub.UnregisterClient(conn)

	// Bucle de lectura para recibir pings o comandos del cliente
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msgType, ok := msg["type"].(string); ok && msgType == "ping" {
			_ = conn.WriteJSON(map[string]string{"type": "pong"})
		}
	}
}
