package config

import (
	"html/template"
	"log"
	"myapp/internal/models"

	"github.com/alexedwards/scs/v2"
)

// App config holds the application config
type AppConfig struct {
	UseCache      bool
	TemplateCache map[string]*template.Template
	InProduction  bool
	InfoLog       *log.Logger
	ErrorLog      *log.Logger
	Session       *scs.SessionManager
	MailChan      chan models.MailData
}
