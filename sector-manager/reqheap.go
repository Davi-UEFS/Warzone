package main

import (
	"container/heap"

	"github.com/Davi-UEFS/Warzone/shared"
)

// ReqHeap implements a priority queue for shared.Requisition.
// Higher Priority value means higher priority (popped first).
type ReqHeap []shared.Requisition

func (h ReqHeap) Len() int { return len(h) }
func (h ReqHeap) Less(i, j int) bool {
	// Primary: higher Priority value pops first (greater-than)
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
	// Tiebreaker: lower LamportTime (older request) pops first (less-than)
	return h[i].LamportTime < h[j].LamportTime
}
func (h ReqHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *ReqHeap) Push(x interface{}) {
	*h = append(*h, x.(shared.Requisition))
}

func (h *ReqHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// Peek returns the top element without removing it. Caller should check Len>0.
func (h *ReqHeap) Peek() shared.Requisition {
	return (*h)[0]
}

// ToSlice returns a copy of the heap as a slice. Order is not guaranteed sorted.
func (h *ReqHeap) ToSlice() []shared.Requisition {
	out := make([]shared.Requisition, len(*h))
	copy(out, *h)
	return out
}

// FromSlice replaces heap content with given slice and inits the heap.
func (h *ReqHeap) FromSlice(s []shared.Requisition) {
	*h = make([]shared.Requisition, len(s))
	copy(*h, s)
	heap.Init(h)
}

// RemoveAt removes the element at index i using heap.Remove.
func (h *ReqHeap) RemoveAt(i int) shared.Requisition {
	x := heap.Remove(h, i).(shared.Requisition)
	return x
}

// ApplyAging increments priority of requests that have waited > threshold seconds.
// threshold: max seconds without dispatch before priority boost
// boostAmount: how much to increment priority per aging cycle
func (h *ReqHeap) ApplyAging(currentTime int64, thresholdSeconds int64, boostAmount int) {
	for i := range *h {
		age := currentTime - (*h)[i].CreatedAt
		if age > thresholdSeconds {
			(*h)[i].Priority += boostAmount
		}
	}
	// Rebuild heap after modifying priorities
	heap.Init(h)
}
