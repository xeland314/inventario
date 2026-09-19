package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"inventario-backend/internal/models"
)

type MovimientoHandler struct {
	DB *sql.DB
}

func (h *MovimientoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	dispositivoID := r.URL.Query().Get("dispositivo_id")
	parentID := r.URL.Query().Get("parent_id")

	query := `SELECT id, dispositivo_id, origen_parent_id, destino_parent_id, tipo_movimiento, descripcion, realizado_por, fecha_movimiento, creado_en FROM movimientos WHERE 1=1`
	args := []interface{}{}
	if dispositivoID != "" {
		query += ` AND dispositivo_id = ?`
		args = append(args, dispositivoID)
	}
	if parentID != "" {
		query += ` AND (origen_parent_id = ? OR destino_parent_id = ?)`
		args = append(args, parentID, parentID)
	}
	query += ` ORDER BY fecha_movimiento DESC, creado_en DESC`

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []models.Movimiento
	for rows.Next() {
		var m models.Movimiento
		if err := rows.Scan(&m.ID, &m.DispositivoID, &m.OrigenParentID, &m.DestinoParentID, &m.TipoMovimiento, &m.Descripcion, &m.RealizadoPor, &m.FechaMovimiento, &m.CreadoEn); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list = append(list, m)
	}
	if list == nil {
		list = []models.Movimiento{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"movimientos": list})
}

func (h *MovimientoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DispositivoID   string  `json:"dispositivo_id"`
		OrigenParentID  *string `json:"origen_parent_id"`
		DestinoParentID *string `json:"destino_parent_id"`
		TipoMovimiento  string  `json:"tipo_movimiento"`
		Descripcion     *string `json:"descripcion"`
		RealizadoPor    *string `json:"realizado_por"`
		FechaMovimiento *string `json:"fecha_movimiento"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.DispositivoID == "" || req.TipoMovimiento == "" {
		http.Error(w, "dispositivo_id y tipo_movimiento son obligatorios", http.StatusBadRequest)
		return
	}
	if !models.EsTipoMovimientoValido(req.TipoMovimiento) {
		http.Error(w, "tipo_movimiento no válido", http.StatusBadRequest)
		return
	}
	// validar dispositivo existe y obtener parent actual
	var currentParent *string
	if err := h.DB.QueryRow(`SELECT parent_id FROM dispositivos WHERE id = ?`, req.DispositivoID).Scan(&currentParent); err == sql.ErrNoRows {
		http.Error(w, "dispositivo no encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// validar origen/destino existen si se proveen
	if req.OrigenParentID != nil && *req.OrigenParentID != "" {
		var tmp string
		if err := h.DB.QueryRow(`SELECT id FROM dispositivos WHERE id = ?`, *req.OrigenParentID).Scan(&tmp); err == sql.ErrNoRows {
			http.Error(w, "origen_parent_id no existe", http.StatusBadRequest)
			return
		}
	}
	if req.DestinoParentID != nil && *req.DestinoParentID != "" {
		var tmp string
		if err := h.DB.QueryRow(`SELECT id FROM dispositivos WHERE id = ?`, *req.DestinoParentID).Scan(&tmp); err == sql.ErrNoRows {
			http.Error(w, "destino_parent_id no existe", http.StatusBadRequest)
			return
		}
		if req.DispositivoID == *req.DestinoParentID {
			http.Error(w, "no se puede asignar dispositivo a sí mismo", http.StatusBadRequest)
			return
		}
	}
	// si es traslado/instalacion, actualizar parent_id del dispositivo dentro de transacción
	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	fechaMov := time.Now().Format(time.RFC3339)
	if req.FechaMovimiento != nil && *req.FechaMovimiento != "" {
		fechaMov = *req.FechaMovimiento
	}
	creadoEn := time.Now().Format(time.RFC3339)
	id := uuid.New().String()

	// normalizar vacíos a NULL
	var origenDB *string = req.OrigenParentID
	if origenDB != nil && *origenDB == "" {
		origenDB = nil
	}
	var destinoDB *string = req.DestinoParentID
	if destinoDB != nil && *destinoDB == "" {
		destinoDB = nil
	}
	// inferir origen si no viene: usar currentParent
	if req.OrigenParentID == nil && (req.TipoMovimiento == "traslado" || req.TipoMovimiento == "retiro" || req.TipoMovimiento == "desasignacion") {
		origenDB = currentParent
	}
	// si no se especifica destino para desasignacion/retiro, es NULL (ya)

	_, err = tx.Exec(`INSERT INTO movimientos (id, dispositivo_id, origen_parent_id, destino_parent_id, tipo_movimiento, descripcion, realizado_por, fecha_movimiento, creado_en) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.DispositivoID, origenDB, destinoDB, req.TipoMovimiento, req.Descripcion, req.RealizadoPor, fechaMov, creadoEn)
	if err != nil {
		http.Error(w, "Error al registrar movimiento: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// actualizar parent_id según tipo
	switch req.TipoMovimiento {
	case "asignacion", "instalacion", "traslado", "canibalizacion":
		if destinoDB != nil {
			if _, err := tx.Exec(`UPDATE dispositivos SET parent_id = ?, actualizado_en = ? WHERE id = ?`, *destinoDB, creadoEn, req.DispositivoID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	case "desasignacion", "retiro":
		if _, err := tx.Exec(`UPDATE dispositivos SET parent_id = NULL, actualizado_en = ? WHERE id = ?`, creadoEn, req.DispositivoID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "movimiento", id, "crear", "", "", "")
	// también auditar traslado a nivel dispositivo
	if req.DestinoParentID != nil && *req.DestinoParentID != "" {
		anterior := ""
		if currentParent != nil {
			anterior = *currentParent
		}
		RegistrarAuditoria(h.DB, r, "dispositivo", req.DispositivoID, "traslado", "parent_id", anterior, *req.DestinoParentID)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "message": "Movimiento registrado"})
}

func (h *MovimientoHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m models.Movimiento
	err := h.DB.QueryRow(`SELECT id, dispositivo_id, origen_parent_id, destino_parent_id, tipo_movimiento, descripcion, realizado_por, fecha_movimiento, creado_en FROM movimientos WHERE id = ?`, id).
		Scan(&m.ID, &m.DispositivoID, &m.OrigenParentID, &m.DestinoParentID, &m.TipoMovimiento, &m.Descripcion, &m.RealizadoPor, &m.FechaMovimiento, &m.CreadoEn)
	if err == sql.ErrNoRows {
		http.Error(w, "Movimiento no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func (h *MovimientoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.DB.Exec(`DELETE FROM movimientos WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "Movimiento no encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Movimiento eliminado"})
}
