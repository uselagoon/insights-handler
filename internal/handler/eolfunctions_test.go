package handler

import (
	"fmt"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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
		{
			name:    "Test Assertion Failure",
			wantErr: true,
			args: args{
				EolArgs: NewEOLDataArgs{
					Packages: []string{
						"alpine",
					},
					CacheLocation:       "testnocache.json",
					ForceCacheRefresh:   true,
					PreventCacheRefresh: true,
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
			if err != nil { // We've got an error, and probably don't need to report it, 'cause it was expected
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
		{
			name:    "Test Nonexistent cache failure",
			wantErr: true,
			args: args{
				EolArgs: NewEOLDataArgs{
					Packages: []string{
						"alpine",
					},
					CacheLocation:       "doesntexist.json",
					ForceCacheRefresh:   false,
					PreventCacheRefresh: true,
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
			if err != nil {
				return
			}
			if len(got.Packages[tt.args.EolArgs.Packages[0]]) == 0 {
				t.Errorf("Could not find any Packages for package '%v'\n", tt.args.EolArgs.Packages[0])
			}
		})
	}
}

func TestEOLData_EolDataForPackage(t1 *testing.T) {
	type fields struct {
		Packages      []string
		CacheLocation string
	}
	type args struct {
		packageName string
		ver         string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    PackageInfo
		wantErr bool
	}{
		{
			name: "Find basic package",
			fields: fields{
				Packages:      []string{"alpine"},
				CacheLocation: "testassets/EOLdata/testnocache.json",
			},
			args: args{
				packageName: "alpine",
				ver:         "3.16",
			},
			want: PackageInfo{
				Cycle:             "3.16",
				ReleaseDate:       "2022-05-23",
				EOL:               "2024-05-23",
				Latest:            "3.16.9",
				LatestReleaseDate: "2024-01-26",
				Link:              "https://alpinelinux.org/posts/Alpine-3.16.9-3.17.7-3.18.6-released.html",
				LTS:               false,
			},
		},
		{
			name: "No match",
			fields: fields{
				Packages:      []string{"alpine"},
				CacheLocation: "testassets/EOLdata/testnocache.json",
			},
			args: args{
				packageName: "NOMATCH",
				ver:         "1.1",
			},
			want:    PackageInfo{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t, err := NewEOLData(NewEOLDataArgs{
				Packages:            tt.fields.Packages,
				CacheLocation:       tt.fields.CacheLocation,
				PreventCacheRefresh: true,
				ForceCacheRefresh:   false,
			})

			if err != nil {
				//This shouldn't happen, so let's just die
				t1.Errorf("Issue with checking packages - have to exit")
			}

			got, err := t.EolDataForPackage(tt.args.packageName, tt.args.ver)
			if (err != nil) != tt.wantErr {
				t1.Errorf("EolDataForPackage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("EolDataForPackage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEOLData_GenerateProblemsForPackages(t1 *testing.T) {
	type initData struct {
		Packages []string
		InitTime time.Time
	}
	type args struct {
		packages      map[string]string
		environmentId int
		service       string
	}
	tests := []struct {
		name     string
		initData initData
		args     args
		want     []lagoonclient.LagoonProblem
		wantErr  bool
	}{
		{
			name: "Check > EOL",
			initData: initData{
				Packages: []string{"alpine"},
				InitTime: time.Now(),
			},
			args: args{
				packages: map[string]string{
					"alpine": "3.19",
				},
				environmentId: 0,
				service:       "",
			},
			wantErr: false,
			want:    nil,
		},
		{
			name: "Check < EOL",
			initData: initData{
				Packages: []string{"alpine"},
				InitTime: time.Date(1990, time.January, 1, 1, 1, 1, 1, time.Local),
			},
			args: args{
				packages: map[string]string{
					"alpine": "3.19",
				},
				environmentId: 0,
				service:       "",
			},
			wantErr: false,
			want: []lagoonclient.LagoonProblem{
				lagoonclient.LagoonProblem{
					Environment:       0,
					Identifier:        fmt.Sprintf("EOL-%v-%v", "alpine", "3.19"),
					Version:           "3.19",
					FixedVersion:      "",
					Source:            "insights-handler-EOLData",
					Service:           "",
					Data:              "{}",
					Severity:          "",
					SeverityScore:     0,
					AssociatedPackage: "",
					Description:       fmt.Sprintf("Package '%v' is at End-of-life as of '%v'", "alpine", "2025-11-01"),
					Links:             "",
				},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t, err := NewEOLData(NewEOLDataArgs{
				Packages:            tt.initData.Packages,
				CacheLocation:       "testassets/EOLdata/cachedata.json",
				PreventCacheRefresh: true,
				ForceCacheRefresh:   false,
			})

			got, err := t.GenerateProblemsForPackages(tt.args.packages, tt.args.environmentId, tt.args.service)
			if (err != nil) != tt.wantErr {
				t1.Errorf("GenerateProblemsForPackages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GenerateProblemsForPackages() got = %v, want %v", got, tt.want)
			}
		})
	}
}
