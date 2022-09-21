package hotkey

// Native Go implement of topk heavykeeper algorithm, Based on paper
// HeavyKeeper: An Accurate Algorithm for Finding Top-k Elephant Flow (https://www.usenix.org/system/files/conference/atc18/atc18-gong.pdf)

import (
	"math"
	"math/rand"
	"unsafe"

	"github.com/go-kratos/aegis/pkg/minheap"

	"github.com/spaolacci/murmur3"
)

const LOOKUP_TABLE = 256

type TopK struct {
	k           uint32
	width       uint32
	depth       uint32
	decay       float64
	lookupTable []float64

	r       *rand.Rand
	buckets [][]bucket
	minHeap *minheap.Heap
}

func NewTopk(k, width, depth uint32, decay float64) *TopK {
	arrays := make([][]bucket, depth)
	for i := range arrays {
		arrays[i] = make([]bucket, width)
	}

	topk := TopK{
		k:           k,
		width:       width,
		depth:       depth,
		decay:       decay,
		lookupTable: make([]float64, LOOKUP_TABLE),
		buckets:     arrays,
		r:           rand.New(rand.NewSource(0)),
		minHeap:     minheap.NewHeap(k),
	}
	for i := 0; i < LOOKUP_TABLE; i++ {
		topk.lookupTable[i] = math.Pow(decay, float64(i))
	}
	return &topk
}

func (topk *TopK) Query(item string) (exist bool) {
	_, exist = topk.minHeap.Find(item)
	return
}

func (topk *TopK) Count(item string) (uint32, bool) {
	if id, exist := topk.minHeap.Find(item); exist {
		return topk.minHeap.Nodes[id].Count, true
	}
	return 0, false
}

func (topk *TopK) List() []minheap.Node {
	return topk.minHeap.Sorted()
}

// Add add item into heavykeeper and return if item had beend add into minheap.
// if item had been add into minheap and some item was expelled, return the expelled item.
func (topk *TopK) Add(item string, incr uint32) (string, bool) {
	bs := StringToBytes(item)
	itemFingerprint := murmur3.Sum32(bs)
	var maxCount uint32

	// compute d hashes
	for i, row := range topk.buckets {

		bucketNumber := murmur3.Sum32WithSeed(bs, uint32(i)) % uint32(topk.width)

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
	minHeap := topk.minHeap.Min()
	if len(topk.minHeap.Nodes) == int(topk.k) && maxCount < minHeap {
		return "", false
	}
	// update minheap
	itemHeapIdx, itemHeapExist := topk.minHeap.Find(item)
	if itemHeapExist {
		topk.minHeap.Fix(itemHeapIdx, maxCount)
		return "", true
	}
	expelled := topk.minHeap.Add(minheap.Node{Item: item, Count: maxCount})
	return expelled, true
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

func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
