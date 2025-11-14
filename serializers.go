package main

import "encoding/json"

type CreateDBRequest struct {
	Name    string `json:"name"`
	Cache   string `json:"cache"`
	Journal string `json:"journal"`
	Sync    string `json:"sync"`
	Lock    string `json:"lock"`
}
type UpdateRequest struct {
	DB     string  `json:"db"`
	Table  string  `json:"table"`
	Row    [][]any `json:"row"`
	Column string  `json:"column"`
	Value  string  `json:"value"`
}
type QueryRequest struct {
	Query    string `json:"query"`
	Editable bool   `json:"editable"`
}

type AppResult struct {
	Err     error `json:"error"`
	Results any   `json:"results"`
}

func (r *AppResult) Error() string {
	if r.Err == nil {
		return ""
	}
	return r.Err.Error()
}

func (r *AppResult) MarshalJSON() ([]byte, error) {
	tmp := &struct {
		Err     string `json:"error,omitempty"`
		Results any    `json:"results"`
	}{Err: r.Error(), Results: r.Results}
	return json.Marshal(tmp)
}
