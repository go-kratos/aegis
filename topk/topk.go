package topk

// Item is topk item.
type Item struct {
	Key   string
	Count uint32
}

// Topk algorithm interface.
type Topk interface {
	// Add item and return if item is in the topk.
	Add(item string, incr uint32) (string, bool)
	// List all topk items.
	List() []Item
	// Expelled watch at the expelled items.
	Expelled() <-chan Item
	Fading()
}
