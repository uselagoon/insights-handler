package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGetEndOfLifeInfo(t *testing.T) {
	type args struct {
		packageNames []string
	}
	tests := []struct {
		name         string
		args         args
		wantResponse bool
	}{
		{
			name: "Get alpine information",
			args: args{packageNames: []string{
				"alpine",
				"ubuntu",
			}},
			wantResponse: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEndOfLifeInfo(tt.args.packageNames)

			if tt.wantResponse == true {
				for _, name := range tt.args.packageNames {
					if len(got[name]) == 0 {
						t.Errorf("Expected data for package %v, got nothing", name)
					}
				}
			}
		})
	}
}

func TestNewEOLData(t *testing.T) {
	type args struct {
		EolArgs NewEOLDataArgs
	}
	tests := []struct {
		name    string
		args    args
		want    *EOLData
		wantErr bool
	}{
		{
			name: "Test No Cache",
			args: args{
				EolArgs: NewEOLDataArgs{
					Packages: []string{
						"alpine",
					},
					CacheLocation: "testnocache.json",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Let's set up a temporary location to save the incoming data
			dir, err := os.MkdirTemp("", "*-test")
			if err != nil {
				t.Errorf("Unable to create test directory")
				return
			}
			defer func(path string) {
				err := os.RemoveAll(path)
				if err != nil {
					fmt.Println("Unable to remove directory: ", path)
				}
			}(dir)
			tt.args.EolArgs.CacheLocation = filepath.Join(dir, tt.args.EolArgs.CacheLocation)
			got, err := NewEOLData(tt.args.EolArgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEOLData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got.Packages[tt.args.EolArgs.Packages[0]]) == 0 {
				t.Errorf("Could not find any Packages for package '%v'\n", tt.args.EolArgs.Packages[0])
			}

		})
	}
}

func TestNewEOLDataWithExistingCache(t *testing.T) {
	type args struct {
		EolArgs NewEOLDataArgs
	}
	tests := []struct {
		name    string
		args    args
		want    *EOLData
		wantErr bool
	}{
		{
			name: "Test No Cache",
			args: args{
				EolArgs: NewEOLDataArgs{
					Packages: []string{
						"alpine",
					},
					CacheLocation: "testassets/EOLdata/testnocache.json",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Let's set up a temporary location to save the incoming data
			got, err := NewEOLData(tt.args.EolArgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEOLData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got.Packages[tt.args.EolArgs.Packages[0]]) == 0 {
				t.Errorf("Could not find any Packages for package '%v'\n", tt.args.EolArgs.Packages[0])
			}
		})
	}
}
