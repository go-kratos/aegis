package minheap

import (
	"container/heap"
	"sort"
)

type Heap struct {
	Nodes Nodes
	K     uint32
}

func NewHeap(k uint32) *Heap {
	h := Nodes{}
	heap.Init(&h)
	return &Heap{Nodes: h, K: k}
}

func (h *Heap) Add(val Node) string {
	if h.K > uint32(len(h.Nodes)) {
		heap.Push(&h.Nodes, val)
	} else if val.Count > h.Nodes[0].Count {
		expelled := heap.Pop(&h.Nodes)
		heap.Push(&h.Nodes, val)
		return expelled.(Node).Item
	}
	return ""
}

func (h *Heap) Pop() Node {
	expelled := heap.Pop(&h.Nodes)
	return expelled.(Node)
}

func (h *Heap) Fix(idx int, count uint32) {
	h.Nodes[idx].Count = count
	heap.Fix(&h.Nodes, idx)
}

func (h *Heap) Min() uint32 {
	if len(h.Nodes) == 0 {
		return 0
	}
	return h.Nodes[0].Count
}

func (h *Heap) Find(item string) (int, bool) {
	for i := range h.Nodes {
		if h.Nodes[i].Item == item {
			return i, true
		}
	}
	return 0, false
}

func (h *Heap) Sorted() Nodes {
	nodes := append([]Node(nil), h.Nodes...)
	sort.Sort(sort.Reverse(Nodes(nodes)))
	return nodes
}

type Nodes []Node

type Node struct {
	Item  string
	Count uint32
}

func (n Nodes) Len() int {
	return len(n)
}

func (n Nodes) Less(i, j int) bool {
	return (n[i].Count < n[j].Count) || (n[i].Count == n[j].Count && n[i].Item > n[j].Item)
}

func (n Nodes) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n *Nodes) Push(val interface{}) {
	*n = append(*n, val.(Node))
}

func (n *Nodes) Pop() interface{} {
	var val Node
	val, *n = (*n)[len((*n))-1], (*n)[:len((*n))-1]
	return val
}
