package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"myapp/internal/config"
	"myapp/internal/driver"
	"myapp/internal/forms"
	"myapp/internal/models"
	"myapp/internal/render"
	"myapp/internal/repository"
	"myapp/internal/repository/dbrepo"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//Repo the repository used by the handlers
var Repo *Repository
var currentCarName string

//Repository is the repository typ
type Repository struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}

//NewRepo create new repository
func NewRepo(a *config.AppConfig, db *driver.DB) *Repository {
	return &Repository{
		App: a,
		DB:  dbrepo.NewPostgresRepo(db.SQL, a),
	}
}

//NewRepo create new repository
func NewTestingRepo(a *config.AppConfig) *Repository {
	return &Repository{
		App: a,
		DB:  dbrepo.NewTestingRepo(a),
	}
}

// Sets the repository
func NewHandlers(r *Repository) {
	Repo = r
}

func (m *Repository) Home(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "home.page.gohtml", &models.TemplateData{})
}

func (m *Repository) Cars(w http.ResponseWriter, r *http.Request) {
	cars, err := m.DB.GetAllCars()

	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find cars")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	data := make(map[string]interface{})
	data["cars"] = cars
	render.Template(w, r, "cars.page.gohtml", &models.TemplateData{
		Data: data,
	})
}
func (m *Repository) Car(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	carName := strings.Replace(url, "/cars/", "", -1)

	car, err := m.DB.GetCarByName(carName)
	m.App.Session.Put(r.Context(), "car", car)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find car in the session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data := make(map[string]interface{})
	data["car"] = car

	render.Template(w, r, "car.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

type jsonResponse struct {
	OK bool `json:"ok"`
}

func (m *Repository) CheckAvailability(w http.ResponseWriter, r *http.Request) {

	car, ok := m.App.Session.Get(r.Context(), "car").(models.Car)
	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't find car in the session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	sd := r.Form.Get("start")
	ed := r.Form.Get("end")

	available, err := m.DB.CheckAvailabilityByDate(car.ID, sd, ed)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't check availability")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	if available {
		layout := "2006-01-02"
		startDate, _ := time.Parse(layout, sd)
		endDate, _ := time.Parse(layout, ed)

		res := models.Reservation{
			CarID:     car.ID,
			StartDate: startDate,
			EndDate:   endDate,
		}

		m.App.Session.Put(r.Context(), "reservation", res)
	}

	resp := jsonResponse{
		OK: available,
	}
	out, _ := json.MarshalIndent(resp, "", "    ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (m *Repository) MakeReservation(w http.ResponseWriter, r *http.Request) {

	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)

	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't find reservation in the session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	car, err := m.DB.GetCarByID(res.CarID)
	m.App.Session.Put(r.Context(), "car", car)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find car in the session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	sd := res.StartDate.Format("2006-01-02")
	ed := res.EndDate.Format("2006-01-02")

	stringMap := make(map[string]string)
	stringMap["start_date"] = sd
	stringMap["end_date"] = ed

	data := make(map[string]interface{})
	data["reservation"] = res

	render.Template(w, r, "reservation.page.gohtml", &models.TemplateData{
		Form:      forms.New(nil),
		Data:      data,
		StringMap: stringMap,
	})
}

func (m *Repository) PostReservation(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	reservation, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)

	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't find reservation in the session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	reservation.FirstName = r.Form.Get("first_name")
	reservation.LastName = r.Form.Get("last_name")
	reservation.Phone = r.Form.Get("phone")
	reservation.Email = r.Form.Get("email")
	form := forms.New(r.PostForm)
	form.Required("first_name", "last_name", "email", "phone")
	form.IsEmail("email")

	if !form.Valid() {
		data := make(map[string]interface{})
		data["reservation"] = reservation

		render.Template(w, r, "reservation.page.gohtml", &models.TemplateData{
			Form: form,
			Data: data,
		})
		return
	}

	id, err := m.DB.InsertReservation(reservation)
	if err != nil {

		m.App.Session.Put(r.Context(), "error", "can't insert reservation into database")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return

	}
	m.App.Session.Put(r.Context(), "reservation", reservation)
	restriction := models.CarRescriction{
		StartDate:     reservation.StartDate,
		EndDate:       reservation.EndDate,
		CarID:         reservation.CarID,
		ReservationID: id,
		RestrictionID: 1,
		Reservation:   reservation,
	}

	err = m.DB.InsertCarRestriction(restriction)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't insert restriction into database")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	content := fmt.Sprintf("<h1>Hi, %s %s!</h1></br>Your car rent is confirmed.</br>Rent info: </br> <ul><li>Start date: %s</li><li>End date: %s</li></ul>", reservation.FirstName, reservation.LastName, reservation.StartDate, reservation.EndDate)
	msg := models.MailData{
		To:       reservation.Email,
		From:     "me@here.com",
		Subject:  "Reserrvation Confirmation",
		Content:  content,
		Template: "drip.html",
	}

	m.App.MailChan <- msg
	m.App.Session.Put(r.Context(), "reservation", reservation)

	http.Redirect(w, r, "/reservation-summary", http.StatusSeeOther)
}

func (m *Repository) ReservationSummary(w http.ResponseWriter, r *http.Request) {
	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)

	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't find car in the session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data := make(map[string]interface{})
	data["reservation"] = res
	render.Template(w, r, "reservation-summary.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

func (m *Repository) ShowLogin(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "login.page.gohtml", &models.TemplateData{
		Form: forms.New(nil),
	})
}
func (m *Repository) PostLogin(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}
	var email string
	var password string

	email = r.Form.Get("email")
	password = r.Form.Get("password")
	form := forms.New(r.PostForm)
	form.Required("email", "password")
	form.IsEmail("email")
	if !form.Valid() {
		render.Template(w, r, "login.page.gohtml", &models.TemplateData{
			Form: form,
		})
		return
	}

	id, _, err := m.DB.Authenticate(email, password)
	if err != nil {

		m.App.Session.Put(r.Context(), "error", "Invalid login credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "user_id", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (m *Repository) Logut(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.Destroy(r.Context())
	_ = m.App.Session.RenewToken(r.Context())

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (m *Repository) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	reservations, err := m.DB.GetReservations()

	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	data := make(map[string]interface{})
	data["res"] = reservations
	render.Template(w, r, "admin-reservations.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

func (m *Repository) AdminCars(w http.ResponseWriter, r *http.Request) {
	cars, err := m.DB.GetAllCars()

	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
	data := make(map[string]interface{})
	data["cars"] = cars
	render.Template(w, r, "admin-cars.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

func (m *Repository) AdminEditCar(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/admin/cars/", "", -1)
	carID, _ := strconv.Atoi(id)

	car, err := m.DB.GetCarByID(carID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find car in the session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data := make(map[string]interface{})
	data["car"] = car
	form := forms.New(nil)
	render.Template(w, r, "admin-edit-car.page.gohtml", &models.TemplateData{
		Data: data,
		Form: form,
	})
}
func (m *Repository) AdminUpdateCar(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	var car models.Car
	car.ID, _ = strconv.Atoi(r.Form.Get("car_id"))
	car.CarName = r.Form.Get("car_name")
	car.Brand = r.Form.Get("brand")
	car.Model = r.Form.Get("model")
	car.Version = r.Form.Get("version")
	car.Fuel = r.Form.Get("fuel")
	car.Power = r.Form.Get("power")
	car.Gearbox = r.Form.Get("gearbox")
	car.MadeAt = r.Form.Get("made_at")
	car.Drive = r.Form.Get("drive")
	car.Combustion = r.Form.Get("combustion")
	car.Body = r.Form.Get("body")
	car.Color = r.Form.Get("color")

	form := forms.New(r.PostForm)
	form.Required("car_name", "brand", "model", "version", "fuel", "power", "gearbox", "made_at",
		"drive", "combustion", "body", "color")

	if !form.Valid() {
		data := make(map[string]interface{})
		data["car"] = car

		log.Println(form.Errors)
		render.Template(w, r, "admin-edit-car.page.gohtml", &models.TemplateData{
			Form: form,
			Data: data,
		})
		return
	}

	err = m.DB.UpdateCar(car)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		log.Println(err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, "/admin/cars", http.StatusSeeOther)
}

func (m *Repository) AdminAddCar(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	form := forms.New(nil)
	render.Template(w, r, "admin-add-new-car.page.gohtml", &models.TemplateData{
		Data: data,
		Form: form,
	})
}

func (m *Repository) AdminPostAddCar(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	var car models.Car
	car.CarName = r.Form.Get("car_name")
	car.Brand = r.Form.Get("brand")
	car.Model = r.Form.Get("model")
	car.Version = r.Form.Get("version")
	car.Fuel = r.Form.Get("fuel")
	car.Power = r.Form.Get("power")
	car.Gearbox = r.Form.Get("gearbox")
	car.MadeAt = r.Form.Get("made_at")
	car.Drive = r.Form.Get("drive")
	car.Combustion = r.Form.Get("combustion")
	car.Body = r.Form.Get("body")
	car.Color = r.Form.Get("color")

	form := forms.New(r.PostForm)
	form.Required("car_name", "brand", "model", "version", "fuel", "power", "gearbox", "made_at",
		"drive", "combustion", "body", "color")

	if !form.Valid() {
		data := make(map[string]interface{})
		data["car"] = car

		log.Println(form.Errors)
		render.Template(w, r, "admin-add-new-car.page.gohtml", &models.TemplateData{
			Form: form,
			Data: data,
		})
		return
	}
	log.Println(car)
	err = m.DB.AddCar(car)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, "/admin/cars", http.StatusSeeOther)
}

func (m *Repository) AdminDeleteCar(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/admin/cars/delete/", "", -1)
	carID, _ := strconv.Atoi(id)

	err := m.DB.DeleteCar(carID)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	m.App.Session.Put(r.Context(), "flash", "hello world")
	http.Redirect(w, r, "/admin/cars", http.StatusSeeOther)
}

func (m *Repository) AdminShowReservation(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/admin/reservations/", "", -1)
	resID, _ := strconv.Atoi(id)

	reservation, err := m.DB.GetReservationByID(resID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find reservation in the session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data := make(map[string]interface{})
	data["reservation"] = reservation
	form := forms.New(nil)

	render.Template(w, r, "admin-reservation.page.gohtml", &models.TemplateData{
		Data: data,
		Form: form,
	})
}

func (m *Repository) AdminDeleteRes(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/admin/reservations/delete/", "", -1)
	resID, _ := strconv.Atoi(id)
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	email := r.Form.Get("email")
	fname := r.Form.Get("first_name")
	lname := r.Form.Get("last_name")

	err = m.DB.DeleteReservation(resID)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	content := fmt.Sprintf("<h1>Hi, %s %s!</h1></br>Your car rent has been canceled.</br><p>Contact admin for more info me@here.com.</p><br>", fname, lname)
	msg := models.MailData{
		To:       email,
		From:     "me@here.com",
		Subject:  "Reserrvation canceled",
		Content:  content,
		Template: "drip.html",
	}

	m.App.MailChan <- msg
	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}
