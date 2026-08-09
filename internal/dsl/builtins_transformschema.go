package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

func init() {
	registerVocabularyWords("transformschema", map[string]builtinFn{
		"ts-forest-valid?": bTSForestValid,
		"ts-schema-apply":  bTSSchemaApply,
		"ts-program-apply": bTSProgramApply,
		"ts-refine":        bTSRefine,
		"ts-digest":        bTSDigest,
	})
}

func bTSForestValid(vm *VM) error {
	value := vm.pop()
	if value.Kind() != VString {
		vm.push(BoolVal(false))
		return nil
	}
	_, err := transformschema.ParseForest([]byte(value.AsString()))
	vm.push(BoolVal(err == nil))
	return nil
}

func bTSSchemaApply(vm *VM) error {
	schemaValue, forestValue := vm.pop(), vm.pop()
	if schemaValue.Kind() != VString || forestValue.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	f, err := transformschema.ParseForest([]byte(forestValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	s, err := transformschema.ParseSchema([]byte(schemaValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	r, _ := s.Apply(f)
	output := ""
	if r.Output != nil {
		b, _ := r.Output.CanonicalJSON()
		output = string(b)
	}
	vm.push(ListVal([]Value{StringVal(r.Terminal), StringVal(output)}))
	return nil
}

func bTSProgramApply(vm *VM) error {
	programValue, forestValue := vm.pop(), vm.pop()
	if programValue.Kind() != VString || forestValue.Kind() != VString {
		vm.push(Nil())
		return nil
	}
	f, err := transformschema.ParseForest([]byte(forestValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	p, err := transformschema.ParseProgram([]byte(programValue.AsString()))
	if err != nil {
		vm.push(Nil())
		return nil
	}
	out, err := p.Apply(f)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	b, _ := out.CanonicalJSON()
	vm.push(StringVal(string(b)))
	return nil
}

func bTSRefine(vm *VM) error {
	value, partialValue := vm.pop(), vm.pop()
	if value.Kind() != VString || partialValue.Kind() != VList {
		vm.push(Nil())
		return nil
	}
	items := partialValue.AsList()
	if len(items) != 6 {
		vm.push(Nil())
		return nil
	}
	values := make([]string, 5)
	if items[0].Kind() != VInt {
		vm.push(Nil())
		return nil
	}
	for i := range values {
		if items[i+1].Kind() != VString {
			vm.push(Nil())
			return nil
		}
		values[i] = items[i+1].AsString()
	}
	p := transformschema.Partial{Stage: items[0].AsInt(), Targets: values[0], Anchor: values[1], ReferenceScope: values[2], OldGuard: values[3], Locality: values[4]}
	next, err := p.Refine(value.AsString())
	if err != nil {
		vm.push(Nil())
		return nil
	}
	vm.push(ListVal([]Value{IntVal(next.Stage), StringVal(next.Targets), StringVal(next.Anchor), StringVal(next.ReferenceScope), StringVal(next.OldGuard), StringVal(next.Locality)}))
	return nil
}

func bTSDigest(vm *VM) error {
	v := vm.pop()
	var material any
	switch v.Kind() {
	case VString:
		material = v.AsString()
	case VInt:
		material = v.AsInt()
	case VBool:
		material = v.AsBool()
	case VList:
		material = tsSerializable(v)
	default:
		vm.push(Nil())
		return nil
	}
	b, err := json.Marshal(material)
	if err != nil {
		vm.push(Nil())
		return nil
	}
	d := sha256.Sum256(b)
	vm.push(StringVal(hex.EncodeToString(d[:])))
	return nil
}

func tsSerializable(v Value) any {
	if v.Kind() != VList {
		return nil
	}
	values := v.AsList()
	out := make([]any, len(values))
	for i, value := range values {
		switch value.Kind() {
		case VString:
			out[i] = value.AsString()
		case VInt:
			out[i] = value.AsInt()
		case VBool:
			out[i] = value.AsBool()
		case VList:
			out[i] = tsSerializable(value)
		default:
			out[i] = nil
		}
	}
	return out
}
