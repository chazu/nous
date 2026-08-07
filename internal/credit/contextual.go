// Package credit represents contextual credit as ordinary Nous units.
package credit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/chazu/nous/internal/unit"
)

const (
	Category            = "ContextualCredit"
	MaxCreditors        = 32
	MaxContextBytes     = 256
	MaxRoleBytes        = 256
	MaxSubjectBytes     = 512
	decisionRole        = "decision"
	recordNamePrefix    = "ContextualCredit-"
	collisionNameFormat = "%s-collision-%d"
)

type Tuple struct {
	Context string
	Subject string
	Role    string
}

type Provenance struct {
	SourceUnit    string
	RewardTaskNum int
}

// ValidDeclaration validates the parallel contextual-attribution slots on a
// source unit. Legacy scalar reward does not depend on this declaration.
func ValidDeclaration(context, decision string, creditors, roles []string) bool {
	if !validText(context, MaxContextBytes) || !validText(decision, MaxSubjectBytes) ||
		len(creditors) > MaxCreditors || len(creditors) != len(roles) {
		return false
	}
	for index := range creditors {
		if !validText(creditors[index], MaxSubjectBytes) || !validText(roles[index], MaxRoleBytes) {
			return false
		}
	}
	return true
}

func validText(value string, max int) bool { return value != "" && len(value) <= max }

func DecisionTuple(context, decision string) Tuple {
	return Tuple{Context: context, Subject: decision, Role: decisionRole}
}

// Lookup finds a record by its semantic tuple, independent of its allocated
// unit name or any collision suffix.
func Lookup(store *unit.Store, tuple Tuple) *unit.Unit {
	if store == nil {
		return nil
	}
	for _, name := range store.Examples(Category) {
		record := store.Get(name)
		if matches(record, tuple) {
			return record
		}
	}
	return nil
}

func RewardTotal(store *unit.Store, tuple Tuple) int {
	if record := Lookup(store, tuple); record != nil {
		return record.GetInt("rewardTotal")
	}
	return 0
}

// Upsert adds positive reward to the compact record for tuple. It is total for
// an in-memory Store and reuses a matching tuple before allocating a name.
func Upsert(store *unit.Store, tuple Tuple, amount int, provenance Provenance) *unit.Unit {
	if store == nil || amount <= 0 || !validTuple(tuple) {
		return Lookup(store, tuple)
	}
	if record := Lookup(store, tuple); record != nil {
		update(record, amount, provenance)
		return record
	}

	base := recordName(tuple)
	name := base
	if store.Has(name) {
		for suffix := 1; ; suffix++ {
			name = fmt.Sprintf(collisionNameFormat, base, suffix)
			if !store.Has(name) {
				break
			}
		}
	}
	record := unit.New(name)
	record.SetWorth(0)
	record.Set("isA", []string{Category, "Anything"})
	record.Set("creditContext", tuple.Context)
	record.Set("creditSubject", tuple.Subject)
	record.Set("creditRole", tuple.Role)
	update(record, amount, provenance)
	store.Put(record)
	return record
}

func update(record *unit.Unit, amount int, provenance Provenance) {
	record.Set("rewardTotal", record.GetInt("rewardTotal")+amount)
	record.Set("evidenceCount", record.GetInt("evidenceCount")+1)
	record.Set("lastSourceUnit", provenance.SourceUnit)
	record.Set("lastRewardTaskNum", provenance.RewardTaskNum)
}

func matches(record *unit.Unit, tuple Tuple) bool {
	return record != nil && record.GetString("creditContext") == tuple.Context &&
		record.GetString("creditSubject") == tuple.Subject &&
		record.GetString("creditRole") == tuple.Role
}

func validTuple(tuple Tuple) bool {
	return validText(tuple.Context, MaxContextBytes) &&
		validText(tuple.Subject, MaxSubjectBytes) && validText(tuple.Role, MaxRoleBytes)
}

func recordName(tuple Tuple) string {
	encoded, _ := json.Marshal([]string{tuple.Context, tuple.Subject, tuple.Role})
	digest := sha256.Sum256(encoded)
	return recordNamePrefix + hex.EncodeToString(digest[:])
}
