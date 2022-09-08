package main

import (
	"myapp/internal/config"
	"myapp/internal/handlers"

	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func routes(app *config.AppConfig) http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)
	//any request without csrf token will not work NoSurf
	mux.Use(NoSurf)
	mux.Use(SessionLoad)
	mux.Get("/", handlers.Repo.Home)
	mux.Get("/cars", handlers.Repo.Cars)
	mux.Post("/check-availability", handlers.Repo.CheckAvailability)
	mux.Get("/login", handlers.Repo.ShowLogin)
	mux.Post("/login", handlers.Repo.PostLogin)
	mux.Get("/logout", handlers.Repo.Logut)

	mux.Get("/make-reservation", handlers.Repo.MakeReservation)
	mux.Get("/cars/{car}", handlers.Repo.Car)
	mux.Post("/make-reservation", handlers.Repo.PostReservation)
	mux.Get("/reservation-summary", handlers.Repo.ReservationSummary)

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	mux.Route("/admin", func(mux chi.Router) {
		// mux.Use(Auth)
		mux.Get("/dashboard", handlers.Repo.AdminDashboard)
		mux.Get("/cars", handlers.Repo.AdminCars)
		mux.Get("/cars/add", handlers.Repo.AdminAddCar)
		mux.Post("/cars/add", handlers.Repo.AdminPostAddCar)
		mux.Get("/cars/{id}", handlers.Repo.AdminEditCar)
		mux.Post("/cars/{id}", handlers.Repo.AdminUpdateCar)
		mux.Post("/cars/delete/{id}", handlers.Repo.AdminDeleteCar)
		mux.Get("/reservations/{id}", handlers.Repo.AdminShowReservation)
		mux.Post("/reservations/delete/{id}", handlers.Repo.AdminDeleteRes)
	})
	return mux
}
