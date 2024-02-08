package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/CycloneDX/cyclonedx-go"
	"io/ioutil"
	"net/http"
	"strings"
)

// getBOMfromPayload is used to extract a *cdx.BOM from an incoming payload
func getBOMfromPayload(v string) (*cyclonedx.BOM, error) {
	bom := new(cyclonedx.BOM)

	// Decode base64
	r := strings.NewReader(v)
	dec := base64.NewDecoder(base64.StdEncoding, r)

	res, err := ioutil.ReadAll(dec)
	if err != nil {
		return nil, err
	}

	fileType := http.DetectContentType(res)

	if fileType != "application/zip" && fileType != "application/x-gzip" && fileType != "application/gzip" {
		decoder := cyclonedx.NewBOMDecoder(bytes.NewReader(res), cyclonedx.BOMFileFormatJSON)
		if err = decoder.Decode(bom); err != nil {
			return nil, err
		}
	} else {
		// Compressed cyclonedx sbom
		result, decErr := decodeGzipString(v)
		if decErr != nil {
			return nil, decErr
		}
		b, mErr := json.MarshalIndent(result, "", " ")
		if mErr != nil {
			return nil, mErr
		}

		decoder := cyclonedx.NewBOMDecoder(bytes.NewReader(b), cyclonedx.BOMFileFormatJSON)
		if err = decoder.Decode(bom); err != nil {
			return nil, err
		}
	}
	return bom, nil
}
