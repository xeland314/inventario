package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	dbpkg "inventario-backend/internal/db"
)

func newAuditRouter(db *sql.DB) chi.Router {
	r := chi.NewRouter()
	dh := &DispositivoHandler{DB: db}
	mh := &MovimientoHandler{DB: db}
	eh := &EtiquetaHandler{DB: db}
	ah := &AuthHandler{DB: db}
	rh := &ReporteHandler{DB: db}
	aud := &AuditoriaHandler{DB: db}
	// public
	r.Post("/api/auth/register", ah.Register)
	r.Post("/api/auth/login", ah.Login)
	r.Group(func(r chi.Router) {
		r.Use(ah.Middleware)
		r.Get("/api/auth/me", ah.Me)
		r.Post("/api/auth/logout", ah.Logout)
		r.Get("/api/auditoria", aud.Listar)
		r.Get("/api/auditoria/{id}", aud.Obtener)
		r.Get("/api/dispositivos/{id}/historial", func(w http.ResponseWriter, req *http.Request) {
			q := req.URL.Query()
			q.Set("entidad", "dispositivo")
			q.Set("entidad_id", chi.URLParam(req, "id"))
			req.URL.RawQuery = q.Encode()
			aud.Listar(w, req)
		})
		r.Get("/api/reportes", rh.Listar)
		r.Post("/api/reportes", rh.Crear)
		r.Post("/api/reportes/{id}/cancelar", rh.Cancelar)
		r.Get("/api/dispositivos", dh.Listar)
		r.Get("/api/dispositivos/{id}", dh.Obtener)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("tecnico"))
			r.Post("/api/dispositivos", dh.Crear)
			r.Put("/api/dispositivos/{id}", dh.Actualizar)
			r.Patch("/api/dispositivos/{id}/baja", dh.DarDeBaja)
			r.Delete("/api/dispositivos/{id}", dh.Eliminar)
			r.Post("/api/etiquetas/generar", eh.Generar)
			r.Post("/api/etiquetas/{id}/asociar", eh.Asociar)
			r.Post("/api/movimientos", mh.Crear)
		})
		r.Get("/api/etiquetas", eh.Listar)
		r.Get("/api/etiquetas/scan/{id}", eh.Escanear)
		r.Get("/api/movimientos", mh.Listar)
	})
	return r
}

func TestAuditoriaCrearDispositivo(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newAuditRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	req := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "AUD-001", "numero_serie": "SN-AUD1", "tipo": "Laptop", "marca": "Dell", "modelo": "XPS"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	id := res["id"].(string)
	// auditoria crear
	req2 := authReq("GET", "/api/auditoria?entidad=dispositivo&entidad_id="+id, tok, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("auditoria list %d %s", w2.Code, w2.Body.String())
	}
	var list map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &list)
	auds := list["auditoria"].([]interface{})
	if len(auds) == 0 {
		t.Fatalf("expected auditoria crear, got 0")
	}
	found := false
	for _, a := range auds {
		am := a.(map[string]interface{})
		if am["accion"] == "crear" {
			found = true
			if am["username"] != "admin" {
				t.Fatalf("username should be admin got %v", am["username"])
			}
		}
	}
	if !found {
		t.Fatalf("crear accion not found")
	}
}

func TestAuditoriaActualizarEstado(t *testing.T) {
	db := setupTestDB(t)
	// need fresh with auditoria table via setupTestDB already via InitDB (setupTestDB uses dbpkg.InitDB)
	// but setupTestDB creates 9 tables now 10, ok
	// use audit router
	// recreate db via setupTestDB already done, but need audit router: setup with same db
	defer db.Close()
	r := newAuditRouter(db)
	// bootstrap
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// create
	body, _ := json.Marshal(map[string]interface{}{"codigo_activo": "AUD-002", "numero_serie": "SN-AUD2", "tipo": "Desktop", "marca": "HP", "modelo": "M1", "estado": "activo"})
	req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	id := res["id"].(string)
	// update estado activo->averiado
	body2, _ := json.Marshal(map[string]interface{}{"estado": "averiado"})
	req2 := httptest.NewRequest("PUT", "/api/dispositivos/"+id, bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("update %d %s", w2.Code, w2.Body.String())
	}
	// check auditoria campo=estado
	req3 := authReq("GET", "/api/auditoria?entidad=dispositivo&entidad_id="+id+"&campo=estado", tok, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var list map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &list)
	auds := list["auditoria"].([]interface{})
	if len(auds) != 1 {
		t.Fatalf("expected 1 estado audit, got %d %v", len(auds), w3.Body.String())
	}
	am := auds[0].(map[string]interface{})
	if am["valor_anterior"] != "activo" || am["valor_nuevo"] != "averiado" {
		t.Fatalf("valor mismatch %v", am)
	}
	if am["accion"] != "actualizar" {
		t.Fatalf("accion should actualizar")
	}
}

func TestAuditoriaNoLogSinCambio(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuditRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	body, _ := json.Marshal(map[string]interface{}{"codigo_activo": "AUD-003", "numero_serie": "SN-AUD3", "tipo": "Monitor", "marca": "LG", "modelo": "M1", "ubicacion": "Matriz"})
	req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	id := res["id"].(string)
	// get baseline count
	reqA := authReq("GET", "/api/auditoria?entidad=dispositivo&entidad_id="+id, tok, nil)
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	var listA map[string]interface{}
	json.Unmarshal(wA.Body.Bytes(), &listA)
	c0 := int(listA["total"].(float64))
	// update with same ubicacion (no change? actual code still logs if value differs; we send same value -> should not log porque comparamos antes)
	body2, _ := json.Marshal(map[string]interface{}{"ubicacion": "Matriz"})
	req2 := httptest.NewRequest("PUT", "/api/dispositivos/"+id, bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	// after
	reqB := authReq("GET", "/api/auditoria?entidad=dispositivo&entidad_id="+id, tok, nil)
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	var listB map[string]interface{}
	json.Unmarshal(wB.Body.Bytes(), &listB)
	c1 := int(listB["total"].(float64))
	if c1 != c0 {
		t.Fatalf("no cambio should not add auditoria: before %d after %d", c0, c1)
	}
}

func TestAuditoriaFiltros(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuditRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// create 2 dispositivos
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]interface{}{"codigo_activo": "AUD-F" + string(rune('1'+i)), "numero_serie": "SN-AUDF", "tipo": "Laptop", "marca": "Dell", "modelo": "M1"})
		req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
	// filter by accion crear
	req := authReq("GET", "/api/auditoria?accion=crear&limit=10", tok, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("filter %d", w.Code)
	}
	var list map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &list)
	if int(list["total"].(float64)) < 2 {
		t.Fatalf("expected >=2 crear")
	}
}

func TestAuditoriaLectorSoloLectura(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuditRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tokAdmin := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tokAdmin)
	tokLector := loginUser(t, r, "lector1", "Lector123!")
	// lector GET ok
	req := authReq("GET", "/api/auditoria", tokLector, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("lector GET auditoria should 200 got %d", w.Code)
	}
	// lector POST no existe (405) — intentar crear dispositivo should 403 y no generar auditoria como tecnico
}

func TestAuditoriaReportes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuditRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tokAdmin := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tokAdmin)
	tokLector := loginUser(t, r, "lector1", "Lector123!")
	// admin crea dispositivo
	body, _ := json.Marshal(map[string]interface{}{"codigo_activo": "AUD-REP", "numero_serie": "SN-REP", "tipo": "Switch", "marca": "Cisco", "modelo": "M1"})
	req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokAdmin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	did := res["id"].(string)
	// lector reporta
	req2 := authReq("POST", "/api/reportes", tokLector, map[string]interface{}{"dispositivo_id": did, "descripcion": "Puerto 3 no da link"})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 201 {
		t.Fatalf("reporte 201 got %d %s", w2.Code, w2.Body.String())
	}
	var rep map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &rep)
	rid := rep["id"].(string)
	// auditoria reporte
	req3 := authReq("GET", "/api/auditoria?entidad=reporte&entidad_id="+rid, tokAdmin, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var list map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &list)
	auds := list["auditoria"].([]interface{})
	if len(auds) == 0 {
		t.Fatalf("expected reporte auditoria")
	}
	found := false
	for _, a := range auds {
		if a.(map[string]interface{})["accion"] == "reportar" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reportar accion not found")
	}
}

func TestAuditoriaBaja(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newAuditRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	body, _ := json.Marshal(map[string]interface{}{"codigo_activo": "AUD-BAJA", "numero_serie": "SN-BAJA", "tipo": "Desktop", "marca": "HP", "modelo": "M1"})
	req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	id := res["id"].(string)
	// baja
	body2, _ := json.Marshal(map[string]interface{}{"motivo_baja": "obsoleto"})
	req2 := httptest.NewRequest("PATCH", "/api/dispositivos/"+id+"/baja", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("baja %d %s", w2.Code, w2.Body.String())
	}
	req3 := authReq("GET", "/api/auditoria?entidad=dispositivo&entidad_id="+id+"&accion=baja", tok, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var list map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &list)
	if int(list["total"].(float64)) == 0 {
		t.Fatalf("baja auditoria missing")
	}
}
