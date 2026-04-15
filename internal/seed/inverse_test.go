package seed

import (
	"fmt"
	"testing"
	
	"github.com/chazu/nous/internal/unit"
)

func TestInverseIndex(t *testing.T) {
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}
	
	set := store.Get("Set")
	if set == nil {
		t.Fatal("Set not found")
	}
	isRangeOf := set.GetStrings("isRangeOf")
	fmt.Printf("Set.isRangeOf = %v\n", isRangeOf)
	if len(isRangeOf) == 0 {
		t.Error("expected Set to have isRangeOf populated from inverse of range")
	}
	
	number := store.Get("Number")
	if number != nil {
		fmt.Printf("Number.isRangeOf = %v\n", number.GetStrings("isRangeOf"))
	}
}
