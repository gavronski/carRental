package handlers

import (
	"myapp/internal/config"
	"myapp/internal/models"
	"myapp/internal/render"
	"net/http"
)

//Repo the repository used by the handlers
var Repo *Repository

//Repository is the repository typ
type Repository struct {
	App *config.AppConfig
	// DB  repository.DatabaseRepo
}

//NewRepo create new repository
// func NewRepo(a *config.AppConfig, db *driver.DB) *Repository {
func NewRepo(a *config.AppConfig) *Repository {
	return &Repository{
		App: a,
		// DB:  dbrepo.NewPostgresRepo(db.SQL, a),
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
	render.Template(w, r, "cars.page.gohtml", &models.TemplateData{})
}

func (m *Repository) Car(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "car.page.gohtml", &models.TemplateData{})
}
