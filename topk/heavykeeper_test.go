package topk

import (
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTopkList(t *testing.T) {
	// zipfan distribution
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().Unix())), 3, 2, 1000)
	topk := NewHeavyKeeper(10, 10000, 5, 0.925, 0)
	dataMap := make(map[string]int)
	for i := 0; i < 10000; i++ {
		key := strconv.FormatUint(zipf.Uint64(), 10)
		dataMap[key] = dataMap[key] + 1
		topk.Add(key, 1)
	}
	var rate float64
	for _, node := range topk.List() {
		rate += math.Abs(float64(node.Count)-float64(dataMap[node.Key])) / float64(dataMap[node.Key])
		t.Logf("item %s, count %d, expect %d", node.Key, node.Count, dataMap[node.Key])
	}
	t.Logf("err rate avg:%f", rate)
	for i, node := range topk.List() {
		assert.Equal(t, strconv.FormatInt(int64(i), 10), node.Key)
		t.Logf("%s: %d", node.Key, node.Count)
	}
}

func BenchmarkAdd(b *testing.B) {
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().Unix())), 2, 2, 1000)
	var data []string = make([]string, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = strconv.FormatUint(zipf.Uint64(), 10)
	}
	topk := NewHeavyKeeper(10, 1000, 5, 0.9, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		topk.Add(data[i%1000], 1)
	}
}
