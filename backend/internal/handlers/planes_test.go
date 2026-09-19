package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	dbpkg "inventario-backend/internal/db"
)

func newPlanesRouter(db *sql.DB) chi.Router {
	r := chi.NewRouter()
	dh := &DispositivoHandler{DB: db}
	ah := &AuthHandler{DB: db}
	mh := &MantenimientoHandler{DB: db}
	ph := &PlanHandler{DB: db}
	aud := &AuditoriaHandler{DB: db}
	r.Post("/api/auth/register", ah.Register)
	r.Post("/api/auth/login", ah.Login)
	r.Group(func(r chi.Router) {
		r.Use(ah.Middleware)
		r.Get("/api/auditoria", aud.Listar)
		r.Get("/api/dispositivos", dh.Listar)
		r.Get("/api/dispositivos/{id}", dh.Obtener)
		// planes lectura
		r.Get("/api/planes", ph.Listar)
		r.Get("/api/planes/proximos", ph.Proximos)
		r.Get("/api/planes/vencidos", ph.Vencidos)
		r.Get("/api/planes/calendario.ics", ph.CalendarioICS)
		r.Get("/api/planes/{id}", ph.Obtener)
		r.Get("/api/mantenimientos", mh.Listar)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("tecnico"))
			r.Post("/api/dispositivos", dh.Crear)
			r.Post("/api/planes", ph.Crear)
			r.Put("/api/planes/{id}", ph.Actualizar)
			r.Patch("/api/planes/{id}", ph.Actualizar)
			r.Delete("/api/planes/{id}", ph.Eliminar)
			r.Post("/api/mantenimientos", mh.Crear)
		})
	})
	return r
}

func TestPlanesCrearPorTipoYDispositivo(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newPlanesRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// crear dispositivo para plan específico
	reqD := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "PLAN-D1", "numero_serie": "SN-PLAN-D1", "tipo": "Desktop", "marca": "HP", "modelo": "M1"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resD map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resD)
	did := resD["id"].(string)
	// plan por tipo UPS
	req1 := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "UPS", "cada_dias": 180, "proxima_fecha": time.Now().AddDate(0, 0, 10).Format("2006-01-02"), "descripcion": "Mantenimiento UPS"})
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != 201 {
		t.Fatalf("plan tipo %d %s", w1.Code, w1.Body.String())
	}
	// plan por dispositivo
	req2 := authReq("POST", "/api/planes", tok, map[string]interface{}{"dispositivo_id": did, "cada_dias": 90, "proxima_fecha": time.Now().AddDate(0, 0, 5).Format("2006-01-02"), "descripcion": "Pasta térmica Desktop"})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 201 {
		t.Fatalf("plan dispositivo %d %s", w2.Code, w2.Body.String())
	}
	// listar filtra por tipo
	reqL := authReq("GET", "/api/planes?tipo=UPS", tok, nil)
	wL := httptest.NewRecorder()
	r.ServeHTTP(wL, reqL)
	var list map[string]interface{}
	json.Unmarshal(wL.Body.Bytes(), &list)
	if len(list["planes"].([]interface{})) != 1 {
		t.Fatalf("filter tipo UPS failed %v", wL.Body.String())
	}
	// filtro dispositivo_id
	reqL2 := authReq("GET", "/api/planes?dispositivo_id="+did, tok, nil)
	wL2 := httptest.NewRecorder()
	r.ServeHTTP(wL2, reqL2)
	var list2 map[string]interface{}
	json.Unmarshal(wL2.Body.Bytes(), &list2)
	if len(list2["planes"].([]interface{})) != 1 {
		t.Fatalf("filter dispositivo failed")
	}
}

func TestPlanesValidacion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newPlanesRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// cada_dias 0
	req := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "UPS", "cada_dias": 0, "proxima_fecha": "2026-01-01", "descripcion": "bad"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("cada_dias 0 should 400 got %d", w.Code)
	}
	// tipo inválido
	req2 := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "TipoInvalido", "cada_dias": 30, "proxima_fecha": "2026-01-01", "descripcion": "bad"})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 400 {
		t.Fatalf("tipo invalid should 400 got %d", w2.Code)
	}
	// dispositivo inexistente
	req3 := authReq("POST", "/api/planes", tok, map[string]interface{}{"dispositivo_id": "no-existe", "cada_dias": 30, "proxima_fecha": "2026-01-01", "descripcion": "x"})
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != 404 {
		t.Fatalf("dispositivo inexistente should 404 got %d", w3.Code)
	}
}

func TestPlanesProximosYVencidos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newPlanesRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	hoy := time.Now().Format("2006-01-02")
	vencida := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	proxima5 := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	proxima40 := time.Now().AddDate(0, 0, 40).Format("2006-01-02")
	// vencida
	reqV := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "UPS", "cada_dias": 30, "proxima_fecha": vencida, "descripcion": "vencida"})
	wV := httptest.NewRecorder()
	r.ServeHTTP(wV, reqV)
	if wV.Code != 201 {
		t.Fatalf("vencida create %d %s", wV.Code, wV.Body.String())
	}
	// proxima 5 días
	reqP := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "Impresora", "cada_dias": 90, "proxima_fecha": proxima5, "descripcion": "proxima5"})
	wP := httptest.NewRecorder()
	r.ServeHTTP(wP, reqP)
	// proxima 40 días (fuera de 30)
	reqF := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "Router", "cada_dias": 60, "proxima_fecha": proxima40, "descripcion": "futura40"})
	wF := httptest.NewRecorder()
	r.ServeHTTP(wF, reqF)

	// proximos dias=30 incluye vencida + proxima5, no futura40 si incluye_vencidos=false? con incluye_vencidos=true incluye ambos. Test both
	reqProx := authReq("GET", "/api/planes/proximos?dias=30&incluye_vencidos=true", tok, nil)
	wProx := httptest.NewRecorder()
	r.ServeHTTP(wProx, reqProx)
	var list map[string]interface{}
	json.Unmarshal(wProx.Body.Bytes(), &list)
	if len(list["planes"].([]interface{})) != 2 {
		t.Fatalf("proximos con vencidos expected 2 got %d hoy %s vencida %s proxima5 %s", len(list["planes"].([]interface{})), hoy, vencida, proxima5)
	}
	// proximos sin vencidos (solo próximos)
	reqProx2 := authReq("GET", "/api/planes/proximos?dias=30", tok, nil)
	wProx2 := httptest.NewRecorder()
	r.ServeHTTP(wProx2, reqProx2)
	var list2 map[string]interface{}
	json.Unmarshal(wProx2.Body.Bytes(), &list2)
	if len(list2["planes"].([]interface{})) != 1 {
		t.Fatalf("proximos sin vencidos expected 1 got %d", len(list2["planes"].([]interface{})))
	}
	// vencidos solo vencida
	reqVen := authReq("GET", "/api/planes/vencidos", tok, nil)
	wVen := httptest.NewRecorder()
	r.ServeHTTP(wVen, reqVen)
	var listV map[string]interface{}
	json.Unmarshal(wVen.Body.Bytes(), &listV)
	if len(listV["planes"].([]interface{})) != 1 {
		t.Fatalf("vencidos expected 1 got %d", len(listV["planes"].([]interface{})))
	}
}

func TestMantenimientoRecalculaProxima(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newPlanesRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// crear dispositivo tipo UPS
	reqD := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "UPS-001", "numero_serie": "SN-UPS001", "tipo": "UPS", "marca": "APC", "modelo": "M1"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resD map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resD)
	did := resD["id"].(string)
	// plan para UPS cada 90 dias, proxima 2026-01-01
	reqP := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "UPS", "cada_dias": 90, "proxima_fecha": "2026-01-01", "descripcion": "Batería UPS"})
	wP := httptest.NewRecorder()
	r.ServeHTTP(wP, reqP)
	var resP map[string]interface{}
	json.Unmarshal(wP.Body.Bytes(), &resP)
	pid := resP["id"].(string)
	// registrar mantenimiento real el 2026-01-10
	bodyM, _ := json.Marshal(map[string]interface{}{"dispositivo_id": did, "realizado_por": "tecnico1", "descripcion": "Cambio batería", "fecha_hora": "2026-01-10"})
	reqM := httptest.NewRequest("POST", "/api/mantenimientos", bytes.NewReader(bodyM))
	reqM.Header.Set("Content-Type", "application/json")
	reqM.Header.Set("Authorization", "Bearer "+tok)
	wM := httptest.NewRecorder()
	r.ServeHTTP(wM, reqM)
	if wM.Code != 201 {
		t.Fatalf("mantenimiento %d %s", wM.Code, wM.Body.String())
	}
	// verificar plan ahora proxima 2026-04-10 (real+90)
	reqG := authReq("GET", "/api/planes/"+pid, tok, nil)
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	var plan map[string]interface{}
	json.Unmarshal(wG.Body.Bytes(), &plan)
	if plan["proxima_fecha"] != "2026-04-10" {
		t.Fatalf("proxima recalculada expected 2026-04-10 got %v", plan["proxima_fecha"])
	}
	// también auditar plan
	reqA := authReq("GET", "/api/auditoria?entidad=plan&entidad_id="+pid+"&campo=proxima_fecha", tok, nil)
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	var listA map[string]interface{}
	json.Unmarshal(wA.Body.Bytes(), &listA)
	if int(listA["total"].(float64)) == 0 {
		t.Fatalf("auditoria proxima_fecha missing")
	}
}

func TestICSFeed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newPlanesRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	reqP := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "Impresora", "cada_dias": 90, "proxima_fecha": time.Now().AddDate(0, 0, 7).Format("2006-01-02"), "descripcion": "Limpieza cabezal"})
	wP := httptest.NewRecorder()
	r.ServeHTTP(wP, reqP)
	reqICS := authReq("GET", "/api/planes/calendario.ics", tok, nil)
	wICS := httptest.NewRecorder()
	r.ServeHTTP(wICS, reqICS)
	if wICS.Code != 200 {
		t.Fatalf("ics %d", wICS.Code)
	}
	if ct := wICS.Header().Get("Content-Type"); !strings.Contains(ct, "text/calendar") {
		t.Fatalf("content type ical %s", ct)
	}
	body := wICS.Body.String()
	if !strings.Contains(body, "BEGIN:VCALENDAR") || !strings.Contains(body, "SUMMARY") {
		t.Fatalf("ics body missing VCALENDAR/SUMMARY %s", body)
	}
}

func TestPlanesRoles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newPlanesRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tokAdmin := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tokAdmin)
	tokLector := loginUser(t, r, "lector1", "Lector123!")
	// lector GET ok
	req := authReq("GET", "/api/planes", tokLector, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("lector GET planes 200 got %d", w.Code)
	}
	// lector POST 403
	req2 := authReq("POST", "/api/planes", tokLector, map[string]interface{}{"tipo": "UPS", "cada_dias": 30, "proxima_fecha": "2026-02-01", "descripcion": "x"})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 403 {
		t.Fatalf("lector POST should 403 got %d", w2.Code)
	}
}

func TestAuditoriaPlan(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newPlanesRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	req := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "NAS", "cada_dias": 365, "proxima_fecha": "2026-12-01", "descripcion": "Revisión NAS"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	pid := res["id"].(string)
	reqA := authReq("GET", "/api/auditoria?entidad=plan&entidad_id="+pid+"&accion=crear", tok, nil)
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	var list map[string]interface{}
	json.Unmarshal(wA.Body.Bytes(), &list)
	if int(list["total"].(float64)) == 0 {
		t.Fatalf("auditoria crear plan missing")
	}
}
