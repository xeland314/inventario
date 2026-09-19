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

type PlanHandler struct {
	DB *sql.DB
}

func isValidDateYYYYMMDD(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func (h *PlanHandler) Listar(w http.ResponseWriter, r *http.Request) {
	dispositivoID := r.URL.Query().Get("dispositivo_id")
	tipo := r.URL.Query().Get("tipo")
	activoStr := r.URL.Query().Get("activo")
	q := r.URL.Query().Get("q")
	query := `SELECT id, dispositivo_id, tipo, cada_dias, proxima_fecha, descripcion, activo, creado_en, actualizado_en, creado_por FROM planes_mantenimiento WHERE 1=1`
	args := []interface{}{}
	if dispositivoID != "" {
		query += ` AND dispositivo_id = ?`
		args = append(args, dispositivoID)
	}
	if tipo != "" {
		query += ` AND tipo = ?`
		args = append(args, tipo)
	}
	if activoStr != "" {
		query += ` AND activo = ?`
		if activoStr == "1" || activoStr == "true" {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if q != "" {
		like := "%" + q + "%"
		query += ` AND descripcion LIKE ?`
		args = append(args, like)
	}
	query += ` ORDER BY proxima_fecha ASC`
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
	var list []models.PlanMantenimiento
	for rows.Next() {
		var p models.PlanMantenimiento
		rows.Scan(&p.ID, &p.DispositivoID, &p.Tipo, &p.CadaDias, &p.ProximaFecha, &p.Descripcion, &p.Activo, &p.CreadoEn, &p.ActualizadoEn, &p.CreadoPor)
		list = append(list, p)
	}
	if list == nil {
		list = []models.PlanMantenimiento{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"planes": list})
}

func (h *PlanHandler) Crear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DispositivoID *string `json:"dispositivo_id"`
		Tipo          *string `json:"tipo"`
		CadaDias      int     `json:"cada_dias"`
		ProximaFecha  string  `json:"proxima_fecha"`
		Descripcion   string  `json:"descripcion"`
		Activo        *int    `json:"activo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Descripcion = strings.TrimSpace(req.Descripcion)
	if req.Descripcion == "" {
		http.Error(w, "descripcion requerida", http.StatusBadRequest)
		return
	}
	if req.CadaDias <= 0 || req.CadaDias > 3650 {
		http.Error(w, "cada_dias debe ser 1..3650", http.StatusBadRequest)
		return
	}
	if !isValidDateYYYYMMDD(req.ProximaFecha) {
		http.Error(w, "proxima_fecha debe ser YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	if req.DispositivoID != nil && *req.DispositivoID == "" {
		req.DispositivoID = nil
	}
	if req.Tipo != nil && *req.Tipo == "" {
		req.Tipo = nil
	}
	if req.Tipo != nil && !models.EsTipoValido(*req.Tipo) {
		http.Error(w, "tipo no válido", http.StatusBadRequest)
		return
	}
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
	activo := 1
	if req.Activo != nil {
		activo = *req.Activo
		if activo != 0 && activo != 1 {
			activo = 1
		}
	}
	id := uuid.NewString()
	creadoEn := time.Now().Format(time.RFC3339)
	var creadoPor *string
	if u, ok := GetAuthUser(r); ok {
		creadoPor = &u.ID
	}
	_, err := h.DB.Exec(`INSERT INTO planes_mantenimiento (id, dispositivo_id, tipo, cada_dias, proxima_fecha, descripcion, activo, creado_en, creado_por) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.DispositivoID, req.Tipo, req.CadaDias, req.ProximaFecha, req.Descripcion, activo, creadoEn, creadoPor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RegistrarAuditoria(h.DB, r, "plan", id, "crear", "", "", "")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id})
}

func (h *PlanHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p models.PlanMantenimiento
	err := h.DB.QueryRow(`SELECT id, dispositivo_id, tipo, cada_dias, proxima_fecha, descripcion, activo, creado_en, actualizado_en, creado_por FROM planes_mantenimiento WHERE id = ?`, id).
		Scan(&p.ID, &p.DispositivoID, &p.Tipo, &p.CadaDias, &p.ProximaFecha, &p.Descripcion, &p.Activo, &p.CreadoEn, &p.ActualizadoEn, &p.CreadoPor)
	if err == sql.ErrNoRows {
		http.Error(w, "plan no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (h *PlanHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var current models.PlanMantenimiento
	if err := h.DB.QueryRow(`SELECT id, dispositivo_id, tipo, cada_dias, proxima_fecha, descripcion, activo, creado_en, actualizado_en, creado_por FROM planes_mantenimiento WHERE id = ?`, id).
		Scan(&current.ID, &current.DispositivoID, &current.Tipo, &current.CadaDias, &current.ProximaFecha, &current.Descripcion, &current.Activo, &current.CreadoEn, &current.ActualizadoEn, &current.CreadoPor); err == sql.ErrNoRows {
		http.Error(w, "plan no encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var req struct {
		DispositivoID *string `json:"dispositivo_id"`
		Tipo          *string `json:"tipo"`
		CadaDias      *int    `json:"cada_dias"`
		ProximaFecha  *string `json:"proxima_fecha"`
		Descripcion   *string `json:"descripcion"`
		Activo        *int    `json:"activo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	sets := []string{}
	args := []interface{}{}
	if req.DispositivoID != nil {
		if *req.DispositivoID == "" {
			sets = append(sets, "dispositivo_id = NULL")
		} else {
			var exists string
			if err := h.DB.QueryRow(`SELECT id FROM dispositivos WHERE id = ?`, *req.DispositivoID).Scan(&exists); err == sql.ErrNoRows {
				http.Error(w, "dispositivo no encontrado", http.StatusNotFound)
				return
			}
			sets = append(sets, "dispositivo_id = ?")
			args = append(args, *req.DispositivoID)
		}
		anterior := ""
		if current.DispositivoID != nil {
			anterior = *current.DispositivoID
		}
		nuevo := ""
		if req.DispositivoID != nil && *req.DispositivoID != "" {
			nuevo = *req.DispositivoID
		}
		RegistrarAuditoria(h.DB, r, "plan", id, "actualizar", "dispositivo_id", anterior, nuevo)
	}
	if req.Tipo != nil {
		if *req.Tipo == "" {
			sets = append(sets, "tipo = NULL")
		} else {
			if !models.EsTipoValido(*req.Tipo) {
				http.Error(w, "tipo no válido", http.StatusBadRequest)
				return
			}
			sets = append(sets, "tipo = ?")
			args = append(args, *req.Tipo)
		}
		anterior := ""
		if current.Tipo != nil {
			anterior = *current.Tipo
		}
		nuevo := ""
		if req.Tipo != nil {
			nuevo = *req.Tipo
		}
		RegistrarAuditoria(h.DB, r, "plan", id, "actualizar", "tipo", anterior, nuevo)
	}
	if req.CadaDias != nil {
		if *req.CadaDias <= 0 || *req.CadaDias > 3650 {
			http.Error(w, "cada_dias 1..3650", http.StatusBadRequest)
			return
		}
		sets = append(sets, "cada_dias = ?")
		args = append(args, *req.CadaDias)
		RegistrarAuditoria(h.DB, r, "plan", id, "actualizar", "cada_dias", fmt.Sprintf("%d", current.CadaDias), fmt.Sprintf("%d", *req.CadaDias))
	}
	if req.ProximaFecha != nil {
		if !isValidDateYYYYMMDD(*req.ProximaFecha) {
			http.Error(w, "proxima_fecha YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		sets = append(sets, "proxima_fecha = ?")
		args = append(args, *req.ProximaFecha)
		RegistrarAuditoria(h.DB, r, "plan", id, "actualizar", "proxima_fecha", current.ProximaFecha, *req.ProximaFecha)
	}
	if req.Descripcion != nil {
		if strings.TrimSpace(*req.Descripcion) == "" {
			http.Error(w, "descripcion no vacía", http.StatusBadRequest)
			return
		}
		sets = append(sets, "descripcion = ?")
		args = append(args, *req.Descripcion)
		RegistrarAuditoria(h.DB, r, "plan", id, "actualizar", "descripcion", current.Descripcion, *req.Descripcion)
	}
	if req.Activo != nil {
		if *req.Activo != 0 && *req.Activo != 1 {
			http.Error(w, "activo 0 o 1", http.StatusBadRequest)
			return
		}
		sets = append(sets, "activo = ?")
		args = append(args, *req.Activo)
		RegistrarAuditoria(h.DB, r, "plan", id, "actualizar", "activo", fmt.Sprintf("%d", current.Activo), fmt.Sprintf("%d", *req.Activo))
	}
	if len(sets) == 0 {
		http.Error(w, "nada para actualizar", http.StatusBadRequest)
		return
	}
	sets = append(sets, "actualizado_en = ?")
	args = append(args, time.Now().Format(time.RFC3339))
	query := "UPDATE planes_mantenimiento SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	args = append(args, id)
	if _, err := h.DB.Exec(query, args...); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Obtener(w, r)
}

func (h *PlanHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := h.DB.Exec(`DELETE FROM planes_mantenimiento WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "plan no encontrado", http.StatusNotFound)
		return
	}
	RegistrarAuditoria(h.DB, r, "plan", id, "eliminar", "", "", "")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "plan eliminado"})
}

func (h *PlanHandler) Proximos(w http.ResponseWriter, r *http.Request) {
	diasStr := r.URL.Query().Get("dias")
	if diasStr == "" {
		diasStr = r.URL.Query().Get("days")
	}
	dias := 30
	if diasStr != "" {
		fmt.Sscanf(diasStr, "%d", &dias)
	}
	incluyeVencidos := r.URL.Query().Get("incluye_vencidos")
	if incluyeVencidos == "" {
		incluyeVencidos = r.URL.Query().Get("vencidos")
	}
	withVencidos := incluyeVencidos == "true" || incluyeVencidos == "1"
	hoy := time.Now().Format("2006-01-02")
	limite := time.Now().AddDate(0, 0, dias).Format("2006-01-02")
	query := `SELECT id, dispositivo_id, tipo, cada_dias, proxima_fecha, descripcion, activo, creado_en, actualizado_en, creado_por FROM planes_mantenimiento WHERE activo=1`
	args := []interface{}{}
	if withVencidos {
		query += ` AND proxima_fecha <= ?`
		args = append(args, limite)
	} else {
		query += ` AND proxima_fecha >= ? AND proxima_fecha <= ?`
		args = append(args, hoy, limite)
	}
	query += ` ORDER BY proxima_fecha ASC`
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.PlanMantenimiento
	for rows.Next() {
		var p models.PlanMantenimiento
		rows.Scan(&p.ID, &p.DispositivoID, &p.Tipo, &p.CadaDias, &p.ProximaFecha, &p.Descripcion, &p.Activo, &p.CreadoEn, &p.ActualizadoEn, &p.CreadoPor)
		list = append(list, p)
	}
	if list == nil {
		list = []models.PlanMantenimiento{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"planes": list, "dias": dias, "hoy": hoy})
}

func (h *PlanHandler) Vencidos(w http.ResponseWriter, r *http.Request) {
	hoy := time.Now().Format("2006-01-02")
	rows, err := h.DB.Query(`SELECT id, dispositivo_id, tipo, cada_dias, proxima_fecha, descripcion, activo, creado_en, actualizado_en, creado_por FROM planes_mantenimiento WHERE activo=1 AND proxima_fecha < ? ORDER BY proxima_fecha ASC`, hoy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.PlanMantenimiento
	for rows.Next() {
		var p models.PlanMantenimiento
		rows.Scan(&p.ID, &p.DispositivoID, &p.Tipo, &p.CadaDias, &p.ProximaFecha, &p.Descripcion, &p.Activo, &p.CreadoEn, &p.ActualizadoEn, &p.CreadoPor)
		list = append(list, p)
	}
	if list == nil {
		list = []models.PlanMantenimiento{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"planes": list, "hoy": hoy})
}

func (h *PlanHandler) CalendarioICS(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, dispositivo_id, tipo, cada_dias, proxima_fecha, descripcion FROM planes_mantenimiento WHERE activo=1 ORDER BY proxima_fecha ASC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Inventario TI//Planes//ES\r\nCALSCALE:GREGORIAN\r\n")
	now := time.Now().UTC().Format("20060102T150405Z")
	for rows.Next() {
		var id string
		var dispID, tipo *string
		var cada int
		var proxima, desc string
		rows.Scan(&id, &dispID, &tipo, &cada, &proxima, &desc)
		dt, err := time.Parse("2006-01-02", proxima)
		if err != nil {
			continue
		}
		dtStart := dt.Format("20060102")
		summary := desc
		if tipo != nil && *tipo != "" {
			summary = *tipo + ": " + desc
		}
		if dispID != nil && *dispID != "" {
			summary = summary + " (" + *dispID + ")"
		}
		// escape
		summary = strings.ReplaceAll(summary, "\n", "\\n")
		descEsc := strings.ReplaceAll(desc, "\n", "\\n")
		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString("UID:" + id + "@inventario.local\r\n")
		b.WriteString("DTSTAMP:" + now + "\r\n")
		b.WriteString("DTSTART;VALUE=DATE:" + dtStart + "\r\n")
		b.WriteString("SUMMARY:" + summary + "\r\n")
		b.WriteString("DESCRIPTION:" + descEsc + " cada " + fmt.Sprintf("%d", cada) + " dias\r\n")
		b.WriteString("END:VEVENT\r\n")
	}
	b.WriteString("END:VCALENDAR\r\n")
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="planes_mantenimiento.ics"`)
	w.Write([]byte(b.String()))
}

// RecalcularProxima actualiza planes matching al registrar mantenimiento real, auditando cambio de proxima_fecha
func RecalcularProxima(db *sql.DB, r *http.Request, dispositivoID string, fechaReal string) error {
	var tipo string
	if err := db.QueryRow(`SELECT tipo FROM dispositivos WHERE id = ?`, dispositivoID).Scan(&tipo); err != nil {
		return nil
	}
	// debug: uncomment if needed
	// fmt.Printf("recalc dispositivo %s tipo %s fechaReal %s\n", dispositivoID, tipo, fechaReal)
	fechaStr := fechaReal
	if t, err := time.Parse(time.RFC3339, fechaReal); err == nil {
		fechaStr = t.Format("2006-01-02")
	} else if len(fechaReal) >= 10 {
		fechaStr = fechaReal[:10]
		if !isValidDateYYYYMMDD(fechaStr) {
			fechaStr = time.Now().Format("2006-01-02")
		}
	}
	base, err := time.Parse("2006-01-02", fechaStr)
	if err != nil {
		base = time.Now()
	}
	rows, err := db.Query(`SELECT id, cada_dias, proxima_fecha FROM planes_mantenimiento WHERE activo=1 AND (dispositivo_id = ? OR tipo = ?)`, dispositivoID, tipo)
	if err != nil {
		return err
	}
	type pending struct {
		id       string
		cada     int
		anterior string
	}
	var pendings []pending
	for rows.Next() {
		var pid string
		var cada int
		var anterior string
		if err := rows.Scan(&pid, &cada, &anterior); err != nil {
			continue
		}
		pendings = append(pendings, pending{pid, cada, anterior})
	}
	rows.Close()
	for _, p := range pendings {
		nueva := base.AddDate(0, 0, p.cada).Format("2006-01-02")
		if _, err := db.Exec(`UPDATE planes_mantenimiento SET proxima_fecha = ?, actualizado_en = ? WHERE id = ?`, nueva, time.Now().Format(time.RFC3339), p.id); err == nil {
			RegistrarAuditoria(db, r, "plan", p.id, "actualizar", "proxima_fecha", p.anterior, nueva)
		}
	}
	return nil
}
