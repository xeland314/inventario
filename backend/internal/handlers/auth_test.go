package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newAuthRouter(db *sql.DB) chi.Router {
	r := chi.NewRouter()
	dh := &DispositivoHandler{DB: db}
	mh := &MovimientoHandler{DB: db}
	eh := &EtiquetaHandler{DB: db}
	ah := &AuthHandler{DB: db}
	rh := &ReporteHandler{DB: db}
	th := &TintaHandler{DB: db}
	// públicos
	r.Post("/api/auth/register", ah.Register)
	r.Post("/api/auth/login", ah.Login)
	// protegidos
	r.Group(func(r chi.Router) {
		r.Use(ah.Middleware)
		r.Get("/api/auth/me", ah.Me)
		r.Post("/api/auth/logout", ah.Logout)
		r.Get("/api/usuarios", ah.ListarUsuarios)
		r.Delete("/api/usuarios/{id}", ah.EliminarUsuario)

		r.Get("/api/reportes", rh.Listar)
		r.Get("/api/reportes/{id}", rh.Obtener)
		r.Post("/api/reportes", rh.Crear)
		r.Post("/api/reportes/{id}/cancelar", rh.Cancelar)
		r.Post("/api/reportes/{id}/resolver", rh.Resolver)

		r.Get("/api/dispositivos", dh.Listar)
		r.Get("/api/dispositivos/{id}", dh.Obtener)
		r.Get("/api/dispositivos/{id}/componentes", dh.ListarComponentes)

		r.Group(func(r chi.Router) {
			r.Use(RequireRole("tecnico"))
			r.Post("/api/dispositivos", dh.Crear)
			r.Put("/api/dispositivos/{id}", dh.Actualizar)
			r.Patch("/api/dispositivos/{id}", dh.Actualizar)
			r.Patch("/api/dispositivos/{id}/baja", dh.DarDeBaja)
			r.Delete("/api/dispositivos/{id}", dh.Eliminar)
			r.Post("/api/movimientos", mh.Crear)
			r.Delete("/api/movimientos/{id}", mh.Eliminar)
			r.Post("/api/etiquetas/generar", eh.Generar)
			r.Post("/api/etiquetas/{id}/asociar", eh.Asociar)
			r.Post("/api/etiquetas/{id}/desasociar", eh.Desasociar)
			r.Post("/api/etiquetas/{id}/impreso", eh.MarcarImpreso)
			r.Post("/api/etiquetas/{id}/anular", eh.Anular)
			r.Delete("/api/etiquetas/{id}", eh.Eliminar)
			r.Put("/api/dispositivos/{id}/tintas", th.Actualizar)
		})
		r.Get("/api/movimientos", mh.Listar)
		r.Get("/api/etiquetas", eh.Listar)
		r.Get("/api/etiquetas/{id}", eh.Obtener)
		r.Get("/api/etiquetas/scan/{id}", eh.Escanear)
	})
	return r
}

func registerUser(t *testing.T, r chi.Router, username, password, rol, token string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"username": username, "password": password, "rol": rol})
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("register %s failed %d %s", username, w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	return res["id"].(string)
}

func loginUser(t *testing.T, r chi.Router, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"username": username, "password": password})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login %s failed %d %s", username, w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	return res["token"].(string)
}

func authReq(method, path, token string, body interface{}) *http.Request {
	var b *bytes.Reader
	if body != nil {
		j, _ := json.Marshal(body)
		b = bytes.NewReader(j)
	} else {
		b = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, b)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestAuthRegisterLoginLogout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuthRouter(db)
	// bootstrap admin sin token
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// me ok
	req := authReq("GET", "/api/auth/me", tok, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("me failed %d %s", w.Code, w.Body.String())
	}
	// login bad password
	body, _ := json.Marshal(map[string]interface{}{"username": "admin", "password": "wrong"})
	req2 := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 401 {
		t.Fatalf("bad password should 401 got %d", w2.Code)
	}
	// logout
	req3 := authReq("POST", "/api/auth/logout", tok, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("logout failed %d %s", w3.Code, w3.Body.String())
	}
	// me after logout 401
	req4 := authReq("GET", "/api/auth/me", tok, nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != 401 {
		t.Fatalf("me after logout should 401 got %d", w4.Code)
	}
}

func TestAuthRoles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuthRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tokAdmin := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tokAdmin)
	tokLector := loginUser(t, r, "lector1", "Lector123!")

	// lector GET ok
	req := authReq("GET", "/api/dispositivos", tokLector, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("lector GET dispositivos should 200 got %d", w.Code)
	}
	// lector POST should 403
	req2 := authReq("POST", "/api/dispositivos", tokLector, map[string]interface{}{"codigo_activo": "X1", "numero_serie": "SN-X1", "tipo": "Laptop", "marca": "Dell", "modelo": "M1"})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 403 {
		t.Fatalf("lector POST should 403 got %d %s", w2.Code, w2.Body.String())
	}
	// tecnico POST ok
	req3 := authReq("POST", "/api/dispositivos", tokAdmin, map[string]interface{}{"codigo_activo": "T1", "numero_serie": "SN-T1", "tipo": "Laptop", "marca": "Dell", "modelo": "M1"})
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != 201 {
		t.Fatalf("tecnico POST should 201 got %d %s", w3.Code, w3.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &res)
	dispID := res["id"].(string)

	// lector puede reportar via etiqueta/dispositivo
	// primero tecnico genera etiqueta
	reqE := authReq("POST", "/api/etiquetas/generar", tokAdmin, map[string]interface{}{"cantidad": 1, "version": 7})
	wE := httptest.NewRecorder()
	r.ServeHTTP(wE, reqE)
	if wE.Code != 201 {
		t.Fatalf("generar etiqueta %d %s", wE.Code, wE.Body.String())
	}
	var gen map[string]interface{}
	json.Unmarshal(wE.Body.Bytes(), &gen)
	etID := gen["ids"].([]interface{})[0].(string)
	// asociar
	reqA := authReq("POST", "/api/etiquetas/"+etID+"/asociar", tokAdmin, map[string]interface{}{"dispositivo_id": dispID})
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	if wA.Code != 200 {
		t.Fatalf("asociar %d %s", wA.Code, wA.Body.String())
	}
	// lector crea reporte con etiqueta_id
	reqR := authReq("POST", "/api/reportes", tokLector, map[string]interface{}{"etiqueta_id": etID, "descripcion": "Pantalla parpadea al mover tapa"})
	wR := httptest.NewRecorder()
	r.ServeHTTP(wR, reqR)
	if wR.Code != 201 {
		t.Fatalf("lector reporte should 201 got %d %s", wR.Code, wR.Body.String())
	}
	var rep map[string]interface{}
	json.Unmarshal(wR.Body.Bytes(), &rep)
	repID := rep["id"].(string)
	// lector cancela propio ok
	reqC := authReq("POST", "/api/reportes/"+repID+"/cancelar", tokLector, nil)
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)
	if wC.Code != 200 {
		t.Fatalf("cancel propio should 200 got %d %s", wC.Code, wC.Body.String())
	}
	// crear otro reporte para probar que lector no puede resolver
	reqR2 := authReq("POST", "/api/reportes", tokLector, map[string]interface{}{"dispositivo_id": dispID, "descripcion": "Teclado no funciona"})
	wR2 := httptest.NewRecorder()
	r.ServeHTTP(wR2, reqR2)
	var rep2 map[string]interface{}
	json.Unmarshal(wR2.Body.Bytes(), &rep2)
	repID2 := rep2["id"].(string)
	reqRes := authReq("POST", "/api/reportes/"+repID2+"/resolver", tokLector, nil)
	wRes := httptest.NewRecorder()
	r.ServeHTTP(wRes, reqRes)
	if wRes.Code != 403 {
		t.Fatalf("lector resolver should 403 got %d", wRes.Code)
	}
	// tecnico resuelve ok
	reqRes2 := authReq("POST", "/api/reportes/"+repID2+"/resolver", tokAdmin, nil)
	wRes2 := httptest.NewRecorder()
	r.ServeHTTP(wRes2, reqRes2)
	if wRes2.Code != 200 {
		t.Fatalf("tecnico resolver should 200 got %d %s", wRes2.Code, wRes2.Body.String())
	}
}

func TestBootstrapSoloPrimerUsuario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuthRouter(db)
	// primer registro sin token ok
	registerUser(t, r, "primero", "Password1!", "tecnico", "")
	// segundo sin token debe 403
	body, _ := json.Marshal(map[string]interface{}{"username": "segundo", "password": "Password1!", "rol": "lector"})
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("segundo registro sin token should 403 got %d %s", w.Code, w.Body.String())
	}
}

func TestSinToken401(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuthRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	// GET sin token
	req := httptest.NewRequest("GET", "/api/dispositivos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("sin token should 401 got %d", w.Code)
	}
}

func TestTecnicoPuedeTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuthRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// crear y borrar
	req := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "DEL-1", "numero_serie": "SN-DEL", "tipo": "Desktop", "marca": "HP", "modelo": "M1"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("create 201 got %d", w.Code)
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	id := res["id"].(string)
	reqD := authReq("DELETE", "/api/dispositivos/"+id, tok, nil)
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	if wD.Code != 200 {
		t.Fatalf("delete should 200 got %d %s", wD.Code, wD.Body.String())
	}
}

func TestLectorNoPuedeCrearUsuario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuthRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tokAdmin := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tokAdmin)
	tokLector := loginUser(t, r, "lector1", "Lector123!")
	body, _ := json.Marshal(map[string]interface{}{"username": "otro", "password": "Password1!", "rol": "lector"})
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokLector)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("lector create user should 403 got %d", w.Code)
	}
}

func TestReportesPermisosCancelAjena(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuthRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tokAdmin := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tokAdmin)
	registerUser(t, r, "lector2", "Lector123!", "lector", tokAdmin)
	tok1 := loginUser(t, r, "lector1", "Lector123!")
	tok2 := loginUser(t, r, "lector2", "Lector123!")
	// tecnico crea dispositivo para reporte
	reqD := authReq("POST", "/api/dispositivos", tokAdmin, map[string]interface{}{"codigo_activo": "REP-1", "numero_serie": "SN-REP", "tipo": "Monitor", "marca": "LG", "modelo": "M1"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var res map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &res)
	did := res["id"].(string)
	// lector1 reporta
	reqR := authReq("POST", "/api/reportes", tok1, map[string]interface{}{"dispositivo_id": did, "descripcion": "No enciende"})
	wR := httptest.NewRecorder()
	r.ServeHTTP(wR, reqR)
	var rep map[string]interface{}
	json.Unmarshal(wR.Body.Bytes(), &rep)
	repID := rep["id"].(string)
	// lector2 intenta cancelar ajeno 403
	reqC := authReq("POST", "/api/reportes/"+repID+"/cancelar", tok2, nil)
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)
	if wC.Code != 403 {
		t.Fatalf("cancel ajeno should 403 got %d %s", wC.Code, wC.Body.String())
	}
	// tecnico cancela ajeno ok
	reqC2 := authReq("POST", "/api/reportes/"+repID+"/cancelar", tokAdmin, nil)
	wC2 := httptest.NewRecorder()
	r.ServeHTTP(wC2, reqC2)
	if wC2.Code != 200 {
		t.Fatalf("tecnico cancel ajeno should 200 got %d %s", wC2.Code, wC2.Body.String())
	}
}
