package main

import (
	"database/sql"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sinedviper/chirpy/internal/database"
)

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env:", err)
	}

	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	queries := database.New(db)

	cfg := &apiConfig{
		queries: queries,
	}

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	mux.Handle("/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./app")))))
	mux.Handle("/assets", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./app/assets")))))
	mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", cfg.fileserverHits.Load())))
	})
	mux.HandleFunc("/admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		if platform != "dev" {
			http.Error(w, "Method Not Found", http.StatusForbidden)
			return
		}

		err := cfg.queries.RemoveUsers(r.Context())
		if err != nil {
			log.Println("Error removing users:", err)
		}
		cfg.fileserverHits.Store(0)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Hits: %d\n", cfg.fileserverHits.Load())))
	})
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg.fileserverHits.Add(1)

		type request struct {
			Body   string `json:"body"`
			UserId string `json:"user_id"`
		}

		var req request

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)

			type returnVal struct {
				Error string `json:"error"`
			}

			respBody := returnVal{
				Error: "Something went wrong",
			}

			resp, _ := json.Marshal(respBody)
			w.Write(resp)
			return
		}

		if len(req.Body) > 140 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			type returnVal struct {
				Error string `json:"error"`
			}

			respBody := returnVal{
				Error: "Chirp is too long",
			}

			resp, _ := json.Marshal(respBody)
			w.Write(resp)
			return
		}

		badWords := []string{"kerfuffle", "sharbert", "fornax"}
		var cleanedBody []string

		for _, word := range strings.Split(req.Body, " ") {
			var isContained string
			for _, badWord := range badWords {
				if strings.Contains(strings.ToLower(word), strings.ToLower(badWord)) {
					isContained = badWord
				}
			}
			if isContained != "" {
				cleanedBody = append(cleanedBody, "****")
			} else {
				cleanedBody = append(cleanedBody, word)
			}
		}

		findUser, errFindUser := cfg.queries.FindUser(r.Context(), uuid.MustParse(req.UserId))
		if errFindUser != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(404)
			type returnVal struct {
				Error string `json:"error"`
			}
			respBody := returnVal{
				Error: "User not found",
			}
			resp, _ := json.Marshal(respBody)
			w.Write(resp)
			return
		}

		createChirp, errCreateChirp := cfg.queries.CreateChirp(r.Context(), database.CreateChirpParams{Body: req.Body, UserID: uuid.NullUUID{UUID: findUser.ID, Valid: true}})
		if errCreateChirp != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			type returnVal struct {
				Error string `json:"error"`
			}
			respBody := returnVal{
				Error: "Something went wrong with create a chirp",
			}
			resp, _ := json.Marshal(respBody)
			w.Write(resp)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		type returnVal struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserId    string    `json:"user_id"`
		}

		respBody := returnVal{
			ID:        createChirp.ID.String(),
			CreatedAt: createChirp.CreatedAt,
			UpdatedAt: createChirp.UpdatedAt,
			Body:      createChirp.Body,
			UserId:    req.UserId,
		}

		resp, _ := json.Marshal(respBody)
		w.Write(resp)
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg.fileserverHits.Add(1)

		getChirps, errGetChirps := cfg.queries.GetChirps(r.Context())
		if errGetChirps != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(404)
			type returnVal struct {
				Error string `json:"error"`
			}
			respBody := returnVal{
				Error: "Can't get chirps",
			}
			resp, _ := json.Marshal(respBody)
			w.Write(resp)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		type returnVal struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserId    string    `json:"user_id"`
		}

		var respBody []returnVal

		for _, chirp := range getChirps {
			respBody = append(respBody, returnVal{
				ID:        chirp.ID.String(),
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserId:    chirp.UserID.UUID.String(),
			})
		}

		resp, _ := json.Marshal(respBody)
		w.Write(resp)
	})

	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg.fileserverHits.Add(1)

		chirpID, err := uuid.Parse(r.PathValue("chirpID"))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			type returnVal struct {
				Error string `json:"error"`
			}

			respBody := returnVal{
				Error: "Invalid chirpID",
			}

			resp, _ := json.Marshal(respBody)
			w.Write(resp)
			return
		}

		getChirp, errGetChirp := cfg.queries.GetChirp(r.Context(), chirpID)
		if errGetChirp != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(404)
			type returnVal struct {
				Error string `json:"error"`
			}
			respBody := returnVal{
				Error: "Not found chirp",
			}
			resp, _ := json.Marshal(respBody)
			w.Write(resp)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		type returnVal struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserId    string    `json:"user_id"`
		}

		resp, _ := json.Marshal(returnVal{
			ID:        getChirp.ID.String(),
			CreatedAt: getChirp.CreatedAt,
			UpdatedAt: getChirp.UpdatedAt,
			Body:      getChirp.Body,
			UserId:    getChirp.UserID.UUID.String(),
		})
		w.Write(resp)
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg.fileserverHits.Add(1)

		type request struct {
			Email string `json:"email"`
		}

		var req request

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)

			type returnVal struct {
				Error string `json:"error"`
			}

			respBody := returnVal{
				Error: "Something went wrong",
			}

			resp, _ := json.Marshal(respBody)
			w.Write(resp)
			return
		}

		userCreate, errCreate := cfg.queries.CreateUser(r.Context(), sql.NullString{
			String: req.Email,
			Valid:  true,
		})

		if errCreate != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			type returnVal struct {
				Error string `json:"error"`
			}
			respBody := returnVal{
				Error: "Something went wrong",
			}
			resp, _ := json.Marshal(respBody)
			w.Write(resp)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		type returnVal struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
		}

		respBody := returnVal{
			ID:        userCreate.ID.String(),
			CreatedAt: userCreate.CreatedAt,
			UpdatedAt: userCreate.UpdatedAt,
			Email:     userCreate.Email.String,
		}

		resp, _ := json.Marshal(respBody)
		w.Write(resp)
	})

	log.Print("Listening on port http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}
