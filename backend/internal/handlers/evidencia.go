package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"inventario-backend/internal/crypto"
)

type EvidenciaHandler struct {
	DB            *sql.DB
	MasterKey     []byte
	StorageFolder string
}

func (h *EvidenciaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Archivos demasiado grandes (max 32MB): "+err.Error(), http.StatusBadRequest)
		return
	}
	entidadTipo := r.FormValue("entidad_tipo")
	entidadID := r.FormValue("entidad_id")
	if entidadTipo == "" || entidadID == "" {
		http.Error(w, "entidad_tipo y entidad_id son obligatorios", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["fotos"]
	if len(files) == 0 {
		// fallback: buscar cualquier campo que sea file
		for _, v := range r.MultipartForm.File {
			files = append(files, v...)
		}
	}
	if len(files) == 0 {
		http.Error(w, "No se enviaron fotos (campo 'fotos')", http.StatusBadRequest)
		return
	}

	os.MkdirAll(h.StorageFolder, 0700)
	var ids []string

	for _, fileHeader := range files {
		f, err := fileHeader.Open()
		if err != nil {
			continue
		}
		plainBytes, _ := io.ReadAll(f)
		f.Close()

		encryptedBytes, err := crypto.EncryptAESGCM(plainBytes, h.MasterKey)
		if err != nil {
			http.Error(w, "Error al cifrar imagen: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fotoID := uuid.New().String()
		filePath := filepath.Join(h.StorageFolder, fmt.Sprintf("%s.enc", fotoID))
		if err := os.WriteFile(filePath, encryptedBytes, 0600); err != nil {
			http.Error(w, "Error al guardar archivo cifrado: "+err.Error(), http.StatusInternalServerError)
			return
		}

		mimeType := fileHeader.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
		cargadoEn := time.Now().Format(time.RFC3339)
		_, err = h.DB.Exec(`INSERT INTO fotos_evidencia (id, entidad_tipo, entidad_id, ruta_archivo_cifrado, nombre_original, mime_type, tamano_bytes, cargado_en)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fotoID, entidadTipo, entidadID, filePath, fileHeader.Filename, mimeType, len(plainBytes), cargadoEn)
		if err != nil {
			http.Error(w, "Error al guardar metadata: "+err.Error(), http.StatusInternalServerError)
			return
		}
		ids = append(ids, fotoID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "foto_ids": ids})
}

func (h *EvidenciaHandler) ServirRaw(w http.ResponseWriter, r *http.Request) {
	fotoID := chi.URLParam(r, "id")
	var filePath, mimeType string
	err := h.DB.QueryRow(`SELECT ruta_archivo_cifrado, mime_type FROM fotos_evidencia WHERE id = ?`, fotoID).Scan(&filePath, &mimeType)
	if err == sql.ErrNoRows {
		http.Error(w, "Foto no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cipherText, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Error al leer disco: "+err.Error(), http.StatusInternalServerError)
		return
	}
	plainBytes, err := crypto.DecryptAESGCM(cipherText, h.MasterKey)
	if err != nil {
		http.Error(w, "Fallo al descifrar (clave incorrecta o archivo corrupto)", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(plainBytes)))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(plainBytes)
}

func (h *EvidenciaHandler) Galeria(w http.ResponseWriter, r *http.Request) {
	tipo := r.URL.Query().Get("tipo")
	id := r.URL.Query().Get("id")
	if tipo == "" {
		tipo = r.URL.Query().Get("entidad_tipo")
	}
	if id == "" {
		id = r.URL.Query().Get("entidad_id")
	}
	if tipo == "" || id == "" {
		http.Error(w, "Parámetros tipo e id son obligatorios", http.StatusBadRequest)
		return
	}
	rows, err := h.DB.Query(`SELECT id, entidad_tipo, entidad_id, ruta_archivo_cifrado, nombre_original, mime_type, tamano_bytes, cargado_en FROM fotos_evidencia WHERE entidad_tipo = ? AND entidad_id = ? ORDER BY cargado_en DESC`, tipo, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var fotos []map[string]interface{}
	for rows.Next() {
		var fID, eTipo, eID, ruta, nombre, mime, cargadoEn string
		var tam int
		rows.Scan(&fID, &eTipo, &eID, &ruta, &nombre, &mime, &tam, &cargadoEn)
		fotos = append(fotos, map[string]interface{}{
			"id": fID, "entidad_tipo": eTipo, "entidad_id": eID,
			"nombre_original": nombre, "mime_type": mime, "tamano_bytes": tam, "cargado_en": cargadoEn,
		})
	}
	if fotos == nil {
		fotos = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"fotos": fotos})
}

func (h *EvidenciaHandler) Eliminar(w http.ResponseWriter, r *http.Request) {
	fotoID := chi.URLParam(r, "id")
	var filePath string
	err := h.DB.QueryRow(`SELECT ruta_archivo_cifrado FROM fotos_evidencia WHERE id = ?`, fotoID).Scan(&filePath)
	if err == sql.ErrNoRows {
		http.Error(w, "Foto no encontrada", http.StatusNotFound)
		return
	}
	os.Remove(filePath)
	h.DB.Exec(`DELETE FROM fotos_evidencia WHERE id = ?`, fotoID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Foto eliminada"})
}
