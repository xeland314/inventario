package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	dbpkg "inventario-backend/internal/db"
)

func newTomaRouter(db *sql.DB) chi.Router {
	r := chi.NewRouter()
	dh := &DispositivoHandler{DB: db}
	ah := &AuthHandler{DB: db}
	eh := &EtiquetaHandler{DB: db}
	th := &TomaHandler{DB: db}
	r.Post("/api/auth/register", ah.Register)
	r.Post("/api/auth/login", ah.Login)
	r.Group(func(r chi.Router) {
		r.Use(ah.Middleware)
		r.Get("/api/tomas", th.Listar)
		r.Get("/api/tomas/{id}", th.Obtener)
		r.Post("/api/tomas/{id}/escanear", th.Escanear)
		r.Get("/api/dispositivos", dh.Listar)
		r.Get("/api/etiquetas/{id}", eh.Obtener)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("tecnico"))
			r.Post("/api/dispositivos", dh.Crear)
			r.Post("/api/tomas", th.Crear)
			r.Post("/api/tomas/{id}/cerrar", th.Cerrar)
			r.Delete("/api/tomas/{id}", th.Eliminar)
			r.Post("/api/etiquetas/generar", eh.Generar)
			r.Post("/api/etiquetas/{id}/asociar", eh.Asociar)
		})
	})
	return r
}

func TestTomaCrearListar(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newTomaRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tok)
	tokLector := loginUser(t, r, "lector1", "Lector123!")
	// tecnico crea toma Icon
	req := authReq("POST", "/api/tomas", tok, map[string]interface{}{"sede": "Icon", "descripcion": "Inventario Icon 2026-09"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("crear toma %d %s", w.Code, w.Body.String())
	}
	// lector lista
	req2 := authReq("GET", "/api/tomas", tokLector, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("listar %d", w2.Code)
	}
	var list map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &list)
	if len(list["tomas"].([]interface{})) != 1 {
		t.Fatalf("expected 1 toma")
	}
}

func TestTomaEscanearYComparar(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newTomaRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// crear 3 Icon (A,B,C) y 1 Matriz (X)
	devIDs := map[string]string{}
	for _, d := range []struct{ codigo, sede string }{
		{"TOMA-A", "Icon"}, {"TOMA-B", "Icon"}, {"TOMA-C", "Icon"}, {"TOMA-X", "Matriz"},
	} {
		req := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": d.codigo, "numero_serie": "SN-" + d.codigo, "tipo": "Laptop", "marca": "Dell", "modelo": "M1", "sede": d.sede})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var res map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &res)
		devIDs[d.codigo] = res["id"].(string)
	}
	// crear etiquetas y asociar
	etIDs := map[string]string{}
	for code, did := range devIDs {
		reqG := authReq("POST", "/api/etiquetas/generar", tok, map[string]interface{}{"cantidad": 1, "version": 7})
		wG := httptest.NewRecorder()
		r.ServeHTTP(wG, reqG)
		var gen map[string]interface{}
		json.Unmarshal(wG.Body.Bytes(), &gen)
		eid := gen["ids"].([]interface{})[0].(string)
		etIDs[code] = eid
		reqA := authReq("POST", "/api/etiquetas/"+eid+"/asociar", tok, map[string]interface{}{"dispositivo_id": did})
		wA := httptest.NewRecorder()
		r.ServeHTTP(wA, reqA)
	}
	// crear toma Icon
	reqT := authReq("POST", "/api/tomas", tok, map[string]interface{}{"sede": "Icon"})
	wT := httptest.NewRecorder()
	r.ServeHTTP(wT, reqT)
	var resT map[string]interface{}
	json.Unmarshal(wT.Body.Bytes(), &resT)
	tomaID := resT["id"].(string)
	// escanear A,B (correctos) + X (sobrante otra sede)
	for _, code := range []string{"TOMA-A", "TOMA-B", "TOMA-X"} {
		reqS := authReq("POST", "/api/tomas/"+tomaID+"/escanear", tok, map[string]interface{}{"etiqueta_id": etIDs[code]})
		wS := httptest.NewRecorder()
		r.ServeHTTP(wS, reqS)
		if wS.Code != 201 {
			t.Fatalf("escanear %s %d %s", code, wS.Code, wS.Body.String())
		}
	}
	// GET toma
	reqG := authReq("GET", "/api/tomas/"+tomaID, tok, nil)
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	var detail map[string]interface{}
	json.Unmarshal(wG.Body.Bytes(), &detail)
	if int(detail["faltantes_count"].(float64)) != 1 {
		t.Fatalf("faltantes 1 expected got %v detail %v", detail["faltantes_count"], detail)
	}
	if int(detail["correctos_count"].(float64)) != 2 {
		t.Fatalf("correctos 2 expected got %v", detail["correctos_count"])
	}
	// sobrantes debe ser 1 (X de Matriz)
	if int(detail["sobrantes_count"].(float64)) != 1 {
		t.Fatalf("sobrantes 1 expected got %v", detail["sobrantes_count"])
	}
	if int(detail["esperados_count"].(float64)) != 3 {
		t.Fatalf("esperados 3 expected got %v", detail["esperados_count"])
	}
}

func TestTomaCodigoBarras(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newTomaRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// dispositivo Icon
	reqD := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "CB-1", "numero_serie": "SN-CB1", "tipo": "Monitor", "marca": "LG", "modelo": "M1", "sede": "Icon"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resD map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resD)
	did := resD["id"].(string)
	// etiqueta
	reqG := authReq("POST", "/api/etiquetas/generar", tok, map[string]interface{}{"cantidad": 1, "version": 7})
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	var gen map[string]interface{}
	json.Unmarshal(wG.Body.Bytes(), &gen)
	eid := gen["ids"].([]interface{})[0].(string)
	// obtener codigo_barras
	reqE := authReq("GET", "/api/etiquetas/"+eid, tok, nil)
	wE := httptest.NewRecorder()
	r.ServeHTTP(wE, reqE)
	var etDetail map[string]interface{}
	json.Unmarshal(wE.Body.Bytes(), &etDetail)
	et := etDetail["etiqueta"].(map[string]interface{})
	cb := et["codigo_barras"].(string)
	// asociar
	reqA := authReq("POST", "/api/etiquetas/"+eid+"/asociar", tok, map[string]interface{}{"dispositivo_id": did})
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	// toma
	reqT := authReq("POST", "/api/tomas", tok, map[string]interface{}{"sede": "Icon"})
	wT := httptest.NewRecorder()
	r.ServeHTTP(wT, reqT)
	var resT map[string]interface{}
	json.Unmarshal(wT.Body.Bytes(), &resT)
	tomaID := resT["id"].(string)
	// escanear via codigo_barras
	reqS := authReq("POST", "/api/tomas/"+tomaID+"/escanear", tok, map[string]interface{}{"codigo_barras": cb})
	wS := httptest.NewRecorder()
	r.ServeHTTP(wS, reqS)
	if wS.Code != 201 {
		t.Fatalf("codigo_barras escanear %d %s", wS.Code, wS.Body.String())
	}
	// escanear via dispositivo_id directo (otro dispositivo)
	reqD2 := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "CB-2", "numero_serie": "SN-CB2", "tipo": "Mouse", "marca": "Logi", "modelo": "M1", "sede": "Icon"})
	wD2 := httptest.NewRecorder()
	r.ServeHTTP(wD2, reqD2)
	var resD2 map[string]interface{}
	json.Unmarshal(wD2.Body.Bytes(), &resD2)
	did2 := resD2["id"].(string)
	reqS2 := authReq("POST", "/api/tomas/"+tomaID+"/escanear", tok, map[string]interface{}{"dispositivo_id": did2})
	wS2 := httptest.NewRecorder()
	r.ServeHTTP(wS2, reqS2)
	if wS2.Code != 201 {
		t.Fatalf("dispositivo_id escanear %d %s", wS2.Code, wS2.Body.String())
	}
	// verificar escaneados 2
	reqG2 := authReq("GET", "/api/tomas/"+tomaID, tok, nil)
	wG2 := httptest.NewRecorder()
	r.ServeHTTP(wG2, reqG2)
	var detail map[string]interface{}
	json.Unmarshal(wG2.Body.Bytes(), &detail)
	if int(detail["escaneados_count"].(float64)) != 2 {
		t.Fatalf("escaneados 2 expected got %v", detail["escaneados_count"])
	}
}

func TestTomaCerrarYCandidatosExtraviado(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newTomaRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// dispositivo Icon
	reqD := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "CLOSE-1", "numero_serie": "SN-C1", "tipo": "Laptop", "marca": "Dell", "modelo": "M1", "sede": "Icon"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	// toma
	reqT := authReq("POST", "/api/tomas", tok, map[string]interface{}{"sede": "Icon"})
	wT := httptest.NewRecorder()
	r.ServeHTTP(wT, reqT)
	var resT map[string]interface{}
	json.Unmarshal(wT.Body.Bytes(), &resT)
	tomaID := resT["id"].(string)
	// cerrar
	reqC := authReq("POST", "/api/tomas/"+tomaID+"/cerrar", tok, nil)
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)
	if wC.Code != 200 {
		t.Fatalf("cerrar %d %s", wC.Code, wC.Body.String())
	}
	// intentar escanear tras cerrar 400
	reqS := authReq("POST", "/api/tomas/"+tomaID+"/escanear", tok, map[string]interface{}{"dispositivo_id": "nope"})
	wS := httptest.NewRecorder()
	r.ServeHTTP(wS, reqS)
	if wS.Code != 400 {
		t.Fatalf("escanear cerrado should 400 got %d", wS.Code)
	}
	// verificar estado cerrado
	reqG := authReq("GET", "/api/tomas/"+tomaID, tok, nil)
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	var detail map[string]interface{}
	json.Unmarshal(wG.Body.Bytes(), &detail)
	toma := detail["toma"].(map[string]interface{})
	if toma["estado"] != "cerrado" {
		t.Fatalf("estado cerrado expected")
	}
}

func TestTomaRoles(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newTomaRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tokAdmin := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tokAdmin)
	tokLector := loginUser(t, r, "lector1", "Lector123!")
	// lector POST toma 403
	req := authReq("POST", "/api/tomas", tokLector, map[string]interface{}{"sede": "Icon"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("lector crear toma should 403 got %d", w.Code)
	}
	// tecnico crea
	req2 := authReq("POST", "/api/tomas", tokAdmin, map[string]interface{}{"sede": "Icon"})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var res map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &res)
	tomaID := res["id"].(string)
	// lector escanear 201 (con dispositivo)
	// primero crear dispositivo Icon via tecnico
	reqD := authReq("POST", "/api/dispositivos", tokAdmin, map[string]interface{}{"codigo_activo": "ROLE-D1", "numero_serie": "SN-RD1", "tipo": "Tablet", "marca": "Samsung", "modelo": "M1", "sede": "Icon"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resD map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resD)
	did := resD["id"].(string)
	reqS := authReq("POST", "/api/tomas/"+tomaID+"/escanear", tokLector, map[string]interface{}{"dispositivo_id": did})
	wS := httptest.NewRecorder()
	r.ServeHTTP(wS, reqS)
	if wS.Code != 201 {
		t.Fatalf("lector escanear should 201 got %d %s", wS.Code, wS.Body.String())
	}
	// lector cerrar 403
	reqC := authReq("POST", "/api/tomas/"+tomaID+"/cerrar", tokLector, nil)
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)
	if wC.Code != 403 {
		t.Fatalf("lector cerrar should 403 got %d", wC.Code)
	}
}

func TestTomaDuplicadoIgnora(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newTomaRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	reqD := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "DUP-1", "numero_serie": "SN-DUP", "tipo": "Laptop", "marca": "Dell", "modelo": "M1", "sede": "Matriz"})
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var resD map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &resD)
	did := resD["id"].(string)
	reqG := authReq("POST", "/api/etiquetas/generar", tok, map[string]interface{}{"cantidad": 1, "version": 7})
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	var gen map[string]interface{}
	json.Unmarshal(wG.Body.Bytes(), &gen)
	eid := gen["ids"].([]interface{})[0].(string)
	reqA := authReq("POST", "/api/etiquetas/"+eid+"/asociar", tok, map[string]interface{}{"dispositivo_id": did})
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	reqT := authReq("POST", "/api/tomas", tok, map[string]interface{}{"sede": "Matriz"})
	wT := httptest.NewRecorder()
	r.ServeHTTP(wT, reqT)
	var resT map[string]interface{}
	json.Unmarshal(wT.Body.Bytes(), &resT)
	tomaID := resT["id"].(string)
	reqS := authReq("POST", "/api/tomas/"+tomaID+"/escanear", tok, map[string]interface{}{"etiqueta_id": eid})
	wS := httptest.NewRecorder()
	r.ServeHTTP(wS, reqS)
	if wS.Code != 201 {
		t.Fatalf("first scan 201 got %d", wS.Code)
	}
	reqS2 := authReq("POST", "/api/tomas/"+tomaID+"/escanear", tok, map[string]interface{}{"etiqueta_id": eid})
	wS2 := httptest.NewRecorder()
	r.ServeHTTP(wS2, reqS2)
	if wS2.Code != 200 {
		// nuestro handler retorna 200 ya escaneado para duplicado UNIQUE
		t.Fatalf("dup scan should 200 ya escaneado got %d %s", wS2.Code, wS2.Body.String())
	}
	// verificar solo 1 escaneado
	reqG2 := authReq("GET", "/api/tomas/"+tomaID, tok, nil)
	wG2 := httptest.NewRecorder()
	r.ServeHTTP(wG2, reqG2)
	var detail map[string]interface{}
	json.Unmarshal(wG2.Body.Bytes(), &detail)
	if int(detail["escaneados_count"].(float64)) != 1 {
		t.Fatalf("escaneados 1 expected got %v", detail["escaneados_count"])
	}
}

func TestAuditoriaToma(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	// need audit router + toma
	rAud := newTomaRouter(db) // shares DB but need audit query via direct DB or via toma router? Use DB query
	registerUser(t, rAud, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, rAud, "admin", "Admin123!")
	req := authReq("POST", "/api/tomas", tok, map[string]interface{}{"sede": "Icon"})
	w := httptest.NewRecorder()
	rAud.ServeHTTP(w, req)
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	tomaID := res["id"].(string)
	// check auditoria via DB directly
	var cnt int
	db.QueryRow(`SELECT COUNT(*) FROM auditoria WHERE entidad='toma' AND entidad_id=? AND accion='crear'`, tomaID).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("auditoria crear toma missing cnt %d", cnt)
	}
}

func TestTomaFiltradaPorSedeSinDadoDeBaja(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newTomaRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	// activo Icon
	reqA := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "FILT-A", "numero_serie": "SN-FA", "tipo": "Laptop", "marca": "Dell", "modelo": "M1", "sede": "Icon", "estado": "activo"})
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	// dado_de_baja Icon no debe contar como esperado
	reqB := authReq("POST", "/api/dispositivos", tok, map[string]interface{}{"codigo_activo": "FILT-B", "numero_serie": "SN-FB", "tipo": "Laptop", "marca": "Dell", "modelo": "M1", "sede": "Icon", "estado": "activo"})
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	var resB map[string]interface{}
	json.Unmarshal(wB.Body.Bytes(), &resB)
	bID := resB["id"].(string)
	// dar de baja B directamente (bypass router, no necesita auditoria)
	db.Exec(`UPDATE dispositivos SET estado='dado_de_baja' WHERE id=?`, bID)
	// toma Icon
	reqT := authReq("POST", "/api/tomas", tok, map[string]interface{}{"sede": "Icon"})
	wT := httptest.NewRecorder()
	r.ServeHTTP(wT, reqT)
	var resT map[string]interface{}
	json.Unmarshal(wT.Body.Bytes(), &resT)
	tomaID := resT["id"].(string)
	reqG := authReq("GET", "/api/tomas/"+tomaID, tok, nil)
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	var detail map[string]interface{}
	json.Unmarshal(wG.Body.Bytes(), &detail)
	if int(detail["esperados_count"].(float64)) != 1 {
		t.Fatalf("esperados should 1 (excluye dado_de_baja) got %v", detail["esperados_count"])
	}
}

func jsonMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
