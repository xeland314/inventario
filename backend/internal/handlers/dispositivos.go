package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"inventario-backend/internal/models"
)

type DispositivoHandler struct {
	DB *sql.DB
}

// helper scan
func scanDispositivo(rows *sql.Rows) (models.Dispositivo, error) {
	var d models.Dispositivo
	err := rows.Scan(&d.ID, &d.CodigoActivo, &d.NumeroSerie, &d.Tipo, &d.Marca, &d.Modelo, &d.Especificaciones, &d.ValorCompra, &d.Proveedor, &d.Estado, &d.Ubicacion, &d.AsignadoA, &d.FechaCompra, &d.FechaBaja, &d.MotivoBaja, &d.ParentID, &d.TipoComponente, &d.Sede, &d.CreadoEn, &d.ActualizadoEn)
	return d, err
}

func scanDispositivoRow(row *sql.Row) (models.Dispositivo, error) {
	var d models.Dispositivo
	err := row.Scan(&d.ID, &d.CodigoActivo, &d.NumeroSerie, &d.Tipo, &d.Marca, &d.Modelo, &d.Especificaciones, &d.ValorCompra, &d.Proveedor, &d.Estado, &d.Ubicacion, &d.AsignadoA, &d.FechaCompra, &d.FechaBaja, &d.MotivoBaja, &d.ParentID, &d.TipoComponente, &d.Sede, &d.CreadoEn, &d.ActualizadoEn)
	return d, err
}

func (h *DispositivoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	estado := r.URL.Query().Get("estado")
	tipo := r.URL.Query().Get("tipo")
	ubicacion := r.URL.Query().Get("ubicacion")
	sede := r.URL.Query().Get("sede")
	parentID := r.URL.Query().Get("parent_id")
	tipoComp := r.URL.Query().Get("tipo_componente")
	soloComponentes := r.URL.Query().Get("solo_componentes")
	soloPadres := r.URL.Query().Get("solo_padres")
	q := r.URL.Query().Get("q")
	if q == "" {
		q = r.URL.Query().Get("search")
	}
	// alias para búsqueda global usada por combobox
	if q == "" {
		q = r.URL.Query().Get("query")
	}

	query := `SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE 1=1`
	args := []interface{}{}
	if estado != "" {
		if !models.EsEstadoValido(estado) {
			http.Error(w, "estado no válido", http.StatusBadRequest)
			return
		}
		query += ` AND estado = ?`
		args = append(args, estado)
	}
	if tipo != "" {
		query += ` AND tipo = ?`
		args = append(args, tipo)
	}
	if ubicacion != "" {
		query += ` AND ubicacion = ?`
		args = append(args, ubicacion)
	}
	if sede != "" {
		query += ` AND sede = ?`
		args = append(args, sede)
	}
	if parentID != "" {
		if parentID == "null" || parentID == "NULL" {
			query += ` AND parent_id IS NULL`
		} else {
			query += ` AND parent_id = ?`
			args = append(args, parentID)
		}
	}
	if tipoComp != "" {
		query += ` AND tipo_componente = ?`
		args = append(args, tipoComp)
	}
	if soloComponentes == "true" || soloComponentes == "1" {
		query += ` AND parent_id IS NOT NULL`
	}
	if soloPadres == "true" || soloPadres == "1" {
		query += ` AND parent_id IS NULL`
	}
	if q != "" {
		like := "%" + q + "%"
		query += ` AND (codigo_activo LIKE ? OR numero_serie LIKE ? OR marca LIKE ? OR modelo LIKE ? OR tipo LIKE ? OR id LIKE ? OR COALESCE(especificaciones,'') LIKE ? OR EXISTS (SELECT 1 FROM etiquetas e WHERE e.dispositivo_id = dispositivos.id AND (e.id LIKE ? OR e.codigo_barras LIKE ?)))`
		args = append(args, like, like, like, like, like, like, like, like, like)
	}
	query += ` ORDER BY creado_en DESC`
	// límite opcional para combobox grande
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

	var list []models.Dispositivo
	for rows.Next() {
		d, err := scanDispositivo(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list = append(list, d)
	}
	if list == nil {
		list = []models.Dispositivo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"dispositivos": list})
}

func (h *DispositivoHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var d models.Dispositivo
	err := h.DB.QueryRow(`SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE id = ?`, id).
		Scan(&d.ID, &d.CodigoActivo, &d.NumeroSerie, &d.Tipo, &d.Marca, &d.Modelo, &d.Especificaciones, &d.ValorCompra, &d.Proveedor, &d.Estado, &d.Ubicacion, &d.AsignadoA, &d.FechaCompra, &d.FechaBaja, &d.MotivoBaja, &d.ParentID, &d.TipoComponente, &d.Sede, &d.CreadoEn, &d.ActualizadoEn)
	if err == sql.ErrNoRows {
		http.Error(w, "Dispositivo no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Cargar tintas si es impresora
	if d.Tipo == "Impresora" {
		rows, _ := h.DB.Query(`SELECT id, dispositivo_id, color, nivel_porcentaje, codigo_tinta, ultima_actualizacion FROM niveles_tinta WHERE dispositivo_id = ?`, id)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var t models.NivelTinta
				rows.Scan(&t.ID, &t.DispositivoID, &t.Color, &t.NivelPorcentaje, &t.CodigoTinta, &t.UltimaActualizacion)
				d.Tintas = append(d.Tintas, t)
			}
		}
	}
	// Cargar componentes hijos (ej: ventilador dentro de CPU)
	rows, _ := h.DB.Query(`SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE parent_id = ? ORDER BY tipo_componente, codigo_activo`, id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			comp, _ := scanDispositivo(rows)
			d.Componentes = append(d.Componentes, comp)
		}
	}
	// Cargar etiqueta asociada (QR/UUID pegado al item)
	var et models.Etiqueta
	errEti := h.DB.QueryRow(`SELECT id, version, estado, dispositivo_id, codigo_barras, generado_en, impreso_en, asignado_en FROM etiquetas WHERE dispositivo_id = ?`, id).Scan(&et.ID, &et.Version, &et.Estado, &et.DispositivoID, &et.CodigoBarras, &et.GeneradoEn, &et.ImpresoEn, &et.AsignadoEn)
	if errEti == nil {
		d.Etiqueta = &et
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

func (h *DispositivoHandler) ListarComponentes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// verificar existe padre
	var exists string
	if err := h.DB.QueryRow(`SELECT id FROM dispositivos WHERE id = ?`, id).Scan(&exists); err == sql.ErrNoRows {
		http.Error(w, "Dispositivo no encontrado", http.StatusNotFound)
		return
	}
	rows, err := h.DB.Query(`SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE parent_id = ? ORDER BY tipo_componente`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.Dispositivo
	for rows.Next() {
		d, _ := scanDispositivo(rows)
		list = append(list, d)
	}
	if list == nil {
		list = []models.Dispositivo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"componentes": list, "parent_id": id})
}

// valida que parent no genere ciclo y existe
func (h *DispositivoHandler) validarParent(dispositivoID string, parentID *string) error {
	if parentID == nil || *parentID == "" {
		return nil
	}
	if dispositivoID != "" && *parentID == dispositivoID {
		return sql.ErrNoRows // usar como señal de error ciclo
	}
	var exists string
	if err := h.DB.QueryRow(`SELECT id FROM dispositivos WHERE id = ?`, *parentID).Scan(&exists); err == sql.ErrNoRows {
		return sql.ErrNoRows
	} else if err != nil {
		return err
	}
	// detectar ciclo: recorrer anc estros del parent hasta raíz
	if dispositivoID != "" {
		current := *parentID
		visited := map[string]bool{}
		for i := 0; i < 20; i++ {
			if visited[current] {
				break
			}
			visited[current] = true
			if current == dispositivoID {
				return sql.ErrNoRows
			}
			var next *string
			if err := h.DB.QueryRow(`SELECT parent_id FROM dispositivos WHERE id = ?`, current).Scan(&next); err != nil || next == nil {
				break
			}
			current = *next
		}
	}
	return nil
}

func (h *DispositivoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CodigoActivo     string  `json:"codigo_activo"`
		NumeroSerie      string  `json:"numero_serie"`
		Tipo             string  `json:"tipo"`
		Marca            string  `json:"marca"`
		Modelo           string  `json:"modelo"`
		Especificaciones string  `json:"especificaciones"`
		ValorCompra      float64 `json:"valor_compra"`
		Proveedor        string  `json:"proveedor"`
		Estado           string  `json:"estado"`
		Ubicacion        string  `json:"ubicacion"`
		AsignadoA        *string `json:"asignado_a"`
		FechaCompra      string  `json:"fecha_compra"`
		ParentID         *string `json:"parent_id"`
		TipoComponente   *string `json:"tipo_componente"`
		Sede             *string `json:"sede"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.CodigoActivo == "" || req.NumeroSerie == "" || req.Tipo == "" || req.Marca == "" || req.Modelo == "" {
		http.Error(w, "Faltan campos obligatorios", http.StatusBadRequest)
		return
	}
	if req.Estado == "" {
		req.Estado = "activo"
	}
	if !models.EsEstadoValido(req.Estado) {
		http.Error(w, "estado no válido. Valores: activo, disponible, reserva, en_mantenimiento, en_reparacion, averiado, obsoleto, dado_de_baja, extraviado", http.StatusBadRequest)
		return
	}
	if !models.EsTipoValido(req.Tipo) {
		http.Error(w, "tipo no válido", http.StatusBadRequest)
		return
	}
	if req.TipoComponente != nil && *req.TipoComponente != "" && !models.EsTipoComponenteValido(*req.TipoComponente) {
		http.Error(w, "tipo_componente no válido", http.StatusBadRequest)
		return
	}
	// normalizar parent_id vacío -> nil
	if req.ParentID != nil && *req.ParentID == "" {
		req.ParentID = nil
	}
	if req.ParentID != nil {
		if err := h.validarParent("", req.ParentID); err == sql.ErrNoRows {
			http.Error(w, "parent_id no existe o genera ciclo", http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if req.FechaCompra == "" {
		req.FechaCompra = time.Now().Format("2006-01-02")
	}
	if req.Proveedor == "" {
		req.Proveedor = "No especificado"
	}
	if req.Ubicacion == "" {
		req.Ubicacion = "Almacén"
	}

	id := uuid.New().String()
	creadoEn := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`INSERT INTO dispositivos (id, codigo_activo, numero_serie, tipo, marca, modelo, especificaciones, valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, creado_en, parent_id, tipo_componente, sede)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.CodigoActivo, req.NumeroSerie, req.Tipo, req.Marca, req.Modelo, req.Especificaciones, req.ValorCompra, req.Proveedor, req.Estado, req.Ubicacion, req.AsignadoA, req.FechaCompra, creadoEn, req.ParentID, req.TipoComponente, req.Sede)
	if err != nil {
		http.Error(w, "Error al insertar: "+err.Error(), http.StatusConflict)
		return
	}
	RegistrarAuditoria(h.DB, r, "dispositivo", id, "crear", "", "", "")
	// Registrar movimiento inicial si tiene parent
	if req.ParentID != nil {
		movID := uuid.New().String()
		h.DB.Exec(`INSERT INTO movimientos (id, dispositivo_id, origen_parent_id, destino_parent_id, tipo_movimiento, descripcion, fecha_movimiento, creado_en) VALUES (?, ?, NULL, ?, 'instalacion', 'Componente creado dentro de equipo', ?, ?)`,
			movID, id, *req.ParentID, creadoEn, creadoEn)
	}
	// Si es impresora, inicializar tintas
	if req.Tipo == "Impresora" {
		for _, color := range []string{"negro", "cian", "magenta", "amarillo"} {
			tintaID := uuid.New().String()
			h.DB.Exec(`INSERT OR IGNORE INTO niveles_tinta (id, dispositivo_id, color, nivel_porcentaje, ultima_actualizacion) VALUES (?, ?, ?, ?, ?)`,
				tintaID, id, color, 100, creadoEn)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "message": "Dispositivo creado"})
}

func (h *DispositivoHandler) DarDeBaja(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.BajaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	if req.MotivoBaja == "" {
		http.Error(w, "motivo_baja es obligatorio", http.StatusBadRequest)
		return
	}
	fechaBaja := time.Now().Format("2006-01-02")
	actualizadoEn := time.Now().Format(time.RFC3339)
	res, err := h.DB.Exec(`UPDATE dispositivos SET estado = 'dado_de_baja', fecha_baja = ?, motivo_baja = ?, actualizado_en = ? WHERE id = ?`, fechaBaja, req.MotivoBaja, actualizadoEn, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ra, _ := res.RowsAffected()
	if ra == 0 {
		http.Error(w, "Dispositivo no encontrado", http.StatusNotFound)
		return
	}
	RegistrarAuditoria(h.DB, r, "dispositivo", id, "baja", "estado", "", "dado_de_baja")
	if req.MotivoBaja != "" {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "baja", "motivo_baja", "", req.MotivoBaja)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Equipo dado de baja correctamente", "fecha_baja": fechaBaja})
}

func (h *DispositivoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.DB.Exec(`DELETE FROM dispositivos WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ra, _ := res.RowsAffected()
	if ra == 0 {
		http.Error(w, "Dispositivo no encontrado", http.StatusNotFound)
		return
	}
	RegistrarAuditoria(h.DB, r, "dispositivo", id, "eliminar", "", "", "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Dispositivo eliminado"})
}

func (h *DispositivoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var current models.Dispositivo
	if err := h.DB.QueryRow(`SELECT id, codigo_activo, numero_serie, tipo, marca, modelo, COALESCE(especificaciones,''), valor_compra, proveedor, estado, ubicacion, asignado_a, fecha_compra, fecha_baja, motivo_baja, parent_id, tipo_componente, sede, creado_en, actualizado_en FROM dispositivos WHERE id = ?`, id).
		Scan(&current.ID, &current.CodigoActivo, &current.NumeroSerie, &current.Tipo, &current.Marca, &current.Modelo, &current.Especificaciones, &current.ValorCompra, &current.Proveedor, &current.Estado, &current.Ubicacion, &current.AsignadoA, &current.FechaCompra, &current.FechaBaja, &current.MotivoBaja, &current.ParentID, &current.TipoComponente, &current.Sede, &current.CreadoEn, &current.ActualizadoEn); err == sql.ErrNoRows {
		http.Error(w, "Dispositivo no encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req struct {
		CodigoActivo     *string  `json:"codigo_activo"`
		NumeroSerie      *string  `json:"numero_serie"`
		Tipo             *string  `json:"tipo"`
		Marca            *string  `json:"marca"`
		Modelo           *string  `json:"modelo"`
		Especificaciones *string  `json:"especificaciones"`
		ValorCompra      *float64 `json:"valor_compra"`
		Proveedor        *string  `json:"proveedor"`
		Estado           *string  `json:"estado"`
		Ubicacion        *string  `json:"ubicacion"`
		AsignadoA        *string  `json:"asignado_a"`
		FechaCompra      *string  `json:"fecha_compra"`
		ParentID         *string  `json:"parent_id"`
		TipoComponente   *string  `json:"tipo_componente"`
		Sede             *string  `json:"sede"`
		ParentIDIsNull   *bool    `json:"-"` // no usado, para distinguir null explícito
	}
	// Necesitamos detectar si parent_id fue enviado como null -> decodificar raw
	raw := map[string]json.RawMessage{}
	bodyBytes := make([]byte, 0)
	// leer body una vez
	// como ya necesitamos decodificar con distinción null vs no enviado, usamos map
	// re-leer: el Body ya fue consumido parcialmente, así que decodificamos desde r.Body de nuevo con Tee
	// simplificación: decodificar req normal y además chequear raw para parent_id
	// primero leemos todo
	// nota: ya consumimos? no, ahora vamos a decodificar raw primero
	// para no complicar, decodificamos dos veces usando buffer
	// --- enfoque simple: leer body ---
	// Como no tenemos body guardado, usamos json.Decoder con UseNumber pero perdemos raw null detection para otros campos
	// Vamos a implementar detección manual: si ParentID == nil pero key existe con null, lo tratamos como desasignación
	// Para eso necesitamos leer raw
	if err := func() error {
		// body ya no está disponible si ya leímos, pero no hemos leído aún en este scope? sí arriba no hemos decodificado req aún
		// entonces decodificamos directamente a req
		return json.NewDecoder(r.Body).Decode(&req)
	}(); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Detectar null explícito para parent_id: si cliente envió "parent_id": null, json decodifica como nil pero igual que no enviado.
	// No distinguimos aquí; para desasignar, cliente debe enviar "" o usar endpoint movimientos desasignacion.
	// Por simplicidad, "" significa desasignar (NULL)
	_ = raw
	_ = bodyBytes

	sets := []string{}
	args := []interface{}{}
	if req.CodigoActivo != nil {
		if *req.CodigoActivo == "" {
			http.Error(w, "codigo_activo no puede estar vacío", http.StatusBadRequest)
			return
		}
		sets = append(sets, "codigo_activo = ?")
		args = append(args, *req.CodigoActivo)
	}
	if req.NumeroSerie != nil {
		sets = append(sets, "numero_serie = ?")
		args = append(args, *req.NumeroSerie)
	}
	if req.Tipo != nil {
		if !models.EsTipoValido(*req.Tipo) {
			http.Error(w, "tipo no válido", http.StatusBadRequest)
			return
		}
		sets = append(sets, "tipo = ?")
		args = append(args, *req.Tipo)
	}
	if req.Marca != nil {
		sets = append(sets, "marca = ?")
		args = append(args, *req.Marca)
	}
	if req.Modelo != nil {
		sets = append(sets, "modelo = ?")
		args = append(args, *req.Modelo)
	}
	if req.Especificaciones != nil {
		sets = append(sets, "especificaciones = ?")
		args = append(args, *req.Especificaciones)
	}
	if req.ValorCompra != nil {
		sets = append(sets, "valor_compra = ?")
		args = append(args, *req.ValorCompra)
	}
	if req.Proveedor != nil {
		sets = append(sets, "proveedor = ?")
		args = append(args, *req.Proveedor)
	}
	if req.Estado != nil {
		if !models.EsEstadoValido(*req.Estado) {
			http.Error(w, "estado no válido", http.StatusBadRequest)
			return
		}
		sets = append(sets, "estado = ?")
		args = append(args, *req.Estado)
	}
	if req.Ubicacion != nil {
		sets = append(sets, "ubicacion = ?")
		args = append(args, *req.Ubicacion)
	}
	if req.AsignadoA != nil {
		if *req.AsignadoA == "" {
			sets = append(sets, "asignado_a = NULL")
		} else {
			sets = append(sets, "asignado_a = ?")
			args = append(args, *req.AsignadoA)
		}
	}
	if req.FechaCompra != nil {
		sets = append(sets, "fecha_compra = ?")
		args = append(args, *req.FechaCompra)
	}
	if req.TipoComponente != nil {
		if *req.TipoComponente != "" && !models.EsTipoComponenteValido(*req.TipoComponente) {
			http.Error(w, "tipo_componente no válido", http.StatusBadRequest)
			return
		}
		if *req.TipoComponente == "" {
			sets = append(sets, "tipo_componente = NULL")
		} else {
			sets = append(sets, "tipo_componente = ?")
			args = append(args, *req.TipoComponente)
		}
	}
	if req.Sede != nil {
		if *req.Sede == "" {
			sets = append(sets, "sede = NULL")
		} else {
			sets = append(sets, "sede = ?")
			args = append(args, *req.Sede)
		}
	}
	// parent_id handling: nil -> no cambio, "" -> NULL (desasignar), valor -> validar y asignar
	parentChange := false
	var nuevoParent *string
	var origenParent *string = current.ParentID
	if req.ParentID != nil {
		parentChange = true
		if *req.ParentID == "" {
			sets = append(sets, "parent_id = NULL")
			nuevoParent = nil
		} else {
			if err := h.validarParent(id, req.ParentID); err == sql.ErrNoRows {
				http.Error(w, "parent_id no existe o genera ciclo/self-reference", http.StatusBadRequest)
				return
			} else if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sets = append(sets, "parent_id = ?")
			args = append(args, *req.ParentID)
			nuevoParent = req.ParentID
		}
	}

	if len(sets) == 0 {
		http.Error(w, "Nada para actualizar", http.StatusBadRequest)
		return
	}

	sets = append(sets, "actualizado_en = ?")
	args = append(args, time.Now().Format(time.RFC3339))

	tipoAnterior := current.Tipo
	query := "UPDATE dispositivos SET " + joinStrings(sets, ", ") + " WHERE id = ?"
	args = append(args, id)
	if _, err := h.DB.Exec(query, args...); err != nil {
		http.Error(w, "Error al actualizar: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Auditoría por campo cambiado
	if req.CodigoActivo != nil && *req.CodigoActivo != current.CodigoActivo {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "codigo_activo", current.CodigoActivo, *req.CodigoActivo)
	}
	if req.NumeroSerie != nil && *req.NumeroSerie != current.NumeroSerie {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "numero_serie", current.NumeroSerie, *req.NumeroSerie)
	}
	if req.Tipo != nil && *req.Tipo != current.Tipo {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "tipo", current.Tipo, *req.Tipo)
	}
	if req.Marca != nil && *req.Marca != current.Marca {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "marca", current.Marca, *req.Marca)
	}
	if req.Modelo != nil && *req.Modelo != current.Modelo {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "modelo", current.Modelo, *req.Modelo)
	}
	if req.Especificaciones != nil && *req.Especificaciones != current.Especificaciones {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "especificaciones", current.Especificaciones, *req.Especificaciones)
	}
	if req.ValorCompra != nil && fmt.Sprintf("%v", *req.ValorCompra) != fmt.Sprintf("%v", current.ValorCompra) {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "valor_compra", fmt.Sprintf("%v", current.ValorCompra), fmt.Sprintf("%v", *req.ValorCompra))
	}
	if req.Proveedor != nil && *req.Proveedor != current.Proveedor {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "proveedor", current.Proveedor, *req.Proveedor)
	}
	if req.Estado != nil && *req.Estado != current.Estado {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "estado", current.Estado, *req.Estado)
	}
	if req.Ubicacion != nil && *req.Ubicacion != current.Ubicacion {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "ubicacion", current.Ubicacion, *req.Ubicacion)
	}
	if req.AsignadoA != nil {
		anterior := ""
		if current.AsignadoA != nil {
			anterior = *current.AsignadoA
		}
		if *req.AsignadoA != anterior {
			RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "asignado_a", anterior, *req.AsignadoA)
		}
	}
	if req.FechaCompra != nil && *req.FechaCompra != current.FechaCompra {
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "fecha_compra", current.FechaCompra, *req.FechaCompra)
	}
	if req.TipoComponente != nil {
		anterior := ""
		if current.TipoComponente != nil {
			anterior = *current.TipoComponente
		}
		nuevo := *req.TipoComponente
		if nuevo != anterior {
			RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "tipo_componente", anterior, nuevo)
		}
	}
	if req.Sede != nil {
		anterior := ""
		if current.Sede != nil {
			anterior = *current.Sede
		}
		nuevo := *req.Sede
		if nuevo != anterior {
			RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "sede", anterior, nuevo)
		}
	}
	if parentChange {
		anterior := ""
		if origenParent != nil {
			anterior = *origenParent
		}
		nuevo := ""
		if nuevoParent != nil {
			nuevo = *nuevoParent
		}
		RegistrarAuditoria(h.DB, r, "dispositivo", id, "actualizar", "parent_id", anterior, nuevo)
	}

	// Registrar movimiento si hubo cambio de parent
	if parentChange {
		// no registrar si origen y destino son iguales (no cambio real)
		origenStr := ""
		if origenParent != nil {
			origenStr = *origenParent
		}
		destStr := ""
		if nuevoParent != nil {
			destStr = *nuevoParent
		}
		if origenStr != destStr {
			var tipoMov string
			var desc string
			switch {
			case origenParent == nil && nuevoParent != nil:
				tipoMov = "instalacion"
				desc = "Componente instalado en equipo"
			case origenParent != nil && nuevoParent == nil:
				tipoMov = "retiro"
				desc = "Componente retirado de equipo"
			case origenParent != nil && nuevoParent != nil:
				tipoMov = "traslado"
				desc = "Componente trasladado entre equipos"
			}
			movID := uuid.New().String()
			now := time.Now().Format(time.RFC3339)
			h.DB.Exec(`INSERT INTO movimientos (id, dispositivo_id, origen_parent_id, destino_parent_id, tipo_movimiento, descripcion, fecha_movimiento, creado_en) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				movID, id, origenParent, nuevoParent, tipoMov, desc, now, now)
		}
	}

	// Si cambió a Impresora y antes no lo era, inicializar tintas
	if req.Tipo != nil && *req.Tipo == "Impresora" && tipoAnterior != "Impresora" {
		creadoEn := time.Now().Format(time.RFC3339)
		for _, color := range []string{"negro", "cian", "magenta", "amarillo"} {
			tintaID := uuid.New().String()
			h.DB.Exec(`INSERT OR IGNORE INTO niveles_tinta (id, dispositivo_id, color, nivel_porcentaje, ultima_actualizacion) VALUES (?, ?, ?, ?, ?)`,
				tintaID, id, color, 100, creadoEn)
		}
	}

	h.Obtener(w, r)
}

func joinStrings(a []string, sep string) string {
	res := ""
	for i, s := range a {
		if i > 0 {
			res += sep
		}
		res += s
	}
	return res
}
