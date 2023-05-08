package subset

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"testing"

	"github.com/go-kratos/aegis/internal/consistent"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type member string

func (m member) String() string {
	return string(m)
}
func TestRedundant(t *testing.T) {
	assert.Equal(t, []member{"2", "3"}, Subset("1", []member{"2", "2", "2", "3"}, 3))
}

func TestDistribution(t *testing.T) {
	var backends []member
	content, err := ioutil.ReadFile("./backends.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(content, &backends)
	if err != nil {
		panic(err)
	}
	res := make(map[member]int64, 0)

	c := consistent.New[member]()
	c.NumberOfReplicas = 160
	c.UseFnv = true
	var max int64
	c.Set(backends)

	for i := 0; i < 8000; i++ {
		id := uuid.New()
		backs := subset(c, id.String()[:12], backends, 25)
		for _, back := range backs {
			res[back] += 1
		}
	}
	for _, c := range res {
		if c > max {
			max = c
		}
	}
	assert.LessOrEqual(t, max, int64(600))
}

func TestRelocation(t *testing.T) {
	var backends []member
	conns := make(map[string]map[member]struct{})

	content, err := ioutil.ReadFile("./backends.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(content, &backends)
	if err != nil {
		panic(err)
	}

	c := consistent.New[member]()
	c.NumberOfReplicas = 160
	c.UseFnv = true
	c.Set(backends)

	var clients []string
	for i := 0; i < 8000; i++ {
		id := uuid.New().String()[:12]
		clients = append(clients, id)
	}
	for _, client := range clients {
		backs := subset(c, client, backends, 25)
		conn := map[member]struct{}{}
		for _, back := range backs {
			conn[back] = struct{}{}
		}
		conns[client] = conn
	}
	var change int64
	re := backends[rand.Intn(len(backends))]
	c.Remove(re)
	for _, client := range clients {
		backs := subset(c, client, backends, 25)
		conn := map[member]struct{}{}
		for _, back := range backs {
			conn[back] = struct{}{}
		}
		old := conns[client]

		var hit int
		for k := range old {
			if _, ok := conn[k]; ok {
				hit++
			}
		}
		change += int64(25 - hit)
	}
	assert.Less(t, float64(change)/float64(200000), 0.005)
}
