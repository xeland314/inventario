package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	dbpkg "inventario-backend/internal/db"
)

func newDashboardRouter(db *sql.DB) chi.Router {
	r := chi.NewRouter()
	dh := &DispositivoHandler{DB: db}
	ah := &AuthHandler{DB: db}
	mh := &MantenimientoHandler{DB: db}
	ph := &PlanHandler{DB: db}
	dash := &DashboardHandler{DB: db}
	th := &TintaHandler{DB: db}
	r.Post("/api/auth/register", ah.Register)
	r.Post("/api/auth/login", ah.Login)
	r.Group(func(r chi.Router) {
		r.Use(ah.Middleware)
		r.Get("/api/dashboard", dash.Resumen)
		r.Get("/api/estadisticas", dash.Resumen)
		r.Get("/api/dispositivos", dh.Listar)
		r.Get("/api/planes", ph.Listar)
		r.Get("/api/planes/proximos", ph.Proximos)
		r.Get("/api/planes/vencidos", ph.Vencidos)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("tecnico"))
			r.Post("/api/dispositivos", dh.Crear)
			r.Post("/api/planes", ph.Crear)
			r.Post("/api/mantenimientos", mh.Crear)
			r.Put("/api/dispositivos/{id}/tintas", th.Actualizar)
		})
	})
	return r
}

func TestDashboardResumen(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newDashboardRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// crear 2 sede Matriz/Icon con valor 100/200, 1 averiado, 1 activo
	req1 := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "DASH-1", "numero_serie": "SN-D1", "tipo": "Laptop", "marca": "Dell", "modelo": "XPS", "estado": "activo", "sede": "Matriz", "valor_compra": 100, "ubicacion": "A1", "proveedor": "P", "fecha_compra": "2023-01-01"})
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != 201 {
		t.Fatalf("create D1 %d %s", w1.Code, w1.Body.String())
	}
	req2 := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "DASH-2", "numero_serie": "SN-D2", "tipo": "UPS", "marca": "APC", "modelo": "M1", "estado": "averiado", "sede": "Icon", "valor_compra": 200, "ubicacion": "B1", "proveedor": "P", "fecha_compra": "2022-01-01"})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	// dashboard
	reqD := authReq("GET", "/api/dashboard", tok, nil)
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	if wD.Code != 200 {
		t.Fatalf("dashboard %d %s", wD.Code, wD.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resp)
	conteoEstado := resp["conteo_estado"].(map[string]interface{})
	if int(conteoEstado["activo"].(float64)) != 1 || int(conteoEstado["averiado"].(float64)) != 1 {
		t.Fatalf("conteo_estado mismatch %v", conteoEstado)
	}
	conteoSede := resp["conteo_sede"].(map[string]interface{})
	if int(conteoSede["Matriz"].(float64)) != 1 || int(conteoSede["Icon"].(float64)) != 1 {
		t.Fatalf("conteo_sede %v", conteoSede)
	}
	if resp["valor_total"].(float64) != 300 {
		t.Fatalf("valor_total expected 300 got %v", resp["valor_total"])
	}
	valorSede := resp["valor_por_sede"].(map[string]interface{})
	if valorSede["Matriz"].(float64) != 100 || valorSede["Icon"].(float64) != 200 {
		t.Fatalf("valor_por_sede %v", valorSede)
	}
}

func TestDashboardCostoPorMes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newDashboardRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// crear dispositivo
	reqD := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "COSTO-1", "numero_serie": "SN-C1", "tipo": "Desktop", "marca": "HP", "modelo": "M1"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resD map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resD)
	did := resD["id"].(string)
	// mantenimientos dos meses
	for _, d := range []struct {
		fecha string
		costo float64
	}{
		{"2026-01-15 10:00:00", 100},
		{"2026-02-10 10:00:00", 200},
	} {
		body := map[string]interface{}{"dispositivo_id": did, "realizado_por": "tec", "descripcion": "mant", "costo": d.costo, "fecha_hora": d.fecha}
		req := authReq("POST", "/api/mantenimientos", tok, body)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("mant %d %s", w.Code, w.Body.String())
		}
	}
	reqDash := authReq("GET", "/api/dashboard", tok, nil)
	wDash := httptest.NewRecorder()
	r.ServeHTTP(wDash, reqDash)
	var resp map[string]interface{}
	json.Unmarshal(wDash.Body.Bytes(), &resp)
	costoMes := resp["costo_por_mes"].([]interface{})
	if len(costoMes) < 2 {
		t.Fatalf("costo_por_mes expected 2 got %d %v", len(costoMes), costoMes)
	}
	// verificar meses
	foundJan := false
	foundFeb := false
	for _, c := range costoMes {
		m := c.(map[string]interface{})
		if m["mes"] == "2026-01" && m["costo"].(float64) == 100 {
			foundJan = true
		}
		if m["mes"] == "2026-02" && m["costo"].(float64) == 200 {
			foundFeb = true
		}
	}
	if !foundJan || !foundFeb {
		t.Fatalf("costo por mes mismatch %v", costoMes)
	}
}

func TestDashboardTiempoMedioReparacion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newDashboardRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	reqD := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "MTTR-1", "numero_serie": "SN-MTTR1", "tipo": "Laptop", "marca": "Lenovo", "modelo": "T14", "estado": "activo"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resD map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resD)
	did := resD["id"].(string)
	// activo -> en_reparacion
	body1 := map[string]interface{}{"estado": "en_reparacion"}
	// newDashboardRouter no expone PUT, usamos audit router que sí tiene PUT
	r2 := newAuditRouter(db) // has PUT
	req1b := authReq("PUT", "/api/dispositivos/"+did, tok, body1)
	w1 := httptest.NewRecorder()
	r2.ServeHTTP(w1, req1b)
	if w1.Code != 200 {
		t.Fatalf("to en_reparacion %d %s", w1.Code, w1.Body.String())
	}
	time.Sleep(1100 * time.Millisecond) // ensure 1s diff
	body2 := map[string]interface{}{"estado": "activo"}
	req2 := authReq("PUT", "/api/dispositivos/"+did, tok, body2)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("back to activo %d %s", w2.Code, w2.Body.String())
	}
	// dashboard should have tiempo >0
	reqDsh := authReq("GET", "/api/dashboard", tok, nil)
	wDsh2 := httptest.NewRecorder()
	r.ServeHTTP(wDsh2, reqDsh)
	var resp map[string]interface{}
	json.Unmarshal(wDsh2.Body.Bytes(), &resp)
	tiempo := resp["tiempo_medio_en_reparacion_horas"].(float64)
	if tiempo <= 0 {
		t.Fatalf("tiempo medio should >0 got %v", tiempo)
	}
	if int(resp["tiempo_medio_conteo"].(float64)) != 1 {
		t.Fatalf("conteo 1 expected")
	}
}

func TestDashboardFrecuenciaFallas(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	rAud := newAuditRouter(db)
	rDash := newDashboardRouter(db)
	registerUser(t, rAud, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, rAud, "admin", "Admin123!")
	// crear 3 dispositivos: 2 Dell, 1 HP
	ids := []string{}
	for _, d := range []struct {
		codigo string
		marca  string
	}{
		{"FREQ-D1", "Dell"}, {"FREQ-D2", "Dell"}, {"FREQ-H1", "HP"},
	} {
		req := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": d.codigo, "numero_serie": "SN-" + d.codigo, "tipo": "Laptop", "marca": d.marca, "modelo": "XPS", "estado": "activo"})
		w := httptest.NewRecorder()
		rAud.ServeHTTP(w, req)
		var res map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &res)
		ids = append(ids, res["id"].(string))
	}
	// provocar fallas: D1 y D2 a averiado, H1 a averiado (so Dell 2, HP 1)
	for _, id := range ids {
		req := authReq("PUT", "/api/dispositivos/"+id, tok, map[string]interface{}{"estado": "averiado"})
		w := httptest.NewRecorder()
		rAud.ServeHTTP(w, req)
	}
	reqD := authReq("GET", "/api/dashboard", tok, nil)
	wD := httptest.NewRecorder()
	rDash.ServeHTTP(wD, reqD)
	var resp map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resp)
	freq := resp["frecuencia_por_marca"].([]interface{})
	if len(freq) == 0 {
		t.Fatalf("freq empty")
	}
	// first should be Dell 2
	first := freq[0].(map[string]interface{})
	if first["marca"] != "Dell" || int(first["fallas"].(float64)) != 2 {
		t.Fatalf("freq marca Dell 2 expected got %v", first)
	}
}

func TestDashboardTintas(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newDashboardRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// crear impresora
	reqD := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "IMP-DASH", "numero_serie": "SN-IMP", "tipo": "Impresora", "marca": "HP", "modelo": "M1"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resD map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resD)
	did := resD["id"].(string)
	// poner tintas bajo 10 y 100
	// need to update tintas via PUT /api/dispositivos/{id}/tintas
	body := map[string]interface{}{"tintas": []map[string]interface{}{{"color": "negro", "nivel_porcentaje": 10}, {"color": "cian", "nivel_porcentaje": 100}}}
	reqT := authReq("PUT", "/api/dispositivos/"+did+"/tintas", tok, body)
	wT := httptest.NewRecorder()
	r.ServeHTTP(wT, reqT)
	if wT.Code != 200 {
		t.Fatalf("tintas update %d %s", wT.Code, wT.Body.String())
	}
	// dashboard umbral 20 should list negro 10
	reqDash := authReq("GET", "/api/dashboard?tinta_umbral=20", tok, nil)
	wDash := httptest.NewRecorder()
	r.ServeHTTP(wDash, reqDash)
	var resp map[string]interface{}
	json.Unmarshal(wDash.Body.Bytes(), &resp)
	tintas := resp["tintas_bajo_umbral"].([]interface{})
	if len(tintas) != 1 {
		t.Fatalf("tintas bajo umbral 20 expected 1 got %d %v", len(tintas), tintas)
	}
}

func TestDashboardPlanes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newDashboardRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	vencida := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	proxima := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	for _, d := range []string{vencida, proxima} {
		req := authReq("POST", "/api/planes", tok, map[string]interface{}{"tipo": "UPS", "cada_dias": 30, "proxima_fecha": d, "descripcion": "plan " + d})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("plan %d", w.Code)
		}
	}
	reqD := authReq("GET", "/api/dashboard", tok, nil)
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resp map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resp)
	if int(resp["planes_vencidos"].(float64)) != 1 {
		t.Fatalf("vencidos 1 expected got %v", resp["planes_vencidos"])
	}
	if int(resp["planes_proximos_30"].(float64)) != 1 {
		t.Fatalf("proximos 1 expected got %v", resp["planes_proximos_30"])
	}
}

func TestDashboardRoles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newDashboardRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tokAdmin := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tokAdmin)
	tokLector := loginUser(t, r, "lector1", "Lector123!")
	for _, tok := range []string{tokAdmin, tokLector} {
		req := authReq("GET", "/api/dashboard", tok, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("dashboard GET should 200 for both roles got %d", w.Code)
		}
	}
}
