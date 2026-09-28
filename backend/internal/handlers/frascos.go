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
)

type FrascoHandler struct {
	DB *sql.DB
}

var coloresValidos = map[string]bool{
	"negro": true, "cian": true, "magenta": true, "amarillo": true, "negro_foto": true, "gris": true, "otro": true,
}

func estadoFrasco(cant, min int) string {
	if cant <= 0 {
		return "agotado"
	}
	if cant <= min {
		return "bajo"
	}
	return "disponible"
}

// Listar GET /api/frascos?sede=&color=&marca=&q=&bajo=1
func (h *FrascoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sede := r.URL.Query().Get("sede")
	color := r.URL.Query().Get("color")
	marca := r.URL.Query().Get("marca")
	bajo := r.URL.Query().Get("bajo") // 1 = solo bajo/agotado

	query := `SELECT id, codigo, marca, modelo_compat, color, capacidad_ml, nivel_porcentaje, cantidad_actual, cantidad_minima, sede, ubicacion, proveedor, lote, vencimiento, costo_unitario, creado_en, actualizado_en FROM frascos WHERE 1=1`
	args := []interface{}{}
	if sede != "" {
		query += ` AND sede = ?`
		args = append(args, sede)
	}
	if color != "" {
		query += ` AND color = ?`
		args = append(args, color)
	}
	if marca != "" {
		query += ` AND marca LIKE ?`
		args = append(args, "%"+marca+"%")
	}
	if q != "" {
		like := "%" + q + "%"
		query += ` AND (codigo LIKE ? OR marca LIKE ? OR modelo_compat LIKE ? OR color LIKE ? OR proveedor LIKE ? OR lote LIKE ?)`
		args = append(args, like, like, like, like, like, like)
	}
	if bajo == "1" || bajo == "true" {
		query += ` AND cantidad_actual <= cantidad_minima`
	}
	query += ` ORDER BY actualizado_en DESC, creado_en DESC`
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id, codigo, marcaV, colorV string
		var modeloCompat, sedeV, ubic, prov, loteV, venc sql.NullString
		var capacidad sql.NullInt64
		var nivel, cant, cantMin int
		var costo sql.NullFloat64
		var creado, actualizado sql.NullString
		rows.Scan(&id, &codigo, &marcaV, &modeloCompat, &colorV, &capacidad, &nivel, &cant, &cantMin, &sedeV, &ubic, &prov, &loteV, &venc, &costo, &creado, &actualizado)
		estado := estadoFrasco(cant, cantMin)
		// for print and UI, estado visual also considers nivel bajo
		item := map[string]interface{}{
			"id": id, "codigo": codigo, "marca": marcaV, "modelo_compat": nil, "color": colorV,
			"capacidad_ml": nil, "nivel_porcentaje": nivel, "cantidad_actual": cant, "cantidad_minima": cantMin, "estado": estado,
			"sede": nil, "ubicacion": nil, "proveedor": nil, "lote": nil, "vencimiento": nil,
			"costo_unitario": 0.0, "creado_en": creado.String, "actualizado_en": actualizado.String,
		}
		if modeloCompat.Valid {
			item["modelo_compat"] = modeloCompat.String
		}
		if capacidad.Valid {
			item["capacidad_ml"] = capacidad.Int64
		}
		if sedeV.Valid {
			item["sede"] = sedeV.String
		}
		if ubic.Valid {
			item["ubicacion"] = ubic.String
		}
		if prov.Valid {
			item["proveedor"] = prov.String
		}
		if loteV.Valid {
			item["lote"] = loteV.String
		}
		if venc.Valid {
			item["vencimiento"] = venc.String
		}
		if costo.Valid {
			item["costo_unitario"] = costo.Float64
		}
		list = append(list, item)
	}
	if list == nil {
		list = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"frascos": list})
}

func (h *FrascoHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var idV, codigo, marcaV, colorV string
	var modeloCompat, sedeV, ubic, prov, loteV, venc sql.NullString
	var capacidad sql.NullInt64
	var nivel, cant, cantMin int
	var costo sql.NullFloat64
	var creado, actualizado sql.NullString
	err := h.DB.QueryRow(`SELECT id, codigo, marca, modelo_compat, color, capacidad_ml, nivel_porcentaje, cantidad_actual, cantidad_minima, sede, ubicacion, proveedor, lote, vencimiento, costo_unitario, creado_en, actualizado_en FROM frascos WHERE id = ?`, id).
		Scan(&idV, &codigo, &marcaV, &modeloCompat, &colorV, &capacidad, &nivel, &cant, &cantMin, &sedeV, &ubic, &prov, &loteV, &venc, &costo, &creado, &actualizado)
	if err == sql.ErrNoRows {
		http.Error(w, "Frasco no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	estado := estadoFrasco(cant, cantMin)
	item := map[string]interface{}{
		"id": idV, "codigo": codigo, "marca": marcaV, "modelo_compat": nil, "color": colorV,
		"capacidad_ml": nil, "nivel_porcentaje": nivel, "cantidad_actual": cant, "cantidad_minima": cantMin, "estado": estado,
		"sede": nil, "ubicacion": nil, "proveedor": nil, "lote": nil, "vencimiento": nil,
		"costo_unitario": 0.0, "creado_en": creado.String, "actualizado_en": actualizado.String,
	}
	if modeloCompat.Valid {
		item["modelo_compat"] = modeloCompat.String
	}
	if capacidad.Valid {
		item["capacidad_ml"] = capacidad.Int64
	}
	if sedeV.Valid {
		item["sede"] = sedeV.String
	}
	if ubic.Valid {
		item["ubicacion"] = ubic.String
	}
	if prov.Valid {
		item["proveedor"] = prov.String
	}
	if loteV.Valid {
		item["lote"] = loteV.String
	}
	if venc.Valid {
		item["vencimiento"] = venc.String
	}
	if costo.Valid {
		item["costo_unitario"] = costo.Float64
	}
	// movimientos recientes
	rows, _ := h.DB.Query(`SELECT id, tipo, cantidad, cantidad_previa, cantidad_nueva, dispositivo_id, realizado_por, nota, fecha FROM frasco_movimientos WHERE frasco_id = ? ORDER BY fecha DESC LIMIT 50`, id)
	var movs []map[string]interface{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var mid, tipo, fecha string
			var cantM, prev, nueva int
			var dispID, realPor, nota sql.NullString
			rows.Scan(&mid, &tipo, &cantM, &prev, &nueva, &dispID, &realPor, &nota, &fecha)
			m := map[string]interface{}{"id": mid, "tipo": tipo, "cantidad": cantM, "cantidad_previa": prev, "cantidad_nueva": nueva, "fecha": fecha}
			if dispID.Valid {
				m["dispositivo_id"] = dispID.String
			}
			if realPor.Valid {
				m["realizado_por"] = realPor.String
			}
			if nota.Valid {
				m["nota"] = nota.String
			}
			movs = append(movs, m)
		}
	}
	if movs == nil {
		movs = []map[string]interface{}{}
	}
	item["movimientos"] = movs
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *FrascoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Codigo          string  `json:"codigo"`
		Marca           string  `json:"marca"`
		ModeloCompat    *string `json:"modelo_compat"`
		Color           string  `json:"color"`
		CapacidadMl     *int    `json:"capacidad_ml"`
		NivelPorcentaje *int    `json:"nivel_porcentaje"`
		CantidadActual  int     `json:"cantidad_actual"`
		CantidadMinima  *int    `json:"cantidad_minima"`
		Sede            *string `json:"sede"`
		Ubicacion       *string `json:"ubicacion"`
		Proveedor       *string `json:"proveedor"`
		Lote            *string `json:"lote"`
		Vencimiento     *string `json:"vencimiento"`
		CostoUnitario   *float64 `json:"costo_unitario"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Marca = strings.TrimSpace(req.Marca)
	req.Codigo = strings.TrimSpace(req.Codigo)
	req.Color = strings.TrimSpace(strings.ToLower(req.Color))
	if req.Marca == "" || req.Color == "" {
		http.Error(w, "marca y color son obligatorios", http.StatusBadRequest)
		return
	}
	if !coloresValidos[req.Color] {
		http.Error(w, "color no válido: negro,cian,magenta,amarillo,negro_foto,gris,otro", http.StatusBadRequest)
		return
	}
	if req.CantidadActual < 0 {
		http.Error(w, "cantidad_actual no puede ser negativa", http.StatusBadRequest)
		return
	}
	cantMin := 2
	if req.CantidadMinima != nil {
		cantMin = *req.CantidadMinima
	}
	if cantMin < 0 {
		http.Error(w, "cantidad_minima no puede ser negativa", http.StatusBadRequest)
		return
	}
	nivel := 100
	if req.NivelPorcentaje != nil {
		nivel = *req.NivelPorcentaje
	}
	if nivel < 0 || nivel > 100 {
		http.Error(w, "nivel_porcentaje debe estar entre 0 y 100", http.StatusBadRequest)
		return
	}
	if req.Codigo == "" {
		req.Codigo = fmt.Sprintf("FR-%s", strings.ToUpper(uuid.New().String()[:8]))
	}
	id := uuid.New().String()
	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`INSERT INTO frascos (id, codigo, marca, modelo_compat, color, capacidad_ml, nivel_porcentaje, cantidad_actual, cantidad_minima, sede, ubicacion, proveedor, lote, vencimiento, costo_unitario, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.Codigo, req.Marca, req.ModeloCompat, req.Color, req.CapacidadMl, nivel, req.CantidadActual, cantMin, req.Sede, req.Ubicacion, req.Proveedor, req.Lote, req.Vencimiento, req.CostoUnitario, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "código ya existe", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req.CantidadActual != 0 {
		movID := uuid.New().String()
		h.DB.Exec(`INSERT INTO frasco_movimientos (id, frasco_id, tipo, cantidad, cantidad_previa, cantidad_nueva, realizado_por, nota, fecha) VALUES (?, ?, 'ingreso', ?, 0, ?, ?, ?, ?)`,
			movID, id, req.CantidadActual, req.CantidadActual, nil, "stock inicial", now)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "codigo": req.Codigo})
}

func (h *FrascoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Marca           *string  `json:"marca"`
		ModeloCompat    *string  `json:"modelo_compat"`
		Color           *string  `json:"color"`
		CapacidadMl     *int     `json:"capacidad_ml"`
		NivelPorcentaje *int     `json:"nivel_porcentaje"`
		CantidadMinima  *int     `json:"cantidad_minima"`
		Sede            *string  `json:"sede"`
		Ubicacion       *string  `json:"ubicacion"`
		Proveedor       *string  `json:"proveedor"`
		Lote            *string  `json:"lote"`
		Vencimiento     *string  `json:"vencimiento"`
		CostoUnitario   *float64 `json:"costo_unitario"`
		Codigo          *string  `json:"codigo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	var exists int
	h.DB.QueryRow(`SELECT COUNT(*) FROM frascos WHERE id = ?`, id).Scan(&exists)
	if exists == 0 {
		http.Error(w, "Frasco no encontrado", http.StatusNotFound)
		return
	}
	if req.Color != nil {
		c := strings.ToLower(strings.TrimSpace(*req.Color))
		if !coloresValidos[c] {
			http.Error(w, "color no válido", http.StatusBadRequest)
			return
		}
		req.Color = &c
	}
	now := time.Now().Format(time.RFC3339)
	// build dynamic update
	sets := []string{}
	args := []interface{}{}
	if req.Marca != nil {
		sets = append(sets, "marca = ?")
		args = append(args, strings.TrimSpace(*req.Marca))
	}
	if req.ModeloCompat != nil {
		sets = append(sets, "modelo_compat = ?")
		args = append(args, *req.ModeloCompat)
	}
	if req.Color != nil {
		sets = append(sets, "color = ?")
		args = append(args, *req.Color)
	}
	if req.CapacidadMl != nil {
		sets = append(sets, "capacidad_ml = ?")
		args = append(args, *req.CapacidadMl)
	}
	if req.NivelPorcentaje != nil {
		if *req.NivelPorcentaje < 0 || *req.NivelPorcentaje > 100 {
			http.Error(w, "nivel_porcentaje debe estar entre 0 y 100", http.StatusBadRequest)
			return
		}
		sets = append(sets, "nivel_porcentaje = ?")
		args = append(args, *req.NivelPorcentaje)
	}
	if req.CantidadMinima != nil {
		if *req.CantidadMinima < 0 {
			http.Error(w, "cantidad_minima no puede ser negativa", http.StatusBadRequest)
			return
		}
		sets = append(sets, "cantidad_minima = ?")
		args = append(args, *req.CantidadMinima)
	}
	if req.Sede != nil {
		sets = append(sets, "sede = ?")
		args = append(args, *req.Sede)
	}
	if req.Ubicacion != nil {
		sets = append(sets, "ubicacion = ?")
		args = append(args, *req.Ubicacion)
	}
	if req.Proveedor != nil {
		sets = append(sets, "proveedor = ?")
		args = append(args, *req.Proveedor)
	}
	if req.Lote != nil {
		sets = append(sets, "lote = ?")
		args = append(args, *req.Lote)
	}
	if req.Vencimiento != nil {
		sets = append(sets, "vencimiento = ?")
		args = append(args, *req.Vencimiento)
	}
	if req.CostoUnitario != nil {
		sets = append(sets, "costo_unitario = ?")
		args = append(args, *req.CostoUnitario)
	}
	if req.Codigo != nil {
		sets = append(sets, "codigo = ?")
		args = append(args, strings.TrimSpace(*req.Codigo))
	}
	if len(sets) == 0 {
		http.Error(w, "nada para actualizar", http.StatusBadRequest)
		return
	}
	sets = append(sets, "actualizado_en = ?")
	args = append(args, now)
	args = append(args, id)
	query := fmt.Sprintf(`UPDATE frascos SET %s WHERE id = ?`, strings.Join(sets, ", "))
	if _, err := h.DB.Exec(query, args...); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *FrascoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.DB.Exec(`DELETE FROM frascos WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "Frasco no encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "eliminado"})
}

// Movimiento POST /api/frascos/{id}/movimiento {tipo, cantidad, dispositivo_id, realizado_por, nota}
func (h *FrascoHandler) Movimiento(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Tipo          string  `json:"tipo"`
		Cantidad      int     `json:"cantidad"`
		DispositivoID *string `json:"dispositivo_id"`
		RealizadoPor  *string `json:"realizado_por"`
		Nota          *string `json:"nota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Tipo = strings.ToLower(strings.TrimSpace(req.Tipo))
	if req.Tipo != "ingreso" && req.Tipo != "consumo" && req.Tipo != "ajuste" && req.Tipo != "traslado" && req.Tipo != "baja" {
		http.Error(w, "tipo debe ser ingreso|consumo|ajuste|traslado|baja", http.StatusBadRequest)
		return
	}
	if req.Cantidad == 0 {
		http.Error(w, "cantidad no puede ser 0", http.StatusBadRequest)
		return
	}
	// ingreso positivo, consumo/baja negativo; ajuste puede ser ambos
	if req.Tipo == "consumo" || req.Tipo == "baja" {
		if req.Cantidad > 0 {
			req.Cantidad = -req.Cantidad
		}
	}
	if req.Tipo == "ingreso" && req.Cantidad < 0 {
		req.Cantidad = -req.Cantidad
	}
	var actual int
	err := h.DB.QueryRow(`SELECT cantidad_actual FROM frascos WHERE id = ?`, id).Scan(&actual)
	if err == sql.ErrNoRows {
		http.Error(w, "Frasco no encontrado", http.StatusNotFound)
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
	_, err = tx.Exec(`UPDATE frascos SET cantidad_actual = ?, actualizado_en = ? WHERE id = ?`, nueva, now, id)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	movID := uuid.New().String()
	_, err = tx.Exec(`INSERT INTO frasco_movimientos (id, frasco_id, tipo, cantidad, cantidad_previa, cantidad_nueva, dispositivo_id, realizado_por, nota, fecha) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		movID, id, req.Tipo, req.Cantidad, actual, nueva, req.DispositivoID, req.RealizadoPor, req.Nota, now)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tx.Commit()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": movID, "cantidad_actual": nueva, "cantidad_previa": actual})
}

func (h *FrascoHandler) ListarMovimientos(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := h.DB.Query(`SELECT id, tipo, cantidad, cantidad_previa, cantidad_nueva, dispositivo_id, realizado_por, nota, fecha FROM frasco_movimientos WHERE frasco_id = ? ORDER BY fecha DESC LIMIT 100`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var mid, tipo, fecha string
		var cant, prev, nueva int
		var dispID, realPor, nota sql.NullString
		rows.Scan(&mid, &tipo, &cant, &prev, &nueva, &dispID, &realPor, &nota, &fecha)
		m := map[string]interface{}{"id": mid, "tipo": tipo, "cantidad": cant, "cantidad_previa": prev, "cantidad_nueva": nueva, "fecha": fecha}
		if dispID.Valid {
			m["dispositivo_id"] = dispID.String
		}
		if realPor.Valid {
			m["realizado_por"] = realPor.String
		}
		if nota.Valid {
			m["nota"] = nota.String
		}
		list = append(list, m)
	}
	if list == nil {
		list = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"movimientos": list})
}
