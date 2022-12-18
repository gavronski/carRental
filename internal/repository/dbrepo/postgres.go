package dbrepo

import (
	"context"
	"errors"
	"log"
	"myapp/internal/models"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// CarSet represents slice of model.Car
type CarSet struct {
	cars []models.Car
}

// init add first element to the cars
func (c *CarSet) init() {
	car := models.Car{}
	c.cars = append(c.cars, car)
}

// contains checks if ccar.ID  (exists) is an index in cars slice
func (c *CarSet) contains(id int) bool {
	contains := false
	count := len(c.cars)

	for i := 0; i < count; i++ {
		if i == id {
			contains = true
			break
		}
	}

	return contains
}

// setCarsWithImages sets all images for car model
func (c *CarSet) setCarsWithImages(car models.Car) {
	if !c.contains(car.ID) {
		car.Images = append(car.Images, car.Image)
		c.cars = append(c.cars, car)
	} else {
		c.cars[car.ID].Images = append(c.cars[car.ID].Images, car.Image)
	}
}

// getCarsWithImages returns slice of non-empty cars
func (c *CarSet) getCarsWithImages() []models.Car {
	return c.cars[1:]
}

// GetAllCars selects car listing
func (m *postgresDBRepo) GetAllCars() ([]models.Car, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	carSet := &CarSet{}
	carSet.init()

	var cars []models.Car

	query := `
	select 
		c.id, c.car_name, c.brand, c.model, c.color, c.gearbox, c.drive,
		i.filename
	from 
		cars c
	left join images i on(i.car_id = c.id)
		order by c.id;
	`
	rows, err := m.DB.QueryContext(ctx, query)

	if err != nil {
		return cars, err
	}

	for rows.Next() {
		var car models.Car
		err := rows.Scan(
			&car.ID,
			&car.CarName,
			&car.Brand,
			&car.Model,
			&car.Color,
			&car.Gearbox,
			&car.Drive,
			&car.Image,
		)

		if err != nil {
			return cars, err
		}

		carSet.setCarsWithImages(car)
	}

	if err = rows.Err(); err != nil {
		return cars, err
	}

	return carSet.getCarsWithImages(), nil
}

// GetCarByID returns car by given ID
func (m *postgresDBRepo) GetCarByID(carID int) (models.Car, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var car = models.Car{}
	images := []string{}

	carSet := &CarSet{}
	carSet.init()
	// query := `select * from cars where id = $1;`
	query := `
		select
			c.*, i.filename
		from
		cars c
			left join images i on(i.car_id = c.id)
		where c.id = $1
			order by c.id
		`
	rows, err := m.DB.QueryContext(ctx, query, carID)
	for rows.Next() {
		err = rows.Scan(
			&car.ID,
			&car.CarName,
			&car.Brand,
			&car.Model,
			&car.Version,
			&car.MadeAt,
			&car.Fuel,
			&car.Power,
			&car.Gearbox,
			&car.Drive,
			&car.Combustion,
			&car.Body,
			&car.Color,
			&car.UpdatedAt,
			&car.CreatedAt,
			&car.Price,
			&car.Image,
		)

		if err != nil {
			return car, err
		}

		images = append(images, car.Image)
	}

	if err = rows.Err(); err != nil {
		return car, err
	}

	car.Images = images

	return car, nil
}

// Getreservation selects res by id
func (m *postgresDBRepo) GetReservationByID(reservationID int) (models.Reservation, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var reservation models.Reservation

	query := `select * from reservations where id = $1`
	row := m.DB.QueryRowContext(ctx, query, reservationID)
	err := row.Scan(
		&reservation.ID,
		&reservation.FirstName,
		&reservation.LastName,
		&reservation.Email,
		&reservation.Phone,
		&reservation.StartDate,
		&reservation.EndDate,
		&reservation.CarID,
		&reservation.CreatedAt,
		&reservation.UpdatedAt,
		&reservation.Cost,
	)

	if err != nil {
		return reservation, err
	}

	return reservation, nil
}

// UpdateCar updates car data by admin
func (m *postgresDBRepo) UpdateCar(car models.Car) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	power, _ := strconv.Atoi(car.Power)

	query := `update cars 
	set car_name = $1, brand = $2, model = $3, version = $4, made_at = $5, 
	fuel = $6, power = $7, gearbox = $8, drive = $9, combustion = $10, body = $11, color = $12, updated_at = $13, price = $14
	 where cars.id = $15;`

	_, err := m.DB.ExecContext(ctx, query,
		car.CarName,
		car.Brand,
		car.Model,
		car.Version,
		car.MadeAt,
		car.Fuel,
		power,
		car.Gearbox,
		car.Drive,
		car.Combustion,
		car.Body,
		car.Color,
		time.Now(),
		car.Price,
		car.ID)

	if err != nil {
		return err
	}

	return nil
}

// AddCar adds new car data by admin
func (m *postgresDBRepo) AddCar(car models.Car) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	id, err := m.GetMaxCarID()

	if err != nil {
		return 0, err
	}
	id = id + 1

	power, _ := strconv.Atoi(car.Power)

	if err != nil {
		return 0, err
	}

	stmt := `insert into cars(
		id, car_name, brand, model, version, made_at, fuel, power, gearbox, drive, combustion, body, color, price, created_at, updated_at)
	values
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err = m.DB.ExecContext(ctx, stmt,
		id,
		car.CarName,
		car.Brand,
		car.Model,
		car.Version,
		car.MadeAt,
		car.Fuel,
		power,
		car.Gearbox,
		car.Drive,
		car.Combustion,
		car.Body,
		car.Color,
		car.Price,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return 0, err
	}

	return id, nil
}

// DeleteCar adds new car data by admin
func (m *postgresDBRepo) DeleteCar(carID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `delete from cars where id = $1`

	_, err := m.DB.ExecContext(ctx, stmt, carID)

	if err != nil {
		return err
	}

	return nil
}

//  CheckAvailabilityByDate check if car can be booked between two dates
func (m *postgresDBRepo) CheckAvailabilityByDate(carID int, startDate, endDate string) (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
	select 
		count(id)
	from 
		car_restrictions
	where
		car_id = $1
		and
		(($2 between start_date and end_date) or ($3 between start_date  and end_date))
		or 
		($2 < start_date and $3 >= end_date);
	`
	var numRows int
	row := m.DB.QueryRowContext(ctx, query, carID, startDate, endDate)

	err := row.Scan(&numRows)

	if err != nil {
		return false, err
	}

	if numRows == 0 {
		return true, nil
	}

	return false, nil
}

// InsertReservation adds new res
func (m *postgresDBRepo) InsertReservation(res models.Reservation) (int, error) {
	var id int
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stmt := `insert into reservations (first_name, last_name, email, phone, start_date, end_date, car_id, created_at, updated_at, cost)
	values($1, $2, $3 ,$4, $5, $6, $7, $8, $9, $10) returning id`

	err := m.DB.QueryRowContext(ctx, stmt,
		res.FirstName,
		res.LastName,
		res.Email,
		res.Phone,
		res.StartDate,
		res.EndDate,
		res.CarID,
		time.Now(),
		time.Now(),
		res.Cost).Scan(&id)

	if err != nil {
		return 0, err
	}
	return id, nil
}

// DeleteReservation deletes res by admin
func (m *postgresDBRepo) DeleteReservation(resID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `delete from reservations where id = $1`

	_, err := m.DB.ExecContext(ctx, stmt, resID)

	if err != nil {
		return err
	}

	return nil
}

// InsertCarRestriction inserts car restriction
func (m *postgresDBRepo) InsertCarRestriction(r models.CarRescriction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `insert into car_restrictions (start_date, end_date, car_id, reservation_id, created_at, updated_at, restriction_id)
		values
		($1, $2, $3, $4, $5, $6, $7)`
	_, err := m.DB.ExecContext(ctx, stmt,
		r.StartDate,
		r.EndDate,
		r.CarID,
		r.ReservationID,
		time.Now(),
		time.Now(),
		r.RestrictionID,
	)

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// GetUserByID returns user by given id
func (m *postgresDBRepo) GetUserByID(id int) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `select id, first_name, last_name, email, password, access_level, created_at, updated_at
		from users where id = $id`

	row := m.DB.QueryRowContext(ctx, query, id)
	var u models.User

	err := row.Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
		&u.Password,
		&u.AccessLevel,
		&u.CreatedAt,
		&u.UpdatetdAt,
	)

	if err != nil {
		return u, err
	}

	return u, nil
}

// UpdateUser updates user info
func (m *postgresDBRepo) UpdateUser(u models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `update users set first_name = $1, last_name = $2, email = $3, access_level = $4, updated_at = $5`

	_, err := m.DB.ExecContext(ctx, query,
		u.FirstName,
		u.LastName,
		u.Email,
		u.AccessLevel,
		time.Now())

	if err != nil {
		return err
	}

	return nil
}

/*
	Authenticate compare data from the form and db
	email, hashed pass and form pass
*/
func (m *postgresDBRepo) Authenticate(email, testPassword string) (int, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	row := m.DB.QueryRowContext(ctx, "select id, password from users where email = $1", email)
	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		return id, "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(testPassword))

	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, "", errors.New("incorrect password")
	} else if err != nil {
		return 0, "", err
	}

	return id, hashedPassword, nil
}

// GetReservation selects reservation data
func (m *postgresDBRepo) GetReservations() ([]models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservations []models.Reservation
	query := `select r.id, r.email, r.start_date, r.end_date, r.cost, r.car_id, c.car_name 
		from reservations r
		left join cars c on(r.car_id = c.id)
		order by r.start_date asc`
	rows, err := m.DB.QueryContext(ctx, query)

	if err != nil {
		return reservations, err
	}

	for rows.Next() {
		var res models.Reservation
		err := rows.Scan(
			&res.ID,
			&res.Email,
			&res.StartDate,
			&res.EndDate,
			&res.Cost,
			&res.CarID,
			&res.Car.CarName,
		)

		if err != nil {
			return reservations, err
		}

		reservations = append(reservations, res)
	}

	if err = rows.Err(); err != nil {
		return reservations, err
	}

	return reservations, nil
}

// GetMax retruns max id from reservations table
func (m *postgresDBRepo) GetMaxCarID() (int, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	query := `select max(id) as id from cars;`
	row := m.DB.QueryRowContext(ctx, query)
	err := row.Scan(
		&id,
	)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *postgresDBRepo) GetMaxImageID() (int, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	query := `select max(id) as id from images;`
	row := m.DB.QueryRowContext(ctx, query)
	err := row.Scan(
		&id,
	)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *postgresDBRepo) InsertCarImage(image models.Image) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	maxID, err := m.GetMaxImageID()
	maxID = maxID + 1

	if err != nil {
		return 0, err
	}
	var newID int

	stmt := `insert into images (id, car_id, filename, updated_at, created_at) 
	values ($1, $2, $3, $4, $5) returning id;`

	err = m.DB.QueryRowContext(ctx, stmt,
		maxID,
		image.CarID,
		image.Filename,
		time.Now(),
		time.Now(),
	).Scan(&newID)

	if err != nil {
		return 0, err
	}

	return newID, nil
}

func (m *postgresDBRepo) DeleteImage(carID int, filename string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `delete from images where filename = $1 and car_id = $2`

	_, err := m.DB.ExecContext(ctx, stmt, filename, carID)

	if err != nil {
		return err
	}

	return nil
}

func (m *postgresDBRepo) GetImagesNumber(carID int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	query := `select count(id) from images where car_id = $1;`
	row := m.DB.QueryRowContext(ctx, query, carID)
	err := row.Scan(
		&id,
	)

	if err != nil {
		return 0, err
	}

	return id, nil
}
