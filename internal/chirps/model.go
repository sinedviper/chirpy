package chirps

import "time"

type CreateRequest struct {
	Body   string `json:"body"`
	UserId string `json:"user_id"`
}

type ChirpResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    string    `json:"user_id"`
}
