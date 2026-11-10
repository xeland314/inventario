package scanner

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ScanRecord registra un evento de lectura de código de barras.
type ScanRecord struct {
	ID            string         `json:"id"`
	Code          string         `json:"code"`
	Timestamp     string         `json:"timestamp"`
	DurationMs    float64        `json:"duration_ms"`
	Source        string         `json:"source"`
	IsScanner     bool           `json:"is_scanner"`
	MatchedType   string         `json:"matched_type"` // dispositivo, conjunto, frasco, etiqueta, demo_catalog, unknown
	MatchedEntity interface{}    `json:"matched_entity,omitempty"`
	Product       *SampleProduct `json:"product,omitempty"`
}

// Hub administra conexiones WebSocket y el historial de escaneos en memoria.
type Hub struct {
	mu            sync.RWMutex
	clients       map[*websocket.Conn]bool
	scanHistory   []ScanRecord
	lastScanDedup map[string]int64
	evdevListener *EvdevListener
	db            *sql.DB
}

// NewHub crea una nueva instancia del Hub.
func NewHub(db *sql.DB) *Hub {
	h := &Hub{
		clients:       make(map[*websocket.Conn]bool),
		scanHistory:   make([]ScanRecord, 0, 100),
		lastScanDedup: make(map[string]int64),
		db:            db,
	}

	// Iniciar listener de evdev con callback a ProcessScan
	h.evdevListener = NewEvdevListener(func(code string, durationMs float64, source string) {
		_, _ = h.ProcessScan(code, durationMs, source)
	})
	h.evdevListener.Start()

	return h
}

// Close detiene el monitor evdev y cierra las conexiones WebSocket.
func (h *Hub) Close() {
	if h.evdevListener != nil {
		h.evdevListener.Stop()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		conn.Close()
	}
	h.clients = make(map[*websocket.Conn]bool)
}

// EvdevActive indica si el listener de hardware está corriendo.
func (h *Hub) EvdevActive() bool {
	if h.evdevListener == nil {
		return false
	}
	return h.evdevListener.IsRunning()
}

// RegisterClient añade una nueva conexión WebSocket.
func (h *Hub) RegisterClient(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Enviar estado inicial
	info := GetScannerInfo(h.EvdevActive())
	_ = conn.WriteJSON(map[string]interface{}{
		"type":   "init",
		"device": info,
		"scans":  h.GetHistoryLimit(15),
	})
}

// UnregisterClient remueve una conexión WebSocket.
func (h *Hub) UnregisterClient(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
}

// Broadcast envía un mensaje a todos los clientes WebSocket activos.
func (h *Hub) Broadcast(msg interface{}) {
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	for _, c := range conns {
		if err := c.WriteJSON(msg); err != nil {
			log.Printf("[ws] Error enviando a cliente: %v", err)
			h.UnregisterClient(c)
		}
	}
}

// ProcessScan procesa, deduplica, enriquece con la base de datos y difunde el código escaneado.
func (h *Hub) ProcessScan(code string, durationMs float64, source string) (*ScanRecord, error) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, nil
	}

	nowMs := time.Now().UnixMilli()

	h.mu.Lock()
	lastTime, exists := h.lastScanDedup[cleanCode]
	if exists && (nowMs-lastTime) < 400 {
		h.mu.Unlock()
		log.Printf("[scanner] Escaneo duplicado descartado (%s, %dms): %s", source, nowMs-lastTime, cleanCode)
		return nil, nil
	}
	h.lastScanDedup[cleanCode] = nowMs
	h.mu.Unlock()

	matchedType, entity, prod := h.Lookup(cleanCode)
	isScanner := durationMs < 90.0 || source == "evdev_hardware" || source == "usb_scanner_wedge"

	rec := ScanRecord{
		ID:            uuid.NewString()[:8],
		Code:          cleanCode,
		Timestamp:     time.Now().Format("15:04:05"),
		DurationMs:    durationMs,
		Source:        source,
		IsScanner:     isScanner,
		MatchedType:   matchedType,
		MatchedEntity: entity,
		Product:       prod,
	}

	h.mu.Lock()
	h.scanHistory = append([]ScanRecord{rec}, h.scanHistory...)
	if len(h.scanHistory) > 100 {
		h.scanHistory = h.scanHistory[:100]
	}
	h.mu.Unlock()

	// Difundir por WebSocket
	h.Broadcast(map[string]interface{}{
		"type": "scan_event",
		"data": rec,
	})

	return &rec, nil
}

// GetHistory retorna todos los escaneos en memoria.
func (h *Hub) GetHistory() []ScanRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()
	res := make([]ScanRecord, len(h.scanHistory))
	copy(res, h.scanHistory)
	return res
}

// GetHistoryLimit retorna los últimos N escaneos.
func (h *Hub) GetHistoryLimit(limit int) []ScanRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if limit <= 0 || limit > len(h.scanHistory) {
		limit = len(h.scanHistory)
	}
	res := make([]ScanRecord, limit)
	copy(res, h.scanHistory[:limit])
	return res
}

// ClearHistory vacía el historial de la sesión.
func (h *Hub) ClearHistory() {
	h.mu.Lock()
	h.scanHistory = h.scanHistory[:0]
	h.mu.Unlock()

	h.Broadcast(map[string]interface{}{
		"type": "history_cleared",
	})
}

// Lookup busca en la BD de inventario y en el catálogo demo.
func (h *Hub) Lookup(code string) (matchedType string, entity interface{}, prod *SampleProduct) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return "unknown", nil, nil
	}

	// 1. Verificar si coincide con catálogo demo
	if p := FindSampleProductByCode(cleanCode); p != nil {
		prod = p
		matchedType = "demo_catalog"
	}

	if h.db == nil {
		if prod != nil {
			return "demo_catalog", nil, prod
		}
		return "unknown", nil, nil
	}

	// 2. Buscar en etiquetas por id (UUID) o codigo_barras (ET-XXXX)
	var eID, estado string
	var dispID, conjID, codBarras sql.NullString
	err := h.db.QueryRow(`
		SELECT id, estado, dispositivo_id, conjunto_id, codigo_barras
		FROM etiquetas
		WHERE id = ? OR codigo_barras = ? OR id = ?
	`, cleanCode, strings.ToUpper(cleanCode), strings.ToLower(cleanCode)).Scan(&eID, &estado, &dispID, &conjID, &codBarras)

	if err == nil {
		// Etiqueta encontrada
		if dispID.Valid && dispID.String != "" {
			d := h.findDispositivo(dispID.String)
			if d != nil {
				d["etiqueta_id"] = eID
				if codBarras.Valid {
					d["etiqueta_codigo_barras"] = codBarras.String
				}
				return "dispositivo", d, prod
			}
		} else if conjID.Valid && conjID.String != "" {
			c := h.findConjunto(conjID.String)
			if c != nil {
				c["etiqueta_id"] = eID
				if codBarras.Valid {
					c["etiqueta_codigo_barras"] = codBarras.String
				}
				return "conjunto", c, prod
			}
		}
		// Etiqueta sin asignar
		et := map[string]interface{}{
			"id":            eID,
			"estado":        estado,
			"codigo_barras": codBarras.String,
		}
		return "etiqueta", et, prod
	}

	// 3. Buscar en dispositivos directamente por codigo_activo, numero_serie o id
	var dID string
	err = h.db.QueryRow(`
		SELECT id FROM dispositivos
		WHERE codigo_activo = ? OR numero_serie = ? OR id = ?
	`, cleanCode, cleanCode, cleanCode).Scan(&dID)
	if err == nil {
		d := h.findDispositivo(dID)
		if d != nil {
			return "dispositivo", d, prod
		}
	}

	// 4. Buscar en conjuntos por codigo (CJ-XXXX) o id
	var cID string
	err = h.db.QueryRow(`
		SELECT id FROM conjuntos
		WHERE codigo = ? OR id = ?
	`, strings.ToUpper(cleanCode), cleanCode).Scan(&cID)
	if err == nil {
		c := h.findConjunto(cID)
		if c != nil {
			return "conjunto", c, prod
		}
	}

	// 5. Buscar en frascos por codigo o id
	var fID, fCodigo, fColor, fMarca string
	var fCantidad int
	err = h.db.QueryRow(`
		SELECT id, codigo, color, COALESCE(marca,''), cantidad_actual
		FROM frascos
		WHERE codigo = ? OR id = ?
	`, strings.ToUpper(cleanCode), cleanCode).Scan(&fID, &fCodigo, &fColor, &fMarca, &fCantidad)
	if err == nil {
		fr := map[string]interface{}{
			"id":              fID,
			"codigo":          fCodigo,
			"color":           fColor,
			"marca":           fMarca,
			"cantidad_actual": fCantidad,
		}
		return "frasco", fr, prod
	}

	if prod != nil {
		return "demo_catalog", nil, prod
	}

	return "unknown", nil, nil
}

func (h *Hub) findDispositivo(id string) map[string]interface{} {
	var dID, codigoActivo, marca, modelo, tipo, estado string
	var serie, ubicacion, sede, asignadoA sql.NullString
	err := h.db.QueryRow(`
		SELECT id, codigo_activo, marca, modelo, tipo, estado, numero_serie, ubicacion, sede, asignado_a
		FROM dispositivos
		WHERE id = ?
	`, id).Scan(&dID, &codigoActivo, &marca, &modelo, &tipo, &estado, &serie, &ubicacion, &sede, &asignadoA)
	if err != nil {
		return nil
	}
	res := map[string]interface{}{
		"id":            dID,
		"codigo_activo": codigoActivo,
		"marca":         marca,
		"modelo":        modelo,
		"tipo":          tipo,
		"estado":        estado,
	}
	if serie.Valid {
		res["numero_serie"] = serie.String
	}
	if ubicacion.Valid {
		res["ubicacion"] = ubicacion.String
	}
	if sede.Valid {
		res["sede"] = sede.String
	}
	if asignadoA.Valid {
		res["asignado_a"] = asignadoA.String
	}
	return res
}

func (h *Hub) findConjunto(id string) map[string]interface{} {
	var cID, codigo, nombre, unidad string
	var cantActual, cantMin int
	var categoria, ubicacion, sede sql.NullString
	err := h.db.QueryRow(`
		SELECT id, codigo, nombre, unidad, cantidad_actual, cantidad_minima, categoria, ubicacion, sede
		FROM conjuntos
		WHERE id = ?
	`, id).Scan(&cID, &codigo, &nombre, &unidad, &cantActual, &cantMin, &categoria, &ubicacion, &sede)
	if err != nil {
		return nil
	}
	res := map[string]interface{}{
		"id":              cID,
		"codigo":          codigo,
		"nombre":          nombre,
		"unidad":          unidad,
		"cantidad_actual": cantActual,
		"cantidad_minima": cantMin,
	}
	if categoria.Valid {
		res["categoria"] = categoria.String
	}
	if ubicacion.Valid {
		res["ubicacion"] = ubicacion.String
	}
	if sede.Valid {
		res["sede"] = sede.String
	}
	return res
}
