package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"inventario-backend/internal/models"
)

type ReporteHandler struct {
	DB *sql.DB
}

// POST /api/reportes {dispositivo_id?, etiqueta_id?, descripcion}
func (h *ReporteHandler) Crear(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthUser(r)
	if !ok {
		http.Error(w, `{"error":"no autenticado"}`, http.StatusUnauthorized)
		return
	}
	var req struct {
		DispositivoID *string `json:"dispositivo_id"`
		EtiquetaID    *string `json:"etiqueta_id"`
		Descripcion   string  `json:"descripcion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Descripcion = strings.TrimSpace(req.Descripcion)
	if len(req.Descripcion) < 5 {
		http.Error(w, "descripcion debe tener al menos 5 caracteres", http.StatusBadRequest)
		return
	}
	// normalizar vacíos a nil
	if req.DispositivoID != nil && *req.DispositivoID == "" {
		req.DispositivoID = nil
	}
	if req.EtiquetaID != nil && *req.EtiquetaID == "" {
		req.EtiquetaID = nil
	}
	if req.DispositivoID == nil && req.EtiquetaID == nil {
		http.Error(w, "dispositivo_id o etiqueta_id requerido", http.StatusBadRequest)
		return
	}
	// validar dispositivo existe si se provee
	if req.DispositivoID != nil {
		var exists string
		if err := h.DB.QueryRow(`SELECT id FROM dispositivos WHERE id = ?`, *req.DispositivoID).Scan(&exists); err == sql.ErrNoRows {
			http.Error(w, "dispositivo no encontrado", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// validar etiqueta existe si se provee
	if req.EtiquetaID != nil {
		var exists string
		if err := h.DB.QueryRow(`SELECT id FROM etiquetas WHERE id = ? OR codigo_barras = ?`, *req.EtiquetaID, strings.ToUpper(*req.EtiquetaID)).Scan(&exists); err == sql.ErrNoRows {
			http.Error(w, "etiqueta no encontrada", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			// normalizar a id real si vino codigo_barras
			if exists != *req.EtiquetaID {
				req.EtiquetaID = &exists
			}
			// si no se dio dispositivo pero etiqueta está asignada, autocompletar
			if req.DispositivoID == nil {
				var dispID *string
				h.DB.QueryRow(`SELECT dispositivo_id FROM etiquetas WHERE id = ?`, *req.EtiquetaID).Scan(&dispID)
				if dispID != nil {
					req.DispositivoID = dispID
				}
			}
		}
	}
	id := uuid.NewString()
	creado := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`INSERT INTO reportes (id, dispositivo_id, etiqueta_id, creado_por, descripcion, estado, creado_en) VALUES (?, ?, ?, ?, ?, 'abierto', ?)`,
		id, req.DispositivoID, req.EtiquetaID, u.ID, req.Descripcion, creado)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "message": "reporte creado"})
}

func (h *ReporteHandler) Listar(w http.ResponseWriter, r *http.Request) {
	estado := r.URL.Query().Get("estado")
	dispositivoID := r.URL.Query().Get("dispositivo_id")
	etiquetaID := r.URL.Query().Get("etiqueta_id")
	creadoPor := r.URL.Query().Get("creado_por")
	query := `SELECT r.id, r.dispositivo_id, r.etiqueta_id, r.creado_por, r.descripcion, r.estado, r.creado_en, r.cancelado_en, u.username FROM reportes r LEFT JOIN usuarios u ON r.creado_por = u.id WHERE 1=1`
	args := []interface{}{}
	if estado != "" {
		if !models.EsEstadoReporteValido(estado) {
			http.Error(w, "estado reporte no válido", http.StatusBadRequest)
			return
		}
		query += ` AND r.estado = ?`
		args = append(args, estado)
	}
	if dispositivoID != "" {
		query += ` AND r.dispositivo_id = ?`
		args = append(args, dispositivoID)
	}
	if etiquetaID != "" {
		query += ` AND r.etiqueta_id = ?`
		args = append(args, etiquetaID)
	}
	if creadoPor != "" {
		query += ` AND r.creado_por = ?`
		args = append(args, creadoPor)
	}
	query += ` ORDER BY r.creado_en DESC`
	if lim := r.URL.Query().Get("limit"); lim != "" {
		query += ` LIMIT ?`
		args = append(args, lim)
	}
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.Reporte
	for rows.Next() {
		var rep models.Reporte
		rows.Scan(&rep.ID, &rep.DispositivoID, &rep.EtiquetaID, &rep.CreadoPor, &rep.Descripcion, &rep.Estado, &rep.CreadoEn, &rep.CanceladoEn, &rep.Username)
		list = append(list, rep)
	}
	if list == nil {
		list = []models.Reporte{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"reportes": list})
}

func (h *ReporteHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var rep models.Reporte
	err := h.DB.QueryRow(`SELECT r.id, r.dispositivo_id, r.etiqueta_id, r.creado_por, r.descripcion, r.estado, r.creado_en, r.cancelado_en, u.username FROM reportes r LEFT JOIN usuarios u ON r.creado_por = u.id WHERE r.id = ?`, id).
		Scan(&rep.ID, &rep.DispositivoID, &rep.EtiquetaID, &rep.CreadoPor, &rep.Descripcion, &rep.Estado, &rep.CreadoEn, &rep.CanceladoEn, &rep.Username)
	if err == sql.ErrNoRows {
		http.Error(w, "reporte no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rep)
}

// POST /api/reportes/{id}/cancelar — solo creador o tecnico
func (h *ReporteHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthUser(r)
	if !ok {
		http.Error(w, `{"error":"no autenticado"}`, http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var creadoPor, estado string
	if err := h.DB.QueryRow(`SELECT creado_por, estado FROM reportes WHERE id = ?`, id).Scan(&creadoPor, &estado); err == sql.ErrNoRows {
		http.Error(w, "reporte no encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if estado != "abierto" {
		http.Error(w, "solo reportes abiertos se pueden cancelar", http.StatusBadRequest)
		return
	}
	if u.Rol != "tecnico" && u.ID != creadoPor {
		http.Error(w, `{"error":"solo el creador o tecnico puede cancelar"}`, http.StatusForbidden)
		return
	}
	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`UPDATE reportes SET estado = 'cancelado', cancelado_en = ? WHERE id = ?`, now, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "reporte cancelado"})
}

// POST /api/reportes/{id}/resolver — solo tecnico
func (h *ReporteHandler) Resolver(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthUser(r)
	if !ok {
		http.Error(w, `{"error":"no autenticado"}`, http.StatusUnauthorized)
		return
	}
	if u.Rol != "tecnico" {
		http.Error(w, `{"error":"solo tecnico puede resolver"}`, http.StatusForbidden)
		return
	}
	id := chi.URLParam(r, "id")
	var estado string
	if err := h.DB.QueryRow(`SELECT estado FROM reportes WHERE id = ?`, id).Scan(&estado); err == sql.ErrNoRows {
		http.Error(w, "reporte no encontrado", http.StatusNotFound)
		return
	}
	if estado != "abierto" {
		http.Error(w, "solo reportes abiertos se pueden resolver", http.StatusBadRequest)
		return
	}
	_, err := h.DB.Exec(`UPDATE reportes SET estado = 'resuelto' WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "reporte resuelto"})
}
