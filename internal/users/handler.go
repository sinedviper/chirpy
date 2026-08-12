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
