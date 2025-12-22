package api

import "github.com/go-chi/chi/v5"

type Handler struct {
	Router *chi.Router
}

func NewHandler() *Handler {
	return &Handler{}
}
