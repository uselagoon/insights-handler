package handler

import (
	"testing"
)

func TestFactDuplicateHandler(t *testing.T) {
	dupfacts := []LagoonFact{
		{Name: "duplicatename"},
		{Name: "duplicatename"},
	}

	outfacts, _ := FactDuplicateHandler(dupfacts)

	if outfacts[0].Name == outfacts[1].Name {
		t.Errorf("Fact names should not be duplicated")
	}
}
