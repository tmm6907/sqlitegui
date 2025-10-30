package models

type DB struct {
	ID          int    `db:"id"`
	Name        string `db:"name"`
	Path        string `db:"path"`
	Root        string `db:"root"`
	App_Created bool   `db:"app_created"`
	Created_At  string `db:"created_at"`
}
