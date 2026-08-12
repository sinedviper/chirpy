package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sinedviper/chirpy/internal/database"
	"github.com/sinedviper/chirpy/internal/response"
)

const expiresAccess = time.Duration(60*60) * time.Second // 1 h

// HandleLogin godoc
// @Summary      Log in
// @Description  Authenticates a user by email/password and returns a JWT access token plus a refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} LoginResponse
// @Failure      401 {object} response.Error "Incorrect email or password"
// @Failure      500 {object} response.Error
// @Router       /api/login [post]
func HandleLogin(queries *database.Queries, publicSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			response.WriteError(w, err, 500, "Something went parse json")
			return
		}

		findUser, errFindUser := queries.FindUserByEmail(r.Context(), sql.NullString{
			String: req.Email,
			Valid:  true,
		})
		if errFindUser != nil {
			response.WriteError(w, errFindUser, 401, "Incorrect email or password")
			return
		}

		hashedPassword, errHash := CheckPasswordHash(req.Password, findUser.HashedPassword)
		if errHash != nil {
			response.WriteError(w, errHash, 500, "Something went hash password")
			return
		}

		if !hashedPassword {
			response.WriteError(w, nil, 401, "Incorrect email or password")
			return
		}

		token, errToken := MakeJWT(findUser.ID, publicSecret, expiresAccess)
		if errToken != nil {
			response.WriteError(w, errToken, 500, "Something went wrong with token")
			return
		}

		var expiresRefresh = 60 * 24 * time.Hour // 60 d
		refreshToken, err := MakeRefreshToken()
		if err != nil {
			response.WriteError(w, err, 500, "Something went wrong with make refresh token")
			return
		}

		createRefresh, errCreateRefresh := queries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			UserID:    uuid.NullUUID{UUID: findUser.ID, Valid: true},
			Token:     refreshToken,
			ExpiresAt: time.Now().Add(expiresRefresh),
		})
		if errCreateRefresh != nil {
			response.WriteError(w, errCreateRefresh, 500, "Something went wrong with create refresh token")
			return
		}

		response.WriteJSON(w, 200, LoginResponse{
			ID:           findUser.ID.String(),
			CreatedAt:    findUser.CreatedAt,
			UpdatedAt:    findUser.UpdatedAt,
			Email:        findUser.Email.String,
			Token:        token,
			RefreshToken: createRefresh.Token,
			IsChirpyRed:  findUser.IsChirpyRed,
		})
	}
}

// HandleRefresh godoc
// @Summary      Refresh access token
// @Description  Exchanges a valid, non-revoked refresh token for a new JWT access token
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} RefreshTokenResponse
// @Failure      401 {object} response.Error "Token missing, expired or revoked"
// @Router       /api/refresh [post]
func HandleRefresh(queries *database.Queries, tokenSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, errToken := GetBearerToken(r.Header)
		if errToken != nil {
			response.WriteError(w, errToken, 401, "Token not found")
			return
		}

		findToken, errFindToken := queries.GetRefreshToken(r.Context(), token)
		if errFindToken != nil {
			response.WriteError(w, errFindToken, 401, "Refresh token not found")
			return
		}

		if findToken.ExpiresAt.Before(time.Now()) || findToken.RevokedAt.Valid {
			response.WriteError(w, nil, 401, "Token is expired")
			return
		}

		tokenAccess, errTokenAccess := MakeJWT(findToken.UserID.UUID, tokenSecret, expiresAccess)
		if errTokenAccess != nil {
			response.WriteError(w, errTokenAccess, 500, "Something went wrong with token")
			return
		}

		response.WriteJSON(w, 200, RefreshTokenResponse{
			Token: tokenAccess,
		})
	}
}

// HandleRevoke godoc
// @Summary      Revoke refresh token
// @Description  Revokes the given refresh token so it can no longer be used
// @Tags         auth
// @Security     BearerAuth
// @Success      204 "No Content"
// @Failure      403 {object} response.Error "Token missing"
// @Failure      404 {object} response.Error "Token not found"
// @Router       /api/revoke [post]
func HandleRevoke(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, errToken := GetBearerToken(r.Header)
		if errToken != nil {
			response.WriteError(w, errToken, 403, "Token not found")
			return
		}

		errFindToken := queries.UpdateRefreshTokenRevoke(r.Context(), token)
		if errFindToken != nil {
			response.WriteError(w, errFindToken, 404, "Token not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
