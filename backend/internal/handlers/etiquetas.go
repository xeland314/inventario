package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"inventario-backend/internal/models"
)

type EtiquetaHandler struct {
	DB *sql.DB
}

// newUUIDv7 genera UUIDv7 (48-bit timestamp ms + version 7 + variant) sin dependencia externa.
func newUUIDv7() string {
	var b [16]byte
	ms := time.Now().UnixMilli()
	// 48-bit timestamp big-endian en bytes 0-5
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	// random para bytes 6-15
	randBytes := make([]byte, 10)
	rand.Read(randBytes)
	// byte 6: version 7 (0111) en high 4 bits
	b[6] = (randBytes[0] & 0x0F) | 0x70
	// byte 7: variant 10xxxxxx
	b[7] = (randBytes[1] & 0x3F) | 0x80
	copy(b[8:], randBytes[2:])
	// formatear como UUID string
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		b[10:16])
}

func generarCodigoBarras(id string) string {
	// ET-XXXXXXXX : 8 hex aleatorios del final del UUID (evita colisión por timestamp en v7)
	clean := strings.ReplaceAll(id, "-", "")
	if len(clean) >= 8 {
		return "ET-" + strings.ToUpper(clean[len(clean)-8:])
	}
	return "ET-" + strings.ToUpper(clean)
}

// POST /api/etiquetas/generar {cantidad, version}
func (h *EtiquetaHandler) Generar(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Cantidad int `json:"cantidad"`
		Version  int `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Cantidad <= 0 || req.Cantidad > 500 {
		http.Error(w, "cantidad debe ser 1..500", http.StatusBadRequest)
		return
	}
	if req.Version != 4 && req.Version != 7 {
		if req.Version == 0 {
			req.Version = 7
		} else {
			http.Error(w, "version debe ser 4 o 7", http.StatusBadRequest)
			return
		}
	}
	now := time.Now().Format(time.RFC3339)
	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	ids := []string{}
	for i := 0; i < req.Cantidad; i++ {
		var id string
		if req.Version == 7 {
			id = newUUIDv7()
			// pequeña pausa para asegurar unicidad ms si se genera muy rápido en mismo ms (random ya asegura pero timestamp igual)
			if i%10 == 0 {
				time.Sleep(time.Millisecond)
			}
		} else {
			id = uuid.NewString()
		}
		codigo := generarCodigoBarras(id)
		_, err := tx.Exec(`INSERT INTO etiquetas (id, version, estado, codigo_barras, generado_en) VALUES (?, ?, 'disponible', ?, ?)`, id, req.Version, codigo, now)
		if err != nil {
			// por colisión improbable, reintentar
			if strings.Contains(err.Error(), "UNIQUE") {
				i--
				continue
			}
			http.Error(w, "Error al generar: "+err.Error(), http.StatusInternalServerError)
			return
		}
		ids = append(ids, id)
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, eid := range ids {
		RegistrarAuditoria(h.DB, r, "etiqueta", eid, "generar", "", "", "")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"generadas": len(ids), "version": req.Version, "ids": ids})
}

func (h *EtiquetaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	estado := r.URL.Query().Get("estado")
	version := r.URL.Query().Get("version")
	disponibleOnly := r.URL.Query().Get("disponible") // legacy alias
	if disponibleOnly == "true" {
		estado = "disponible"
	}
	query := `SELECT id, version, estado, dispositivo_id, codigo_barras, generado_en, impreso_en, asignado_en FROM etiquetas WHERE 1=1`
	args := []interface{}{}
	if estado != "" {
		if !models.EsEstadoEtiquetaValido(estado) {
			http.Error(w, "estado no válido", http.StatusBadRequest)
			return
		}
		query += ` AND estado = ?`
		args = append(args, estado)
	}
	if version != "" {
		query += ` AND version = ?`
		args = append(args, version)
	}
	query += ` ORDER BY generado_en DESC`
	// paginación opcional
	limit := r.URL.Query().Get("limit")
	if limit != "" {
		query += ` LIMIT ?`
		args = append(args, limit)
		if off := r.URL.Query().Get("offset"); off != "" {
			query += ` OFFSET ?`
			args = append(args, off)
		}
	}
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.Etiqueta
	for rows.Next() {
		var e models.Etiqueta
		rows.Scan(&e.ID, &e.Version, &e.Estado, &e.DispositivoID, &e.CodigoBarras, &e.GeneradoEn, &e.ImpresoEn, &e.AsignadoEn)
		list = append(list, e)
	}
	if list == nil {
		list = []models.Etiqueta{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"etiquetas": list})
}

func (h *EtiquetaHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var e models.Etiqueta
	err := h.DB.QueryRow(`SELECT id, version, estado, dispositivo_id, codigo_barras, generado_en, impreso_en, asignado_en FROM etiquetas WHERE id = ?`, id).Scan(&e.ID, &e.Version, &e.Estado, &e.DispositivoID, &e.CodigoBarras, &e.GeneradoEn, &e.ImpresoEn, &e.AsignadoEn)
	if err == sql.ErrNoRows {
		http.Error(w, "Etiqueta no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Si está asignada, adjuntar info dispositivo
	var dispositivo *models.Dispositivo
	if e.DispositivoID != nil {
		var d models.Dispositivo
		err := h.DB.QueryRow(`SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE id = ?`, *e.DispositivoID).
			Scan(&d.ID, &d.CodigoActivo, &d.NumeroSerie, &d.Tipo, &d.Marca, &d.Modelo, &d.Especificaciones, &d.ValorCompra, &d.Proveedor, &d.Estado, &d.Ubicacion, &d.AsignadoA, &d.FechaCompra, &d.FechaBaja, &d.MotivoBaja, &d.ParentID, &d.TipoComponente, &d.Sede, &d.CreadoEn, &d.ActualizadoEn)
		if err == nil {
			dispositivo = &d
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"etiqueta": e, "dispositivo": dispositivo})
}

// POST /api/etiquetas/{id}/asociar {dispositivo_id}
func (h *EtiquetaHandler) Asociar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		DispositivoID string `json:"dispositivo_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.DispositivoID == "" {
		http.Error(w, "dispositivo_id requerido", http.StatusBadRequest)
		return
	}
	// verificar etiqueta existe y está disponible/impreso
	var estado string
	var dispActual *string
	if err := h.DB.QueryRow(`SELECT estado, dispositivo_id FROM etiquetas WHERE id = ?`, id).Scan(&estado, &dispActual); err == sql.ErrNoRows {
		http.Error(w, "Etiqueta no encontrada", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if estado == "anulado" {
		http.Error(w, "Etiqueta anulada no se puede asociar", http.StatusBadRequest)
		return
	}
	if estado == "asignado" {
		http.Error(w, "Etiqueta ya asignada", http.StatusConflict)
		return
	}
	// verificar dispositivo existe y no tiene otra etiqueta
	var exists string
	if err := h.DB.QueryRow(`SELECT id FROM dispositivos WHERE id = ?`, req.DispositivoID).Scan(&exists); err == sql.ErrNoRows {
		http.Error(w, "Dispositivo no encontrado", http.StatusNotFound)
		return
	}
	var otra string
	if err := h.DB.QueryRow(`SELECT id FROM etiquetas WHERE dispositivo_id = ?`, req.DispositivoID).Scan(&otra); err == nil {
		http.Error(w, "Dispositivo ya tiene etiqueta asociada: "+otra, http.StatusConflict)
		return
	}
	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`UPDATE etiquetas SET dispositivo_id = ?, estado = 'asignado', asignado_en = ? WHERE id = ?`, req.DispositivoID, now, id)
	if err != nil {
		http.Error(w, "Error al asociar: "+err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "etiqueta", id, "asociar", "dispositivo_id", "", req.DispositivoID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Etiqueta asociada", "etiqueta_id": id, "dispositivo_id": req.DispositivoID})
}

func (h *EtiquetaHandler) Desasociar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var estado string
	if err := h.DB.QueryRow(`SELECT estado FROM etiquetas WHERE id = ?`, id).Scan(&estado); err == sql.ErrNoRows {
		http.Error(w, "Etiqueta no encontrada", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if estado != "asignado" {
		http.Error(w, "Etiqueta no está asignada", http.StatusBadRequest)
		return
	}
	_, err := h.DB.Exec(`UPDATE etiquetas SET dispositivo_id = NULL, estado = 'disponible', asignado_en = NULL WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "etiqueta", id, "desasociar", "", "", "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Etiqueta desasociada"})
}

func (h *EtiquetaHandler) MarcarImpreso(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var estado string
	if err := h.DB.QueryRow(`SELECT estado FROM etiquetas WHERE id = ?`, id).Scan(&estado); err == sql.ErrNoRows {
		http.Error(w, "Etiqueta no encontrada", http.StatusNotFound)
		return
	}
	if estado != "disponible" {
		http.Error(w, "Solo etiquetas disponibles pueden marcarse impresas", http.StatusBadRequest)
		return
	}
	now := time.Now().Format(time.RFC3339)
	h.DB.Exec(`UPDATE etiquetas SET estado = 'impreso', impreso_en = ? WHERE id = ?`, now, id)
	RegistrarAuditoria(h.DB, r, "etiqueta", id, "impreso", "estado", "disponible", "impreso")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Etiqueta marcada como impresa"})
}

func (h *EtiquetaHandler) Anular(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.DB.Exec(`UPDATE etiquetas SET estado = 'anulado', dispositivo_id = NULL WHERE id = ? AND estado != 'asignado'`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// verificar si existe
		var e string
		if err := h.DB.QueryRow(`SELECT estado FROM etiquetas WHERE id = ?`, id).Scan(&e); err == sql.ErrNoRows {
			http.Error(w, "Etiqueta no encontrada", http.StatusNotFound)
			return
		}
		http.Error(w, "Etiqueta asignada no se puede anular; desasocie primero", http.StatusBadRequest)
		return
	}
	RegistrarAuditoria(h.DB, r, "etiqueta", id, "anulado", "estado", "", "anulado")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Etiqueta anulada"})
}

func (h *EtiquetaHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, _ := h.DB.Exec(`DELETE FROM etiquetas WHERE id = ? AND estado IN ('disponible','anulado')`, id)
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "Solo etiquetas disponibles/anuladas se pueden eliminar", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Etiqueta eliminada"})
}

// Escanear: GET /api/etiquetas/scan/{id} o /api/escaner/{id} -> redirige/info
func (h *EtiquetaHandler) Escanear(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// id puede venir con prefijo ET-xxxx o UUID completo; buscar por id o codigo_barras
	var e models.Etiqueta
	err := h.DB.QueryRow(`SELECT id, version, estado, dispositivo_id, codigo_barras, generado_en, impreso_en, asignado_en FROM etiquetas WHERE id = ? OR codigo_barras = ?`, id, strings.ToUpper(id)).Scan(&e.ID, &e.Version, &e.Estado, &e.DispositivoID, &e.CodigoBarras, &e.GeneradoEn, &e.ImpresoEn, &e.AsignadoEn)
	if err == sql.ErrNoRows {
		http.Error(w, "Etiqueta no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// si no está asignada, retornar etiqueta sin dispositivo
	if e.DispositivoID == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"etiqueta": e, "dispositivo": nil, "mensaje": "Etiqueta sin asignar. Asóciela a un dispositivo."})
		return
	}
	var d models.Dispositivo
	err = h.DB.QueryRow(`SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE id = ?`, *e.DispositivoID).
		Scan(&d.ID, &d.CodigoActivo, &d.NumeroSerie, &d.Tipo, &d.Marca, &d.Modelo, &d.Especificaciones, &d.ValorCompra, &d.Proveedor, &d.Estado, &d.Ubicacion, &d.AsignadoA, &d.FechaCompra, &d.FechaBaja, &d.MotivoBaja, &d.ParentID, &d.TipoComponente, &d.Sede, &d.CreadoEn, &d.ActualizadoEn)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"etiqueta": e, "error": "Dispositivo asociado no encontrado"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"etiqueta": e, "dispositivo": d})
}
