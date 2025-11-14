package models

type ColumnInfo struct {
	Name       string  `db:"name"`
	Type       string  `db:"type"`
	NotNull    bool    `db:"notnull"`
	PK         int     `db:"pk"`
	CID        string  `db:"cid"`
	DFLT_value *string `db:"dflt_value"`
}
