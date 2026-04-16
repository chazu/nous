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

// TestGeneralizationsInverse verifies that the specializations/generalizations
// inverse pair is populated correctly after loading.
//
// Structure declares specializations=[Set,List,Bag] so Set,List,Bag should
// each have generalizations populated with [Structure].
func TestGeneralizationsInverse(t *testing.T) {
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "math"); err != nil {
		t.Fatal(err)
	}

	structure := store.Get("Structure")
	if structure == nil {
		t.Fatal("Structure not found")
	}
	fmt.Printf("Structure.specializations = %v\n", structure.GetStrings("specializations"))

	for _, name := range []string{"Set", "List", "Bag"} {
		u := store.Get(name)
		if u == nil {
			t.Errorf("%s not found", name)
			continue
		}
		gens := u.GetStrings("generalizations")
		fmt.Printf("%s.generalizations = %v\n", name, gens)
		if len(gens) == 0 {
			t.Errorf("expected %s.generalizations to be populated", name)
		}
	}

	// Also check SetOfNumbers via specializations inverse.
	num := store.Get("Number")
	if num != nil {
		fmt.Printf("Number.specializations = %v\n", num.GetStrings("specializations"))
	}
	evenNum := store.Get("EvenNum")
	if evenNum != nil {
		fmt.Printf("EvenNum.generalizations = %v\n", evenNum.GetStrings("generalizations"))
	}
}
