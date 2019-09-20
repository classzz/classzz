package cross

import (
	"github.com/classzz/classzz/database"
)

var (
	BucketKey = "entangle-tx"
)

type CacheEntangleInfo struct {
	DB database.DB
}

func (c *CacheEntangleInfo) FetchEntangleUtxoView(info *EntangleTxInfo) bool {

	var err error
	var txExist bool

	ExTxType := byte(info.ExTxType)
	key := append(info.ExtTxHash, ExTxType)
	err = c.DB.Update(func(tx database.Tx) error {
		entangleBucket := tx.Metadata().Bucket([]byte(BucketKey))
		if entangleBucket == nil {
			if entangleBucket, err = tx.Metadata().CreateBucketIfNotExists([]byte(BucketKey)); err != nil {
				return err
			}
		}

		value := entangleBucket.Get(key)
		if value != nil {
			txExist = false
		}
		return nil
	})

	if err != nil {
		return true
	}

	return txExist
}
