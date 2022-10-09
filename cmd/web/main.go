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
	// Server settings
	srv := &http.Server{
		Addr:    portNumber,
		Handler: routes(&app),
	}
	defer db.SQL.Close()
	defer close(app.MailChan)
	// Start listen
	listenForMail()
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

// run sets app attributes, creates session and returns driver
func run() (*driver.DB, error) {
	// Register models
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Car{})
	gob.Register(models.Restriction{})

	// Create channel for mails
	mailChan := make(chan models.MailData)
	app.MailChan = mailChan

	app.InProduction = false

	// Create a seesion
	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction
	app.Session = session

	// Connect to database
	log.Println("Connecting to databse ...")
	db, err := driver.ConnectSQL("host=localhost port=5432 dbname=car_rental user=postgres password=root")
	if err != nil {
		log.Fatal(err)
	}
	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Println("Chache rendering error", err)
	}
	app.TemplateCache = tc
	app.UseCache = false

	repo := handlers.NewRepo(&app, db)
	helpers.NewHelpers(&app)
	handlers.NewHandlers(repo)
	render.NewRenderer(&app)
	return db, nil
}
