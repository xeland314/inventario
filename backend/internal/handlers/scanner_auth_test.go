package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dbpkg "inventario-backend/internal/db"
	"inventario-backend/internal/scanner"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

func setupScannerAuth(t *testing.T) (*httptest.Server, string) {
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if err := dbpkg.InitDB(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO usuarios (id, username, password_hash, rol, creado_en, activo) VALUES ('u1','tester','x','tecnico',datetime('now'),1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sesiones (id, user_id, token, creado_en) VALUES ('s1','u1','tok-valido',datetime('now'))`); err != nil {
		t.Fatal(err)
	}
	auth := &AuthHandler{DB: db}
	hub := scanner.NewHub(db)
	t.Cleanup(hub.Close)
	sh := &ScannerHandler{DB: db, Hub: hub, Auth: auth}

	r := chi.NewRouter()
	r.HandleFunc("/ws", sh.WebSocketEndpoint)
	r.Get("/api/barcode/{format}/{code}.svg", sh.BarcodeSVG)
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Get("/api/scanner/lookup/{code}", sh.Lookup)
		r.Delete("/api/scanner/scans", sh.ClearScans)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
}

func TestScannerRoutesRequireAuth(t *testing.T) {
	srv, _ := setupScannerAuth(t)
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/scanner/lookup/abc"},
		{"DELETE", "/api/scanner/scans"},
	} {
		req, _ := http.NewRequest(c.method, srv.URL+c.path, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s sin token: esperaba 401, obtuve %d", c.method, c.path, res.StatusCode)
		}
		req, _ = http.NewRequest(c.method, srv.URL+c.path, nil)
		req.Header.Set("Authorization", "Bearer tok-valido")
		res, _ = http.DefaultClient.Do(req)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("%s %s con token: esperaba 200, obtuve %d", c.method, c.path, res.StatusCode)
		}
	}
	// El token por query string ya no se acepta
	res0, _ := http.Get(srv.URL + "/api/scanner/lookup/abc?token=tok-valido")
	res0.Body.Close()
	if res0.StatusCode != http.StatusUnauthorized {
		t.Errorf("?token= debe rechazarse, obtuve %d", res0.StatusCode)
	}
	res, _ := http.Get(srv.URL + "/api/barcode/code128/ABC123.svg")
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("barcode debe seguir público, obtuve %d", res.StatusCode)
	}
}

func TestScannerWebSocketAuth(t *testing.T) {
	_, wsURL := setupScannerAuth(t)

	// Token inválido: recibe error y se cierra, sin "init"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.WriteJSON(map[string]string{"type": "auth", "token": "falso"})
	var msg map[string]interface{}
	if err := c.ReadJSON(&msg); err != nil || msg["type"] != "error" {
		t.Fatalf("esperaba mensaje error, obtuve %v (%v)", msg, err)
	}
	c.Close()

	// Primer mensaje distinto de auth: rechazado
	c, _, _ = websocket.DefaultDialer.Dial(wsURL, nil)
	c.WriteJSON(map[string]string{"type": "ping"})
	msg = nil
	if err := c.ReadJSON(&msg); err != nil || msg["type"] != "error" {
		t.Fatalf("ping sin auth debe rechazarse, obtuve %v (%v)", msg, err)
	}
	c.Close()

	// Token válido: recibe init
	c, _, _ = websocket.DefaultDialer.Dial(wsURL, nil)
	defer c.Close()
	c.WriteJSON(map[string]string{"type": "auth", "token": "tok-valido"})
	msg = nil
	if err := c.ReadJSON(&msg); err != nil || msg["type"] != "init" {
		t.Fatalf("esperaba init, obtuve %v (%v)", msg, err)
	}
}
