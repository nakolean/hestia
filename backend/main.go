package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"artemis/admin"
	"artemis/api"
	"artemis/auth"
	"artemis/db"
	"artemis/middleware"
	"artemis/public"
	"artemis/services"
)

func main() {
	// Open database
	databasePath := os.Getenv("DB_PATH")
	if databasePath == "" {
		databasePath = "data/artemis.db"
	}
	sqlDB, err := db.Open(databasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()

	// Run migrations
	if err = db.Migrate(sqlDB); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Seed initial user from environment variables
	if err = db.SeedInitialUser(sqlDB); err != nil {
		log.Printf("Warning: failed to seed initial user: %v", err)
	}

	// Initialize components
	sessionStore := middleware.NewSessionStore(24 * time.Hour)
	rateLimiter := middleware.NewRateLimiter(10 * time.Minute)

	// Start reminder service
	reminderService := services.NewReminderService(sqlDB)
	reminderService.Start()
	defer reminderService.Stop()

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		MaxAge:           3600,
		AllowCredentials: false,
	}))

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Login / logout
	r.Post("/api/login", func(w http.ResponseWriter, req *http.Request) {
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		if input.Username == "" || input.Password == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "username and password required"})
			return
		}

		if !auth.ValidateLogin(sqlDB, input.Username, input.Password) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
			return
		}

		token := sessionStore.Add()
		middleware.SetSessionCookie(w, token)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	})
	r.Post("/api/logout", func(w http.ResponseWriter, req *http.Request) {
		middleware.ClearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	})

	// Public API routes — API key required, GET-only
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAPIKey(sqlDB, rateLimiter, 100.0, 100.0/60.0))
		r.Use(middleware.RequirePermission("read"))
		r.Use(public.AllowGETOnly)
		public.RegisterPublicChores(r, sqlDB)
		public.RegisterPublicItems(r, sqlDB)
	})

	// Session-authenticated routes (web UI)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireSession(sessionStore))
		api.RegisterChores(r, sqlDB)
		api.RegisterItems(r, sqlDB)
	})

	// Admin routes — session auth only
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireSession(sessionStore))
		admin.RegisterAPIKeys(r, sqlDB)
	})

	// SPA fallback — serve frontend (check ../frontend/dist for local dev, ./dist for Docker)
	frontendDir := "./dist"
	if _, err := os.Stat("../frontend/dist"); err == nil {
		frontendDir = "../frontend/dist"
	}

	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err := os.Stat(frontendDir + "/index.html"); os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"frontend not built (run npm run build in frontend/)","available_routes":["GET /health","POST /api/login","POST /api/logout","GET /api/chores (session)","POST /api/chores (session)","PUT /api/chores/:id (session)","DELETE /api/chores/:id (session)","POST /api/chores/:id/complete (session)","GET /api/chores/overdue (session)","GET /api/items (session)","POST /api/items (session)","DELETE /api/items/:id (session)","PATCH /api/items/:id (session)","GET /api/admin/keys (session)","POST /api/admin/keys (session)","PUT /api/admin/keys/:id (session)","DELETE /api/admin/keys/:id (session)","GET /api/v1/public/chores (key)","GET /api/v1/public/chores/overdue (key)","GET /api/v1/public/chores/next (key)","GET /api/v1/public/items (key)","GET /api/v1/public/items/pending (key)"]}`))
			return
		}
		requested := strings.TrimPrefix(req.URL.Path, "/")
		candidate := frontendDir + "/" + requested
		if _, err := os.Stat(candidate); err == nil {
			http.ServeFile(w, req, candidate)
			return
		}
		http.ServeFile(w, req, frontendDir+"/index.html")
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}