package main

import (
	"encoding/json"
)

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
