package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// eolfunctions.go contains all the basic functionality for checking key facts' end of life status against https://endoflife.date/docs/api

const eloCacheLocation = "/tmp/eol.cache"

type PackageInfo struct {
	Cycle             string `json:"cycle"`
	ReleaseDate       string `json:"releaseDate"`
	EOL               string `json:"eol"`
	Latest            string `json:"latest"`
	LatestReleaseDate string `json:"latestReleaseDate"`
	Link              string `json:"link"`
	LTS               bool   `json:"lts"`
}

type EOLData struct {
	Packages      map[string][]PackageInfo
	CacheLocation string
}

type NewEOLDataArgs struct {
	Packages            []string
	CacheLocation       string
	PreventCacheRefresh bool
	ForceCacheRefresh   bool
}

// NewEOLData creates a new EOLData struct with end-of-life information for the provided Packages.
func NewEOLData(args NewEOLDataArgs) (*EOLData, error) {

	// basic assertions of logic
	if args.ForceCacheRefresh && args.PreventCacheRefresh {
		return nil, fmt.Errorf("you cannot Force Cache Refresh AND Prevent Cache Refresh")
	}

	packages := args.Packages
	cacheLocation := args.CacheLocation
	data := &EOLData{
		CacheLocation: cacheLocation,
	}

	// Check if cache file exists
	if _, err := os.Stat(data.CacheLocation); err == nil || args.ForceCacheRefresh {
		// Cache file exists, load data from file
		if err := loadDataFromFile(data.CacheLocation, data); err != nil {
			return nil, err
		}
	} else if os.IsNotExist(err) {
		if args.PreventCacheRefresh {
			return nil, fmt.Errorf("cache not found and Prevent Cache Refresh enabled")
		}
		// Cache file does not exist, fetch data and write to file
		endOfLifeInfo := GetEndOfLifeInfo(packages)
		data.Packages = endOfLifeInfo

		// Write to cache file
		if err := writeDataToFile(data.CacheLocation, data); err != nil {
			return nil, err
		}
	} else {
		// Some other error occurred
		return nil, err
	}

	return data, nil
}

func GetEndOfLifeInfo(packageNames []string) map[string][]PackageInfo {
	endOfLifeInfo := make(map[string][]PackageInfo)

	for _, packageName := range packageNames {
		url := fmt.Sprintf("https://endoflife.date/api/%s.json", packageName)
		response, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error getting end of life info for %s: %v\n", packageName, err)
			continue
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("Error reading response body for %s: %v\n", packageName, err)
			continue
		}

		var data []PackageInfo
		if err := json.Unmarshal(body, &data); err != nil {
			fmt.Printf("error parsing JSON for %s: %v\n", packageName, err)
			continue
		}

		// Assuming the API returns an array of PackageInfo
		if len(data) > 0 {
			endOfLifeInfo[packageName] = data // Assuming we're interested in the first entry
		}
	}

	return endOfLifeInfo
}

// loadDataFromFile loads data from a file into an EOLData struct.
func loadDataFromFile(filename string, data *EOLData) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(data); err != nil {
		return err
	}

	return nil
}

// writeDataToFile writes data from an EOLData struct to a file.
func writeDataToFile(filename string, data *EOLData) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, jsonData, 0644); err != nil {
		return err
	}

	return nil
}
