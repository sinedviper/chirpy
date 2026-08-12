package users

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/sinedviper/chirpy/internal/auth"
	"github.com/sinedviper/chirpy/internal/database"
	"github.com/sinedviper/chirpy/internal/response"
)

// HandleCreate godoc
// @Summary      Create a user
// @Description  Registers a new user with an email and password
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "New user"
// @Success      201 {object} UserResponse
// @Failure      500 {object} response.Error
// @Router       /api/users [post]
func HandleCreate(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateRequest

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			response.WriteError(w, err, 500, "Something went parse json")
			return
		}

		hashedPassword, errHash := auth.HashPassword(req.Password)
		if errHash != nil {
			response.WriteError(w, errHash, 500, "Something went hash password")
			return
		}

		userCreate, errCreate := queries.CreateUser(r.Context(), database.CreateUserParams{
			Email: sql.NullString{
				String: req.Email,
				Valid:  true,
			},
			HashedPassword: hashedPassword,
		})

		if errCreate != nil {
			response.WriteError(w, errCreate, 500, "Something went create the user")
			return
		}

		response.WriteJSON(w, 201, UserResponse{
			ID:          userCreate.ID.String(),
			CreatedAt:   userCreate.CreatedAt,
			UpdatedAt:   userCreate.UpdatedAt,
			Email:       userCreate.Email.String,
			IsChirpyRed: userCreate.IsChirpyRed,
		})
	}
}

// HandlePut godoc
// @Summary      Update the authenticated user
// @Description  Updates the email/password of the currently authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body PutRequest true "Updated user fields"
// @Security     BearerAuth
// @Success      200 {object} UserResponse
// @Failure      404 {object} response.Error "User not found"
// @Failure      500 {object} response.Error
// @Router       /api/users [put]
func HandlePut(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID").(uuid.UUID)

		findUser, errFindUser := queries.FindUserById(r.Context(), userID)
		if errFindUser != nil {
			response.WriteError(w, errFindUser, 404, "User not found")
			return
		}

		var req PutRequest

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			response.WriteError(w, err, 500, "Something went parse json")
			return
		}

		hashedPassword, errHash := auth.HashPassword(req.Password)
		if errHash != nil {
			response.WriteError(w, errHash, 500, "Something went hash password")
			return
		}

		errUpdate := queries.UpdateUsers(r.Context(), database.UpdateUsersParams{
			Email: sql.NullString{
				String: req.Email,
				Valid:  true,
			},
			HashedPassword: hashedPassword,
			ID:             findUser.ID,
		})

		if errUpdate != nil {
			response.WriteError(w, errUpdate, 500, "Something went create the user")
			return
		}

		findUser, errFindUser = queries.FindUserById(r.Context(), userID)
		if errFindUser != nil {
			response.WriteError(w, errFindUser, 404, "User not found")
			return
		}

		response.WriteJSON(w, 200, UserResponse{
			ID:          findUser.ID.String(),
			CreatedAt:   findUser.CreatedAt,
			UpdatedAt:   findUser.UpdatedAt,
			Email:       findUser.Email.String,
			IsChirpyRed: findUser.IsChirpyRed,
		})
	}
}
