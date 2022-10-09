package repository

import "myapp/internal/models"

type DatabaseRepo interface {
	GetAllCars() ([]models.Car, error)
	GetCarByName(carName string) (models.Car, error)
	CheckAvailabilityByDate(carID int, startDate, endDate string) (bool, error)
	GetCarByID(carID int) (models.Car, error)
	InsertReservation(res models.Reservation) (int, error)
	InsertCarRestriction(r models.CarRescriction) error
	GetUserByID(id int) (models.User, error)
	UpdateUser(u models.User) error
	Authenticate(email, testPassword string) (int, string, error)
	GetReservations() ([]models.Reservation, error)
	UpdateCar(car models.Car) error
	AddCar(car models.Car) error
	DeleteCar(carID int) error
	GetReservationByID(reservationID int) (models.Reservation, error)
	DeleteReservation(resID int) error
}
