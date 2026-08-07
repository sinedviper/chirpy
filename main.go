package main

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
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg := &apiConfig{}

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

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", cfg.fileserverHits.Load())))
	})
	mux.HandleFunc("/admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg.fileserverHits.Store(0)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(fmt.Sprintf("Hits: %d\n", cfg.fileserverHits.Load())))
	})
	mux.HandleFunc("/api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg.fileserverHits.Add(1)

		type request struct {
			Body string `json:"body"`
		}

		var req request

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")

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
			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		type returnVal struct {
			CleanedBody string `json:"cleaned_body"`
		}

		respBody := returnVal{
			CleanedBody: strings.Join(cleanedBody, " "),
		}

		resp, _ := json.Marshal(respBody)
		w.Write(resp)
	})

	log.Print("Listening on port http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}
