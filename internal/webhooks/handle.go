package webhooks

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/sinedviper/chirpy/internal/database"
	"github.com/sinedviper/chirpy/internal/response"
)

// HandleChirpyUpgrade godoc
// @Summary      Polka webhook
// @Description  Handles Polka's "user.upgraded" event to mark a user as Chirpy Red. Other events are ignored
// @Tags         webhooks
// @Accept       json
// @Param        request body WebhookChirpyRequest true "Webhook payload"
// @Security     ApiKeyAuth
// @Success      204 "No Content"
// @Failure      400 {object} response.Error "Invalid user_id"
// @Failure      404 {object} response.Error "User not found"
// @Failure      500 {object} response.Error
// @Router       /api/polka/webhooks [post]
func HandleChirpyUpgrade(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req WebhookChirpyRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			response.WriteError(w, err, 500, "Something went wrong while parsing the request body")
			return
		}

		if req.EVENT == "user.upgraded" {
			uuidUser, errUuidUser := uuid.Parse(req.DATA.USERID)
			if errUuidUser != nil {
				response.WriteError(w, errUuidUser, 400, "User ID should be a correct")
			}

			errFindUser := queries.UpdateChirpyUsers(r.Context(), database.UpdateChirpyUsersParams{ID: uuidUser, IsChirpyRed: true})
			if errFindUser != nil {
				response.WriteError(w, errFindUser, 404, "User not found")
				return
			}
		}

		w.WriteHeader(204)
		return
	}
}
