package middleware

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/sinedviper/chirpy/internal/auth"
	"github.com/sinedviper/chirpy/internal/response"
)

type Middleware struct {
	Hits atomic.Int32
}

func (m *Middleware) Inc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.Hits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) Reset(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.Hits.Store(0)
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) Authentication(next http.Handler, publicSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, errToken := auth.GetBearerToken(r.Header)
		if errToken != nil {
			response.WriteError(w, errToken, 401, "Token not found")
			return
		}

		idUser, err := auth.ValidateJWT(token, publicSecret)
		if err != nil {
			response.WriteError(w, err, 401, "Validation Error")
			return
		}

		ctx := context.WithValue(r.Context(), "userID", idUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
