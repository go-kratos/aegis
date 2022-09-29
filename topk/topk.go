package topk

import "github.com/go-kratos/aegis/pkg/minheap"

type Item = minheap.Node

// Topk algorithm interface
type Topk interface {
	// Add item and return if item is in the topk.
	Add(item string, incr uint32) bool
	// List all topk items.
	List() []Item
	// Expelled watch at the expelled items.
	Expelled() <-chan string
}
