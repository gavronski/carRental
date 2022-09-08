package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"myapp/internal/config"
	"myapp/internal/driver"
	"myapp/internal/handlers"
	"myapp/internal/helpers"
	"myapp/internal/models"
	"myapp/internal/render"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
)

const portNumber = ":8088"

var app config.AppConfig
var session *scs.SessionManager
var infoLog *log.Logger
var errorLog *log.Logger

func main() {
	db, err := run()
	fmt.Println(fmt.Sprintf("Starting an app on port num %s", portNumber))
	// _ = if ts error i dont care
	// _ = http.ListenAndServe(portNumber, nil)
	srv := &http.Server{
		Addr:    portNumber,
		Handler: routes(&app),
	}
	defer db.SQL.Close()
	defer close(app.MailChan)

	listenForMail()
	err = srv.ListenAndServe()
	if err != nil {
		log.Println(err)
	}
}

func run() (*driver.DB, error) {
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Car{})
	gob.Register(models.Restriction{})

	mailChan := make(chan models.MailData)
	app.MailChan = mailChan
	app.InProduction = false
	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	app.InfoLog = infoLog
	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	app.ErrorLog = errorLog

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction
	app.Session = session

	//connect to database
	log.Println("Connecting to databse ...")
	db, err := driver.ConnectSQL("host=localhost port=5432 dbname=car_rental user=postgres password=root")
	if err != nil {
		log.Fatal(err)
	}
	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Println("Błąd z cachem tempalta")
	}
	app.TemplateCache = tc
	app.UseCache = false

	//zwraca strukturę repozytorium
	// repo := handlers.NewRepo(&app, db)
	repo := handlers.NewRepo(&app, db)
	helpers.NewHelpers(&app)
	handlers.NewHandlers(repo)
	render.NewRenderer(&app)
	return db, nil
}
