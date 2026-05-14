package main

import (
	"container/heap"

	"github.com/Davi-UEFS/Warzone/shared"
)

type ReqHeap []shared.Requisition

func (h ReqHeap) Len() int { return len(h) }
func (h ReqHeap) Less(i, j int) bool {
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
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

func (h *ReqHeap) Peek() shared.Requisition {
	return (*h)[0]
}

func (h *ReqHeap) ToSlice() []shared.Requisition {
	out := make([]shared.Requisition, len(*h))
	copy(out, *h)
	return out
}

func (h *ReqHeap) FromSlice(s []shared.Requisition) {
	*h = make([]shared.Requisition, len(s))
	copy(*h, s)
	heap.Init(h)
}

func (h *ReqHeap) RemoveAt(i int) shared.Requisition {
	x := heap.Remove(h, i).(shared.Requisition)
	return x
}

func (h *ReqHeap) ApplyAging(currentTime int64, thresholdSeconds int64, boostAmount int) {
	for i := range *h {
		age := currentTime - (*h)[i].CreatedAt
		if age > thresholdSeconds {
			(*h)[i].Priority += boostAmount
		}
	}
	heap.Init(h)
}
