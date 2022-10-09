package handlers

import (
	"encoding/gob"
	"fmt"
	"log"
	"myapp/internal/config"
	"myapp/internal/models"
	"myapp/internal/render"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/justinas/nosurf"
)

var app config.AppConfig
var session *scs.SessionManager
var pathToTemplates = "./../../templates"
var functions = template.FuncMap{}

func TestMain(m *testing.M) {
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Car{})
	gob.Register(models.Restriction{})
	app.InProduction = false

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction
	app.Session = session

	mailChan := make(chan models.MailData)
	app.MailChan = mailChan
	defer close(mailChan)

	listenForMail()
	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Println("Błąd z cahchem tempalta")
	}
	app.TemplateCache = tc
	app.UseCache = true

	repo := NewTestingRepo(&app)
	NewHandlers(repo)
	render.NewRenderer(&app)

	os.Exit(m.Run())
}
func getRoutes() http.Handler {
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Car{})
	gob.Register(models.Restriction{})
	app.InProduction = false

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction
	app.Session = session

	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Println("Błąd z cahchem tempalta")
	}
	app.TemplateCache = tc
	app.UseCache = true

	repo := NewTestingRepo(&app)
	NewHandlers(repo)
	render.NewRenderer(&app)

	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)
	//  mux.Use(NoSurf)
	mux.Use(SessionLoad)
	mux.Get("/", Repo.Home)
	mux.Get("/cars", Repo.Cars)
	mux.Post("/check-availability", Repo.CheckAvailability)
	mux.Get("/login", Repo.ShowLogin)
	mux.Post("/login", Repo.PostLogin)
	mux.Get("/logout", Repo.Logut)

	mux.Get("/make-reservation", Repo.MakeReservation)
	mux.Get("/cars/{car}", Repo.Car)
	mux.Post("/make-reservation", Repo.PostReservation)
	mux.Get("/reservation-summary", Repo.ReservationSummary)

	mux.Route("/admin", func(mux chi.Router) {
		//  mux.Use(Auth)
		mux.Get("/dashboard", Repo.AdminDashboard)
		mux.Get("/cars", Repo.AdminCars)
		mux.Get("/cars/edit/{id}", Repo.AdminEditCar)
		mux.Post("/cars/edit/{id}", Repo.AdminUpdateCar)
		mux.Get("/cars/add", Repo.AdminAddCar)
		mux.Post("/cars/add", Repo.AdminPostCar)
		mux.Post("/cars/delete/{id}", Repo.AdminDeleteCar)
		mux.Get("/reservations/{id}", Repo.AdminShowReservation)
		mux.Post("/reservations/delete/{id}", Repo.AdminDeleteRes)
	})

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return mux
}

func NoSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   app.InProduction,
		SameSite: http.SameSiteLaxMode,
	})
	return csrfHandler
}

func SessionLoad(next http.Handler) http.Handler {
	return session.LoadAndSave(next)
}

func CreateTemplateCache() (map[string]*template.Template, error) {
	myCache := map[string]*template.Template{}

	//  filepath.Glob zwraca tablicę z dopasowanymi do wzoru stringami - wyszytkie nazwy plików z folderu template
	pages, err := filepath.Glob(fmt.Sprintf("%s/*.page.gohtml", pathToTemplates))
	if err != nil {
		return myCache, err
	}

	for _, page := range pages {
		// Base return part after last '/'
		// page = templates\summary-reservation.page.gohtml
		name := filepath.Base(page)
		// name = summary-reservation.page.gohtml
		// alokuje name w pamięci wynik - &{<nil> 0xc000177ac0 0xc000107680 0xc00004a2a0}
		ts, err := template.New(name).Funcs(functions).ParseFiles(page)
		if err != nil {
			return myCache, err
		}

		//  zwraca tablicę []string z głównym layoutem
		matches, err := filepath.Glob(fmt.Sprintf("%s/*.layout.gohtml", pathToTemplates))
		if err != nil {
			return myCache, err
		}

		if len(matches) > 0 {
			//  jest tym samym to ParseFiles i alokuje plik w pamięci
			ts, err = ts.ParseGlob(fmt.Sprintf("%s/*.layout.gohtml", pathToTemplates))
			if err != nil {
				return myCache, err
			}
		}
		myCache[name] = ts
	}
	// myCache = map[about.page.gohtml:0xc00028c330 contact.page.gohtml:0xc00028cc30 generals.page.gohtml:0xc00008e960 home.page.gohtml:0xc0003043f0 majors.page.gohtml:0xc00028d6e0 make-reservation.page.gohtml:0xc00008f4d0 search-avability.page.gohtml:0xc000305380 summary-reservation.page.gohtml:0xc0000e2330]

	return myCache, nil
}

func TestRepository_Reservation(t *testing.T) {
	reservation := models.Reservation{
		CarID: 1,
	}

	req := httptest.NewRequest("GET", "/make-reservation", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	session.Put(ctx, "reservation", reservation)

	handler := http.HandlerFunc(Repo.MakeReservation)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Reservation handler returns wrong response code %d", rr.Code)
	}

	//  case where reservation is not in the session

	req = httptest.NewRequest("GET", "/make-reservation", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Reservation handler returns wrong response code %d", rr.Code)
	}

	// case with non-existent car
	req = httptest.NewRequest("GET", "/make-reservation", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)

	rr = httptest.NewRecorder()
	reservation.CarID = 100
	session.Put(ctx, "reservation", reservation)
	handler.ServeHTTP(rr, req)

}

func TestRepository_Cars(t *testing.T) {

	req := httptest.NewRequest("GET", "/cars", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))
	req = req.WithContext(ctx)
	handler := http.HandlerFunc(Repo.Cars)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Cars handler returns wrong response code %d", rr.Code)
	}

}

func TestRepository_Car(t *testing.T) {
	// case where address is invalid
	req := httptest.NewRequest("GET", "/cars/", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))
	req = req.WithContext(ctx)
	handler := http.HandlerFunc(Repo.Car)
	// respone and request
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Cars handler returns wrong response code %d", rr.Code)
	}

	// case where address is valid
	req = httptest.NewRequest("GET", "/cars/1", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))
	session.Put(ctx, "car_name", "bullitt")

	req = req.WithContext(ctx)
	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Cars handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_ReservationSummary(t *testing.T) {
	// case where reservation is in session
	req := httptest.NewRequest("GET", "/reservation-summary", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))
	res := models.Reservation{}
	session.Put(ctx, "reservation", res)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(Repo.ReservationSummary)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Reservation summary handler returns wrong response code %d", rr.Code)
	}
	// case where reservation is not in session
	req = httptest.NewRequest("GET", "/reservation-summary", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))
	req = req.WithContext(ctx)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Reservation summary handler returns wrong response code %d", rr.Code)
	}

}

func TestRepository_PostReservation(t *testing.T) {
	// case where everything is correct
	reqBody := "start_date=2030-01-01"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "end_date=2030-01-10")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "first_name=Paul")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "last_name=Johns")

	reqBody = fmt.Sprintf("%s&%s", reqBody, "email=paul@johns.com")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "phone=123456789")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "car_id=1")

	req, _ := http.NewRequest("POST", "/make-reservation", strings.NewReader(reqBody))
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	res := models.Reservation{}
	session.Put(ctx, "reservation", res)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.PostReservation)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}
	// case without POST body
	req, _ = http.NewRequest("POST", "/make-reservation", nil)
	ctx = req.Context()

	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))
	res = models.Reservation{}
	session.Put(ctx, "reservation", res)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}

	// case without reservation in session
	req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	res = models.Reservation{}
	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}

	// case with invalid form

	req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(""))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	res = models.Reservation{}
	session.Put(ctx, "reservation", res)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}

	// case when insert reservation to database
	reqBody = ""
	reqBody = "start_date=2029-01-01"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "end_date=2029-01-10")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "first_name=Tom")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "last_name=Johns")

	reqBody = fmt.Sprintf("%s&%s", reqBody, "email=paul@johns.com")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "phone=123456789")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "car_id=2")
	req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	res = models.Reservation{}
	session.Put(ctx, "reservation", res)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}
	// case while inserting restriction to database
	reqBody = ""
	reqBody = "start_date=2029-01-01"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "end_date=2029-01-10")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "first_name=Paul")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "last_name=Lake")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "car_id=2")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "email=paul@johns.com")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "phone=123456789")
	req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	res = models.Reservation{}
	session.Put(ctx, "reservation", res)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_CheckAvailabilityByDate(t *testing.T) {
	// case where everything is correct
	reqBody := "start=2030-01-01"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2030-01-10")

	req, _ := http.NewRequest("POST", "/check-availability", strings.NewReader(reqBody))
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	car := models.Car{}
	car.ID = 1
	session.Put(ctx, "car", car)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.CheckAvailability)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}

	// case with empty request body
	req, _ = http.NewRequest("POST", "/check-availability", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	car = models.Car{}
	car.ID = 1
	session.Put(ctx, "car", car)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.CheckAvailability)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}

	//  case  with error while checking availability
	req, _ = http.NewRequest("POST", "/check-availability", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	car = models.Car{}
	car.ID = 2
	session.Put(ctx, "car", car)

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.CheckAvailability)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}

	//  case  with no car in session
	req, _ = http.NewRequest("POST", "/check-availability", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.CheckAvailability)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("PostReservation  handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_AdminDashboard(t *testing.T) {
	req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminDashboard)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminDashboard handler returns wrong response code %d", rr.Code)
	}

}

func TestRepository_AdminCars(t *testing.T) {
	req, _ := http.NewRequest("GET", "/admin/cars", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminCars)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminDashboard handler returns wrong response code %d", rr.Code)
	}

}

func TestRepository_AdminEditCar(t *testing.T) {
	// case when carID is empty
	req, _ := http.NewRequest("GET", "/admin/cars/edit/", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminEditCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminEditCar handler returns wrong response code %d", rr.Code)
	}

	// case when carID has value
	req, _ = http.NewRequest("GET", "/admin/cars/edit/1", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminEditCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("AdminEditCar handler returns wrong response code %d", rr.Code)
	}

}

func TestRepository_AdminUpdateCar(t *testing.T) {
	// case with correct data and form
	reqBody := "car_id=1"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "car_name=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "brand=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "model=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "version=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "fuel=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "power=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "gearbox=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "made_at=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "drive=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "combustion=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "body=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "color=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "price=2")

	req, _ := http.NewRequest("POST", "/admin/cars/edit/1", strings.NewReader(reqBody))
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminUpdateCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminUpdateCar handler returns wrong response code %d", rr.Code)
	}

	// case without car id
	req, _ = http.NewRequest("POST", "/admin/cars/edit/", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminUpdateCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminUpdateCar handler returns wrong response code %d", rr.Code)
	}

	// case with not completed form
	reqBody = "car_id=1"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "car_name=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "brand=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "model=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "version=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "fuel=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "power=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "gearbox=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "made_at=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "drive=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "combustion=test")

	req, _ = http.NewRequest("POST", "/admin/cars/edit/1", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminUpdateCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("AdminUpdateCar handler returns wrong response code %d", rr.Code)
	}
	// case with correct form but id is equal to 0
	reqBody = "car_id=0"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "car_name=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "brand=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "model=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "version=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "fuel=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "power=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "gearbox=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "made_at=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "drive=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "combustion=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "body=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "color=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "price=1")

	req, _ = http.NewRequest("POST", "/admin/cars/edit/", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminUpdateCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminUpdateCar handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_AdminAddtCar(t *testing.T) {
	// case when carID is empty
	req, _ := http.NewRequest("GET", "/admin/cars/add", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminAddCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("AdminAddCar handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_AdminPostCar(t *testing.T) {
	// case with correct form
	reqBody := "car_name=test"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "brand=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "model=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "version=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "fuel=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "power=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "gearbox=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "made_at=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "drive=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "combustion=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "body=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "color=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "price=90")

	req, _ := http.NewRequest("POST", "/admin/cars/add", strings.NewReader(reqBody))
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminPostCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminPostCar handler returns wrong response code %d", rr.Code)
	}

	// case without form
	req, _ = http.NewRequest("POST", "/admin/cars/add", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminPostCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminPostCar handler returns wrong response code %d", rr.Code)
	}

	// case with not completed form
	reqBody = "car_name=test"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "brand=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "model=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "version=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "fuel=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "power=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "gearbox=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "made_at=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "drive=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "combustion=test")

	req, _ = http.NewRequest("POST", "/admin/cars/add", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminPostCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("AdminPostCar handler returns wrong response code %d", rr.Code)
	}
	// case with error while adding to db "car.CarName != test"
	reqBody = "car_name=fox"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "brand=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "model=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "version=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "fuel=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "power=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "gearbox=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "made_at=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "drive=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "combustion=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "body=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "color=test")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "price=90")

	req, _ = http.NewRequest("POST", "/admin/cars/add", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminPostCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminPostCar handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_AdminDeleteCar(t *testing.T) {
	// case car id equal to 0
	req, _ := http.NewRequest("POST", "/admin/cars/delete/", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminDeleteCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminDeleteCar handler returns wrong response code %d", rr.Code)
	}

	// case with car id
	req, _ = http.NewRequest("POST", "/admin/cars/delete/1", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminDeleteCar)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("AdminDeleteCar handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_AdminShowReservation(t *testing.T) {
	// case resvervation id equal to 0
	req, _ := http.NewRequest("GET", "/admin/reservations/", nil)
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminShowReservation)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminShowReservation handler returns wrong response code %d", rr.Code)
	}

	// case with car id
	req, _ = http.NewRequest("GET", "/admin/reservations/1", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminShowReservation)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("AdminShowReservation handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_AdminDeleteRes(t *testing.T) {
	reqBody := "email=me@here.com"
	reqBody = fmt.Sprintf("%s&%s", reqBody, "first_name=Paul")
	reqBody = fmt.Sprintf("%s&%s", reqBody, "last_name=Johnes")

	// case res id equal to 0
	req, _ := http.NewRequest("POST", "/admin/reservations/delete/", strings.NewReader(reqBody))
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.AdminDeleteRes)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminDeleteRes handler returns wrong response code %d", rr.Code)
	}

	// case with res id
	req, _ = http.NewRequest("POST", "/admin/reservations/delete/1", strings.NewReader(reqBody))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminDeleteRes)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("AdminDeleteRes handler returns wrong response code %d", rr.Code)
	}

	// case when url is correct but body is nil
	req, _ = http.NewRequest("POST", "/admin/reservations/delete/1", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.AdminDeleteRes)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("AdminDeleteRes handler returns wrong response code %d", rr.Code)
	}
}

func TestRepository_PostLogin(t *testing.T) {
	postForm := url.Values{}

	postForm.Add("email", "me@here.com")
	postForm.Add("password", "admin")

	// case when data is propper
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(postForm.Encode()))
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.PostLogin)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Login form handler returns wrong response code %d", rr.Code)
	}

	// case with wrong credentials
	postForm = url.Values{}

	postForm.Add("email", "me@here.go")
	postForm.Add("password", "admin")

	req, _ = http.NewRequest("POST", "/login", strings.NewReader(postForm.Encode()))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.PostLogin)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Login form handler returns wrong response code %d", rr.Code)
	}

	// case with no post data
	req, _ = http.NewRequest("POST", "/login", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.PostLogin)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Login form handler returns wrong response code %d", rr.Code)
	}

	// case with wrong email
	postForm = url.Values{}

	postForm.Add("email", "wrong email")
	postForm.Add("password", "admin")
	req, _ = http.NewRequest("POST", "/login", strings.NewReader(postForm.Encode()))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.PostLogin)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Login form handler returns wrong response code %d", rr.Code)
	}
}
func TestRepository_PostContactUs(t *testing.T) {
	// case when form data is propper
	postForm := url.Values{}

	postForm.Add("email", "test@here.com")
	postForm.Add("message", "message")

	req, _ := http.NewRequest("POST", "/contact-us", strings.NewReader(postForm.Encode()))
	ctx := req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Repo.PostContactUs)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Login form handler returns wrong response code %d", rr.Code)
	}

	//  case without form
	req, _ = http.NewRequest("POST", "/contact-us", nil)
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.PostContactUs)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Login form handler returns wrong response code %d", rr.Code)
	}
	//  case with wrogn form data
	postForm = url.Values{}
	postForm.Add("message", "a")

	req, _ = http.NewRequest("POST", "/contact-us", strings.NewReader(postForm.Encode()))
	ctx = req.Context()
	ctx, _ = session.Load(ctx, req.Header.Get("X-Session"))

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()

	handler = http.HandlerFunc(Repo.PostContactUs)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Login form handler returns wrong response code %d", rr.Code)
	}

}
func listenForMail() {
	go func() {
		for {
			_ = <-app.MailChan
		}
	}()

}
