package dbrepo

import (
	"database/sql"
	"myapp/internal/config"
	"myapp/internal/repository"
)

type postgresDBRepo struct {
	App *config.AppConfig
	DB  *sql.DB
}

func NewPostgresRepo(conn *sql.DB, a *config.AppConfig) repository.DatabaseRepo {
	return &postgresDBRepo{
		App: a,
		DB:  conn,
	}
}

// testDBREpo created for testing db functions
type testDBRepo struct {
	App *config.AppConfig
	DB  *sql.DB
}

// NewTestingRepo sets testDBRepo
func NewTestingRepo(a *config.AppConfig) repository.DatabaseRepo {
	return &testDBRepo{
		App: a,
	}
}
