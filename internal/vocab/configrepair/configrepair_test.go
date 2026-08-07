package configrepair

import (
	"reflect"
	"strings"
	"testing"
)

func TestConfigValidationAndCanonicalization(t *testing.T) {
	input := []string{"tls=true", "environment=production", "replicas=2"}
	want := []string{"environment=production", "replicas=2", "tls=true"}
	got, err := Canonicalize(input)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Canonicalize = (%v,%v), want %v", got, err, want)
	}
	invalid := [][]string{
		{""}, {"a"}, {"=x"}, {"a="}, {"a=x=y"}, {"a=x", "a=y"},
		{"1a=x"}, {"a=x y"}, {strings.Repeat("a", MaxKeyBytes+1) + "=x"},
	}
	for _, data := range invalid {
		if ValidConfig(data) {
			t.Fatalf("ValidConfig(%v) = true", data)
		}
	}
}

func TestSchemaValidationAndSatisfaction(t *testing.T) {
	schema := seedSchema(false)
	if !ValidSchema(schema) {
		t.Fatal("seed schema rejected")
	}
	valid := []string{
		"environment=production", "tls=true", "service_port=443",
		"replicas=2", "admin_public=false", "redirect_http=false",
	}
	if satisfied, err := Satisfies(valid, schema); err != nil || !satisfied {
		t.Fatalf("valid config = (%v,%v)", satisfied, err)
	}
	for _, change := range []Repair{{"service_port", "80"}, {"replicas", "1"}, {"admin_public", "true"}} {
		config, _ := Apply(valid, change)
		if satisfied, err := Satisfies(config, schema); err != nil || satisfied {
			t.Fatalf("constraint violation %v = (%v,%v)", change, satisfied, err)
		}
	}
	if satisfied, err := Satisfies(append(valid, "unknown=x"), schema); err != nil || satisfied {
		t.Fatalf("unknown field = (%v,%v), want false,nil", satisfied, err)
	}
}

func TestSchemaRejectsMalformedTypedAndDuplicateRecords(t *testing.T) {
	invalid := [][]string{
		{"field:n:int:00:10"},
		{"field:n:int:0:+10"},
		{"field:n:int:10:0"},
		{"field:b:bool", "eq-if:b=yes,b=true"},
		{"field:n:int:0:10", "min-if:n=2,n=11"},
		{"field:a:string", "field:a:bool"},
		{"field:a:string", "required:a", "required:a"},
		{"field:a:string", "protected:a", "protected:a"},
		{"required:missing"},
		{"field:a:string", "eq-if:a=x,missing=y"},
	}
	for _, schema := range invalid {
		if ValidSchema(schema) {
			t.Fatalf("ValidSchema(%v) = true", schema)
		}
	}
	unsatisfiable := []string{"field:b:bool", "required:b", "eq-if:b=true,b=false"}
	if !ValidSchema(unsatisfiable) {
		t.Fatal("structurally valid unsatisfiable schema rejected")
	}
}

func TestRepairPlanIsCanonicalIdempotentAndPermutationInvariant(t *testing.T) {
	input := seedConfig("80", "0", "true", false, 0)
	repairs := []Repair{
		{"service_port", "443"}, {"replicas", "2"}, {"admin_public", "false"},
	}
	first, err := ApplyPlan(input, repairs)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ApplyPlan(first, repairs)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("second application = (%v,%v), want %v", second, err, first)
	}
	changed, _ := ChangedKeys(input, first)
	secondChanged, _ := ChangedKeys(first, second)
	if changed != 3 || secondChanged != 0 {
		t.Fatalf("changed counts = %d then %d", changed, secondChanged)
	}
	permuted, err := ApplyPlan(input, []Repair{repairs[2], repairs[0], repairs[1]})
	if err != nil || !reflect.DeepEqual(first, permuted) {
		t.Fatalf("permuted plan = (%v,%v), want %v", permuted, err, first)
	}
	if ValidPlan([]Repair{{"a", "x"}, {"a", "y"}}) || ValidPlan(nil) ||
		ValidPlan([]Repair{{"a", "x"}, {"b", "y"}, {"c", "z"}, {"d", "q"}}) {
		t.Fatal("invalid plan accepted")
	}
}

func TestProtectedIntentIsIndependentOfSatisfaction(t *testing.T) {
	schema := seedSchema(false)
	input := seedConfig("80", "0", "true", false, 0)
	shortcut, err := ApplyPlan(input, []Repair{{"tls", "false"}, {"environment", "development"}})
	if err != nil {
		t.Fatal(err)
	}
	if satisfied, err := Satisfies(shortcut, schema); err != nil || !satisfied {
		t.Fatalf("shortcut satisfaction = (%v,%v)", satisfied, err)
	}
	if preserved, err := PreservesProtected(input, shortcut, schema); err != nil || preserved {
		t.Fatalf("shortcut intent = (%v,%v), want false,nil", preserved, err)
	}
}

func TestProtectedComparisonRejectsTypeInvalidValues(t *testing.T) {
	schema := []string{"field:tls:bool", "field:count:int:0:10", "protected:tls", "protected:count"}
	for _, configuration := range [][]string{{"tls=yes", "count=2"}, {"tls=true", "count=02"}} {
		if preserved, err := PreservesProtected(configuration, configuration, schema); err == nil || preserved {
			t.Fatalf("type-invalid protected values = (%v,%v), want error", preserved, err)
		}
	}
}

func TestDecisionKeyUsesAssignmentsNotOrder(t *testing.T) {
	a := []Repair{{"port", "443"}, {"replicas", "2"}, {"admin", "false"}}
	b := []Repair{{"admin", "false"}, {"port", "443"}, {"replicas", "2"}}
	first, err := DecisionKey(a)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DecisionKey(b)
	if err != nil || first != second || !strings.HasPrefix(first, "sha256:v1:") || len(first) >= 512 {
		t.Fatalf("decision keys = %q %q err=%v", first, second, err)
	}
	different, _ := DecisionKey([]Repair{{"admin", "true"}, {"port", "443"}, {"replicas", "2"}})
	if different == first {
		t.Fatal("different semantic assignment reused decision key")
	}
}

func TestSeedCorpusHasOneQualifyingPlanAndIntentGateIsEssential(t *testing.T) {
	repairs := []Repair{
		{"service_port", "443"}, {"replicas", "2"}, {"admin_public", "false"},
		{"redirect_http", "true"}, {"tls", "false"}, {"environment", "development"},
	}
	examples := []struct {
		config []string
		schema []string
	}{
		{seedConfig("80", "0", "true", false, 0), seedSchema(false)},
		{seedConfig("443", "0", "true", false, 0), seedSchema(false)},
		{seedConfig("80", "2", "true", true, 12), seedSchema(true)},
		{seedConfig("80", "2", "false", true, 5), seedSchema(true)},
	}
	qualified := 0
	validityOnly := 0
	for _, plan := range subsets(repairs, MaxPlanSize) {
		support, constraintSupport := 0, 0
		for _, example := range examples {
			result, err := ApplyPlan(example.config, plan)
			if err != nil {
				t.Fatal(err)
			}
			satisfied, _ := Satisfies(result, example.schema)
			preserved, _ := PreservesProtected(example.config, result, example.schema)
			second, _ := ApplyPlan(result, plan)
			if satisfied {
				constraintSupport++
			}
			if satisfied && preserved && reflect.DeepEqual(result, second) {
				support++
			}
		}
		if constraintSupport == len(examples) {
			validityOnly++
		}
		if support == len(examples) {
			qualified++
			if !reflect.DeepEqual(plan, repairs[:3]) {
				t.Fatalf("unexpected qualifying plan %v", plan)
			}
		}
	}
	if qualified != 1 || validityOnly < 2 {
		t.Fatalf("qualified=%d validity-only=%d", qualified, validityOnly)
	}
}

func seedSchema(gateway bool) []string {
	schema := []string{
		"field:environment:string", "field:tls:bool", "field:service_port:int:1:65535",
		"field:replicas:int:0:10", "field:admin_public:bool", "field:redirect_http:bool",
		"required:environment", "required:tls", "required:service_port", "required:replicas",
		"required:admin_public", "required:redirect_http", "protected:environment", "protected:tls",
		"eq-if:tls=true,service_port=443", "min-if:environment=production,replicas=2",
		"eq-if:environment=production,admin_public=false",
	}
	if gateway {
		schema = append(schema, "field:route_count:int:0:100", "required:route_count")
	}
	return schema
}

func seedConfig(port, replicas, admin string, gateway bool, routes int) []string {
	config := []string{
		"environment=production", "tls=true", "service_port=" + port,
		"replicas=" + replicas, "admin_public=" + admin, "redirect_http=false",
	}
	if gateway {
		config = append(config, "route_count="+strconvItoa(routes))
	}
	return config
}

func subsets(repairs []Repair, maximum int) [][]Repair {
	var out [][]Repair
	var visit func(int, []Repair)
	visit = func(index int, selected []Repair) {
		if len(selected) > 0 {
			out = append(out, append([]Repair(nil), selected...))
		}
		if len(selected) == maximum {
			return
		}
		for next := index; next < len(repairs); next++ {
			visit(next+1, append(selected, repairs[next]))
		}
	}
	visit(0, nil)
	return out
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
