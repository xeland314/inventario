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
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	dbpkg "inventario-backend/internal/db"
	"inventario-backend/internal/handlers"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func seedDefaultAdmin(db *sql.DB) {
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM usuarios`).Scan(&cnt); err != nil {
		log.Printf("WARN seed check: %v", err)
		return
	}
	if cnt > 0 {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("Admin123!"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("WARN seed hash: %v", err)
		return
	}
	id := uuid.NewString()
	_, err = db.Exec(`INSERT INTO usuarios (id, username, password_hash, rol, activo) VALUES (?, ?, ?, 'tecnico', 1)`, id, "admin", string(hash))
	if err != nil {
		log.Printf("WARN seed insert: %v", err)
		return
	}
	log.Printf("✓ Usuario admin creado (username: admin password: Admin123! rol: tecnico)")
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
	seedDefaultAdmin(database)

	os.MkdirAll(storageFolder, 0700)
	os.MkdirAll("./uploads/pdf", 0755)

	// --- Handlers ---
	dispHandler := &handlers.DispositivoHandler{DB: database}
	mantHandler := &handlers.MantenimientoHandler{DB: database}
	tintaHandler := &handlers.TintaHandler{DB: database}
	evHandler := &handlers.EvidenciaHandler{DB: database, MasterKey: []byte(encryptionKey), StorageFolder: storageFolder}
	movHandler := &handlers.MovimientoHandler{DB: database}
	etiquetaHandler := &handlers.EtiquetaHandler{DB: database}
	authHandler := &handlers.AuthHandler{DB: database}
	reporteHandler := &handlers.ReporteHandler{DB: database}

	// --- Router ---
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Auth-Token"},
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
		// Públicos
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Protegidos — requieren token
		r.Group(func(r chi.Router) {
			r.Use(authHandler.Middleware)

			// Auth propias
			r.Get("/auth/me", authHandler.Me)
			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/usuarios", authHandler.ListarUsuarios)
			r.Delete("/usuarios/{id}", authHandler.EliminarUsuario)

			// Reportes — lector puede crear y cancelar propio
			r.Get("/reportes", reporteHandler.Listar)
			r.Get("/reportes/{id}", reporteHandler.Obtener)
			r.Post("/reportes", reporteHandler.Crear)
			r.Post("/reportes/{id}/cancelar", reporteHandler.Cancelar)
			r.Post("/reportes/{id}/resolver", reporteHandler.Resolver)

			// Dispositivos — GET para todos, escritura solo tecnico
			r.Get("/dispositivos", dispHandler.Listar)
			r.Get("/dispositivos/{id}", dispHandler.Obtener)
			r.Get("/dispositivos/{id}/componentes", dispHandler.ListarComponentes)
			r.Get("/dispositivos/etiqueta/{id}", func(w http.ResponseWriter, req *http.Request) {
				etiquetaHandler.Escanear(w, req)
			})
			// tecnico only
			r.Group(func(r chi.Router) {
				r.Use(handlers.RequireRole("tecnico"))
				r.Post("/dispositivos", dispHandler.Crear)
				r.Put("/dispositivos/{id}", dispHandler.Actualizar)
				r.Patch("/dispositivos/{id}", dispHandler.Actualizar)
				r.Patch("/dispositivos/{id}/baja", dispHandler.DarDeBaja)
				r.Delete("/dispositivos/{id}", dispHandler.Eliminar)

				// Movimientos escritura
				r.Post("/movimientos", movHandler.Crear)
				r.Delete("/movimientos/{id}", movHandler.Eliminar)

				// Etiquetas escritura
				r.Post("/etiquetas/generar", etiquetaHandler.Generar)
				r.Post("/etiquetas/{id}/asociar", etiquetaHandler.Asociar)
				r.Post("/etiquetas/{id}/desasociar", etiquetaHandler.Desasociar)
				r.Post("/etiquetas/{id}/impreso", etiquetaHandler.MarcarImpreso)
				r.Post("/etiquetas/{id}/anular", etiquetaHandler.Anular)
				r.Delete("/etiquetas/{id}", etiquetaHandler.Eliminar)

				// Mantenimientos escritura
				r.Post("/mantenimientos", mantHandler.Crear)
				r.Handle("/mantenimiento", http.HandlerFunc(mantHandler.Crear))
				r.Put("/mantenimientos/{id}", mantHandler.Actualizar)
				r.Patch("/mantenimientos/{id}", mantHandler.Actualizar)
				r.Delete("/mantenimientos/{id}", mantHandler.Eliminar)

				// Tintas escritura
				r.Put("/impresoras/{id}/tintas", tintaHandler.Actualizar)
				r.Put("/dispositivos/{id}/tintas", tintaHandler.Actualizar)

				// Evidencia escritura
				r.Post("/evidencia/upload", evHandler.Upload)
				r.Delete("/evidencia/{id}", evHandler.Eliminar)
			})

			// Lectura permitida a lector y tecnico
			r.Get("/movimientos", movHandler.Listar)
			r.Get("/movimientos/{id}", movHandler.Obtener)

			r.Get("/etiquetas", etiquetaHandler.Listar)
			r.Get("/etiquetas/{id}", etiquetaHandler.Obtener)
			r.Get("/etiquetas/scan/{id}", etiquetaHandler.Escanear)
			r.Get("/escaner/{id}", etiquetaHandler.Escanear)

			r.Get("/mantenimientos", mantHandler.Listar)
			r.Get("/mantenimientos/{id}", mantHandler.Obtener)

			r.Get("/impresoras/{id}/tintas", tintaHandler.Obtener)
			r.Get("/dispositivos/{id}/tintas", tintaHandler.Obtener)

			r.Get("/evidencia/galeria", evHandler.Galeria)
			r.Get("/galeria", evHandler.Galeria)
			r.Get("/evidencia/{id}/raw", evHandler.ServirRaw)
			r.Get("/evidencia/{id}", evHandler.ServirRaw)
		})
	})

	// Servir PDFs (solo lectura, sin cifrado) — requiere auth? público para PDF reportes, dejamos público
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// --- Frontend estático servido por Go ---
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
		if st, err := os.Stat(filepath.Join(".", c)); err == nil && st.IsDir() {
			frontendDist, _ = filepath.Abs(filepath.Join(".", c))
			break
		}
	}
	if frontendDist == "" {
		frontendDist, _ = filepath.Abs("../frontend/dist")
	}
	if st, err := os.Stat(frontendDist); err == nil && st.IsDir() {
		log.Printf("✓ Frontend estático: %s", frontendDist)
		fileServer := http.FileServer(http.Dir(frontendDist))
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			path := req.URL.Path
			fullPath := filepath.Join(frontendDist, filepath.FromSlash(path))
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, req)
				return
			}
			if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
				if _, err := os.Stat(filepath.Join(fullPath, "index.html")); err == nil {
					http.ServeFile(w, req, filepath.Join(fullPath, "index.html"))
					return
				}
			}
			if strings.HasPrefix(path, "/dispositivos/") {
				placeholder := filepath.Join(frontendDist, "dispositivos", "placeholder", "index.html")
				if _, err := os.Stat(placeholder); err == nil {
					http.ServeFile(w, req, placeholder)
					return
				}
			}
			index := filepath.Join(frontendDist, "index.html")
			if _, err := os.Stat(index); err == nil {
				http.ServeFile(w, req, index)
				return
			}
			http.NotFound(w, req)
		})
	} else {
		log.Printf("⚠ Frontend no encontrado en %s — ejecuta 'npm run build' en frontend/", frontendDist)
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
