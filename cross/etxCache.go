package cross

import (
	"github.com/classzz/classzz/database"
)

var (
	BucketKey = []byte("entangle-tx")
)

type CacheEntangleInfo struct {
	DB database.DB
}

func (c *CacheEntangleInfo) FetchEntangleUtxoView(info *EntangleTxInfo) bool {

	var err error
	txExist := false

	ExTxType := byte(info.ExTxType)
	key := append(info.ExtTxHash, ExTxType)
	err = c.DB.Update(func(tx database.Tx) error {
		entangleBucket := tx.Metadata().Bucket(BucketKey)
		if entangleBucket == nil {
			if entangleBucket, err = tx.Metadata().CreateBucketIfNotExists(BucketKey); err != nil {
				return err
			}
		}

		value := entangleBucket.Get(key)
		if value != nil {
			txExist = true
		}
		return nil
	})

	return txExist
}
