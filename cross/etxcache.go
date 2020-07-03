package cross

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/database"
	"github.com/classzz/classzz/rlp"
	"log"
)

var (
	BucketKey        = []byte("entangle-tx")
	EntangleStateKey = []byte("entanglestate")
)

type CacheEntangleInfo struct {
	DB database.DB
}

func (c *CacheEntangleInfo) FetchExChangeUtxoView(info *ExChangeTxInfo) bool {

	var err error
	txExist := false

	ExTxType := byte(info.ExTxType)
	ExTxHash := []byte(info.ExtTxHash)
	key := append(ExTxHash, ExTxType)
	err = c.DB.View(func(tx database.Tx) error {
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

func (c *CacheEntangleInfo) LoadEntangleState(height int32, hash chainhash.Hash) *EntangleState {

	var err error
	es := NewEntangleState()

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, height)
	buf.Write(hash.CloneBytes())

	err = c.DB.Update(func(tx database.Tx) error {
		entangleBucket := tx.Metadata().Bucket(EntangleStateKey)
		if entangleBucket == nil {
			if entangleBucket, err = tx.Metadata().CreateBucketIfNotExists(EntangleStateKey); err != nil {
				return err
			}
		}

		value := entangleBucket.Get(buf.Bytes())
		if value != nil {
			err := rlp.DecodeBytes(value, es)
			if err != nil {
				log.Fatal("Failed to RLP encode EntangleState", "err", err)
				return err
			}
			return nil
		}
		return errors.New("value is nil")
	})
	if err != nil {
		return nil
	}
	return es
}
