package admin

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sinedviper/chirpy/internal/database"
	"github.com/sinedviper/chirpy/internal/response"
)

func HandleMetrics(numberRequests int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.WriteTextHtml(w, 200, fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", numberRequests))
	}
}

func HandleReset(queries *database.Queries, platform string, numberRequests int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if platform != "dev" {
			response.WriteError(w, nil, 403, "Method not allowed")
			return
		}

		err := queries.RemoveUsers(r.Context())
		if err != nil {
			log.Println("Error removing users:", err)
		}

		response.WriteTextPlain(w, 200, fmt.Sprintf("Hits: %d\n", numberRequests))
	}
}

func HandleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.WriteTextPlain(w, 200, "OK")
	}
}
