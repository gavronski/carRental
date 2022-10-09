package helpers

import (
	"myapp/internal/config"
	"net/http"
)

var app *config.AppConfig

// NewHelpers sets up helper
func NewHelpers(a *config.AppConfig) {
	app = a
}

/*
	IsAuthanticated checks if user is authenticated
	by searching "user_id" in session.
	Returns true if is and false otherwise
*/
func IsAuthanticated(r *http.Request) bool {
	exists := app.Session.Exists(r.Context(), "user_id")
	return exists
}
