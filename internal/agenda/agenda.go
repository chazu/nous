// Package agenda implements the priority queue of tasks for the nous engine.
// Tasks are merged when duplicates are proposed, with priority boosting.
package agenda

import "container/heap"

// Task represents a unit of work: explore a particular slot of a particular unit.
type Task struct {
	Priority  int
	UnitName  string
	SlotName  string
	Reasons   []string
	Extra     map[string]any
	index     int // heap index
	sequence  uint64
	lookupKey string
}

func taskKey(unitName, slotName string) string {
	return unitName + "|" + slotName
}

// Agenda is a priority queue of tasks with duplicate merging.
type Agenda struct {
	tasks        taskHeap
	lookup       map[string]*Task
	nextSequence uint64
}

// New creates an empty agenda.
func New() *Agenda {
	return &Agenda{
		lookup: make(map[string]*Task),
	}
}

// Push adds a task. If a task with the same unit+slot exists,
// merge: boost priority and append reasons.
// Tasks with different Extra["SlotToChange"] values are NOT merged.
func (a *Agenda) Push(t *Task) {
	key := taskKey(t.UnitName, t.SlotName)
	if existing, ok := a.lookup[key]; ok {
		// Don't merge if both have Extra with different SlotToChange
		if extraSlotsDiffer(existing.Extra, t.Extra) {
			// Use a disambiguated key so both stay in the queue
			key = key + "|" + t.Extra["SlotToChange"].(string)
			if _, ok2 := a.lookup[key]; ok2 {
				// Already have this exact variant — merge into it
				a.mergeInto(a.lookup[key], t)
				return
			}
			a.enqueue(key, t)
			a.lookup[key] = t
			return
		}
		// Merge: take max priority + boost, append reasons
		a.mergeInto(existing, t)
		return
	}
	a.enqueue(key, t)
	a.lookup[key] = t
}

func (a *Agenda) enqueue(key string, t *Task) {
	t.sequence = a.nextSequence
	a.nextSequence++
	t.lookupKey = key
	heap.Push(&a.tasks, t)
}

func (a *Agenda) mergeInto(existing, t *Task) {
	// A bare task is a request to explore a slot; a populated task is the
	// concrete proposal produced by that exploration. Never let merging erase
	// the proposal merely because the bare request reached the queue first.
	if len(existing.Extra) == 0 && len(t.Extra) > 0 {
		existing.Extra = make(map[string]any, len(t.Extra))
		for key, value := range t.Extra {
			existing.Extra[key] = value
		}
	}
	newPri := existing.Priority
	if t.Priority > newPri {
		newPri = t.Priority
	}
	newPri += 50 // merge boost
	if newPri > 1000 {
		newPri = 1000
	}
	existing.Priority = newPri
	existing.Reasons = append(existing.Reasons, t.Reasons...)
	heap.Fix(&a.tasks, existing.index)
}

// extraSlotsDiffer returns true if both extras have SlotToChange and the values differ.
func extraSlotsDiffer(a, b map[string]any) bool {
	if a == nil || b == nil {
		return false
	}
	aSlot, aOk := a["SlotToChange"]
	bSlot, bOk := b["SlotToChange"]
	if !aOk || !bOk {
		return false
	}
	return aSlot != bSlot
}

// Pop removes and returns the highest-priority task, or nil if empty.
func (a *Agenda) Pop() *Task {
	if len(a.tasks) == 0 {
		return nil
	}
	t := heap.Pop(&a.tasks).(*Task)
	delete(a.lookup, t.lookupKey)
	return t
}

// PurgeUnit removes every pending task whose UnitName matches. Returns the
// number of tasks dropped. Call this when a unit is killed — otherwise
// orphan tasks keep firing heuristics on a non-existent unit, and any
// "slot=nil" guard will match indefinitely because get-slot on a missing
// unit returns nil for every slot.
func (a *Agenda) PurgeUnit(unitName string) int {
	removed := 0
	kept := a.tasks[:0]
	for _, t := range a.tasks {
		if t.UnitName == unitName {
			removed++
			continue
		}
		kept = append(kept, t)
	}
	a.tasks = kept
	for k := range a.lookup {
		if a.lookup[k].UnitName == unitName {
			delete(a.lookup, k)
		}
	}
	heap.Init(&a.tasks)
	// Reassign indices after heap.Init (heap.Init already sets them).
	return removed
}

// Len returns the number of tasks.
func (a *Agenda) Len() int {
	return len(a.tasks)
}

// Peek returns the highest-priority task without removing it, or nil.
func (a *Agenda) Peek() *Task {
	if len(a.tasks) == 0 {
		return nil
	}
	return a.tasks[0]
}

// taskHeap implements heap.Interface for max-priority ordering.
type taskHeap []*Task

func (h taskHeap) Len() int { return len(h) }
func (h taskHeap) Less(i, j int) bool {
	if h[i].Priority == h[j].Priority {
		return h[i].sequence < h[j].sequence
	}
	return h[i].Priority > h[j].Priority
}
func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *taskHeap) Push(x any) {
	t := x.(*Task)
	t.index = len(*h)
	*h = append(*h, t)
}

func (h *taskHeap) Pop() any {
	old := *h
	n := len(old)
	t := old[n-1]
	old[n-1] = nil
	t.index = -1
	*h = old[:n-1]
	return t
}
