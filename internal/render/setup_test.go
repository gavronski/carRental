package render

import (
	"encoding/gob"
	"myapp/internal/config"
	"myapp/internal/models"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
)

var session *scs.SessionManager
var testApp config.AppConfig

func TestMain(m *testing.M) {

	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Car{})
	gob.Register(models.Restriction{})
	testApp.InProduction = false
	testApp.UseCache = true
	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = false
	testApp.Session = session
	app = &testApp
	os.Exit(m.Run())
}

type writer struct{}

func (m *writer) Header() http.Header {
	var header http.Header
	return header
}
func (m *writer) WriteHeader(statusCode int) {

}
func (m *writer) Write(b []byte) (int, error) {
	length := len(b)
	return length, nil
}
