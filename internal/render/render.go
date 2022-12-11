package render

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"myapp/internal/config"

	"myapp/internal/models"
	"net/http"
	"path/filepath"

	"github.com/justinas/nosurf"
)

var functions = template.FuncMap{}
var app *config.AppConfig
var pathToTemplates = "./templates"

func NewRenderer(a *config.AppConfig) {
	app = a
}

//  AddDefaultData adds default data to templates
func AddDefaultData(td *models.TemplateData, r *http.Request) *models.TemplateData {
	td.Flash = app.Session.PopString(r.Context(), "flash")
	td.Error = app.Session.PopString(r.Context(), "error")
	td.Warning = app.Session.PopString(r.Context(), "warning")
	td.CSRFToken = nosurf.Token(r)
	if app.Session.Exists(r.Context(), "user_id") {
		td.IsAuthenticated = 1
	}
	return td
}

//  RenderTemplate renders template using html
func Template(w http.ResponseWriter, r *http.Request, tmpl string, td *models.TemplateData) error {
	var tc map[string]*template.Template
	if app.UseCache {
		tc = app.TemplateCache
	} else {
		tc, _ = CreateTemplateCache()
	}

	t, ok := tc[tmpl]

	if !ok {
		return errors.New("cannot get template from cache")
	}

	buf := new(bytes.Buffer)

	td = AddDefaultData(td, r)

	_ = t.Execute(buf, td)

	_, err := buf.WriteTo(w)

	if err != nil {
		return err
	}
	return nil
}

func CreateTemplateCache() (map[string]*template.Template, error) {
	myCache := map[string]*template.Template{}

	//  filepath.Glob retruns slice witch matched pattern
	pages, err := filepath.Glob(fmt.Sprintf("%s/*.page.gohtml", pathToTemplates))
	if err != nil {
		return myCache, err
	}

	for _, page := range pages {
		// Base return part after last '/'
		// page = templates\summary-reservation.page.gohtml
		name := filepath.Base(page)
		// name = summary-reservation.page.gohtml
		// allocates html template - &{<nil> 0xc000177ac0 0xc000107680 0xc00004a2a0}
		ts, err := template.New(name).Funcs(functions).ParseFiles(page)
		if err != nil {
			return myCache, err
		}

		//  returns slice with main layout
		matches, err := filepath.Glob(fmt.Sprintf("%s/*.layout.gohtml", pathToTemplates))
		if err != nil {
			return myCache, err
		}

		if len(matches) > 0 {
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
