package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"inventario-backend/internal/models"
)

type AuthHandler struct {
	DB *sql.DB
}

type ctxKey string

const ctxUserKey ctxKey = "auth_user"

type AuthUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Rol      string `json:"rol"`
}

func GetAuthUser(r *http.Request) (*AuthUser, bool) {
	u, ok := r.Context().Value(ctxUserKey).(*AuthUser)
	return u, ok
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,32}$`)

func isValidUsername(u string) bool { return usernameRe.MatchString(u) }

// Middleware verifica Authorization: Bearer <token> o X-Auth-Token
func (h *AuthHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token = strings.TrimSpace(auth[7:])
		} else {
			token = r.Header.Get("X-Auth-Token")
			if token == "" {
				token = r.URL.Query().Get("token")
			}
		}
		if token == "" {
			http.Error(w, `{"error":"no autenticado"}`, http.StatusUnauthorized)
			return
		}
		var u AuthUser
		var activo int
		err := h.DB.QueryRow(`SELECT u.id, u.username, u.rol, u.activo FROM sesiones s JOIN usuarios u ON s.user_id = u.id WHERE s.token = ?`, token).Scan(&u.ID, &u.Username, &u.Rol, &activo)
		if err != nil {
			http.Error(w, `{"error":"token inválido"}`, http.StatusUnauthorized)
			return
		}
		if activo != 1 {
			http.Error(w, `{"error":"usuario deshabilitado"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, &u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := GetAuthUser(r)
			if !ok {
				http.Error(w, `{"error":"no autenticado"}`, http.StatusUnauthorized)
				return
			}
			if !allowed[u.Rol] {
				http.Error(w, `{"error":"rol no autorizado"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// POST /api/auth/register {username,password,rol}
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Rol      string `json:"rol"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if !isValidUsername(req.Username) {
		http.Error(w, "username debe ser 3-32 caracteres alfanuméricos ._-", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "password debe tener al menos 8 caracteres", http.StatusBadRequest)
		return
	}
	if req.Rol == "" {
		req.Rol = "lector"
	}
	if !models.EsRolValido(req.Rol) {
		http.Error(w, "rol no válido: tecnico o lector", http.StatusBadRequest)
		return
	}
	// bootstrap: si no hay usuarios, permitir sin auth
	var cnt int
	h.DB.QueryRow(`SELECT COUNT(*) FROM usuarios`).Scan(&cnt)
	if cnt > 0 {
		// exigir tecnico
		u, ok := GetAuthUser(r)
		// intentar leer token aunque middleware no esté aplicado: manual check
		if !ok {
			// intentar leer token manual si handler fue llamado sin middleware
			token := ""
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				token = strings.TrimSpace(auth[7:])
			} else {
				token = r.Header.Get("X-Auth-Token")
			}
			if token != "" {
				var tmp AuthUser
				var activo int
				err := h.DB.QueryRow(`SELECT u.id, u.username, u.rol, u.activo FROM sesiones s JOIN usuarios u ON s.user_id = u.id WHERE s.token = ?`, token).Scan(&tmp.ID, &tmp.Username, &tmp.Rol, &activo)
				if err == nil && activo == 1 {
					u = &tmp
					ok = true
				}
			}
		}
		if !ok || u.Rol != "tecnico" {
			http.Error(w, `{"error":"solo tecnico puede crear usuarios"}`, http.StatusForbidden)
			return
		}
	}
	// hash
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "error al hashear password", http.StatusInternalServerError)
		return
	}
	id := uuid.NewString()
	creado := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(`INSERT INTO usuarios (id, username, password_hash, rol, creado_en, activo) VALUES (?, ?, ?, ?, ?, 1)`, id, req.Username, string(hash), req.Rol, creado)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "username ya existe", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "username": req.Username, "rol": req.Rol})
}

// POST /api/auth/login {username,password}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	var id, hash, rol string
	var activo int
	err := h.DB.QueryRow(`SELECT id, password_hash, rol, activo FROM usuarios WHERE username = ?`, req.Username).Scan(&id, &hash, &rol, &activo)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"credenciales inválidas"}`, http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if activo != 1 {
		http.Error(w, `{"error":"usuario deshabilitado"}`, http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"credenciales inválidas"}`, http.StatusUnauthorized)
		return
	}
	token, err := generateToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sid := uuid.NewString()
	creado := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(`INSERT INTO sesiones (id, user_id, token, creado_en) VALUES (?, ?, ?, ?)`, sid, id, token, creado)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"usuario": map[string]interface{}{
			"id":       id,
			"username": req.Username,
			"rol":      rol,
		},
	})
}

// POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthUser(r)
	if !ok {
		http.Error(w, `{"error":"no autenticado"}`, http.StatusUnauthorized)
		return
	}
	// token del header
	token := ""
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token = strings.TrimSpace(auth[7:])
	} else {
		token = r.Header.Get("X-Auth-Token")
	}
	if token == "" {
		http.Error(w, "token requerido", http.StatusBadRequest)
		return
	}
	// borrar solo sesión del token (no todas del usuario) — anular sesión específica
	res, err := h.DB.Exec(`DELETE FROM sesiones WHERE token = ? AND user_id = ?`, token, u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "sesión no encontrada", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "sesión anulada"})
}

// GET /api/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthUser(r)
	if !ok {
		http.Error(w, `{"error":"no autenticado"}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"usuario": u})
}

// GET /api/usuarios
func (h *AuthHandler) ListarUsuarios(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, username, rol, creado_en, activo FROM usuarios ORDER BY creado_en DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []models.Usuario
	for rows.Next() {
		var u models.Usuario
		rows.Scan(&u.ID, &u.Username, &u.Rol, &u.CreadoEn, &u.Activo)
		list = append(list, u)
	}
	if list == nil {
		list = []models.Usuario{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"usuarios": list})
}

// DELETE /api/usuarios/{id}
func (h *AuthHandler) EliminarUsuario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, _ := GetAuthUser(r)
	if u != nil && u.ID == id {
		http.Error(w, "no puedes borrar tu propia cuenta", http.StatusBadRequest)
		return
	}
	res, err := h.DB.Exec(`DELETE FROM usuarios WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "usuario no encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "usuario eliminado"})
}
