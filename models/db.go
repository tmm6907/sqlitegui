package models

type DB struct {
	ID         int    `db:"id"`
	Name       string `db:"name"`
	Path       string `db:"path"`
	Created_At string `db:"created_at"`
}
