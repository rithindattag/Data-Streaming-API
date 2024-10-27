package api

import (
	"github.com/gorilla/mux"
)

func NewRouter(h *Handlers) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/stream/start", h.StartStream).Methods("POST")
	r.HandleFunc("/stream/{stream_id}/send", h.SendData).Methods("POST")
	r.HandleFunc("/stream/{stream_id}/results", h.StreamResults).Methods("GET")

	return r
}
