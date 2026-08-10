package actionrelationutility

import (
	"reflect"

	"github.com/chazu/nous/internal/unit"
)

// cloneUtilityStore creates the utility namespace from frozen acquisition
// authority without adding a general Store cloning surface outside this lane.
func cloneUtilityStore(source *unit.Store) *unit.Store {
	result := unit.NewStore()
	for _, name := range source.All() {
		sourceUnit := source.Get(name)
		if sourceUnit == nil {
			continue
		}
		target := unit.New(name)
		for slot, value := range sourceUnit.Slots {
			target.Slots[slot] = cloneUtilitySlot(reflect.ValueOf(value)).Interface()
		}
		result.Put(target)
	}
	return result
}

func cloneUtilitySlot(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return reflect.Zero(reflect.TypeOf((*any)(nil)).Elem())
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.New(value.Type()).Elem()
		result.Set(cloneUtilitySlot(value.Elem()))
		return result
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			result.SetMapIndex(cloneUtilitySlot(iterator.Key()), cloneUtilitySlot(iterator.Value()))
		}
		return result
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := 0; index < value.Len(); index++ {
			result.Index(index).Set(cloneUtilitySlot(value.Index(index)))
		}
		return result
	case reflect.Array:
		result := reflect.New(value.Type()).Elem()
		for index := 0; index < value.Len(); index++ {
			result.Index(index).Set(cloneUtilitySlot(value.Index(index)))
		}
		return result
	default:
		return value
	}
}
