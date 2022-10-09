package dbrepo

import (
	"errors"
	"myapp/internal/models"
)

func (m *testDBRepo) GetAllCars() ([]models.Car, error) {
	return nil, errors.New("can't find cars")
}

func (m *testDBRepo) GetCarByName(carName string) (models.Car, error) {
	var car models.Car
	if carName != "" {
		return car, nil
	}
	return car, errors.New("no such car")
}

func (m *testDBRepo) GetCarByID(carID int) (models.Car, error) {
	var car models.Car

	if carID > 3 || carID == 0 {
		return car, errors.New("car does not exists")
	}

	return car, nil
}

func (m *testDBRepo) CheckAvailabilityByDate(carID int, startDate, endDate string) (bool, error) {
	if carID == 2 {
		return true, errors.New("error in check availability")
	}
	return true, nil
}

func (m *testDBRepo) InsertReservation(res models.Reservation) (int, error) {
	var id int
	id = 1
	if res.FirstName == "Tom" {
		return id, errors.New("error while inserting data")
	}

	return id, nil
}

func (m *testDBRepo) InsertCarRestriction(r models.CarRescriction) error {
	if r.Reservation.LastName == "Lake" {
		return errors.New("error while inserting data")
	}
	return nil
}

func (m *testDBRepo) GetUserByID(id int) (models.User, error) {
	var u models.User
	return u, nil
}

func (m *testDBRepo) UpdateUser(u models.User) error {
	return nil
}

func (m *testDBRepo) Authenticate(email, testPassword string) (int, string, error) {
	if email == "me@here.go" {
		return 0, "", errors.New("wrong credentials")
	}
	return 1, "", nil
}

func (m *testDBRepo) GetReservations() ([]models.Reservation, error) {
	return nil, errors.New("can't get all reservations")
}

func (m *testDBRepo) UpdateCar(car models.Car) error {
	if car.ID == 0 {
		return errors.New("can't update ar with id 0")
	}

	return nil
}
func (m *testDBRepo) AddCar(car models.Car) error {
	if car.CarName != "test" {
		return errors.New("can't add a car")
	}
	return nil
}

func (m *testDBRepo) DeleteCar(carID int) error {
	if carID == 0 {
		return errors.New("can't delete car")
	}
	return nil
}

func (m *testDBRepo) GetReservationByID(reservationID int) (models.Reservation, error) {
	var res models.Reservation
	if reservationID == 0 {
		return res, errors.New("can't get reservation")
	}
	return res, nil
}

func (m *testDBRepo) DeleteReservation(resID int) error {
	if resID == 0 {
		return errors.New("can't delete reservation")
	}
	return nil
}
