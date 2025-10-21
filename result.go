package main

type Result struct {
	ErrStr  string `json:"error"`
	Results any    `json:"results"`
}
