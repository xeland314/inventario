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

type TomaHandler struct {
	DB *sql.DB
}

func (h *TomaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sede        string  `json:"sede"`
		Descripcion *string `json:"descripcion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Sede = strings.TrimSpace(req.Sede)
	if req.Sede == "" {
		http.Error(w, "sede requerida", http.StatusBadRequest)
		return
	}
	id := uuid.NewString()
	creadoEn := time.Now().Format(time.RFC3339)
	var creadoPor *string
	if u, ok := GetAuthUser(r); ok {
		creadoPor = &u.ID
	}
	_, err := h.DB.Exec(`INSERT INTO tomas_fisicas (id, sede, descripcion, estado, creado_por, creado_en) VALUES (?, ?, ?, 'abierto', ?, ?)`,
		id, req.Sede, req.Descripcion, creadoPor, creadoEn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "toma", id, "crear", "sede", "", req.Sede)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "sede": req.Sede})
}

func (h *TomaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	sede := r.URL.Query().Get("sede")
	estado := r.URL.Query().Get("estado")
	query := `SELECT id, sede, descripcion, estado, creado_por, creado_en, cerrado_en, cerrado_por FROM tomas_fisicas WHERE 1=1`
	args := []interface{}{}
	if sede != "" {
		query += ` AND sede = ?`
		args = append(args, sede)
	}
	if estado != "" {
		query += ` AND estado = ?`
		args = append(args, estado)
	}
	query += ` ORDER BY creado_en DESC`
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.TomaFisica
	for rows.Next() {
		var t models.TomaFisica
		rows.Scan(&t.ID, &t.Sede, &t.Descripcion, &t.Estado, &t.CreadoPor, &t.CreadoEn, &t.CerradoEn, &t.CerradoPor)
		list = append(list, t)
	}
	if list == nil {
		list = []models.TomaFisica{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tomas": list})
}

func (h *TomaHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var t models.TomaFisica
	err := h.DB.QueryRow(`SELECT id, sede, descripcion, estado, creado_por, creado_en, cerrado_en, cerrado_por FROM tomas_fisicas WHERE id = ?`, id).
		Scan(&t.ID, &t.Sede, &t.Descripcion, &t.Estado, &t.CreadoPor, &t.CreadoEn, &t.CerradoEn, &t.CerradoPor)
	if err == sql.ErrNoRows {
		http.Error(w, "toma no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// calcular esperados, escaneados, faltantes, sobrantes, correctos
	esperados := map[string]models.Dispositivo{}
	rows, _ := h.DB.Query(`SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE COALESCE(sede,'Sin sede') = ? AND estado != 'dado_de_baja'`, t.Sede)
	if rows != nil {
		for rows.Next() {
			var d models.Dispositivo
			rows.Scan(&d.ID, &d.CodigoActivo, &d.NumeroSerie, &d.Tipo, &d.Marca, &d.Modelo, &d.Especificaciones, &d.ValorCompra, &d.Proveedor, &d.Estado, &d.Ubicacion, &d.AsignadoA, &d.FechaCompra, &d.FechaBaja, &d.MotivoBaja, &d.ParentID, &d.TipoComponente, &d.Sede, &d.CreadoEn, &d.ActualizadoEn)
			esperados[d.ID] = d
		}
		rows.Close()
	}
	escaneadosMap := map[string]bool{}
	escaneadosList := []string{}
	// items con dispositivo_id
	rows2, _ := h.DB.Query(`SELECT dispositivo_id FROM toma_items WHERE toma_id = ? AND dispositivo_id IS NOT NULL`, id)
	if rows2 != nil {
		for rows2.Next() {
			var did *string
			rows2.Scan(&did)
			if did != nil {
				escaneadosMap[*did] = true
				escaneadosList = append(escaneadosList, *did)
			}
		}
		rows2.Close()
	}
	// contar escaneados con etiqueta sin dispositivo (huérfanas) para sobrantes
	var escaneadosHuérfanos int
	h.DB.QueryRow(`SELECT COUNT(*) FROM toma_items WHERE toma_id = ? AND dispositivo_id IS NULL`, id).Scan(&escaneadosHuérfanos)

	correctos := []models.Dispositivo{}
	faltantes := []models.Dispositivo{}
	for eid, dev := range esperados {
		if escaneadosMap[eid] {
			correctos = append(correctos, dev)
		} else {
			faltantes = append(faltantes, dev)
		}
	}
	sobrantes := []models.Dispositivo{}
	for _, eid := range escaneadosList {
		if _, ok := esperados[eid]; !ok {
			// buscar dispositivo para detalle
			var d models.Dispositivo
			err := h.DB.QueryRow(`SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE id = ?`, eid).Scan(&d.ID, &d.CodigoActivo, &d.NumeroSerie, &d.Tipo, &d.Marca, &d.Modelo, &d.Especificaciones, &d.ValorCompra, &d.Proveedor, &d.Estado, &d.Ubicacion, &d.AsignadoA, &d.FechaCompra, &d.FechaBaja, &d.MotivoBaja, &d.ParentID, &d.TipoComponente, &d.Sede, &d.CreadoEn, &d.ActualizadoEn)
			if err == nil {
				sobrantes = append(sobrantes, d)
			}
		}
	}
	escaneadosCount := len(escaneadosList) + escaneadosHuérfanos
	totalFaltantes := len(faltantes)
	totalSobrantes := len(sobrantes) + escaneadosHuérfanos
	totalCorrectos := len(correctos)
	// limitar arrays a 100 para no sobrecargar (counts mantienen totales)
	limit := 100
	if len(correctos) > limit {
		correctos = correctos[:limit]
	}
	if len(faltantes) > limit {
		faltantes = faltantes[:limit]
	}
	if len(sobrantes) > limit {
		sobrantes = sobrantes[:limit]
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"toma":              t,
		"esperados_count":   len(esperados),
		"escaneados_count":  escaneadosCount,
		"correctos_count":   totalCorrectos,
		"faltantes_count":   totalFaltantes,
		"sobrantes_count":   totalSobrantes,
		"faltantes":         faltantes,
		"sobrantes":         sobrantes,
		"correctos":         correctos,
		"escaneados_huerfanos": escaneadosHuérfanos,
		"total_esperados":   len(esperados),
	})
}

func (h *TomaHandler) Escanear(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var tomaEstado string
	if err := h.DB.QueryRow(`SELECT estado FROM tomas_fisicas WHERE id = ?`, id).Scan(&tomaEstado); err == sql.ErrNoRows {
		http.Error(w, "toma no encontrada", http.StatusNotFound)
		return
	}
	if tomaEstado == "cerrado" {
		http.Error(w, "toma cerrada, no se puede escanear", http.StatusBadRequest)
		return
	}
	var req struct {
		EtiquetaID    *string `json:"etiqueta_id"`
		DispositivoID *string `json:"dispositivo_id"`
		CodigoBarras  *string `json:"codigo_barras"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	// resolver dispositivo_id
	var dispID *string
	var etID *string
	if req.CodigoBarras != nil && *req.CodigoBarras != "" {
		cb := strings.ToUpper(strings.TrimSpace(*req.CodigoBarras))
		var eid, did *string
		h.DB.QueryRow(`SELECT id, dispositivo_id FROM etiquetas WHERE codigo_barras = ?`, cb).Scan(&eid, &did)
		if eid != nil {
			etID = eid
			dispID = did
		} else {
			// intentar como id directo
			h.DB.QueryRow(`SELECT id, dispositivo_id FROM etiquetas WHERE id = ?`, cb).Scan(&eid, &did)
			if eid != nil {
				etID = eid
				dispID = did
			}
		}
	}
	if etID == nil && req.EtiquetaID != nil && *req.EtiquetaID != "" {
		eid := strings.TrimSpace(*req.EtiquetaID)
		var did *string
		var foundID string
		// probar id directo o codigo_barras
		err := h.DB.QueryRow(`SELECT id, dispositivo_id FROM etiquetas WHERE id = ? OR codigo_barras = ?`, eid, strings.ToUpper(eid)).Scan(&foundID, &did)
		if err == nil {
			etID = &foundID
			dispID = did
		} else {
			http.Error(w, "etiqueta no encontrada", http.StatusNotFound)
			return
		}
	}
	if dispID == nil && req.DispositivoID != nil && *req.DispositivoID != "" {
		did := strings.TrimSpace(*req.DispositivoID)
		var exists string
		if err := h.DB.QueryRow(`SELECT id FROM dispositivos WHERE id = ?`, did).Scan(&exists); err == sql.ErrNoRows {
			http.Error(w, "dispositivo no encontrado", http.StatusNotFound)
			return
		}
		dispID = &did
	}
	if etID == nil && dispID == nil {
		http.Error(w, "etiqueta_id, dispositivo_id o codigo_barras requerido", http.StatusBadRequest)
		return
	}
	// si tenemos etiqueta pero no dispositivo, intentar resolver dispositivo via etiqueta ya tiene (dispID)
	// si tenemos dispositivo pero no etiqueta, buscar etiqueta asociada
	if etID == nil && dispID != nil {
		var eid string
		h.DB.QueryRow(`SELECT id FROM etiquetas WHERE dispositivo_id = ?`, *dispID).Scan(&eid)
		if eid != "" {
			etID = &eid
		}
	}
	// insertar, tolerar duplicado UNIQUE
	itemID := uuid.NewString()
	escaneadoEn := time.Now().Format(time.RFC3339)
	var escaneadoPor *string
	if u, ok := GetAuthUser(r); ok {
		escaneadoPor = &u.ID
	}
	_, err := h.DB.Exec(`INSERT INTO toma_items (id, toma_id, etiqueta_id, dispositivo_id, escaneado_en, escaneado_por) VALUES (?, ?, ?, ?, ?, ?)`,
		itemID, id, etID, dispID, escaneadoEn, escaneadoPor)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "ya escaneado", "toma_id": id})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "toma", id, "escanear", "", "", "")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": itemID, "toma_id": id, "etiqueta_id": etID, "dispositivo_id": dispID})
}

func (h *TomaHandler) Cerrar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var estado string
	if err := h.DB.QueryRow(`SELECT estado FROM tomas_fisicas WHERE id = ?`, id).Scan(&estado); err == sql.ErrNoRows {
		http.Error(w, "toma no encontrada", http.StatusNotFound)
		return
	}
	if estado == "cerrado" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "ya cerrada"})
		return
	}
	var cerradoPor *string
	if u, ok := GetAuthUser(r); ok {
		cerradoPor = &u.ID
	}
	cerradoEn := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`UPDATE tomas_fisicas SET estado='cerrado', cerrado_en=?, cerrado_por=? WHERE id=?`, cerradoEn, cerradoPor, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "toma", id, "cerrar", "estado", "abierto", "cerrado")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "toma cerrada"})
}

func (h *TomaHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var estado string
	if err := h.DB.QueryRow(`SELECT estado FROM tomas_fisicas WHERE id=?`, id).Scan(&estado); err == sql.ErrNoRows {
		http.Error(w, "toma no encontrada", http.StatusNotFound)
		return
	}
	if estado != "abierto" {
		http.Error(w, "solo tomas abiertas se pueden eliminar", http.StatusBadRequest)
		return
	}
	res, err := h.DB.Exec(`DELETE FROM tomas_fisicas WHERE id=?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "toma no encontrada", http.StatusNotFound)
		return
	}
	RegistrarAuditoria(h.DB, r, "toma", id, "eliminar", "", "", "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "toma eliminada"})
}
