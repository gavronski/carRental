package main

import (
	"fmt"
	"myapp/internal/config"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRoutes(t *testing.T) {
	var app config.AppConfig
	mux := routes(&app)

	switch v := mux.(type) {
	case *chi.Mux:
		// Fne, do nothing
	default:
		t.Error(fmt.Sprintf("type is not http.Handler, but is type %s", v))
	}
}
