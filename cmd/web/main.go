package main

import (
	"fmt"
	"log"
	"myapp/internal/config"
	"myapp/internal/handlers"
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
	run()
	fmt.Println(fmt.Sprintf("Starting an app on port num %s", portNumber))
	// _ = if ts error i dont care
	// _ = http.ListenAndServe(portNumber, nil)
	srv := &http.Server{
		Addr:    portNumber,
		Handler: routes(&app),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Println(err)
	}
}

func run() {
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
	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Println("Błąd z cahchem tempalta")
	}
	app.TemplateCache = tc
	app.UseCache = false

	//zwraca strukturę repozytorium
	// repo := handlers.NewRepo(&app, db)
	repo := handlers.NewRepo(&app)

	handlers.NewHandlers(repo)
	render.NewRenderer(&app)
}
