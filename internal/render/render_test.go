package render

import (
	"myapp/internal/models"
	"net/http"
	"testing"
)

func TestAddDefaultData(t *testing.T) {
	var td models.TemplateData
	r, err := getSession()
	if err != nil {
		t.Error(err)
	}
	result := AddDefaultData(&td, r)

	if result.Flash != "test" {
		t.Error("expected result is: test")
	}
}

func TestTemplate(t *testing.T) {
	pathToTemplates = "./../../templates"
	tc, err := CreateTemplateCache()
	if err != nil {
		t.Error(err)
	}
	app.TemplateCache = tc
	var w writer
	rw := w
	r, err := getSession()
	if err != nil {
		t.Error(err)
	}

	err = Template(&rw, r, "home.page.gohtml", &models.TemplateData{})
	if err != nil {
		t.Error(err)
	}
}

func getSession() (*http.Request, error) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	// new ctx with session data
	ctx, err = session.Load(ctx, r.Header.Get("X-Session"))
	if err != nil {
		return nil, err

	}
	r = r.WithContext(ctx)

	session.Put(r.Context(), "flash", "test")

	return r, nil
}

func TesCreateTemplateCache(t *testing.T) {
	_, err := CreateTemplateCache()
	if err != nil {
		t.Error(err)
	}
}

func TestNewRender(t *testing.T) {
	NewRenderer(app)
}
