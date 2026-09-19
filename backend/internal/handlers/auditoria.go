package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"inventario-backend/internal/models"
)

type AuditoriaHandler struct {
	DB *sql.DB
}

func RegistrarAuditoria(db *sql.DB, r *http.Request, entidad, entidadID, accion, campo, valorAnterior, valorNuevo string) {
	if db == nil {
		return
	}
	var usuarioID *string
	var username *string
	if u, ok := GetAuthUser(r); ok && u != nil {
		usuarioID = &u.ID
		username = &u.Username
	} else {
		s := "sistema"
		username = &s
	}
	id := uuid.NewString()
	fecha := time.Now().Format(time.RFC3339)
	var campoPtr *string
	if campo != "" {
		campoPtr = &campo
	}
	var antPtr *string
	if valorAnterior != "" || campo != "" {
		// registrar incluso vacío si hay campo (para distinguir null)
		// si campo vacío (creación), anterior siempre nil
		if campo == "" {
			antPtr = nil
		} else {
			v := valorAnterior
			antPtr = &v
		}
	}
	var nuevoPtr *string
	if valorNuevo != "" {
		v := valorNuevo
		nuevoPtr = &v
	} else if campo != "" {
		// para campos que se borran (NULL) registrar vacío como null? dejamos nil si valor_nuevo vacío y campo no es estado eliminado?
		// mejor registrar nil si valorNuevo == "" y campo != ""
		// pero para asignado_a = NULL necesitamos distinguir, así que registrar "" como NULL es ok
		// dejamos nil para no ruido
	}
	// para acciones globales (crear), nuevo = entidadID
	if campo == "" && valorNuevo == "" && accion == "crear" {
		v := entidadID
		nuevoPtr = &v
	}
	_, err := db.Exec(`INSERT INTO auditoria (id, entidad, entidad_id, accion, campo, valor_anterior, valor_nuevo, usuario_id, username, fecha) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, entidad, entidadID, accion, campoPtr, antPtr, nuevoPtr, usuarioID, username, fecha)
	if err != nil {
		log.Printf("WARN auditoria insert %s %s %s: %v", entidad, entidadID, accion, err)
	}
}

func (h *AuditoriaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	entidad := r.URL.Query().Get("entidad")
	entidadID := r.URL.Query().Get("entidad_id")
	accion := r.URL.Query().Get("accion")
	campo := r.URL.Query().Get("campo")
	usuarioID := r.URL.Query().Get("usuario_id")
	desde := r.URL.Query().Get("desde")
	hasta := r.URL.Query().Get("hasta")
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = r.URL.Query().Get("limite")
	}
	offsetStr := r.URL.Query().Get("offset")
	limit := 100
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}
	if limit > 500 {
		limit = 500
	}
	if limit <= 0 {
		limit = 100
	}
	offset := 0
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			offset = v
		}
	}
	query := `SELECT id, entidad, entidad_id, accion, campo, valor_anterior, valor_nuevo, usuario_id, username, fecha FROM auditoria WHERE 1=1`
	args := []interface{}{}
	if entidad != "" {
		query += ` AND entidad = ?`
		args = append(args, entidad)
	}
	if entidadID != "" {
		query += ` AND entidad_id = ?`
		args = append(args, entidadID)
	}
	if accion != "" {
		query += ` AND accion = ?`
		args = append(args, accion)
	}
	if campo != "" {
		query += ` AND campo = ?`
		args = append(args, campo)
	}
	if usuarioID != "" {
		query += ` AND usuario_id = ?`
		args = append(args, usuarioID)
	}
	if desde != "" {
		query += ` AND fecha >= ?`
		args = append(args, desde)
	}
	if hasta != "" {
		query += ` AND fecha <= ?`
		args = append(args, hasta)
	}
	// total count
	countQuery := `SELECT COUNT(*) FROM (` + query + `)`
	var total int
	if err := h.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		total = 0
	}
	query += ` ORDER BY fecha DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.Auditoria
	for rows.Next() {
		var a models.Auditoria
		rows.Scan(&a.ID, &a.Entidad, &a.EntidadID, &a.Accion, &a.Campo, &a.ValorAnterior, &a.ValorNuevo, &a.UsuarioID, &a.Username, &a.Fecha)
		list = append(list, a)
	}
	if list == nil {
		list = []models.Auditoria{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"auditoria": list, "total": total, "limit": limit, "offset": offset})
}

func (h *AuditoriaHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var a models.Auditoria
	err := h.DB.QueryRow(`SELECT id, entidad, entidad_id, accion, campo, valor_anterior, valor_nuevo, usuario_id, username, fecha FROM auditoria WHERE id = ?`, id).Scan(&a.ID, &a.Entidad, &a.EntidadID, &a.Accion, &a.Campo, &a.ValorAnterior, &a.ValorNuevo, &a.UsuarioID, &a.Username, &a.Fecha)
	if err == sql.ErrNoRows {
		http.Error(w, "auditoria no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}
