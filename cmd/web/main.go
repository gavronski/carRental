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
	"strconv"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
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

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbName := os.Getenv("DB_NAME")
	dbPass := os.Getenv("DB_PASS")
	dbUser := os.Getenv("DB_USER")
	dbPort := os.Getenv("DB_PORT")
	dbHost := os.Getenv("DB_HOST")
	dbSSL := os.Getenv("DB_SSL")

	cacheENV := os.Getenv("CACHE")
	useCache, _ := strconv.ParseBool(cacheENV)
	prodENV := os.Getenv("IN_PRODUCTION")
	inProduction, _ := strconv.ParseBool(prodENV)
	app.InProduction = inProduction

	// Create a seesion
	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction
	app.Session = session

	// Connect to database
	log.Println("Connecting to databse ...")
	connectionString := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s", dbHost, dbPort, dbName, dbUser, dbPass, dbSSL)
	db, err := driver.ConnectSQL(connectionString)
	if err != nil {
		log.Fatal(err)
	}
	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Println("Chache rendering error", err)
	}
	app.TemplateCache = tc
	app.UseCache = useCache

	repo := handlers.NewRepo(&app, db)
	helpers.NewHelpers(&app)
	handlers.NewHandlers(repo)
	render.NewRenderer(&app)
	return db, nil
}
