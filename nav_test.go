package main

import "testing"

func TestGetNavData(t *testing.T) {
	res := test_app.GetNavData()
	if res.Err != nil {
		t.Errorf("error setting getting nav data: %s", res.Err.Error())
	}
}
