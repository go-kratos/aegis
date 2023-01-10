package subset

import (
	"stathat.com/c/consistent"
)

func Subset(selectKey string, inss []string, num int) []string {
	if len(inss) <= num {
		return inss
	}
	c := consistent.New()
	c.NumberOfReplicas = 160
	c.UseFnv = true
	for _, ins := range inss {
		c.Add(ins)
	}
	backends, err := c.GetN(selectKey, num)
	if err != nil {
		return inss
	}
	return backends
}

func subset(c *consistent.Consistent, selectKey string, inss []string, num int) []string {
	backends, err := c.GetN(selectKey, num)
	if err != nil {
		return inss
	}
	return backends
}
