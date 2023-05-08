package subset

import (
	"github.com/go-kratos/aegis/internal/consistent"
)

func Subset[M consistent.Member](selectKey string, inss []M, num int) []M {
	if len(inss) <= num {
		return inss
	}

	c := consistent.New[M]()
	c.NumberOfReplicas = 160
	c.UseFnv = true
	c.Set(inss)

	return subset(c, selectKey, inss, num)
}

func subset[M consistent.Member](c *consistent.Consistent[M], selectKey string, inss []M, num int) []M {
	backends, err := c.GetN(selectKey, num)
	if err != nil {
		return inss
	}
	return backends
}
