package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"myapp/internal/config"
	"myapp/internal/driver"
	"myapp/internal/forms"
	"myapp/internal/models"
	"myapp/internal/render"
	"myapp/internal/repository"
	"myapp/internal/repository/dbrepo"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Repo the repository used by the handlers
var Repo *Repository

var uploadPath = "./static/images/"

// Repository is the repository type
type Repository struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}

// NewRepo creates new repository
func NewRepo(a *config.AppConfig, db *driver.DB) *Repository {
	return &Repository{
		App: a,
		DB:  dbrepo.NewPostgresRepo(db.SQL, a),
	}
}

// NewRepo creates new repository
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

// Home handles request for home page and renders template
func (m *Repository) Home(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "home.page.gohtml", &models.TemplateData{})
}

// About handles request for about page and renders template
func (m *Repository) About(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "about.page.gohtml", &models.TemplateData{})
}

// Cars handles request for cars offer
func (m *Repository) Cars(w http.ResponseWriter, r *http.Request) {
	cars, err := m.DB.GetAllCars()

	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find cars")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Find front image for all cars
	for i, car := range cars {
		// If car has no image set default
		if len(car.Images) == 0 {
			cars[i].Image = fmt.Sprintf("%s%s", uploadPath, "image-icon.png")
		} else {
			cars[i].Image = fmt.Sprintf("%s%s", uploadPath, car.Images[0])
		}
	}

	data := make(map[string]interface{})
	data["cars"] = cars

	render.Template(w, r, "cars.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

// Car handles request for choosen car
func (m *Repository) Car(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/cars/", "", -1)
	carID, _ := strconv.Atoi(id)
	car, err := m.DB.GetCarByID(carID)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find car")
		http.Redirect(w, r, "/cars", http.StatusSeeOther)
		return
	}

	// If car has no image set default
	if len(car.Images) == 0 {
		car.Image = fmt.Sprintf("%s%s", uploadPath, "image-icon.png")
	} else {
		car.Image = fmt.Sprintf("%s%s", uploadPath, car.Images[0])
	}

	m.App.Session.Put(r.Context(), "car", car)

	data := make(map[string]interface{})
	data["car"] = car

	render.Template(w, r, "car.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

type jsonResponse struct {
	OK bool `json:"ok"`
}

// CheckAvailability handlers requests for availability and send JSON response
func (m *Repository) CheckAvailability(w http.ResponseWriter, r *http.Request) {

	// Get car from the session
	car, ok := m.App.Session.Get(r.Context(), "car").(models.Car)
	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't find car in the session")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Parse form
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't check availability")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sd := r.Form.Get("start")
	ed := r.Form.Get("end")

	// Parse cost field to integer
	cost, _ := strconv.Atoi(r.Form.Get("cost"))

	available, err := m.DB.CheckAvailabilityByDate(car.ID, sd, ed)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't check availability")
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
			Cost:      cost,
		}

		m.App.Session.Put(r.Context(), "reservation", res)
	}

	// Return json response
	resp := jsonResponse{
		OK: available,
	}
	out, _ := json.MarshalIndent(resp, "", "    ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

// MakeReservation handles request for form reservation
func (m *Repository) MakeReservation(w http.ResponseWriter, r *http.Request) {
	// Get reservation from the session
	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)

	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't find reservation in the session")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	car, err := m.DB.GetCarByID(res.CarID)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find car in the session")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	res.Car = car
	m.App.Session.Put(r.Context(), "car", car)
	m.App.Session.Put(r.Context(), "reservation", res)

	// Change dates format
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

/*
	PostReservation handles request for posting reservation
	validates form,
	adds data to reservations,
	sends email if form is valid
*/
func (m *Repository) PostReservation(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	reservation, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)

	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't find reservation in the session")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	reservation.FirstName = r.Form.Get("first_name")
	reservation.LastName = r.Form.Get("last_name")
	reservation.Phone = r.Form.Get("phone")
	reservation.Email = r.Form.Get("email")
	form := forms.New(r.PostForm)

	form.Required("first_name", "last_name", "email", "phone")
	form.IsEmail("email")
	form.IsNum("phone")

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
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return

	}
	fmt.Println(id)
	m.App.Session.Put(r.Context(), "reservation", reservation)
	// restriction := models.CarRescriction{
	// 	StartDate:     reservation.StartDate,
	// 	EndDate:       reservation.EndDate,
	// 	CarID:         reservation.CarID,
	// 	ReservationID: id,
	// 	RestrictionID: 1,
	// 	Reservation:   reservation,
	// }

	// err = m.DB.InsertCarRestriction(restriction)
	// if err != nil {
	// 	m.App.Session.Put(r.Context(), "error", "can't insert restriction into database")
	// 	http.Redirect(w, r, "/", http.StatusSeeOther)
	// 	return
	// }
	// Email content
	content := fmt.Sprintf("<h1>Hi, %s %s!</h1></br>Your car rent is confirmed.</br>Rent info: </br> <ul><li>Start date: %s</li><li>End date: %s</li></ul>", reservation.FirstName, reservation.LastName, reservation.StartDate, reservation.EndDate)
	msg := models.MailData{
		To:       reservation.Email,
		From:     "me@here.com",
		Subject:  "Reserrvation Confirmation",
		Content:  content,
		Template: "drip.html",
	}

	// Set message and send email
	m.App.MailChan <- msg
	m.App.Session.Put(r.Context(), "reservation", reservation)
	m.App.Session.Put(r.Context(), "flash", "Reservation has been completed")
	http.Redirect(w, r, "/reservation-summary", http.StatusSeeOther)
}

// ReservationSummary handles request for reservation summary
func (m *Repository) ReservationSummary(w http.ResponseWriter, r *http.Request) {
	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)

	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't find car in the session")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	sd := res.StartDate.Format("2006-01-02")
	ed := res.EndDate.Format("2006-01-02")

	data := make(map[string]interface{})
	data["reservation"] = res
	data["start"] = sd
	data["end"] = ed
	render.Template(w, r, "reservation-summary.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

// ShowLogin renders form to login
func (m *Repository) ShowLogin(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "login.page.gohtml", &models.TemplateData{
		Form: forms.New(nil),
	})
}

// PostLogin insert credentials
func (m *Repository) PostLogin(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	var email string
	var password string

	email = r.Form.Get("email")
	password = r.Form.Get("password")
	form := forms.New(r.PostForm)
	// Form validation
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
	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

// Logout handles request for logout
func (m *Repository) Logut(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.Destroy(r.Context())
	_ = m.App.Session.RenewToken(r.Context())

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ContactUs renders contact template
func (m *Repository) ContactUs(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	data["message"] = ""
	data["email"] = ""

	render.Template(w, r, "contact.page.gohtml", &models.TemplateData{
		Form: forms.New(nil),
		Data: data,
	})
}

// PostContactUs handles message to company via email
func (m *Repository) PostContactUs(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("email", "message")
	form.IsEmail("email")

	if !form.Valid() {
		data := make(map[string]interface{})
		data["message"] = form.Get("message")
		data["email"] = form.Get("email")
		render.Template(w, r, "contact.page.gohtml", &models.TemplateData{
			Form: form,
			Data: data,
		})
		return
	}
	email := r.Form.Get("email")
	message := r.Form.Get("message")

	content := fmt.Sprintf("<p>%s</p>", message)
	msg := models.MailData{
		To:       email,
		From:     "me@here.com",
		Subject:  "Customer message",
		Content:  content,
		Template: "drip.html",
	}

	m.App.MailChan <- msg
	m.App.Session.Put(r.Context(), "flash", "message has been sent")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

// AdminDashboard renders dashboard fullfilled wtih reservations
func (m *Repository) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	reservations, err := m.DB.GetReservations()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	data := make(map[string]interface{})
	data["res"] = reservations
	render.Template(w, r, "admin-reservations.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

// AdminCars handles car listing
func (m *Repository) AdminCars(w http.ResponseWriter, r *http.Request) {
	cars, err := m.DB.GetAllCars()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}
	data := make(map[string]interface{})
	data["cars"] = cars
	render.Template(w, r, "admin-cars.page.gohtml", &models.TemplateData{
		Data: data,
	})
}

// AdminEditCar renders form with data
func (m *Repository) AdminEditCar(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/admin/cars/edit/", "", -1)
	carID, _ := strconv.Atoi(id)

	car, err := m.DB.GetCarByID(carID)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// If car has no image set default
	if len(car.Images) == 0 {
		car.Image = fmt.Sprintf("%s%s", uploadPath, "image-icon.png")
	} else {
		car.Image = fmt.Sprintf("%s%s", uploadPath, car.Images[0])
	}

	data := make(map[string]interface{})
	data["car"] = car
	form := forms.New(nil)
	render.Template(w, r, "admin-edit-car.page.gohtml", &models.TemplateData{
		Data: data,
		Form: form,
	})
}

// AdminUpdateCar updates car's data
func (m *Repository) AdminUpdateCar(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	form := forms.New(r.PostForm)

	var car models.Car
	strID := r.Form.Get("car_id")
	car.ID, _ = strconv.Atoi(strID)
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
	car.Price, _ = strconv.Atoi(r.Form.Get("price"))

	if err != nil {
		form.Errors.Add("price", err.Error())
	}
	form.Required("car_name", "brand", "model", "version", "fuel", "power", "gearbox", "made_at",
		"drive", "combustion", "body", "color", "price")

	form.IsNum("power")
	if !form.Valid() {
		data := make(map[string]interface{})
		data["car"] = car

		render.Template(w, r, "admin-edit-car.page.gohtml", &models.TemplateData{
			Form: form,
			Data: data,
		})
		return
	}

	err = m.DB.UpdateCar(car)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Car info has been updated")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

// AdminAddCar renders form to add new car
func (m *Repository) AdminAddCar(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	form := forms.New(nil)
	render.Template(w, r, "admin-add-new-car.page.gohtml", &models.TemplateData{
		Data: data,
		Form: form,
	})
}

// AdminPostCar adds new car to db
func (m *Repository) AdminPostCar(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	form := forms.New(r.PostForm)

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
	car.Price, err = strconv.Atoi(r.Form.Get("price"))

	if err != nil {
		form.Errors.Add("price", err.Error())
	}

	form.Required("car_name", "brand", "model", "version", "fuel", "power", "gearbox", "made_at",
		"drive", "combustion", "body", "color", "price")
	form.IsNum("power")

	if !form.Valid() {
		data := make(map[string]interface{})
		data["car"] = car

		render.Template(w, r, "admin-add-new-car.page.gohtml", &models.TemplateData{
			Form: form,
			Data: data,
		})
		return
	}
	carID, err := m.DB.AddCar(car)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	defaultImage := "image-icon.png"
	var image = models.Image{
		CarID:    carID,
		Filename: defaultImage,
	}

	_, err = m.DB.InsertCarImage(image)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Car has been added")
	http.Redirect(w, r, "/admin/cars", http.StatusSeeOther)
}

// AdminDeleteCar deletes car from db
func (m *Repository) AdminDeleteCar(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/admin/cars/delete/", "", -1)
	carID, _ := strconv.Atoi(id)

	err := m.DB.DeleteCar(carID)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/admin/cars", http.StatusSeeOther)
		return
	}
	m.App.Session.Put(r.Context(), "flash", "Car has been deleted")
	resp := jsonResponse{
		OK: true,
	}
	out, _ := json.MarshalIndent(resp, "", "    ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

// AdminShowReservations retruns reservations data
func (m *Repository) AdminShowReservation(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/admin/reservations/", "", -1)
	resID, _ := strconv.Atoi(id)

	reservation, err := m.DB.GetReservationByID(resID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
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

// AdminDeleteRes deletes res data and notifies customer about reservation canceling, returns json
func (m *Repository) AdminDeleteRes(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	id := strings.Replace(url, "/admin/reservations/delete/", "", -1)
	resID, _ := strconv.Atoi(id)
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	email := r.Form.Get("email")
	fname := r.Form.Get("first_name")
	lname := r.Form.Get("last_name")

	err = m.DB.DeleteReservation(resID)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "deleted bruh")

	content := fmt.Sprintf("<h1>Hi, %s %s!</h1></br>Your car rent has been canceled.</br><p>Contact admin for more info me@here.com.</p><br>", fname, lname)
	msg := models.MailData{
		To:       email,
		From:     "me@here.com",
		Subject:  "Reserrvation canceled",
		Content:  content,
		Template: "drip.html",
	}
	m.App.Session.Put(r.Context(), "flash", "Reservation has been deleted")

	m.App.MailChan <- msg

	resp := jsonResponse{
		OK: true,
	}
	out, _ := json.MarshalIndent(resp, "", "    ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

// AdminUploadImage uploads for car images to images folder and saves in db
func (m *Repository) AdminUploadImage(w http.ResponseWriter, r *http.Request) {
	files, err := m.UploadFiles(r, uploadPath)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	carID, _ := strconv.Atoi(r.Form.Get("car_id"))
	var image = models.Image{
		CarID:    carID,
		Filename: files[0].OrginalFilename,
	}

	_, err = m.DB.InsertCarImage(image)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

type UploadedFile struct {
	OrginalFilename string
	FileSize        int64
}

// UploadFiles upload image form form
func (m *Repository) UploadFiles(r *http.Request, uploadDir string) ([]*UploadedFile, error) {
	var uploadedFiles []*UploadedFile
	err := r.ParseMultipartForm(int64(1024 * 1024 * 5))

	if err != nil {
		return nil, fmt.Errorf("the uploaded file is too big, and must be less than %d bytes", 1024*1024*5)
	}
	for _, fHeaders := range r.MultipartForm.File {
		for _, hdr := range fHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile
				infile, err := hdr.Open()
				if err != nil {
					return nil, err
				}
				defer infile.Close()

				uploadedFile.OrginalFilename = hdr.Filename

				var outfile *os.File
				defer outfile.Close()

				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.OrginalFilename)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, infile)

					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = fileSize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}
	return uploadedFiles, nil
}

// AdminDeleteImage deletes an image from car gallery
func (m *Repository) AdminDeleteImage(w http.ResponseWriter, r *http.Request) {
	name := r.Form.Get("del_image")
	id := r.Form.Get("car_id")
	carID, _ := strconv.Atoi(id)

	// remove image form db
	err := m.DB.DeleteImage(carID, name)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	// remove image from filesystem
	if name != "image-icon.png" {
		err = os.Remove(uploadPath + name)

		if err != nil {
			m.App.Session.Put(r.Context(), "error", err)
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	// check count of car images
	num, err := m.DB.GetImagesNumber(carID)

	if err != nil {
		m.App.Session.Put(r.Context(), "error", err)
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	//if car has no image set default image
	if num == 0 {
		defaultImage := "image-icon.png"
		var image = models.Image{
			CarID:    carID,
			Filename: defaultImage,
		}

		_, err = m.DB.InsertCarImage(image)

		if err != nil {
			m.App.Session.Put(r.Context(), "error", err)
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	m.App.Session.Put(r.Context(), "flash", "Image has been deleted")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
