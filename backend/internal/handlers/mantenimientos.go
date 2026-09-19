package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type MantenimientoHandler struct {
	DB *sql.DB
}

func (h *MantenimientoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	dispositivoID := r.URL.Query().Get("dispositivo_id")
	query := `SELECT id, dispositivo_id, realizado_por, descripcion, costo, fecha_hora, url_pdf, creado_en FROM mantenimientos`
	args := []interface{}{}
	if dispositivoID != "" {
		query += ` WHERE dispositivo_id = ?`
		args = append(args, dispositivoID)
	}
	query += ` ORDER BY fecha_hora DESC`

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var id, dispID, realizadoPor, descripcion, fechaHora, creadoEn string
		var costo float64
		var urlPdf *string
		rows.Scan(&id, &dispID, &realizadoPor, &descripcion, &costo, &fechaHora, &urlPdf, &creadoEn)
		list = append(list, map[string]interface{}{
			"id": id, "dispositivo_id": dispID, "realizado_por": realizadoPor,
			"descripcion": descripcion, "costo": costo, "fecha_hora": fechaHora, "url_pdf": urlPdf, "creado_en": creadoEn,
		})
	}
	if list == nil {
		list = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"mantenimientos": list})
}

func (h *MantenimientoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	// Soporta JSON y multipart/form-data (con PDF)
	contentType := r.Header.Get("Content-Type")

	var dispositivoID, realizadoPor, descripcion, fechaHora string
	var costo float64
	var urlPdf *string

	if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "Error al parsear form: "+err.Error(), http.StatusBadRequest)
			return
		}
		dispositivoID = r.FormValue("dispositivo_id")
		realizadoPor = r.FormValue("realizado_por")
		descripcion = r.FormValue("descripcion")
		fechaHora = r.FormValue("fecha_hora")
		fmt.Sscanf(r.FormValue("costo"), "%f", &costo)

		file, header, err := r.FormFile("pdf")
		if err == nil {
			defer file.Close()
			os.MkdirAll("./uploads/pdf", 0755)
			filename := fmt.Sprintf("%s_%s", uuid.New().String(), header.Filename)
			dest := filepath.Join("./uploads/pdf", filename)
			dst, _ := os.Create(dest)
			io.Copy(dst, file)
			dst.Close()
			p := "/uploads/pdf/" + filename
			urlPdf = &p
		} else if v := strings.TrimSpace(r.FormValue("url_pdf")); v != "" {
			// Permitir URL externa si no se subió archivo
			urlPdf = &v
		}
	} else {
		var req struct {
			DispositivoID string  `json:"dispositivo_id"`
			RealizadoPor  string  `json:"realizado_por"`
			Descripcion   string  `json:"descripcion"`
			Costo         float64 `json:"costo"`
			FechaHora     string  `json:"fecha_hora"`
			UrlPdf        *string `json:"url_pdf"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
			return
		}
		dispositivoID = req.DispositivoID
		realizadoPor = req.RealizadoPor
		descripcion = req.Descripcion
		costo = req.Costo
		fechaHora = req.FechaHora
		if req.UrlPdf != nil && strings.TrimSpace(*req.UrlPdf) != "" {
			v := strings.TrimSpace(*req.UrlPdf)
			urlPdf = &v
		}
	}

	if dispositivoID == "" || realizadoPor == "" || descripcion == "" {
		http.Error(w, "dispositivo_id, realizado_por y descripcion son obligatorios", http.StatusBadRequest)
		return
	}
	if fechaHora == "" {
		fechaHora = time.Now().Format("2006-01-02 15:04:05")
	}

	id := uuid.New().String()
	creadoEn := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`INSERT INTO mantenimientos (id, dispositivo_id, realizado_por, descripcion, costo, fecha_hora, url_pdf, creado_en) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, dispositivoID, realizadoPor, descripcion, costo, fechaHora, urlPdf, creadoEn)
	if err != nil {
		http.Error(w, "Error al insertar mantenimiento: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "message": "Mantenimiento registrado", "url_pdf": urlPdf})
}

func (h *MantenimientoHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var mID, dispID, realizadoPor, descripcion, fechaHora, creadoEn string
	var costo float64
	var urlPdf *string
	err := h.DB.QueryRow(`SELECT id, dispositivo_id, realizado_por, descripcion, costo, fecha_hora, url_pdf, creado_en FROM mantenimientos WHERE id = ?`, id).
		Scan(&mID, &dispID, &realizadoPor, &descripcion, &costo, &fechaHora, &urlPdf, &creadoEn)
	if err == sql.ErrNoRows {
		http.Error(w, "Mantenimiento no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": mID, "dispositivo_id": dispID, "realizado_por": realizadoPor,
		"descripcion": descripcion, "costo": costo, "fecha_hora": fechaHora, "url_pdf": urlPdf, "creado_en": creadoEn,
	})
}

func (h *MantenimientoHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Opcional: borrar archivo PDF local si existe
	var urlPdf *string
	h.DB.QueryRow(`SELECT url_pdf FROM mantenimientos WHERE id = ?`, id).Scan(&urlPdf)
	res, err := h.DB.Exec(`DELETE FROM mantenimientos WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "Mantenimiento no encontrado", http.StatusNotFound)
		return
	}
	if urlPdf != nil && strings.HasPrefix(*urlPdf, "/uploads/") {
		// intentar borrar archivo físico (no crítico si falla)
		os.Remove(filepath.Join(".", *urlPdf))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Mantenimiento eliminado"})
}

func (h *MantenimientoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Verificar existe y obtener url actual para fallback
	var currentUrl *string
	var exists string
	if err := h.DB.QueryRow(`SELECT id, url_pdf FROM mantenimientos WHERE id = ?`, id).Scan(&exists, &currentUrl); err == sql.ErrNoRows {
		http.Error(w, "Mantenimiento no encontrado", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("Content-Type")
	var realizadoPor, descripcion, fechaHora *string
	var costo *float64
	var urlPdf *string
	var urlPdfProvided bool // para distinguir no enviado vs vacío (borrar)

	if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "Error al parsear form: "+err.Error(), http.StatusBadRequest)
			return
		}
		if v := r.FormValue("realizado_por"); v != "" {
			realizadoPor = &v
		}
		if v := r.FormValue("descripcion"); v != "" {
			descripcion = &v
		}
		if v := r.FormValue("fecha_hora"); v != "" {
			fechaHora = &v
		}
		if v := r.FormValue("costo"); v != "" {
			var c float64
			fmt.Sscanf(v, "%f", &c)
			costo = &c
		}
		// url_pdf puede venir aunque haya archivo
		if _, ok := r.MultipartForm.Value["url_pdf"]; ok {
			urlPdfProvided = true
			v := strings.TrimSpace(r.FormValue("url_pdf"))
			if v == "" {
				urlPdf = nil // borrar
			} else {
				urlPdf = &v
			}
		}
		// Si se sube nuevo PDF, tiene prioridad sobre url_pdf
		if file, header, err := r.FormFile("pdf"); err == nil {
			defer file.Close()
			os.MkdirAll("./uploads/pdf", 0755)
			filename := fmt.Sprintf("%s_%s", uuid.New().String(), header.Filename)
			dest := filepath.Join("./uploads/pdf", filename)
			dst, _ := os.Create(dest)
			io.Copy(dst, file)
			dst.Close()
			p := "/uploads/pdf/" + filename
			urlPdf = &p
			urlPdfProvided = true
			// borrar archivo anterior si era local
			if currentUrl != nil && strings.HasPrefix(*currentUrl, "/uploads/") {
				os.Remove(filepath.Join(".", *currentUrl))
			}
		}
	} else {
		var req struct {
			RealizadoPor *string  `json:"realizado_por"`
			Descripcion  *string  `json:"descripcion"`
			Costo        *float64 `json:"costo"`
			FechaHora    *string  `json:"fecha_hora"`
			UrlPdf       *string  `json:"url_pdf"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
			return
		}
		realizadoPor = req.RealizadoPor
		descripcion = req.Descripcion
		costo = req.Costo
		fechaHora = req.FechaHora
		if req.UrlPdf != nil {
			urlPdfProvided = true
			v := strings.TrimSpace(*req.UrlPdf)
			if v == "" {
				urlPdf = nil
			} else {
				urlPdf = &v
			}
			// si cambia de archivo local a URL externa, borrar archivo viejo
			if urlPdf != nil && currentUrl != nil && strings.HasPrefix(*currentUrl, "/uploads/") && !strings.HasPrefix(*urlPdf, "/uploads/") {
				os.Remove(filepath.Join(".", *currentUrl))
			}
			if urlPdf == nil && currentUrl != nil && strings.HasPrefix(*currentUrl, "/uploads/") {
				os.Remove(filepath.Join(".", *currentUrl))
			}
		}
	}

	// Construir UPDATE dinámico
	sets := []string{}
	args := []interface{}{}
	if realizadoPor != nil {
		sets = append(sets, "realizado_por = ?")
		args = append(args, *realizadoPor)
	}
	if descripcion != nil {
		sets = append(sets, "descripcion = ?")
		args = append(args, *descripcion)
	}
	if costo != nil {
		sets = append(sets, "costo = ?")
		args = append(args, *costo)
	}
	if fechaHora != nil {
		sets = append(sets, "fecha_hora = ?")
		args = append(args, *fechaHora)
	}
	if urlPdfProvided {
		sets = append(sets, "url_pdf = ?")
		args = append(args, urlPdf)
	}
	if len(sets) == 0 {
		http.Error(w, "Nada para actualizar", http.StatusBadRequest)
		return
	}
	args = append(args, id)
	query := fmt.Sprintf(`UPDATE mantenimientos SET %s WHERE id = ?`, strings.Join(sets, ", "))
	if _, err := h.DB.Exec(query, args...); err != nil {
		http.Error(w, "Error al actualizar: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Devolver actualizado
	h.Obtener(w, r)
}
