package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TintaHandler struct {
	DB *sql.DB
}

func (h *TintaHandler) Obtener(w http.ResponseWriter, r *http.Request) {
	dispositivoID := chi.URLParam(r, "id")
	rows, err := h.DB.Query(`SELECT id, dispositivo_id, color, nivel_porcentaje, codigo_tinta, ultima_actualizacion FROM niveles_tinta WHERE dispositivo_id = ? ORDER BY color`, dispositivoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tintas []map[string]interface{}
	for rows.Next() {
		var id, dispID, color, fecha string
		var nivel int
		var codigo *int
		rows.Scan(&id, &dispID, &color, &nivel, &codigo, &fecha)
		tintas = append(tintas, map[string]interface{}{
			"id": id, "dispositivo_id": dispID, "color": color, "nivel_porcentaje": nivel, "codigo_tinta": codigo, "ultima_actualizacion": fecha,
		})
	}
	if tintas == nil {
		tintas = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tintas": tintas})
}

func (h *TintaHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	dispositivoID := chi.URLParam(r, "id")

	var req struct {
		Tintas []struct {
			Color           string `json:"color"`
			NivelPorcentaje int    `json:"nivel_porcentaje"`
			CodigoTinta     *int   `json:"codigo_tinta"`
		} `json:"tintas"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Tintas) == 0 {
		http.Error(w, "tintas no puede estar vacío", http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)
	for _, t := range req.Tintas {
		if t.NivelPorcentaje < 0 || t.NivelPorcentaje > 100 {
			http.Error(w, "nivel_porcentaje debe estar entre 0 y 100", http.StatusBadRequest)
			return
		}
		if t.CodigoTinta != nil && *t.CodigoTinta < 0 {
			http.Error(w, "codigo_tinta debe ser numérico positivo", http.StatusBadRequest)
			return
		}
		// Upsert: si existe actualiza, si no inserta. Incluye codigo_tinta
		_, err := h.DB.Exec(`
			INSERT INTO niveles_tinta (id, dispositivo_id, color, nivel_porcentaje, codigo_tinta, ultima_actualizacion)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(dispositivo_id, color) DO UPDATE SET nivel_porcentaje=excluded.nivel_porcentaje, codigo_tinta=excluded.codigo_tinta, ultima_actualizacion=excluded.ultima_actualizacion`,
			uuid.New().String(), dispositivoID, t.Color, t.NivelPorcentaje, t.CodigoTinta, now)
		if err != nil {
			http.Error(w, "Error al actualizar tinta: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Retornar estado actualizado
	h.Obtener(w, r)
}
