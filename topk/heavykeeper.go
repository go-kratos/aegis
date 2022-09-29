package topk

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

// Topk implement by heavykeeper algorithm.
type HeavyKeeper struct {
	k           uint32
	width       uint32
	depth       uint32
	decay       float64
	lookupTable []float64

	r        *rand.Rand
	buckets  [][]bucket
	minHeap  *minheap.Heap
	expelled chan string
}

func NewHeavyKeeper(k, width, depth uint32, decay float64) Topk {
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
		expelled:    make(chan string, 32),
	}
	for i := 0; i < LOOKUP_TABLE; i++ {
		topk.lookupTable[i] = math.Pow(decay, float64(i))
	}
	return topk
}

func (topk *HeavyKeeper) Expelled() <-chan string {
	return topk.expelled
}

func (topk *HeavyKeeper) List() []Item {
	return topk.minHeap.Sorted()
}

func (topk *HeavyKeeper) expell(item string) {
	select {
	case topk.expelled <- item:
	default:
	}
}

// Add add item into heavykeeper and return if item had beend add into minheap.
// if item had been add into minheap and some item was expelled, return the expelled item.
func (topk *HeavyKeeper) Add(item string, incr uint32) bool {
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
		return false
	}
	// update minheap
	itemHeapIdx, itemHeapExist := topk.minHeap.Find(item)
	if itemHeapExist {
		topk.minHeap.Fix(itemHeapIdx, maxCount)
		return true
	}
	expelled := topk.minHeap.Add(minheap.Node{Item: item, Count: maxCount})
	topk.expell(expelled)
	return true
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