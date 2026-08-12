package main

import (
	"database/sql"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "github.com/sinedviper/chirpy/docs"
	"github.com/sinedviper/chirpy/internal/admin"
	"github.com/sinedviper/chirpy/internal/auth"
	"github.com/sinedviper/chirpy/internal/chirps"
	"github.com/sinedviper/chirpy/internal/database"
	"github.com/sinedviper/chirpy/internal/middleware"
	"github.com/sinedviper/chirpy/internal/users"
	"github.com/sinedviper/chirpy/internal/webhooks"
	httpSwagger "github.com/swaggo/http-swagger"
)

import (
	"log"
	"net/http"
)

type apiConfig struct {
	middleware *middleware.Middleware
	queries    *database.Queries
}

// @title           Chirpy API
// @version         1.0
// @description     Twitter-like microblogging API built with Go, PostgreSQL and sqlc.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT access token, format: "Bearer {token}"
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Polka webhook key, format: "ApiKey {key}"
func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env:", err)
	}

	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	publicKey := os.Getenv("PUBLIC_KEY")
	polkaKey := os.Getenv("POLKA_KEY")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	queries := database.New(db)

	cfg := &apiConfig{
		middleware: &middleware.Middleware{},
		queries:    queries,
	}

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// SWAGGER
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// WEBHOOK
	mux.Handle("POST /api/polka/webhooks", cfg.middleware.AuthenticationWebhook(cfg.middleware.Inc(webhooks.HandleChirpyUpgrade(cfg.queries)), polkaKey))

	// SITE
	mux.Handle("/", cfg.middleware.Inc(http.StripPrefix("/app", http.FileServer(http.Dir("./app")))))
	mux.Handle("/assets", cfg.middleware.Inc(http.StripPrefix("/app", http.FileServer(http.Dir("./app/assets")))))

	// API
	mux.HandleFunc("GET /api/healthz", admin.HandleHealth())
	mux.Handle("POST /api/login", cfg.middleware.Inc(auth.HandleLogin(cfg.queries, publicKey)))
	mux.Handle("POST /api/refresh", cfg.middleware.Inc(auth.HandleRefresh(cfg.queries, publicKey)))
	mux.Handle("POST /api/revoke", cfg.middleware.Inc(auth.HandleRevoke(cfg.queries)))

	// ADMIN
	mux.HandleFunc("GET /admin/metrics", admin.HandleMetrics(cfg.middleware.Hits.Load()))
	mux.Handle("POST /admin/reset", cfg.middleware.Reset(admin.HandleReset(cfg.queries, platform, cfg.middleware.Hits.Load())))

	// CHIRPS
	mux.Handle("POST /api/chirps", cfg.middleware.Authentication(cfg.middleware.Inc(chirps.HandleCreate(cfg.queries)), publicKey))
	mux.Handle("DELETE /api/chirps/{chirpID}", cfg.middleware.Authentication(cfg.middleware.Inc(chirps.HandleDelete(cfg.queries)), publicKey))
	mux.Handle("GET /api/chirps", cfg.middleware.Inc(chirps.HandleGetAll(cfg.queries)))
	mux.Handle("GET /api/chirps/{chirpID}", cfg.middleware.Inc(chirps.HandleGetOne(cfg.queries)))

	// USERS
	mux.Handle("POST /api/users", cfg.middleware.Inc(users.HandleCreate(cfg.queries)))
	mux.Handle("PUT /api/users", cfg.middleware.Authentication(cfg.middleware.Inc(users.HandlePut(cfg.queries)), publicKey))

	log.Print("Listening on port http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}
