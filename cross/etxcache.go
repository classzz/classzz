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
	BucketKey         = []byte("entangle-tx")
	EntangleStateKey  = []byte("entanglestate")
	CommitteeStateKey = []byte("committeestate")
)

type CacheCommitteeState struct {
	DB database.DB
}

func (c *CacheCommitteeState) FetchExtUtxoView(info *ConvertTxInfo) bool {

	txExist := false
	AssetType := info.AssetType
	ExTxHash := []byte(info.ExtTxHash)
	key := append(ExTxHash, AssetType)
	c.DB.View(func(tx database.Tx) error {
		committeeBucket := tx.Metadata().Bucket(CommitteeStateKey)
		if committeeBucket == nil {
			return nil
		}
		value := committeeBucket.Get(key)
		if value != nil {
			txExist = true
		}
		return nil
	})
	return txExist
}

func (c *CacheCommitteeState) LoadCommitteeState(height int32, hash chainhash.Hash) *CommitteeState {

	cs := NewCommitteeState()

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, height)
	buf.Write(hash.CloneBytes())

	err := c.DB.Update(func(tx database.Tx) error {
		committeeBucket := tx.Metadata().Bucket(CommitteeStateKey)
		if committeeBucket == nil {
			return nil
		}
		value := committeeBucket.Get(buf.Bytes())
		if value != nil {
			err := rlp.DecodeBytes(value, cs)
			if err != nil {
				return err
			}
			return nil
		}
		return errors.New("value is nil")
	})
	if err != nil {
		log.Fatal("Failed to LoadCommitteeState ", "err", err)
		return nil
	}
	return cs
}

func (c *CacheCommitteeState) LoadEntangleState(height int32, hash chainhash.Hash) *EntangleState {

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
				log.Fatal("Failed to RLP encode LoadEntangleState EntangleState2 ", "err", err)
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
