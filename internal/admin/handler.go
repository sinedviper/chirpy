package admin

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sinedviper/chirpy/internal/database"
	"github.com/sinedviper/chirpy/internal/response"
)

// HandleMetrics godoc
// @Summary      Hit counter
// @Description  Returns an HTML page showing how many times the server has been hit
// @Tags         admin
// @Produce      html
// @Success      200 {string} string "HTML page"
// @Router       /admin/metrics [get]
func HandleMetrics(numberRequests int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.WriteTextHtml(w, 200, fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", numberRequests))
	}
}

// HandleReset godoc
// @Summary      Reset app state
// @Description  Resets the hit counter and deletes all users. Only available when PLATFORM=dev
// @Tags         admin
// @Produce      plain
// @Success      200 {string} string "Hits: N"
// @Failure      403 {object} response.Error "Not running in dev mode"
// @Router       /admin/reset [post]
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

// HandleHealth godoc
// @Summary      Health check
// @Description  Returns OK if the server is up
// @Tags         health
// @Produce      plain
// @Success      200 {string} string "OK"
// @Router       /api/healthz [get]
func HandleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.WriteTextPlain(w, 200, "OK")
	}
}
