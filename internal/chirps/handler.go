package chirps

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/sinedviper/chirpy/internal/database"
	"github.com/sinedviper/chirpy/internal/response"
)

// HandleCreate godoc
// @Summary      Create a chirp
// @Description  Creates a new chirp (max 140 chars) for the authenticated user, filtering out profanity
// @Tags         chirps
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "Chirp body"
// @Security     BearerAuth
// @Success      201 {object} ChirpResponse
// @Failure      400 {object} response.Error "Chirp too long"
// @Failure      404 {object} response.Error "User not found"
// @Failure      500 {object} response.Error
// @Router       /api/chirps [post]
func HandleCreate(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateRequest

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			response.WriteError(w, err, 500, "Something went wrong")
			return
		}

		if len(req.Body) > 140 {
			response.WriteError(w, nil, 400, "Chirp is too long")
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

		userID := r.Context().Value("userID").(uuid.UUID)

		findUser, errFindUser := queries.FindUserById(r.Context(), userID)
		if errFindUser != nil {
			response.WriteError(w, errFindUser, 404, "User not found")
			return
		}

		createChirp, errCreateChirp := queries.CreateChirp(r.Context(), database.CreateChirpParams{Body: req.Body, UserID: uuid.NullUUID{UUID: findUser.ID, Valid: true}})
		if errCreateChirp != nil {
			response.WriteError(w, errCreateChirp, 500, "Something went wrong with create a chirp")
			return
		}

		response.WriteJSON(w, http.StatusCreated, ChirpResponse{
			ID:        createChirp.ID.String(),
			CreatedAt: createChirp.CreatedAt,
			UpdatedAt: createChirp.UpdatedAt,
			Body:      createChirp.Body,
			UserId:    userID.String(),
		})
	}
}

// HandleGetAll godoc
// @Summary      List chirps
// @Description  Lists all chirps, optionally filtered by author and sorted by creation date
// @Tags         chirps
// @Produce      json
// @Param        author_id query string false "Filter by author UUID"
// @Param        sort      query string false "Sort direction: asc (default) or desc"
// @Success      200 {array} ChirpResponse
// @Failure      400 {object} response.Error "Invalid author_id"
// @Failure      404 {object} response.Error "Can't get chirps"
// @Router       /api/chirps [get]
func HandleGetAll(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorIDStr := r.URL.Query().Get("author_id")
		sortStr := r.URL.Query().Get("sort")

		sort := "asc"
		var getChirps []database.Chirp

		if sortStr != "" {
			if sortStr == "asc" || sortStr == "desc" {
				sort = sortStr
			}
		}

		if authorIDStr != "" {
			authorID, err := uuid.Parse(authorIDStr)
			if err != nil {
				response.WriteError(w, err, 400, "Invalid author_id")
				return
			}
			getChirpsByAuthor, errGetChirps := queries.GetChirpsByAuthor(r.Context(), database.GetChirpsByAuthorParams{UserID: uuid.NullUUID{UUID: authorID, Valid: true}, SortDirection: sort})
			if errGetChirps != nil {
				response.WriteError(w, errGetChirps, 404, "Can't get chirps")
				return
			}
			getChirps = getChirpsByAuthor
		} else {
			getChirpsAll, errGetChirps := queries.GetChirps(r.Context(), sort)
			if errGetChirps != nil {
				response.WriteError(w, errGetChirps, 404, "Can't get chirps")
				return
			}

			getChirps = getChirpsAll
		}

		var respBody []ChirpResponse

		for _, chirp := range getChirps {
			respBody = append(respBody, ChirpResponse{
				ID:        chirp.ID.String(),
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserId:    chirp.UserID.UUID.String(),
			})
		}

		response.WriteJSON(w, http.StatusOK, respBody)
	}
}

// HandleGetOne godoc
// @Summary      Get a chirp
// @Description  Returns a single chirp by ID
// @Tags         chirps
// @Produce      json
// @Param        chirpID path string true "Chirp UUID"
// @Success      200 {object} ChirpResponse
// @Failure      400 {object} response.Error "Invalid chirpID"
// @Failure      404 {object} response.Error "Not found"
// @Router       /api/chirps/{chirpID} [get]
func HandleGetOne(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := uuid.Parse(r.PathValue("chirpID"))
		if err != nil {
			response.WriteError(w, err, 400, "Invalid chirpID")
			return
		}

		getChirp, errGetChirp := queries.GetChirp(r.Context(), chirpID)
		if errGetChirp != nil {
			response.WriteError(w, errGetChirp, 404, "Not found chirp")
			return
		}

		response.WriteJSON(w, http.StatusOK, ChirpResponse{
			ID:        getChirp.ID.String(),
			CreatedAt: getChirp.CreatedAt,
			UpdatedAt: getChirp.UpdatedAt,
			Body:      getChirp.Body,
			UserId:    getChirp.UserID.UUID.String(),
		})
	}
}

// HandleDelete godoc
// @Summary      Delete a chirp
// @Description  Deletes a chirp. Only the chirp's author may delete it
// @Tags         chirps
// @Param        chirpID path string true "Chirp UUID"
// @Security     BearerAuth
// @Success      204 "No Content"
// @Failure      400 {object} response.Error "Invalid chirpID"
// @Failure      403 {object} response.Error "Not the chirp's author"
// @Failure      404 {object} response.Error "User or chirp not found"
// @Router       /api/chirps/{chirpID} [delete]
func HandleDelete(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chirpID, err := uuid.Parse(r.PathValue("chirpID"))
		if err != nil {
			response.WriteError(w, err, 400, "Invalid chirpID")
			return
		}

		userID := r.Context().Value("userID").(uuid.UUID)

		findUser, errFindUser := queries.FindUserById(r.Context(), userID)
		if errFindUser != nil {
			response.WriteError(w, errFindUser, 404, "User not found")
			return
		}

		getChirp, errGetChirp := queries.GetChirp(r.Context(), chirpID)
		if errGetChirp != nil {
			response.WriteError(w, errGetChirp, 404, "Not found chirp")
			return
		}

		if getChirp.UserID.UUID != findUser.ID {
			response.WriteError(w, nil, 403, "This user can't delete this chirp")
			return
		} else {
			err := queries.DeleteChirpById(r.Context(), chirpID)
			if err != nil {
				response.WriteError(w, err, 500, "Something went wrong")
				return
			}
		}

		w.WriteHeader(204)
	}
}
