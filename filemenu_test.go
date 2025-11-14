package main

import (
	"os"
	"testing"
)

func TestExport(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"csv", false},
	}

	for _, tt := range tests {
		test_app.exportDB(tt.format)
		dest := ""
		if _, err := os.Stat(dest); (err != nil) != tt.wantErr {
			t.Error("Have %s want expected error %v", err.Error(), tt.wantErr)
		}
	}
}
