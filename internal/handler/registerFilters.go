package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var KeyFactFilters []func(filter parserFilter) parserFilter

type FactTransformNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type FactTransform struct {
	Type            string                   `json:"type"`
	Lookupvalue     []FactTransformNameValue `json:"lookupvalue"`
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

		for i, _ := range transform.Lookupvalue {

			n := transform.Lookupvalue[i].Name
			v := transform.Lookupvalue[i].Value
			filter = filter.fieldContains(n, v)
		}

		for i, _ := range transform.Transformations {
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

func init() {

	transforms, err := LoadTransformsFromDisk("./keyfacts.json")
	if err != nil {
		fmt.Println(err)
	}
	for _, transform := range transforms {
		filter, _ := GenerateFilterFromTransform(transform)
		registerFilter(filter)
	}

	//Some further examples of registering
	// Let's register the standard key fact filters as they currently exist
	//factRegexes, err := scanKeyFactsFile("./syft_key_facts.txt")
	//if err != nil {
	//	fmt.Errorf("scan file error: %w", err)
	//}
	//for _, k := range factRegexes {
	//	workingRegex := k
	//	registerFilter(func(filter parserFilter) parserFilter {
	//		return filter.fieldContains("Name", workingRegex).setKeyFact(true)
	//	})
	//}
	//
	//// Let's register the standard key fact filters as they currently exist
	//factRegexes, err = scanKeyFactsFile("./env_var_key_facts.txt")
	//if err != nil {
	//	fmt.Errorf("scan file error: %w", err)
	//}
	//for _, k := range factRegexes {
	//	workingRegex := k
	//	registerFilter(func(filter parserFilter) parserFilter {
	//		return filter.fieldContains("Key", workingRegex).setKeyFact(true)
	//	})
	//}

}
