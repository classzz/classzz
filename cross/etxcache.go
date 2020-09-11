package cross

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/database"
	"github.com/classzz/classzz/rlp"
	"log"
)

var (
	BucketKey        = []byte("entangle-tx")
	EntangleStateKey = []byte("entanglestate")
	BurnTxInfoKey    = []byte("burntxinfo")
)

type CacheEntangleInfo struct {
	DB database.DB
}

func (c *CacheEntangleInfo) FetchExChangeUtxoView(info *ExChangeTxInfo) bool {

	var err error
	txExist := false

	AssetType := byte(info.AssetType)
	ExTxHash := []byte(info.ExtTxHash)
	key := append(ExTxHash, AssetType)
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

func (c *CacheEntangleInfo) LoadEntangleState2(height int32, hash chainhash.Hash) *EntangleState2 {

	var err error
	es := NewEntangleState2()

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

func (c *CacheEntangleInfo) LoadBurnTxInfo(address string) *BurnTxInfo {

	bti := &BurnTxInfo{}
	buf, err := hex.DecodeString(address)

	err = c.DB.View(func(tx database.Tx) error {
		BurnTxInfoBucket := tx.Metadata().Bucket(BurnTxInfoKey)
		if BurnTxInfoBucket == nil {
			if BurnTxInfoBucket, err = tx.Metadata().CreateBucketIfNotExists(BurnTxInfoKey); err != nil {
				return err
			}
		}
		value := BurnTxInfoBucket.Get(buf)
		if value != nil {
			err := rlp.DecodeBytes(value, bti)
			if err != nil {
				log.Fatal("Failed to RLP encode BurnTxInfo", "err", err)
				return err
			}
			return nil
		}
		return errors.New("value is nil")
	})
	if err != nil {
		return nil
	}
	return bti
}

func (c *CacheEntangleInfo) LoadBurnTxInfoAll(BeaconID uint64) []*BurnTxInfo {

	btis := make([]*BurnTxInfo, 0)

	err := c.DB.View(func(tx database.Tx) error {
		BurnTxInfoBucket := tx.Metadata().Bucket(BurnTxInfoKey)
		var err error
		if BurnTxInfoBucket == nil {
			if BurnTxInfoBucket, err = tx.Metadata().CreateBucketIfNotExists(BurnTxInfoKey); err != nil {
				return err
			}
		}

		cursor := BurnTxInfoBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			fmt.Printf("key=%s, value=%s\n", cursor.Key(), cursor.Value())
			bti := &BurnTxInfo{}
			err := rlp.DecodeBytes(cursor.Value(), bti)
			if err != nil {
				log.Fatal("Failed to RLP encode BurnTxInfo", "err", err)
				return err
			}
			if bti.BeaconID == BeaconID {
				btis = append(btis, bti)
			}
		}

		return nil
	})

	if err != nil {
		return nil
	}
	return btis
}
