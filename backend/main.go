package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/mattn/go-sqlite3"

	dbpkg "inventario-backend/internal/db"
	"inventario-backend/internal/handlers"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// --- Config ---
	dbPath := getEnv("DB_PATH", "./inventario.db")
	storageFolder := getEnv("STORAGE_PATH", "./encrypted_storage")
	encryptionKey := getEnv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef") // 32 bytes
	port := getEnv("PORT", "8080")

	if len(encryptionKey) != 32 {
		log.Fatalf("ENCRYPTION_KEY debe tener 32 bytes, tiene %d", len(encryptionKey))
	}

	// --- DB ---
	database, err := sql.Open("sqlite3", dbPath+"?_fk=1&_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := dbpkg.InitDB(database); err != nil {
		log.Fatal(err)
	}

	os.MkdirAll(storageFolder, 0700)
	os.MkdirAll("./uploads/pdf", 0755)

	// --- Handlers ---
	dispHandler := &handlers.DispositivoHandler{DB: database}
	mantHandler := &handlers.MantenimientoHandler{DB: database}
	tintaHandler := &handlers.TintaHandler{DB: database}
	evHandler := &handlers.EvidenciaHandler{DB: database, MasterKey: []byte(encryptionKey), StorageFolder: storageFolder}
	movHandler := &handlers.MovimientoHandler{DB: database}

	// --- Router ---
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"inventario-backend"}`))
	})

	// API
	r.Route("/api", func(r chi.Router) {
		// Dispositivos
		r.Get("/dispositivos", dispHandler.Listar)
		r.Post("/dispositivos", dispHandler.Crear)
		r.Get("/dispositivos/{id}", dispHandler.Obtener)
		r.Put("/dispositivos/{id}", dispHandler.Actualizar)
		r.Patch("/dispositivos/{id}", dispHandler.Actualizar)
		r.Patch("/dispositivos/{id}/baja", dispHandler.DarDeBaja)
		r.Delete("/dispositivos/{id}", dispHandler.Eliminar)
		r.Get("/dispositivos/{id}/componentes", dispHandler.ListarComponentes)

		// Movimientos / historial trazabilidad componentes
		r.Get("/movimientos", movHandler.Listar)
		r.Post("/movimientos", movHandler.Crear)
		r.Get("/movimientos/{id}", movHandler.Obtener)
		r.Delete("/movimientos/{id}", movHandler.Eliminar)

		// Mantenimientos
		r.Get("/mantenimientos", mantHandler.Listar)
		r.Post("/mantenimientos", mantHandler.Crear)
		r.Handle("/mantenimiento", http.HandlerFunc(mantHandler.Crear)) // alias singular
		r.Get("/mantenimientos/{id}", mantHandler.Obtener)
		r.Put("/mantenimientos/{id}", mantHandler.Actualizar)
		r.Patch("/mantenimientos/{id}", mantHandler.Actualizar)
		r.Delete("/mantenimientos/{id}", mantHandler.Eliminar)

		// Tintas - ambas rutas por compatibilidad
		r.Get("/impresoras/{id}/tintas", tintaHandler.Obtener)
		r.Put("/impresoras/{id}/tintas", tintaHandler.Actualizar)
		r.Get("/dispositivos/{id}/tintas", tintaHandler.Obtener)
		r.Put("/dispositivos/{id}/tintas", tintaHandler.Actualizar)

		// Evidencia cifrada
		r.Post("/evidencia/upload", evHandler.Upload)
		r.Get("/evidencia/galeria", evHandler.Galeria)
		r.Get("/galeria", evHandler.Galeria) // alias
		r.Get("/evidencia/{id}/raw", evHandler.ServirRaw)
		r.Delete("/evidencia/{id}", evHandler.Eliminar)

		// Aliases adicionales para frontend Astro
		r.Get("/evidencia/{id}", evHandler.ServirRaw)
	})

	// Servir PDFs (solo lectura, sin cifrado)
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// --- Frontend estático servido por Go ---
	// Busca dist en varias ubicaciones relativas (dev vs binary)
	candidates := []string{
		getEnv("FRONTEND_DIST", ""),
		"../frontend/dist",
		"./frontend/dist",
		"frontend/dist",
		filepath.Join(".", "dist"),
	}
	frontendDist := ""
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			frontendDist, _ = filepath.Abs(c)
			break
		}
		// También probar ruta relativa al cwd
		if st, err := os.Stat(filepath.Join(".", c)); err == nil && st.IsDir() {
			frontendDist, _ = filepath.Abs(filepath.Join(".", c))
			break
		}
	}
	if frontendDist == "" {
		// fallback por defecto
		frontendDist, _ = filepath.Abs("../frontend/dist")
	}
	if st, err := os.Stat(frontendDist); err == nil && st.IsDir() {
		log.Printf("✓ Frontend estático: %s", frontendDist)
		// FileServer para assets
		fileServer := http.FileServer(http.Dir(frontendDist))
		// Handler SPA con fallback
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			// Evitar colisión con /api y /health ya definidos (chi las matchea antes)
			path := req.URL.Path
			// Intentar servir archivo exacto
			fullPath := filepath.Join(frontendDist, filepath.FromSlash(path))
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, req)
				return
			}
			// Si es directorio con index.html
			if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
				if _, err := os.Stat(filepath.Join(fullPath, "index.html")); err == nil {
					http.ServeFile(w, req, filepath.Join(fullPath, "index.html"))
					return
				}
			}
			// Fallback para rutas dinámicas /dispositivos/:id → servir placeholder pre-render
			if strings.HasPrefix(path, "/dispositivos/") {
				placeholder := filepath.Join(frontendDist, "dispositivos", "placeholder", "index.html")
				if _, err := os.Stat(placeholder); err == nil {
					http.ServeFile(w, req, placeholder)
					return
				}
			}
			// Fallback general SPA → index.html
			index := filepath.Join(frontendDist, "index.html")
			if _, err := os.Stat(index); err == nil {
				http.ServeFile(w, req, index)
				return
			}
			http.NotFound(w, req)
		})
	} else {
		log.Printf("⚠ Frontend no encontrado en %s — ejecuta 'npm run build' en frontend/", frontendDist)
		// Ruta informativa
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(`<h1>Frontend no compilado</h1><p>Ejecuta <code>npm run build</code> en <code>frontend/</code> y reinicia Go.</p><p><a href="/api/dispositivos">Probar API</a> | <a href="/health">health</a></p>`))
		})
	}

	log.Printf("✓ Servidor Go iniciado")
	log.Printf("  DB: %s", dbPath)
	log.Printf("  Storage cifrado: %s", storageFolder)
	log.Printf("  Puerto: :%s", port)
	if frontendDist != "" {
		log.Printf("  Frontend: http://localhost:%s/ (servido por Go)", port)
	}
	log.Printf("  API: http://localhost:%s/api/dispositivos", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
