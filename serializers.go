package main

type CreateDBRequest struct {
	Name    string `json:"name"`
	Cache   string `json:"cache"`
	Journal string `json:"journal"`
	Sync    string `json:"sync"`
	Lock    string `json:"lock"`
}

type ImportRequest struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type DBResult struct {
	Tables      []string `json:"tables"`
	App_Created bool     `json:"app_created"`
}
