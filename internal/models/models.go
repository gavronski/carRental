package models

import (
	"time"
)

type User struct {
	ID          int
	FirstName   string
	LastName    string
	Email       string
	Password    string
	AccessLevel int
	CreatedAt   time.Time
	UpdatetdAt  time.Time
}

type Car struct {
	ID         int
	CarName    string
	Brand      string
	Model      string
	Version    string
	MadeAt     string
	Fuel       string
	Power      string
	Gearbox    string
	Drive      string
	Combustion string
	Body       string
	Color      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Images     []string
	Price      int
	Filename   string
}
type Restriction struct {
	ID              int
	RestrictionName string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Reservation struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	Phone     string
	StartDate time.Time
	EndDate   time.Time
	CarID     int
	CreatedAt time.Time
	UpdatedAt time.Time
	Car       Car
	Cost      int
}

type CarRescriction struct {
	ID            int
	StartDate     time.Time
	EndDate       time.Time
	CarID         int
	ReservationID int
	RestrictionID int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Reservation   Reservation
	Restriction   Restriction
}

type MailData struct {
	To       string
	From     string
	Subject  string
	Content  string
	Template string
}
