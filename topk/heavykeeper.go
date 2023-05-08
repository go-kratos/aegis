package topk

// Native Go implement of topk heavykeeper algorithm, Based on paper
// HeavyKeeper: An Accurate Algorithm for Finding Top-k Elephant Flow (https://www.usenix.org/system/files/conference/atc18/atc18-gong.pdf)

import (
	"math"
	"math/rand"

	"github.com/go-kratos/aegis/internal/minheap"

	"github.com/twmb/murmur3"
)

const LOOKUP_TABLE = 256

// Topk implement by heavykeeper algorithm.
type HeavyKeeper struct {
	k           uint32
	width       uint32
	depth       uint32
	decay       float64
	lookupTable []float64
	minCount    uint32

	r        *rand.Rand
	buckets  [][]bucket
	minHeap  *minheap.Heap
	expelled chan Item
	total    uint64
}

func NewHeavyKeeper(k, width, depth uint32, decay float64, min uint32) Topk {
	arrays := make([][]bucket, depth)
	for i := range arrays {
		arrays[i] = make([]bucket, width)
	}

	topk := &HeavyKeeper{
		k:           k,
		width:       width,
		depth:       depth,
		decay:       decay,
		lookupTable: make([]float64, LOOKUP_TABLE),
		buckets:     arrays,
		r:           rand.New(rand.NewSource(0)),
		minHeap:     minheap.NewHeap(k),
		expelled:    make(chan Item, 32),
		minCount:    min,
	}
	for i := 0; i < LOOKUP_TABLE; i++ {
		topk.lookupTable[i] = math.Pow(decay, float64(i))
	}
	return topk
}

func (topk *HeavyKeeper) Expelled() <-chan Item {
	return topk.expelled
}

func (topk *HeavyKeeper) List() []Item {
	items := topk.minHeap.Sorted()
	res := make([]Item, 0, len(items))
	for _, item := range items {
		res = append(res, Item{Key: item.Key, Count: item.Count})
	}
	return res
}

// Add add item into heavykeeper and return if item had beend add into minheap.
// if item had been add into minheap and some item was expelled, return the expelled item.
func (topk *HeavyKeeper) Add(key string, incr uint32) (string, bool) {
	keyBytes := []byte(key)
	itemFingerprint := murmur3.Sum32(keyBytes)
	var maxCount uint32

	// compute d hashes
	for i, row := range topk.buckets {

		bucketNumber := murmur3.SeedSum32(uint32(i), keyBytes) % uint32(topk.width)
		fingerprint := row[bucketNumber].fingerprint
		count := row[bucketNumber].count

		if count == 0 {
			row[bucketNumber].fingerprint = itemFingerprint
			row[bucketNumber].count = incr
			maxCount = max(maxCount, incr)

		} else if fingerprint == itemFingerprint {
			row[bucketNumber].count += incr
			maxCount = max(maxCount, row[bucketNumber].count)

		} else {
			for localIncr := incr; localIncr > 0; localIncr-- {
				var decay float64
				curCount := row[bucketNumber].count
				if row[bucketNumber].count < LOOKUP_TABLE {
					decay = topk.lookupTable[curCount]
				} else {
					// decr pow caculate cost
					decay = topk.lookupTable[LOOKUP_TABLE-1]
				}
				if topk.r.Float64() < decay {
					row[bucketNumber].count--
					if row[bucketNumber].count == 0 {
						row[bucketNumber].fingerprint = itemFingerprint
						row[bucketNumber].count = localIncr
						maxCount = max(maxCount, localIncr)
						break
					}
				}
			}
		}
	}
	topk.total += uint64(incr)
	if maxCount < topk.minCount {
		return "", false
	}
	minHeap := topk.minHeap.Min()
	if len(topk.minHeap.Nodes) == int(topk.k) && maxCount < minHeap {
		return "", false
	}
	// update minheap
	itemHeapIdx, itemHeapExist := topk.minHeap.Find(key)
	if itemHeapExist {
		topk.minHeap.Fix(itemHeapIdx, maxCount)
		return "", true
	}
	var exp string
	expelled := topk.minHeap.Add(&minheap.Node{Key: key, Count: maxCount})
	if expelled != nil {
		topk.expell(Item{Key: expelled.Key, Count: expelled.Count})
		exp = expelled.Key
	}

	return exp, true
}

func (topk *HeavyKeeper) expell(item Item) {
	select {
	case topk.expelled <- item:
	default:
	}
}

type bucket struct {
	fingerprint uint32
	count       uint32
}

func (b *bucket) Get() (uint32, uint32) {
	return b.fingerprint, b.count
}

func (b *bucket) Set(fingerprint, count uint32) {
	b.fingerprint = fingerprint
	b.count = count
}

func (b *bucket) Inc(val uint32) uint32 {
	b.count += val
	return b.count
}

func max(x, y uint32) uint32 {
	if x > y {
		return x
	}
	return y
}

func (topk *HeavyKeeper) Fading() {
	for _, row := range topk.buckets {
		for i := range row {
			row[i].count = row[i].count >> 1
		}
	}
	for i := 0; i < len(topk.minHeap.Nodes); i++ {
		topk.minHeap.Nodes[i].Count = topk.minHeap.Nodes[i].Count >> 1
	}
	topk.total = topk.total >> 1
}

func (topk *HeavyKeeper) Total() uint64 {
	return topk.total
}
