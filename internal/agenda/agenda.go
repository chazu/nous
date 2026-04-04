// Package agenda implements the priority queue of tasks for the nous engine.
// Tasks are merged when duplicates are proposed, with priority boosting.
package agenda

import "container/heap"

// Task represents a unit of work: explore a particular slot of a particular unit.
type Task struct {
	Priority int
	UnitName string
	SlotName string
	Reasons  []string
	index    int // heap index
}

func taskKey(unitName, slotName string) string {
	return unitName + "|" + slotName
}

// Agenda is a priority queue of tasks with duplicate merging.
type Agenda struct {
	tasks  taskHeap
	lookup map[string]*Task
}

// New creates an empty agenda.
func New() *Agenda {
	return &Agenda{
		lookup: make(map[string]*Task),
	}
}

// Push adds a task. If a task with the same unit+slot exists,
// merge: boost priority and append reasons.
func (a *Agenda) Push(t *Task) {
	key := taskKey(t.UnitName, t.SlotName)
	if existing, ok := a.lookup[key]; ok {
		// Merge: take max priority + boost, append reasons
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
		return
	}
	heap.Push(&a.tasks, t)
	a.lookup[key] = t
}

// Pop removes and returns the highest-priority task, or nil if empty.
func (a *Agenda) Pop() *Task {
	if len(a.tasks) == 0 {
		return nil
	}
	t := heap.Pop(&a.tasks).(*Task)
	delete(a.lookup, taskKey(t.UnitName, t.SlotName))
	return t
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

func (h taskHeap) Len() int           { return len(h) }
func (h taskHeap) Less(i, j int) bool { return h[i].Priority > h[j].Priority } // max-heap
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
