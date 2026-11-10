package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dbpkg "inventario-backend/internal/db"
	"inventario-backend/internal/scanner"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func setupScannerTestRouter(t *testing.T) (*chi.Mux, *sql.DB, *scanner.Hub) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatalf("error abriendo DB in-memory: %v", err)
	}
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatalf("error inicializando DB: %v", err)
	}

	hub := scanner.NewHub(db)
	sh := &ScannerHandler{DB: db, Hub: hub}

	r := chi.NewRouter()
	r.Get("/api/scanner/device", sh.DeviceInfo)
	r.Post("/api/scanner/scan", sh.RecordScan)
	r.Get("/api/scanner/scans", sh.ListScans)
	r.Delete("/api/scanner/scans", sh.ClearScans)
	r.Get("/api/scanner/catalog", sh.Catalog)
	r.Get("/api/scanner/lookup/{code}", sh.Lookup)
	r.Get("/api/barcode/{format}/{code}", sh.BarcodePNG)
	r.Get("/api/barcode/{format}/{code}.svg", sh.BarcodeSVG)

	return r, db, hub
}

func TestScannerDeviceInfo(t *testing.T) {
	r, db, hub := setupScannerTestRouter(t)
	defer db.Close()
	defer hub.Close()

	req := httptest.NewRequest("GET", "/api/scanner/device", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d", w.Code)
	}

	var info scanner.ScannerDeviceInfo
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
		t.Fatalf("error parseando json: %v", err)
	}

	if info.VendorID != "05e0" {
		t.Errorf("esperado vendor_id 05e0, obtenido %s", info.VendorID)
	}
	if info.ProductID != "1200" {
		t.Errorf("esperado product_id 1200, obtenido %s", info.ProductID)
	}
}

func TestScannerRecordScanAndHistory(t *testing.T) {
	r, db, hub := setupScannerTestRouter(t)
	defer db.Close()
	defer hub.Close()

	// 1. Insertar un dispositivo de prueba
	_, err := db.Exec(`
		INSERT INTO dispositivos (id, codigo_activo, numero_serie, tipo, marca, modelo, estado, sede, proveedor, ubicacion, fecha_compra, creado_en)
		VALUES ('disp-test-1', 'GGM-LAP-099', 'SN-TEST-888', 'Laptop', 'Lenovo', 'ThinkPad T14', 'activo', 'Matriz', 'Lenovo Oficial', 'Piso 2', '2026-01-01', '2026-11-04 10:00:00')
	`)
	if err != nil {
		t.Fatalf("error insertando dispositivo: %v", err)
	}

	// 2. Escanear por codigo_activo
	body, _ := json.Marshal(map[string]interface{}{
		"code":        "GGM-LAP-099",
		"duration_ms": 28.5,
		"source":      "usb_scanner_wedge",
	})
	req := httptest.NewRequest("POST", "/api/scanner/scan", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Status string             `json:"status"`
		Data   scanner.ScanRecord `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("error unmarshaling response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("esperado status success, obtenido %s", resp.Status)
	}
	if resp.Data.MatchedType != "dispositivo" {
		t.Errorf("esperado matched_type dispositivo, obtenido %s", resp.Data.MatchedType)
	}
	if !resp.Data.IsScanner {
		t.Errorf("esperado is_scanner=true")
	}

	// 3. Consultar historial
	reqH := httptest.NewRequest("GET", "/api/scanner/scans", nil)
	wH := httptest.NewRecorder()
	r.ServeHTTP(wH, reqH)

	var list []scanner.ScanRecord
	if err := json.Unmarshal(wH.Body.Bytes(), &list); err != nil {
		t.Fatalf("error leyendo historial: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("esperado 1 escaneo en historial, obtenido %d", len(list))
	}
	if list[0].Code != "GGM-LAP-099" {
		t.Errorf("codigo incorrecto: %s", list[0].Code)
	}

	// 4. Limpiar historial
	reqC := httptest.NewRequest("DELETE", "/api/scanner/scans", nil)
	wC := httptest.NewRecorder()
	r.ServeHTTP(wC, reqC)

	if wC.Code != http.StatusOK {
		t.Errorf("esperado 200 en DELETE scans")
	}

	// 5. Historial debe estar vacío
	reqH2 := httptest.NewRequest("GET", "/api/scanner/scans", nil)
	wH2 := httptest.NewRecorder()
	r.ServeHTTP(wH2, reqH2)
	var list2 []scanner.ScanRecord
	json.Unmarshal(wH2.Body.Bytes(), &list2)
	if len(list2) != 0 {
		t.Errorf("esperado historial vacío tras DELETE, hay %d", len(list2))
	}
}

func TestScannerLookupConjuntoAndDemo(t *testing.T) {
	r, db, hub := setupScannerTestRouter(t)
	defer db.Close()
	defer hub.Close()

	// Insertar conjunto
	_, err := db.Exec(`
		INSERT INTO conjuntos (id, codigo, nombre, unidad, cantidad_actual, cantidad_minima)
		VALUES ('conj-1', 'CJ-00000042', 'Tornillos M3x10', 'pieza', 100, 20)
	`)
	if err != nil {
		t.Fatalf("error insertando conjunto: %v", err)
	}

	// Lookup de conjunto
	reqL := httptest.NewRequest("GET", "/api/scanner/lookup/CJ-00000042", nil)
	wL := httptest.NewRecorder()
	r.ServeHTTP(wL, reqL)

	var lResp struct {
		MatchedType   string                 `json:"matched_type"`
		MatchedEntity map[string]interface{} `json:"matched_entity"`
	}
	if err := json.Unmarshal(wL.Body.Bytes(), &lResp); err != nil {
		t.Fatalf("error unmarshaling: %v", err)
	}
	if lResp.MatchedType != "conjunto" {
		t.Errorf("esperado tipo conjunto, obtenido %s", lResp.MatchedType)
	}
	if lResp.MatchedEntity["nombre"] != "Tornillos M3x10" {
		t.Errorf("nombre incorrecto: %v", lResp.MatchedEntity["nombre"])
	}

	// Lookup de producto demo de escaner_codigo_barras (Coca-Cola 7501031311309)
	reqDemo := httptest.NewRequest("GET", "/api/scanner/lookup/7501031311309", nil)
	wDemo := httptest.NewRecorder()
	r.ServeHTTP(wDemo, reqDemo)

	var dResp struct {
		MatchedType string                 `json:"matched_type"`
		Product     *scanner.SampleProduct `json:"product"`
	}
	if err := json.Unmarshal(wDemo.Body.Bytes(), &dResp); err != nil {
		t.Fatalf("error parseando demo: %v", err)
	}
	if dResp.Product == nil || dResp.Product.Name != "Refresco Coca-Cola 600ml" {
		t.Errorf("esperado producto demo Coca-Cola, obtenido: %+v", dResp.Product)
	}
}

func TestBarcodeGeneration(t *testing.T) {
	r, db, hub := setupScannerTestRouter(t)
	defer db.Close()
	defer hub.Close()

	// 1. PNG Code 128
	reqPNG := httptest.NewRequest("GET", "/api/barcode/code128/SKU-LAPTOP-M3.png", nil)
	wPNG := httptest.NewRecorder()
	r.ServeHTTP(wPNG, reqPNG)

	if wPNG.Code != http.StatusOK {
		t.Errorf("esperado 200 PNG, obtenido %d: %s", wPNG.Code, wPNG.Body.String())
	}
	if wPNG.Header().Get("Content-Type") != "image/png" {
		t.Errorf("esperado Content-Type image/png, obtenido %s", wPNG.Header().Get("Content-Type"))
	}
	if wPNG.Body.Len() < 50 {
		t.Errorf("cuerpo PNG demasiado pequeño: %d bytes", wPNG.Body.Len())
	}

	// 2. SVG EAN-13
	reqSVG := httptest.NewRequest("GET", "/api/barcode/ean13/7501031311309.svg", nil)
	wSVG := httptest.NewRecorder()
	r.ServeHTTP(wSVG, reqSVG)

	if wSVG.Code != http.StatusOK {
		t.Errorf("esperado 200 SVG, obtenido %d: %s", wSVG.Code, wSVG.Body.String())
	}
	if wSVG.Header().Get("Content-Type") != "image/svg+xml" {
		t.Errorf("esperado Content-Type image/svg+xml, obtenido %s", wSVG.Header().Get("Content-Type"))
	}
	if !bytes.Contains(wSVG.Body.Bytes(), []byte("<svg")) {
		t.Errorf("SVG inválido, no contiene tag <svg")
	}
}
