package handlers

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"inventario-backend/internal/crypto"
)

// IAHandler: contador/identificador por imagen.
// Vía navegador: transformers.js en el cliente (cero backend).
// Vía VLM local: proxy a Ollama (nativo) o llama.cpp/LM Studio (OpenAI-compatible).
// La foto viaja cifrada en disco: el backend descifra y envía al VLM, no el cliente.
type IAHandler struct {
	DB            *sql.DB
	MasterKey     []byte
	StorageFolder string
	VLMURL        string // ej: http://localhost:11434
	VLMModel      string // ej: qwen2.5vl:7b
	VLMType       string // "ollama" | "openai"
}

const iaPromptDefault = `Analiza la foto de una caja u organizador de inventario técnico. Identifica los elementos visibles (tipo, modelo, valores si se leen) y estima la cantidad de cada uno. Responde SOLO con JSON válido, sin texto adicional, con esta forma exacta: {"descripcion": "...", "cantidad_estimada": 0, "items": [{"nombre": "...", "cantidad": 0}], "confianza": "alta|media|baja"}. En cantidad_estimada usa el total estimado del elemento principal.`

func (h *IAHandler) resolvePath(stored string) string {
	cleaned := strings.ReplaceAll(stored, "\\", "/")
	return filepath.Join(h.StorageFolder, filepath.Base(cleaned))
}

// Config GET /api/ia/config — el frontend muestra/oculta opciones según esto
func (h *IAHandler) Config(w http.ResponseWriter, r *http.Request) {
	tipo := strings.ToLower(strings.TrimSpace(h.VLMType))
	if tipo != "openai" {
		tipo = "ollama"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"navegador": map[string]interface{}{"disponible": true, "modelo": "owlvit-base-patch32"},
		"vlm": map[string]interface{}{
			"disponible": h.VLMURL != "",
			"tipo":       tipo,
			"modelo":     h.VLMModel,
		},
	})
}

// Identificar POST /api/ia/identificar {foto_evidencia_id, prompt?}
func (h *IAHandler) Identificar(w http.ResponseWriter, r *http.Request) {
	if h.VLMURL == "" {
		http.Error(w, "VLM no configurado: define VLM_URL y VLM_MODEL en .env", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		FotoEvidenciaID string `json:"foto_evidencia_id"`
		Prompt          string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.FotoEvidenciaID) == "" {
		http.Error(w, "foto_evidencia_id requerido (sube la foto primero a /api/evidencia/upload)", http.StatusBadRequest)
		return
	}
	prompt := iaPromptDefault
	if strings.TrimSpace(req.Prompt) != "" {
		prompt = req.Prompt
	}
	// foto: descifrar en servidor (nunca viaja descifrada por el cliente al VLM)
	var filePath, mimeType string
	err := h.DB.QueryRow(`SELECT ruta_archivo_cifrado, mime_type FROM fotos_evidencia WHERE id = ?`, strings.TrimSpace(req.FotoEvidenciaID)).Scan(&filePath, &mimeType)
	if err == sql.ErrNoRows {
		http.Error(w, "Foto no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cipherText, err := os.ReadFile(h.resolvePath(filePath))
	if err != nil {
		http.Error(w, "Error al leer disco: "+err.Error(), http.StatusInternalServerError)
		return
	}
	plainBytes, err := crypto.DecryptAESGCM(cipherText, h.MasterKey)
	if err != nil {
		http.Error(w, "Fallo al descifrar (clave incorrecta o archivo corrupto)", http.StatusForbidden)
		return
	}
	b64 := base64.StdEncoding.EncodeToString(plainBytes)

	tipo := strings.ToLower(strings.TrimSpace(h.VLMType))
	if tipo != "openai" {
		tipo = "ollama"
	}
	client := &http.Client{Timeout: 180 * time.Second}
	var contenido string
	if tipo == "openai" {
		contenido, err = h.llamarOpenAICompatible(client, prompt, b64, mimeType)
	} else {
		contenido, err = h.llamarOllama(client, prompt, b64)
	}
	if err != nil {
		http.Error(w, "Error al consultar VLM local: "+err.Error(), http.StatusBadGateway)
		return
	}
	// extraer JSON del texto (los VLM a veces envuelven en ```json ... ```)
	resultado := extraerJSON(contenido)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"vlm":      true,
		"tipo":     tipo,
		"modelo":   h.VLMModel,
		"foto_id":  strings.TrimSpace(req.FotoEvidenciaID),
		"resultado": resultado,
		"texto":     contenido,
	})
}

// llamarOllama: POST {VLM_URL}/api/chat {model, messages:[{role, content, images:[b64]}], stream:false, format:json}
func (h *IAHandler) llamarOllama(client *http.Client, prompt, b64 string) (string, error) {
	body := map[string]interface{}{
		"model": h.VLMModel,
		"messages": []map[string]interface{}{
			{"role": "user", "content": prompt, "images": []string{b64}},
		},
		"stream": false,
		"format": "json",
	}
	raw, _ := json.Marshal(body)
	resp, err := client.Post(strings.TrimRight(h.VLMURL, "/")+"/api/chat", "application/json", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Message.Content, nil
}

// llamarOpenAICompatible: POST {VLM_URL}/v1/chat/completions (llama.cpp llama-server, LM Studio...)
func (h *IAHandler) llamarOpenAICompatible(client *http.Client, prompt, b64, mimeType string) (string, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	body := map[string]interface{}{
		"model": h.VLMModel,
		"messages": []map[string]interface{}{
			{"role": "user", "content": []map[string]interface{}{
				{"type": "text", "text": prompt},
				{"type": "image_url", "image_url": map[string]string{"url": "data:" + mimeType + ";base64," + b64}},
			}},
		},
		"stream": false,
	}
	raw, _ := json.Marshal(body)
	resp, err := client.Post(strings.TrimRight(h.VLMURL, "/")+"/v1/chat/completions", "application/json", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vlm %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("respuesta sin choices")
	}
	return out.Choices[0].Message.Content, nil
}

// extraerJSON saca el primer objeto JSON balanceado de un texto
func extraerJSON(s string) map[string]interface{} {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "```json", "")
	s = strings.ReplaceAll(s, "```", "")
	start := strings.Index(s, "{")
	if start < 0 {
		return nil
	}
	depth := 0
	for i := start; i < len(s) && i-start < 5_000_000; i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				candidate := s[start : i+1]
				var m map[string]interface{}
				if err := json.Unmarshal([]byte(candidate), &m); err == nil {
					return m
				}
				// seguir buscando si el candidato no parsea
				start = strings.Index(s[i+1:], "{")
				if start < 0 {
					return nil
				}
				start = start + i + 1
				i = start - 1
				depth = 0
			}
		}
	}
	return nil
}
