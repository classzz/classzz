package cross

// "fmt"

type CacheEntangleInfo struct {
}

func (c *CacheEntangleInfo) TxExist(info *EntangleTxInfo) bool {
	return false
}
