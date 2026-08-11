package response

import (
	"encoding/json"
	"log"
	"net/http"
)

type Error struct {
	Error string `json:"error"`
}

func WriteError(w http.ResponseWriter, err error, status int, msg string) {
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Error{Error: msg})
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func WriteTextPlain(w http.ResponseWriter, status int, data string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(data))
}

func WriteTextHtml(w http.ResponseWriter, status int, data string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(data))
}
