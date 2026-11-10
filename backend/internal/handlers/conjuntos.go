package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"inventario-backend/internal/models"
)

// ConjuntoHandler: stock a granel por caja (tornillos, resistores...).
// La caja lleva la etiqueta QR del pool; cada pieza NO se etiqueta.
// Patrón frascos: cantidad + libro de movimientos + mínimo de alerta.
type ConjuntoHandler struct {
	DB *sql.DB
}

func estadoConjunto(cant, min int) string {
	if cant <= 0 {
		return "agotado"
	}
	if cant <= min {
		return "bajo"
	}
	return "disponible"
}

const conjuntosCols = `c.id, c.codigo, c.nombre, c.categoria, c.especificaciones, c.unidad, c.cantidad_actual, c.cantidad_minima, c.ubicacion, c.sede, c.proveedor, c.costo_unitario, c.creado_en, c.actualizado_en, e.id, e.codigo_barras`

func scanConjuntoRow(scan func(dest ...interface{}) error) (models.Conjunto, error) {
	var c models.Conjunto
	var categoria, especificaciones, ubicacion, sede, proveedor, actualizado sql.NullString
	err := scan(&c.ID, &c.Codigo, &c.Nombre, &categoria, &especificaciones, &c.Unidad, &c.CantidadActual, &c.CantidadMinima, &ubicacion, &sede, &proveedor, &c.CostoUnitario, &c.CreadoEn, &actualizado, &c.EtiquetaID, &c.EtiquetaCodigo)
	if err != nil {
		return c, err
	}
	c.Categoria = nullableString(categoria)
	c.Especificaciones = nullableString(especificaciones)
	c.Ubicacion = nullableString(ubicacion)
	c.Sede = nullableString(sede)
	c.Proveedor = nullableString(proveedor)
	c.ActualizadoEn = nullableString(actualizado)
	c.Estado = estadoConjunto(c.CantidadActual, c.CantidadMinima)
	return c, nil
}

func nullableString(n sql.NullString) *string {
	if n.Valid {
		v := n.String
		if v == "" {
			return nil
		}
		return &v
	}
	return nil
}

// Listar GET /api/conjuntos?q=&categoria=&sede=&bajo=1
func (h *ConjuntoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	categoria := r.URL.Query().Get("categoria")
	sede := r.URL.Query().Get("sede")
	ubicacion := r.URL.Query().Get("ubicacion")
	bajo := r.URL.Query().Get("bajo") // 1 = solo bajo/agotado

	query := `SELECT ` + conjuntosCols + ` FROM conjuntos c LEFT JOIN etiquetas e ON e.conjunto_id = c.id WHERE 1=1`
	args := []interface{}{}
	if categoria != "" {
		query += ` AND c.categoria = ?`
		args = append(args, categoria)
	}
	if sede != "" {
		query += ` AND c.sede = ?`
		args = append(args, sede)
	}
	if ubicacion != "" {
		query += ` AND c.ubicacion LIKE ?`
		args = append(args, "%"+ubicacion+"%")
	}
	if q != "" {
		like := "%" + q + "%"
		query += ` AND (c.codigo LIKE ? OR c.nombre LIKE ? OR c.categoria LIKE ? OR c.especificaciones LIKE ? OR c.ubicacion LIKE ? OR c.proveedor LIKE ?)`
		args = append(args, like, like, like, like, like, like)
	}
	if bajo == "1" || bajo == "true" {
		query += ` AND c.cantidad_actual <= c.cantidad_minima`
	}
	query += ` ORDER BY COALESCE(c.actualizado_en, c.creado_en) DESC`
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.Conjunto
	for rows.Next() {
		c, err := scanConjuntoRow(rows.Scan)
		if err != nil {
			continue
		}
		list = append(list, c)
	}
	if list == nil {
		list = []models.Conjunto{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"conjuntos": list})
}

// Obtener GET /api/conjuntos/{id}
func (h *ConjuntoHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	query := `SELECT ` + conjuntosCols + ` FROM conjuntos c LEFT JOIN etiquetas e ON e.conjunto_id = c.id WHERE c.id = ? OR c.codigo = ?`
	row := h.DB.QueryRow(query, id, strings.ToUpper(id))
	c, err := scanConjuntoRow(row.Scan)
	if err == sql.ErrNoRows {
		http.Error(w, "Conjunto no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// movimientos recientes
	rows, _ := h.DB.Query(`SELECT id, conjunto_id, tipo, cantidad, cantidad_previa, cantidad_nueva, realizado_por, nota, fecha FROM conjunto_movimientos WHERE conjunto_id = ? ORDER BY fecha DESC, rowid DESC LIMIT 50`, c.ID)
	c.Movimientos = []models.ConjuntoMovimiento{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var m models.ConjuntoMovimiento
			rows.Scan(&m.ID, &m.ConjuntoID, &m.Tipo, &m.Cantidad, &m.CantidadPrevia, &m.CantidadNueva, &m.RealizadoPor, &m.Nota, &m.Fecha)
			c.Movimientos = append(c.Movimientos, m)
		}
	}
	// conteos IA/humano
	rowsC, _ := h.DB.Query(`SELECT id, conjunto_id, foto_evidencia_id, cantidad_detectada, cantidad_confirmada, modelo, detecciones_json, nota, creado_por, creado_en FROM conteos WHERE conjunto_id = ? ORDER BY creado_en DESC, rowid DESC LIMIT 50`, c.ID)
	c.Conteos = []models.Conteo{}
	if rowsC != nil {
		defer rowsC.Close()
		for rowsC.Next() {
			var ct models.Conteo
			rowsC.Scan(&ct.ID, &ct.ConjuntoID, &ct.FotoEvidenciaID, &ct.CantidadDetectada, &ct.CantidadConfirmada, &ct.Modelo, &ct.DeteccionesJSON, &ct.Nota, &ct.CreadoPor, &ct.CreadoEn)
			c.Conteos = append(c.Conteos, ct)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

// Crear POST /api/conjuntos {nombre*, categoria?, unidad?, cantidad_actual?, cantidad_minima?, etiqueta_id?...}
func (h *ConjuntoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Codigo           string   `json:"codigo"`
		Nombre           string   `json:"nombre"`
		Categoria        *string  `json:"categoria"`
		Especificaciones *string  `json:"especificaciones"`
		Unidad           string   `json:"unidad"`
		CantidadActual   int      `json:"cantidad_actual"`
		CantidadMinima   int      `json:"cantidad_minima"`
		Ubicacion        *string  `json:"ubicacion"`
		Sede             *string  `json:"sede"`
		Proveedor        *string  `json:"proveedor"`
		CostoUnitario    *float64 `json:"costo_unitario"`
		EtiquetaID       *string  `json:"etiqueta_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Nombre = strings.TrimSpace(req.Nombre)
	req.Codigo = strings.TrimSpace(req.Codigo)
	if req.Nombre == "" {
		http.Error(w, "nombre es obligatorio (ej: Tornillos M3x10)", http.StatusBadRequest)
		return
	}
	req.Unidad = strings.TrimSpace(strings.ToLower(req.Unidad))
	if req.Unidad == "" {
		req.Unidad = "pieza"
	}
	if !models.EsUnidadConjuntoValida(req.Unidad) {
		http.Error(w, "unidad no válida: pieza,gr,ml,m,paquete", http.StatusBadRequest)
		return
	}
	if req.CantidadActual < 0 {
		http.Error(w, "cantidad_actual no puede ser negativa", http.StatusBadRequest)
		return
	}
	if req.CantidadMinima < 0 {
		http.Error(w, "cantidad_minima no puede ser negativa", http.StatusBadRequest)
		return
	}
	if req.Codigo == "" {
		req.Codigo = fmt.Sprintf("CJ-%s", strings.ToUpper(uuid.New().String()[:8]))
	}
	costo := 0.0
	if req.CostoUnitario != nil {
		costo = *req.CostoUnitario
	}
	id := uuid.New().String()
	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`INSERT INTO conjuntos (id, codigo, nombre, categoria, especificaciones, unidad, cantidad_actual, cantidad_minima, ubicacion, sede, proveedor, costo_unitario, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.Codigo, req.Nombre, req.Categoria, req.Especificaciones, req.Unidad, req.CantidadActual, req.CantidadMinima, req.Ubicacion, req.Sede, req.Proveedor, costo, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "código ya existe", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "conjunto", id, "crear", "", "", "")
	if req.CantidadActual != 0 {
		movID := uuid.New().String()
		h.DB.Exec(`INSERT INTO conjunto_movimientos (id, conjunto_id, tipo, cantidad, cantidad_previa, cantidad_nueva, realizado_por, nota, fecha) VALUES (?, ?, 'ingreso', ?, 0, ?, ?, ?, ?)`,
			movID, id, req.CantidadActual, req.CantidadActual, realizadoPor(r), "stock inicial", now)
	}
	// Asociar etiqueta escaneada si se envió (atomico con la creación) — el QR se pega en la caja
	if req.EtiquetaID != nil && strings.TrimSpace(*req.EtiquetaID) != "" {
		if err := asociarEtiquetaAConjunto(h.DB, r, strings.TrimSpace(*req.EtiquetaID), id, now); err != nil {
			http.Error(w, err.Error(), err.status)
			return
		}
		RegistrarAuditoria(h.DB, r, "etiqueta", strings.TrimSpace(*req.EtiquetaID), "asociar", "conjunto_id", "", id)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "codigo": req.Codigo})
}

// Actualizar PUT/PATCH /api/conjuntos/{id} — cantidad_actual NO editable (solo via libro)
func (h *ConjuntoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Codigo           *string  `json:"codigo"`
		Nombre           *string  `json:"nombre"`
		Categoria        *string  `json:"categoria"`
		Especificaciones *string  `json:"especificaciones"`
		Unidad           *string  `json:"unidad"`
		CantidadMinima   *int     `json:"cantidad_minima"`
		Ubicacion        *string  `json:"ubicacion"`
		Sede             *string  `json:"sede"`
		Proveedor        *string  `json:"proveedor"`
		CostoUnitario    *float64 `json:"costo_unitario"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Unidad != nil {
		u := strings.ToLower(strings.TrimSpace(*req.Unidad))
		if !models.EsUnidadConjuntoValida(u) {
			http.Error(w, "unidad no válida: pieza,gr,ml,m,paquete", http.StatusBadRequest)
			return
		}
		req.Unidad = &u
	}
	if req.CantidadMinima != nil && *req.CantidadMinima < 0 {
		http.Error(w, "cantidad_minima no puede ser negativa", http.StatusBadRequest)
		return
	}
	// verificar existencia (por id o codigo)
	var realID string
	if err := h.DB.QueryRow(`SELECT id FROM conjuntos WHERE id = ? OR codigo = ?`, id, strings.ToUpper(id)).Scan(&realID); err == sql.ErrNoRows {
		http.Error(w, "Conjunto no encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now().Format(time.RFC3339)
	sets := []string{}
	args := []interface{}{}
	if req.Codigo != nil {
		sets = append(sets, "codigo = ?")
		args = append(args, strings.TrimSpace(*req.Codigo))
	}
	if req.Nombre != nil {
		v := strings.TrimSpace(*req.Nombre)
		if v == "" {
			http.Error(w, "nombre no puede quedar vacío", http.StatusBadRequest)
			return
		}
		sets = append(sets, "nombre = ?")
		args = append(args, v)
	}
	if req.Categoria != nil {
		sets = append(sets, "categoria = ?")
		args = append(args, nullableStr(*req.Categoria))
	}
	if req.Especificaciones != nil {
		sets = append(sets, "especificaciones = ?")
		args = append(args, nullableStr(*req.Especificaciones))
	}
	if req.Unidad != nil {
		sets = append(sets, "unidad = ?")
		args = append(args, *req.Unidad)
	}
	if req.CantidadMinima != nil {
		sets = append(sets, "cantidad_minima = ?")
		args = append(args, *req.CantidadMinima)
	}
	if req.Ubicacion != nil {
		sets = append(sets, "ubicacion = ?")
		args = append(args, nullableStr(*req.Ubicacion))
	}
	if req.Sede != nil {
		sets = append(sets, "sede = ?")
		args = append(args, nullableStr(*req.Sede))
	}
	if req.Proveedor != nil {
		sets = append(sets, "proveedor = ?")
		args = append(args, nullableStr(*req.Proveedor))
	}
	if req.CostoUnitario != nil {
		sets = append(sets, "costo_unitario = ?")
		args = append(args, *req.CostoUnitario)
	}
	if len(sets) == 0 {
		http.Error(w, "nada para actualizar", http.StatusBadRequest)
		return
	}
	sets = append(sets, "actualizado_en = ?")
	args = append(args, now)
	args = append(args, realID)
	query := fmt.Sprintf(`UPDATE conjuntos SET %s WHERE id = ?`, strings.Join(sets, ", "))
	if _, err := h.DB.Exec(query, args...); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "código ya existe", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "conjunto", realID, "actualizar", "", "", "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func nullableStr(s string) interface{} {
	v := strings.TrimSpace(s)
	if v == "" {
		return nil
	}
	return v
}

func (h *ConjuntoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// liberar etiqueta de la caja antes de borrar (la FK solo NULLea, dejaría estado 'asignado' huérfano)
	h.DB.Exec(`UPDATE etiquetas SET conjunto_id = NULL, estado = 'disponible', asignado_en = NULL WHERE conjunto_id = ? OR conjunto_id IN (SELECT id FROM conjuntos WHERE codigo = ?)`, id, strings.ToUpper(id))
	res, err := h.DB.Exec(`DELETE FROM conjuntos WHERE id = ? OR codigo = ?`, id, strings.ToUpper(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "Conjunto no encontrado", http.StatusNotFound)
		return
	}
	RegistrarAuditoria(h.DB, r, "conjunto", id, "eliminar", "", "", "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "eliminado"})
}

// Movimiento POST /api/conjuntos/{id}/movimiento {tipo, cantidad, realizado_por?, nota?}
func (h *ConjuntoHandler) Movimiento(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Tipo         string  `json:"tipo"`
		Cantidad     int     `json:"cantidad"`
		RealizadoPor *string `json:"realizado_por"`
		Nota         *string `json:"nota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Tipo = strings.ToLower(strings.TrimSpace(req.Tipo))
	if !models.EsTipoConjuntoMovValido(req.Tipo) {
		http.Error(w, "tipo debe ser ingreso|consumo|ajuste|traslado|baja", http.StatusBadRequest)
		return
	}
	if req.Cantidad == 0 {
		http.Error(w, "cantidad no puede ser 0", http.StatusBadRequest)
		return
	}
	// ingreso positivo, consumo/baja negativo; ajuste puede ser ambos
	if (req.Tipo == "consumo" || req.Tipo == "baja") && req.Cantidad > 0 {
		req.Cantidad = -req.Cantidad
	}
	if req.Tipo == "ingreso" && req.Cantidad < 0 {
		req.Cantidad = -req.Cantidad
	}
	var actual int
	err := h.DB.QueryRow(`SELECT cantidad_actual FROM conjuntos WHERE id = ? OR codigo = ?`, id, strings.ToUpper(id)).Scan(&actual)
	if err == sql.ErrNoRows {
		http.Error(w, "Conjunto no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nueva := actual + req.Cantidad
	if nueva < 0 {
		http.Error(w, fmt.Sprintf("stock insuficiente: actual %d, intento %d", actual, req.Cantidad), http.StatusBadRequest)
		return
	}
	now := time.Now().Format(time.RFC3339)
	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res, err := tx.Exec(`UPDATE conjuntos SET cantidad_actual = ?, actualizado_en = ? WHERE id = ? OR codigo = ?`, nueva, now, id, strings.ToUpper(id))
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		tx.Rollback()
		http.Error(w, "Conjunto no encontrado", http.StatusNotFound)
		return
	}
	movID := uuid.New().String()
	_, err = tx.Exec(`INSERT INTO conjunto_movimientos (id, conjunto_id, tipo, cantidad, cantidad_previa, cantidad_nueva, realizado_por, nota, fecha) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		movID, id, req.Tipo, req.Cantidad, actual, nueva, req.RealizadoPor, req.Nota, now)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tx.Commit()
	RegistrarAuditoria(h.DB, r, "conjunto", id, "actualizar", "movimiento", fmt.Sprintf("%d", actual), fmt.Sprintf("%d", nueva))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": movID, "cantidad_actual": nueva, "cantidad_previa": actual})
}

// ListarMovimientos GET /api/conjuntos/{id}/movimientos
func (h *ConjuntoHandler) ListarMovimientos(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := h.DB.Query(`SELECT id, conjunto_id, tipo, cantidad, cantidad_previa, cantidad_nueva, realizado_por, nota, fecha FROM conjunto_movimientos WHERE conjunto_id = ? ORDER BY fecha DESC, rowid DESC LIMIT 100`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.ConjuntoMovimiento
	for rows.Next() {
		var m models.ConjuntoMovimiento
		rows.Scan(&m.ID, &m.ConjuntoID, &m.Tipo, &m.Cantidad, &m.CantidadPrevia, &m.CantidadNueva, &m.RealizadoPor, &m.Nota, &m.Fecha)
		list = append(list, m)
	}
	if list == nil {
		list = []models.ConjuntoMovimiento{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"movimientos": list})
}

// CrearConteo POST /api/conjuntos/{id}/conteos {cantidad_confirmada*, cantidad_detectada?, modelo?, detecciones?, foto_evidencia_id?, nota?, aplicar_ajuste?}
func (h *ConjuntoHandler) CrearConteo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		CantidadDetectada int    `json:"cantidad_detectada"`
		CantidadConfirmada int   `json:"cantidad_confirmada"`
		Modelo            string `json:"modelo"`
		Detecciones        []struct {
			Box   [4]float64 `json:"box"`
			Score float64    `json:"score"`
			Label string     `json:"label"`
		} `json:"detecciones"`
		FotoEvidenciaID *string `json:"foto_evidencia_id"`
		Nota            *string `json:"nota"`
		AplicarAjuste   bool    `json:"aplicar_ajuste"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.CantidadConfirmada < 0 {
		http.Error(w, "cantidad_confirmada no puede ser negativa", http.StatusBadRequest)
		return
	}
	if req.CantidadDetectada < 0 {
		req.CantidadDetectada = 0
	}
	if strings.TrimSpace(req.Modelo) == "" {
		req.Modelo = "manual"
	}
	var actual int
	err := h.DB.QueryRow(`SELECT cantidad_actual FROM conjuntos WHERE id = ? OR codigo = ?`, id, strings.ToUpper(id)).Scan(&actual)
	if err == sql.ErrNoRows {
		http.Error(w, "Conjunto no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// foto debe existir si se envía
	if req.FotoEvidenciaID != nil && strings.TrimSpace(*req.FotoEvidenciaID) != "" {
		var exists string
		if err := h.DB.QueryRow(`SELECT id FROM fotos_evidencia WHERE id = ?`, strings.TrimSpace(*req.FotoEvidenciaID)).Scan(&exists); err == sql.ErrNoRows {
			http.Error(w, "foto_evidencia_id no encontrado", http.StatusNotFound)
			return
		}
	}
	deteccionesJSON := interface{}(nil)
	if req.Detecciones != nil {
		if b, err := json.Marshal(req.Detecciones); err == nil {
			s := string(b)
			deteccionesJSON = s
		}
	}
	conteoID := uuid.New().String()
	now := time.Now().Format(time.RFC3339)
	creadoPor := creadoPorUser(r)

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = tx.Exec(`INSERT INTO conteos (id, conjunto_id, foto_evidencia_id, cantidad_detectada, cantidad_confirmada, modelo, detecciones_json, nota, creado_por, creado_en) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		conteoID, id, req.FotoEvidenciaID, req.CantidadDetectada, req.CantidadConfirmada, req.Modelo, deteccionesJSON, req.Nota, creadoPor, now)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	movID := ""
	// ajuste atómico opcional: delta = confirmada - actual
	delta := req.CantidadConfirmada - actual
	if req.AplicarAjuste && delta != 0 {
		movID = uuid.New().String()
		nota := fmt.Sprintf("conteo %s (IA sugirió %d, humano confirmó %d)", req.Modelo, req.CantidadDetectada, req.CantidadConfirmada)
		_, err = tx.Exec(`UPDATE conjuntos SET cantidad_actual = ?, actualizado_en = ? WHERE id = ? OR codigo = ?`, req.CantidadConfirmada, now, id, strings.ToUpper(id))
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec(`INSERT INTO conjunto_movimientos (id, conjunto_id, tipo, cantidad, cantidad_previa, cantidad_nueva, realizado_por, nota, fecha) VALUES (?, ?, 'ajuste', ?, ?, ?, ?, ?, ?)`,
			movID, id, delta, actual, req.CantidadConfirmada, creadoPor, nota, now)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	tx.Commit()
	RegistrarAuditoria(h.DB, r, "conjunto", id, "crear", "conteo", fmt.Sprintf("%d", req.CantidadDetectada), fmt.Sprintf("%d", req.CantidadConfirmada))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": conteoID, "cantidad_confirmada": req.CantidadConfirmada, "cantidad_previa": actual, "ajuste_aplicado": req.AplicarAjuste && delta != 0, "delta": delta})
}

// ListarConteos GET /api/conjuntos/{id}/conteos
func (h *ConjuntoHandler) ListarConteos(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := h.DB.Query(`SELECT id, conjunto_id, foto_evidencia_id, cantidad_detectada, cantidad_confirmada, modelo, detecciones_json, nota, creado_por, creado_en FROM conteos WHERE conjunto_id = ? ORDER BY creado_en DESC, rowid DESC LIMIT 100`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.Conteo
	for rows.Next() {
		var ct models.Conteo
		rows.Scan(&ct.ID, &ct.ConjuntoID, &ct.FotoEvidenciaID, &ct.CantidadDetectada, &ct.CantidadConfirmada, &ct.Modelo, &ct.DeteccionesJSON, &ct.Nota, &ct.CreadoPor, &ct.CreadoEn)
		list = append(list, ct)
	}
	if list == nil {
		list = []models.Conteo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"conteos": list})
}

// asociarEtiquetaAConjunto valida estado/uniqueness y pega la etiqueta (QR) en la caja.
type conjuntosHTTPErr struct {
	msg    string
	status int
}

func (e *conjuntosHTTPErr) Error() string { return e.msg }

func asociarEtiquetaAConjunto(db *sql.DB, r *http.Request, etiInput, conjuntoID, now string) *conjuntosHTTPErr {
	// resolver etiqueta por UUID o ET-XXXX
	var etiID, estado string
	var dispActual, conjActual sql.NullString
	err := db.QueryRow(`SELECT id, estado, dispositivo_id, conjunto_id FROM etiquetas WHERE id = ? OR codigo_barras = ?`, etiInput, strings.ToUpper(etiInput)).Scan(&etiID, &estado, &dispActual, &conjActual)
	if err == sql.ErrNoRows {
		return &conjuntosHTTPErr{"Etiqueta no encontrada", http.StatusNotFound}
	}
	if err != nil {
		return &conjuntosHTTPErr{err.Error(), http.StatusInternalServerError}
	}
	if estado == "anulado" {
		return &conjuntosHTTPErr{"Etiqueta anulada no se puede asociar", http.StatusBadRequest}
	}
	if dispActual.Valid || conjActual.Valid {
		return &conjuntosHTTPErr{"Etiqueta ya asignada", http.StatusConflict}
	}
	// la caja no debe tener ya otra etiqueta
	var otra string
	if err := db.QueryRow(`SELECT id FROM etiquetas WHERE conjunto_id = ?`, conjuntoID).Scan(&otra); err == nil {
		return &conjuntosHTTPErr{"El conjunto ya tiene etiqueta asociada: " + otra, http.StatusConflict}
	}
	_, err = db.Exec(`UPDATE etiquetas SET conjunto_id = ?, estado = 'asignado', asignado_en = ? WHERE id = ?`, conjuntoID, now, etiID)
	if err != nil {
		return &conjuntosHTTPErr{err.Error(), http.StatusInternalServerError}
	}
	return nil
}

func creadoPorUser(r *http.Request) interface{} {
	if u, ok := GetAuthUser(r); ok && u != nil {
		return u.ID
	}
	return nil
}

func realizadoPor(r *http.Request) interface{} {
	if u, ok := GetAuthUser(r); ok && u != nil {
		return u.Username
	}
	return nil
}
