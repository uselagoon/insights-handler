package handler

import (
	"encoding/json"
	"io/ioutil"
)

var KeyFactFilters []func(filter parserFilter) parserFilter

type FactTransformNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type FactLookupNameValue struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	ExactMatch bool   `json:"exactMatch"`
}

type FactTransform struct {
	Type            string                   `json:"type"`
	Lookupvalue     []FactLookupNameValue    `json:"lookupvalue"`
	Transformations []FactTransformNameValue `json:"transformations"`
	Keyfact         bool                     `json:"keyfact"`
}

type FactTransforms struct {
	Transforms []FactTransform `json:"transforms"`
}

func registerFilter(filter func(filter parserFilter) parserFilter) {
	KeyFactFilters = append(KeyFactFilters, filter)
}

func ProcessLagoonFactAgainstRegisteredFilters(fact LagoonFact, insightsRawData interface{}) (LagoonFact, error) {
	for _, filter := range KeyFactFilters {
		pf := FactProcessor{
			Fact:          fact,
			InsightsData:  insightsRawData,
			hasErrorState: false,
			theError:      nil,
			filteredOut:   false,
		}
		pfout := filter(&pf)
		if !pfout.isFilteredOut() {
			fact = pfout.getFact()
		}

	}
	return fact, nil
}

func LoadTransformsFromDisk(filename string) ([]FactTransform, error) {
	ret := FactTransforms{}

	file, err := ioutil.ReadFile(filename)

	if err != nil {
		return ret.Transforms, err
	}
	err = json.Unmarshal(file, &ret)

	if err != nil {
		return ret.Transforms, err
	}
	return ret.Transforms, nil
}

func GenerateFilterFromTransform(transform FactTransform) (func(filter parserFilter) parserFilter, error) {

	return func(filter parserFilter) parserFilter {

		if transform.Type != "" {
			filter = filter.isOfType(transform.Type)
		}

		for i := range transform.Lookupvalue {

			n := transform.Lookupvalue[i].Name
			v := transform.Lookupvalue[i].Value
			em := transform.Lookupvalue[i].ExactMatch
			if em {
				filter = filter.fieldContainsExactMatch(n, v)
			} else {
				filter = filter.fieldContains(n, v)
			}

		}

		for i := range transform.Transformations {
			n := transform.Transformations[i].Name
			v := transform.Transformations[i].Value
			filter = filter.setFactField(n, v)
		}

		if transform.Keyfact {
			filter = filter.setKeyFact(true)
		}

		return filter
	}, nil

}

func RegisterFiltersFromJson(filename string) error {

	transforms, err := LoadTransformsFromDisk(filename)
	if err != nil {
		return err
	}
	for _, transform := range transforms {
		filter, _ := GenerateFilterFromTransform(transform)
		registerFilter(filter)
	}

	return nil
}
