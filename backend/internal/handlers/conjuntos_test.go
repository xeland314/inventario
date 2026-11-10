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

func newConjuntosRouter(db *sql.DB) chi.Router {
	r := chi.NewRouter()
	ah := &AuthHandler{DB: db}
	eh := &EtiquetaHandler{DB: db}
	ch := &ConjuntoHandler{DB: db}
	r.Post("/api/auth/register", ah.Register)
	r.Post("/api/auth/login", ah.Login)
	r.Group(func(r chi.Router) {
		r.Use(ah.Middleware)
		r.Get("/api/conjuntos", ch.Listar)
		r.Get("/api/conjuntos/{id}", ch.Obtener)
		r.Get("/api/conjuntos/{id}/movimientos", ch.ListarMovimientos)
		r.Get("/api/conjuntos/{id}/conteos", ch.ListarConteos)
		r.Get("/api/etiquetas/scan/{id}", eh.Escanear)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("tecnico"))
			r.Post("/api/conjuntos", ch.Crear)
			r.Put("/api/conjuntos/{id}", ch.Actualizar)
			r.Delete("/api/conjuntos/{id}", ch.Eliminar)
			r.Post("/api/conjuntos/{id}/movimiento", ch.Movimiento)
			r.Post("/api/conjuntos/{id}/conteos", ch.CrearConteo)
			r.Post("/api/etiquetas/generar", eh.Generar)
			r.Post("/api/etiquetas/{id}/asociar", eh.Asociar)
			r.Post("/api/etiquetas/{id}/desasociar", eh.Desasociar)
		})
	})
	return r
}

func crearConjunto(t *testing.T, r chi.Router, tok string, payload map[string]interface{}) map[string]interface{} {
	t.Helper()
	req := authReq("POST", "/api/conjuntos", tok, payload)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("crear conjunto %d %s", w.Code, w.Body.String())
	}
	var out map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &out)
	return out
}

func TestConjuntoCrearListarBajo(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newConjuntosRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")

	out := crearConjunto(t, r, tok, map[string]interface{}{
		"nombre": "Tornillos M3x10 acero inox", "categoria": "tornillos",
		"unidad": "pieza", "cantidad_actual": 500, "cantidad_minima": 50,
		"ubicacion": "Caja organizadora A, cajón 3", "sede": "Matriz",
	})
	cid := out["id"].(string)
	if out["codigo"].(string)[:3] != "CJ-" {
		t.Fatalf("código auto CJ-XXXX esperado: %v", out["codigo"])
	}

	// stock inicial en el libro
	reqM := authReq("GET", "/api/conjuntos/"+cid+"/movimientos", tok, nil)
	wM := httptest.NewRecorder()
	r.ServeHTTP(wM, reqM)
	var movs map[string]interface{}
	json.Unmarshal(wM.Body.Bytes(), &movs)
	if len(movs["movimientos"].([]interface{})) != 1 {
		t.Fatalf("stock inicial en libro esperado: %v", movs)
	}

	// ?bajo=1 vacío con 500 > mín 50
	reqB := authReq("GET", "/api/conjuntos?bajo=1", tok, nil)
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	var bajos map[string]interface{}
	json.Unmarshal(wB.Body.Bytes(), &bajos)
	if len(bajos["conjuntos"].([]interface{})) != 0 {
		t.Fatalf("bajo=1 debería estar vacío: %v", bajos)
	}

	// consumo 480 → queda 20 ≤ 50 → aparece en bajo
	reqC := authReq("POST", "/api/conjuntos/"+cid+"/movimiento", tok, map[string]interface{}{"tipo": "consumo", "cantidad": 480})
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)
	if wC.Code != 200 {
		t.Fatalf("consumo %d %s", wC.Code, wC.Body.String())
	}
	reqB2 := authReq("GET", "/api/conjuntos?bajo=1", tok, nil)
	wB2 := httptest.NewRecorder()
	r.ServeHTTP(wB2, reqB2)
	json.Unmarshal(wB2.Body.Bytes(), &bajos)
	if len(bajos["conjuntos"].([]interface{})) != 1 {
		t.Fatalf("bajo=1 debería tener 1: %v", bajos)
	}
	if bajos["conjuntos"].([]interface{})[0].(map[string]interface{})["estado"] != "bajo" {
		t.Fatalf("estado esperado bajo")
	}
}

func TestConjuntoMovimientoAritmetica(t *testing.T) {
	db, _ := sql.Open("sqlite3", ":memory:?_fk=1")
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newConjuntosRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	out := crearConjunto(t, r, tok, map[string]interface{}{"nombre": "Resistores 10kΩ 1/4W", "cantidad_actual": 100})
	cid := out["id"].(string)

	// consumo mayor al stock → 400
	reqX := authReq("POST", "/api/conjuntos/"+cid+"/movimiento", tok, map[string]interface{}{"tipo": "consumo", "cantidad": 101})
	wX := httptest.NewRecorder()
	r.ServeHTTP(wX, reqX)
	if wX.Code != 400 {
		t.Fatalf("stock insuficiente debería 400, got %d %s", wX.Code, wX.Body.String())
	}

	// ajuste negativo (faltan 5) → 95
	reqA := authReq("POST", "/api/conjuntos/"+cid+"/movimiento", tok, map[string]interface{}{"tipo": "ajuste", "cantidad": -5, "nota": "conteo físico"})
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	if wA.Code != 200 {
		t.Fatalf("ajuste %d %s", wA.Code, wA.Body.String())
	}
	var resA map[string]interface{}
	json.Unmarshal(wA.Body.Bytes(), &resA)
	if int(resA["cantidad_actual"].(float64)) != 95 || int(resA["cantidad_previa"].(float64)) != 100 {
		t.Fatalf("previa/nueva esperado 100→95: %v", resA)
	}

	// PUT no permite cantidad_actual
	reqP := authReq("PUT", "/api/conjuntos/"+cid, tok, map[string]interface{}{"nombre": "Resistores 10kΩ 1/4W 5%"})
	wP := httptest.NewRecorder()
	r.ServeHTTP(wP, reqP)
	if wP.Code != 200 {
		t.Fatalf("PUT %d %s", wP.Code, wP.Body.String())
	}

	// detalle
	reqD := authReq("GET", "/api/conjuntos/"+cid, tok, nil)
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	var det map[string]interface{}
	json.Unmarshal(wD.Body.Bytes(), &det)
	if int(det["cantidad_actual"].(float64)) != 95 {
		t.Fatalf("cantidad_actual esperado 95: %v", det["cantidad_actual"])
	}
}

func TestConjuntoEtiquetaAsociarScan(t *testing.T) {
	db, _ := sql.Open("sqlite3", ":memory:?_fk=1")
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newConjuntosRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")

	out := crearConjunto(t, r, tok, map[string]interface{}{"nombre": "Tuercas M3", "cantidad_actual": 200})
	cid := out["id"].(string)

	// generar 2 etiquetas del pool
	reqG := authReq("POST", "/api/etiquetas/generar", tok, map[string]interface{}{"cantidad": 2, "version": 7})
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	if wG.Code != 201 {
		t.Fatalf("generar %d %s", wG.Code, wG.Body.String())
	}
	var gen map[string]interface{}
	json.Unmarshal(wG.Body.Bytes(), &gen)
	ids := gen["ids"].([]interface{})
	et1 := ids[0].(string)
	et2 := ids[1].(string)

	// asociar et1 al conjunto
	reqA := authReq("POST", "/api/etiquetas/"+et1+"/asociar", tok, map[string]interface{}{"conjunto_id": cid})
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	if wA.Code != 200 {
		t.Fatalf("asociar conjunto %d %s", wA.Code, wA.Body.String())
	}

	// scan de la etiqueta → retorna conjunto
	reqS := authReq("GET", "/api/etiquetas/scan/"+et1, tok, nil)
	wS := httptest.NewRecorder()
	r.ServeHTTP(wS, reqS)
	if wS.Code != 200 {
		t.Fatalf("scan %d %s", wS.Code, wS.Body.String())
	}
	var scan map[string]interface{}
	json.Unmarshal(wS.Body.Bytes(), &scan)
	conj, ok := scan["conjunto"].(map[string]interface{})
	if !ok || conj == nil {
		t.Fatalf("scan debería retornar conjunto: %v", scan)
	}
	if conj["nombre"] != "Tuercas M3" || int(conj["cantidad_actual"].(float64)) != 200 {
		t.Fatalf("conjunto del scan incorrecto: %v", conj)
	}

	// la caja ya tiene etiqueta → et2 → 409
	reqA2 := authReq("POST", "/api/etiquetas/"+et2+"/asociar", tok, map[string]interface{}{"conjunto_id": cid})
	wA2 := httptest.NewRecorder()
	r.ServeHTTP(wA2, reqA2)
	if wA2.Code != 409 {
		t.Fatalf("caja ya etiquetada debería 409, got %d %s", wA2.Code, wA2.Body.String())
	}

	// listado incluye etiqueta_codigo (ET-XXXX)
	reqL := authReq("GET", "/api/conjuntos", tok, nil)
	wL := httptest.NewRecorder()
	r.ServeHTTP(wL, reqL)
	var list map[string]interface{}
	json.Unmarshal(wL.Body.Bytes(), &list)
	first := list["conjuntos"].([]interface{})[0].(map[string]interface{})
	if first["etiqueta_id"] != et1 {
		t.Fatalf("listado debería incluir etiqueta_id: %v", first)
	}

	// desasociar libera la etiqueta
	reqD := authReq("POST", "/api/etiquetas/"+et1+"/desasociar", tok, nil)
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	if wD.Code != 200 {
		t.Fatalf("desasociar %d %s", wD.Code, wD.Body.String())
	}
	reqS2 := authReq("GET", "/api/etiquetas/scan/"+et1, tok, nil)
	wS2 := httptest.NewRecorder()
	r.ServeHTTP(wS2, reqS2)
	json.Unmarshal(wS2.Body.Bytes(), &scan)
	if scan["conjunto"] != nil {
		t.Fatalf("tras desasociar scan no debería traer conjunto: %v", scan)
	}
}

func TestConjuntoConteosAjuste(t *testing.T) {
	db, _ := sql.Open("sqlite3", ":memory:?_fk=1")
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newConjuntosRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	out := crearConjunto(t, r, tok, map[string]interface{}{"nombre": "Tornillos M4", "cantidad_actual": 500})
	cid := out["id"].(string)

	// conteo: IA detectó 473, humano confirmó 480, aplicar ajuste → delta -20
	reqC := authReq("POST", "/api/conjuntos/"+cid+"/conteos", tok, map[string]interface{}{
		"cantidad_detectada": 473, "cantidad_confirmada": 480,
		"modelo": "browser:owlvit-base-patch32", "aplicar_ajuste": true,
		"detecciones": []map[string]interface{}{
			{"box": [4]float64{0.1, 0.2, 0.3, 0.4}, "score": 0.31, "label": "screw"},
		},
	})
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)
	if wC.Code != 201 {
		t.Fatalf("conteo %d %s", wC.Code, wC.Body.String())
	}
	var resC map[string]interface{}
	json.Unmarshal(wC.Body.Bytes(), &resC)
	if int(resC["delta"].(float64)) != -20 || resC["ajuste_aplicado"] != true {
		t.Fatalf("delta esperado -20 ajustado: %v", resC)
	}

	// el ajuste quedó en el libro como movimiento tipo ajuste
	reqM := authReq("GET", "/api/conjuntos/"+cid+"/movimientos", tok, nil)
	wM := httptest.NewRecorder()
	r.ServeHTTP(wM, reqM)
	var movs map[string]interface{}
	json.Unmarshal(wM.Body.Bytes(), &movs)
	movsList := movs["movimientos"].([]interface{})
	if len(movsList) != 2 {
		t.Fatalf("esperado ingreso inicial + ajuste: %v", movs)
	}
	ajuste := movsList[0].(map[string]interface{})
	if ajuste["tipo"] != "ajuste" || int(ajuste["cantidad"].(float64)) != -20 {
		t.Fatalf("ajuste en libro incorrecto: %v", ajuste)
	}

	// historial de conteos visible
	reqH := authReq("GET", "/api/conjuntos/"+cid+"/conteos", tok, nil)
	wH := httptest.NewRecorder()
	r.ServeHTTP(wH, reqH)
	var hist map[string]interface{}
	json.Unmarshal(wH.Body.Bytes(), &hist)
	conteos := hist["conteos"].([]interface{})
	if len(conteos) != 1 {
		t.Fatalf("esperado 1 conteo: %v", hist)
	}
	ct := conteos[0].(map[string]interface{})
	if int(ct["cantidad_detectada"].(float64)) != 473 || int(ct["cantidad_confirmada"].(float64)) != 480 {
		t.Fatalf("conteo detectada/confirmada: %v", ct)
	}
	if ct["modelo"] != "browser:owlvit-base-patch32" {
		t.Fatalf("modelo: %v", ct)
	}
}

func TestConjuntoRoles(t *testing.T) {
	db, _ := sql.Open("sqlite3", ":memory:?_fk=1")
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newConjuntosRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	registerUser(t, r, "lector1", "Lector123!", "lector", tok)
	tokLector := loginUser(t, r, "lector1", "Lector123!")

	// lector GET 200
	reqG := authReq("GET", "/api/conjuntos", tokLector, nil)
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	if wG.Code != 200 {
		t.Fatalf("lector GET debería 200, got %d", wG.Code)
	}
	// lector POST 403
	reqP := authReq("POST", "/api/conjuntos", tokLector, map[string]interface{}{"nombre": "X"})
	wP := httptest.NewRecorder()
	r.ServeHTTP(wP, reqP)
	if wP.Code != 403 {
		t.Fatalf("lector POST debería 403, got %d", wP.Code)
	}
	// lector movimiento 403
	crearConjunto(t, r, tok, map[string]interface{}{"nombre": "Fusibles 5x20", "cantidad_actual": 10})
	reqM := authReq("GET", "/api/conjuntos", tok, nil)
	wM := httptest.NewRecorder()
	r.ServeHTTP(wM, reqM)
	var list map[string]interface{}
	json.Unmarshal(wM.Body.Bytes(), &list)
	cid := list["conjuntos"].([]interface{})[0].(map[string]interface{})["id"].(string)
	reqMov := authReq("POST", "/api/conjuntos/"+cid+"/movimiento", tokLector, map[string]interface{}{"tipo": "consumo", "cantidad": 1})
	wMov := httptest.NewRecorder()
	r.ServeHTTP(wMov, reqMov)
	if wMov.Code != 403 {
		t.Fatalf("lector movimiento debería 403, got %d", wMov.Code)
	}
}

func TestConjuntoEliminar(t *testing.T) {
	db, _ := sql.Open("sqlite3", ":memory:?_fk=1")
	defer db.Close()
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	r := newConjuntosRouter(db)
	registerUser(t, r, "admin", "Admin123!", "tecnico", "")
	tok := loginUser(t, r, "admin", "Admin123!")
	out := crearConjunto(t, r, tok, map[string]interface{}{"nombre": "Jumpers", "cantidad_actual": 30})
	cid := out["id"].(string)

	// etiqueta asociada
	reqG := authReq("POST", "/api/etiquetas/generar", tok, map[string]interface{}{"cantidad": 1, "version": 4})
	wG := httptest.NewRecorder()
	r.ServeHTTP(wG, reqG)
	var gen map[string]interface{}
	json.Unmarshal(wG.Body.Bytes(), &gen)
	et := gen["ids"].([]interface{})[0].(string)
	reqA := authReq("POST", "/api/etiquetas/"+et+"/asociar", tok, map[string]interface{}{"conjunto_id": cid})
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	if wA.Code != 200 {
		t.Fatalf("asociar %d %s", wA.Code, wA.Body.String())
	}

	// eliminar conjunto → etiqueta liberada a disponible
	reqD := authReq("DELETE", "/api/conjuntos/"+cid, tok, nil)
	wD := httptest.NewRecorder()
	r.ServeHTTP(wD, reqD)
	if wD.Code != 200 {
		t.Fatalf("eliminar %d %s", wD.Code, wD.Body.String())
	}
	reqS := authReq("GET", "/api/etiquetas/scan/"+et, tok, nil)
	wS := httptest.NewRecorder()
	r.ServeHTTP(wS, reqS)
	var scan map[string]interface{}
	json.Unmarshal(wS.Body.Bytes(), &scan)
	if scan["conjunto"] != nil {
		t.Fatalf("tras eliminar conjunto, etiqueta debería quedar libre: %v", scan)
	}
	eti := scan["etiqueta"].(map[string]interface{})
	if eti["estado"] != "disponible" {
		t.Fatalf("etiqueta debería volver a disponible: %v", eti)
	}
}
