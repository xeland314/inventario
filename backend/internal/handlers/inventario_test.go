package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
	dbpkg "inventario-backend/internal/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return db
}

func newRouter(db *sql.DB) chi.Router {
	r := chi.NewRouter()
	dh := &DispositivoHandler{DB: db}
	mh := &MovimientoHandler{DB: db}
	r.Get("/api/dispositivos", dh.Listar)
	r.Post("/api/dispositivos", dh.Crear)
	r.Get("/api/dispositivos/{id}", dh.Obtener)
	r.Put("/api/dispositivos/{id}", dh.Actualizar)
	r.Patch("/api/dispositivos/{id}", dh.Actualizar)
	r.Patch("/api/dispositivos/{id}/baja", dh.DarDeBaja)
	r.Get("/api/dispositivos/{id}/componentes", dh.ListarComponentes)
	r.Delete("/api/dispositivos/{id}", dh.Eliminar)
	r.Get("/api/movimientos", mh.Listar)
	r.Post("/api/movimientos", mh.Crear)
	return r
}

func createDispositivo(t *testing.T, router chi.Router, payload map[string]interface{}) string {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("create failed %d %s payload %v", w.Code, w.Body.String(), payload)
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	return res["id"].(string)
}

// ---------- Estados ----------
func TestEstadosValidos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)

	estadosValidos := []string{"activo", "disponible", "reserva", "en_mantenimiento", "en_reparacion", "averiado", "obsoleto", "extraviado"}
	for _, estado := range estadosValidos {
		payload := map[string]interface{}{
			"codigo_activo": "EST-" + estado,
			"numero_serie":  "SN-" + estado,
			"tipo":          "Laptop",
			"marca":         "Dell",
			"modelo":        "XPS",
			"estado":        estado,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Errorf("estado %s should be valid, got %d %s", estado, w.Code, w.Body.String())
		}
	}
}

func TestEstadoInvalido(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	payload := map[string]interface{}{
		"codigo_activo": "BAD-001",
		"numero_serie":  "SN-BAD",
		"tipo":          "Laptop",
		"marca":         "HP",
		"modelo":        "ProBook",
		"estado":        "estado_inventado",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid estado, got %d %s", w.Code, w.Body.String())
	}
}

func TestListarFiltroEstado(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	createDispositivo(t, r, map[string]interface{}{"codigo_activo": "F1", "numero_serie": "SN1", "tipo": "Laptop", "marca": "Dell", "modelo": "M1", "estado": "averiado"})
	createDispositivo(t, r, map[string]interface{}{"codigo_activo": "F2", "numero_serie": "SN2", "tipo": "Laptop", "marca": "Dell", "modelo": "M2", "estado": "obsoleto"})
	req := httptest.NewRequest("GET", "/api/dispositivos?estado=averiado", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list filter failed %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	list := resp["dispositivos"].([]interface{})
	if len(list) != 1 {
		t.Fatalf("expected 1 averiado, got %d", len(list))
	}
}

func TestActualizarEstadoInvalido(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	id := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "U1", "numero_serie": "SN-U1", "tipo": "Desktop", "marca": "HP", "modelo": "M1"})
	body, _ := json.Marshal(map[string]interface{}{"estado": "no_existe"})
	req := httptest.NewRequest("PUT", "/api/dispositivos/"+id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 invalid update estado, got %d %s", w.Code, w.Body.String())
	}
}

// ---------- Parent / Componentes ----------
func TestCrearComponenteConParent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	parentID := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "PARENT-001", "numero_serie": "SN-P", "tipo": "Desktop", "marca": "Intel", "modelo": "i5", "ubicacion": "Matriz", "sede": "Matriz"})
	childID := createDispositivo(t, r, map[string]interface{}{
		"codigo_activo": "FAN-001", "numero_serie": "SN-FAN", "tipo": "Otro", "marca": "Generico", "modelo": "Ventilador 120mm",
		"parent_id": parentID, "tipo_componente": "Ventilador", "sede": "Matriz",
	})
	// verificar que padre lista componentes
	req := httptest.NewRequest("GET", "/api/dispositivos/"+parentID+"/componentes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list componentes %d %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	comps := resp["componentes"].([]interface{})
	if len(comps) != 1 {
		t.Fatalf("expected 1 componente, got %d", len(comps))
	}
	// verificar Obtener incluye componentes
	req2 := httptest.NewRequest("GET", "/api/dispositivos/"+parentID, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var disp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &disp)
	if comps2, ok := disp["componentes"].([]interface{}); !ok || len(comps2) != 1 {
		t.Fatalf("obtener should include componentes")
	}
	// child debe tener parent_id
	req3 := httptest.NewRequest("GET", "/api/dispositivos/"+childID, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var child map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &child)
	if child["parent_id"] != parentID {
		t.Fatalf("child parent_id mismatch %v != %v", child["parent_id"], parentID)
	}
	if child["tipo_componente"] != "Ventilador" {
		t.Fatalf("tipo_componente mismatch")
	}
}

func TestCrearComponenteParentInexistente(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	body, _ := json.Marshal(map[string]interface{}{
		"codigo_activo": "BAD-CHILD", "numero_serie": "SN-BAD", "tipo": "Otro", "marca": "X", "modelo": "Y",
		"parent_id": "no-existe-uuid", "tipo_componente": "SSD",
	})
	req := httptest.NewRequest("POST", "/api/dispositivos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for nonexistent parent, got %d %s", w.Code, w.Body.String())
	}
}

func TestSelfReferenceCiclo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	id := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "CICLO-001", "numero_serie": "SN-C", "tipo": "Desktop", "marca": "HP", "modelo": "M1"})
	// intentar asignarse a sí mismo
	body, _ := json.Marshal(map[string]interface{}{"parent_id": id})
	req := httptest.NewRequest("PUT", "/api/dispositivos/"+id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("self-reference should fail 400, got %d %s", w.Code, w.Body.String())
	}
	// ciclo A->B, intentar B->A
	aID := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "A-001", "numero_serie": "SN-A", "tipo": "Desktop", "marca": "HP", "modelo": "A"})
	bID := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "B-001", "numero_serie": "SN-B", "tipo": "Otro", "marca": "HP", "modelo": "B", "parent_id": aID, "tipo_componente": "RAM"})
	// ahora intentar A parent = B (ciclo)
	body2, _ := json.Marshal(map[string]interface{}{"parent_id": bID})
	req2 := httptest.NewRequest("PUT", "/api/dispositivos/"+aID, bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 400 {
		t.Fatalf("cycle should fail 400, got %d %s", w2.Code, w2.Body.String())
	}
}

func TestTrasladoGeneraMovimiento(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	desk1 := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "DESK-1", "numero_serie": "SN-D1", "tipo": "Desktop", "marca": "Intel", "modelo": "i5"})
	desk2 := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "DESK-2", "numero_serie": "SN-D2", "tipo": "Desktop", "marca": "Intel", "modelo": "i7"})
	comp := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "SSD-001", "numero_serie": "SN-SSD", "tipo": "Otro", "marca": "Samsung", "modelo": "860", "parent_id": desk1, "tipo_componente": "SSD"})
	// mover comp de desk1 a desk2 via PUT
	body, _ := json.Marshal(map[string]interface{}{"parent_id": desk2})
	req := httptest.NewRequest("PUT", "/api/dispositivos/"+comp, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("traslado failed %d %s", w.Code, w.Body.String())
	}
	// verificar movimientos para comp
	req2 := httptest.NewRequest("GET", "/api/movimientos?dispositivo_id="+comp, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	movs := resp["movimientos"].([]interface{})
	// debe haber al menos 2: instalacion inicial + traslado
	if len(movs) < 2 {
		t.Fatalf("expected >=2 movimientos, got %d", len(movs))
	}
	foundTraslado := false
	for _, m := range movs {
		mm := m.(map[string]interface{})
		if mm["tipo_movimiento"] == "traslado" {
			foundTraslado = true
			if mm["origen_parent_id"] != desk1 || mm["destino_parent_id"] != desk2 {
				t.Fatalf("traslado origen/destino mismatch")
			}
		}
	}
	if !foundTraslado {
		t.Fatalf("traslado movimiento not found")
	}
}

func TestRetiroGeneraMovimiento(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	desk := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "DESK-R", "numero_serie": "SN-DR", "tipo": "Desktop", "marca": "Intel", "modelo": "i5"})
	comp := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "RAM-001", "numero_serie": "SN-RAM", "tipo": "Otro", "marca": "Kingston", "modelo": "8GB", "parent_id": desk, "tipo_componente": "RAM"})
	// retirar (parent_id = "")
	body, _ := json.Marshal(map[string]interface{}{"parent_id": ""})
	req := httptest.NewRequest("PUT", "/api/dispositivos/"+comp, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("retiro failed %d %s", w.Code, w.Body.String())
	}
	req2 := httptest.NewRequest("GET", "/api/movimientos?dispositivo_id="+comp, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	movs := resp["movimientos"].([]interface{})
	found := false
	for _, m := range movs {
		if m.(map[string]interface{})["tipo_movimiento"] == "retiro" {
			found = true
		}
	}
	if !found {
		t.Fatalf("retiro movimiento not found")
	}
}

// ---------- Movimientos via POST ----------
func TestCrearMovimientoTraslado(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	desk1 := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "D1", "numero_serie": "SN-D1", "tipo": "Desktop", "marca": "HP", "modelo": "M1"})
	desk2 := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "D2", "numero_serie": "SN-D2", "tipo": "Desktop", "marca": "HP", "modelo": "M2"})
	comp := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "C-T", "numero_serie": "SN-C", "tipo": "Otro", "marca": "X", "modelo": "Y", "parent_id": desk1, "tipo_componente": "HDD"})
	payload := map[string]interface{}{
		"dispositivo_id":    comp,
		"destino_parent_id": desk2,
		"tipo_movimiento":   "traslado",
		"descripcion":       "canibalizacion HDD",
		"realizado_por":     "tester",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/movimientos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("create movimiento traslado %d %s", w.Code, w.Body.String())
	}
	// verificar parent actualizado
	req2 := httptest.NewRequest("GET", "/api/dispositivos/"+comp, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var child map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &child)
	if child["parent_id"] != desk2 {
		t.Fatalf("after traslado parent should be desk2")
	}
}

func TestCrearMovimientoInvalido(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	comp := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "C-BAD", "numero_serie": "SN-CBAD", "tipo": "Otro", "marca": "X", "modelo": "Y"})
	body, _ := json.Marshal(map[string]interface{}{
		"dispositivo_id":  comp,
		"tipo_movimiento": "tipo_invalido",
	})
	req := httptest.NewRequest("POST", "/api/movimientos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("invalid tipo_movimiento should 400 got %d %s", w.Code, w.Body.String())
	}
}

// ---------- Sede y filtros ----------
func TestFiltroSedeYComponente(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := newRouter(db)
	createDispositivo(t, r, map[string]interface{}{"codigo_activo": "S1", "numero_serie": "SN-S1", "tipo": "Laptop", "marca": "Dell", "modelo": "M1", "sede": "Matriz"})
	createDispositivo(t, r, map[string]interface{}{"codigo_activo": "S2", "numero_serie": "SN-S2", "tipo": "Laptop", "marca": "Dell", "modelo": "M2", "sede": "Icon"})
	req := httptest.NewRequest("GET", "/api/dispositivos?sede=Icon", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp["dispositivos"].([]interface{})) != 1 {
		t.Fatalf("filter sede failed")
	}
	// filtro solo_componentes
	desk := createDispositivo(t, r, map[string]interface{}{"codigo_activo": "DESK-F", "numero_serie": "SN-DF", "tipo": "Desktop", "marca": "HP", "modelo": "M1"})
	createDispositivo(t, r, map[string]interface{}{"codigo_activo": "COMP-F", "numero_serie": "SN-CF", "tipo": "Otro", "marca": "X", "modelo": "Y", "parent_id": desk, "tipo_componente": "SSD"})
	req2 := httptest.NewRequest("GET", "/api/dispositivos?solo_componentes=true", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var resp2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if len(resp2["dispositivos"].([]interface{})) != 1 {
		t.Fatalf("solo_componentes filter failed got %d", len(resp2["dispositivos"].([]interface{})))
	}
}

// ---------- Migración segura ----------
func TestMigracionNoPierdeDatos(t *testing.T) {
	// Simula BD antigua sin columnas nuevas, InitDB debe migrar sin perder
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// crear esquema antiguo manualmente
	_, err = db.Exec(`CREATE TABLE dispositivos (id TEXT PRIMARY KEY, codigo_activo TEXT UNIQUE NOT NULL, numero_serie TEXT NOT NULL, tipo TEXT NOT NULL, marca TEXT NOT NULL, modelo TEXT NOT NULL, especificaciones TEXT, valor_compra REAL, proveedor TEXT, estado TEXT DEFAULT 'activo', ubicacion TEXT, asignado_a TEXT, fecha_compra TEXT, fecha_baja TEXT, motivo_baja TEXT, creado_en TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO dispositivos (id, codigo_activo, numero_serie, tipo, marca, modelo, estado, ubicacion, fecha_compra, creado_en) VALUES ('id-old','OLD-001','SN-OLD','Desktop','HP','M1','activo','Matriz','2026-01-01', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	// ahora ejecutar InitDB (debería añadir columnas y tabla movimientos)
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatalf("InitDB migration failed: %v", err)
	}
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM dispositivos WHERE id='id-old'`).Scan(&cnt); err != nil || cnt != 1 {
		t.Fatalf("dato antiguo perdido")
	}
	// verificar columnas nuevas existen
	_, err = db.Exec(`SELECT parent_id, tipo_componente, sede, actualizado_en FROM dispositivos LIMIT 1`)
	if err != nil {
		t.Fatalf("columnas nuevas no existen tras migración: %v", err)
	}
	_, err = db.Exec(`SELECT id FROM movimientos LIMIT 1`)
	if err != nil {
		t.Fatalf("tabla movimientos no creada: %v", err)
	}
}
