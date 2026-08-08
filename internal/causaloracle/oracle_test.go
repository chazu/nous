package causaloracle

import "testing"

func TestUniverse(t *testing.T) {
	u := Enumerate()
	if len(u) != 72 {
		t.Fatalf("models=%d", len(u))
	}
	classes := map[string]int{}
	for _, m := range u {
		c, _ := Code(m)
		s, _ := Signature(c)
		classes[s]++
	}
	if len(classes) != 58 {
		t.Fatalf("classes=%d", len(classes))
	}
}
func TestTeacherRights(t *testing.T) {
	m := Enumerate()[0]
	code, _ := Code(m)
	teacher, e := NewTeacher("opaque", code)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = teacher.Respond("wrong", "do:0=0"); e == nil {
		t.Fatal("wrong token accepted")
	}
	if _, e = teacher.Respond("opaque", "do:0=0"); e != nil {
		t.Fatal(e)
	}
}
