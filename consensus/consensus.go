// Package consensus implements the czz consensus engine.
package consensus

import (
	"errors"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"math/big"
)

type CzzConsensusParam struct {
	HeadHash chainhash.Hash
	Target   *big.Int
}

type MiningParam struct {
	Info    *CzzConsensusParam
	MinerID int
	Begin   uint64
	Loops   uint64
	Done    uint64
	Abort   chan struct{}
}

// CZZhashFull aggregates data from the full dataset (using the full in-memory
// dataset) in order to produce our final value for a particular header hash and
// nonce.
func CZZhashFull(hash []byte, nonce uint64) []byte {
	return HashCZZ(hash, nonce)
}

func MineBlock(conf *MiningParam) (uint64, bool) {
	var (
		nonce = conf.Begin
		found = false
	)

	for i := uint64(0); i < conf.Loops; i++ {
		conf.Done = nonce + 1
		select {
		case <-conf.Abort:
			return nonce, found
		default:
			result := CZZhashFull(conf.Info.HeadHash[:], nonce)

			if new(big.Int).SetBytes(result).Cmp(conf.Info.Target) <= 0 {
				found = true
				return nonce, found
			}
		}

		nonce++
	}
	return nonce, found
}
func VerifyBlockSeal(Info *CzzConsensusParam, nonce uint64) error {
	result := CZZhashFull(Info.HeadHash[:], nonce)
	if new(big.Int).SetBytes(result).Cmp(Info.Target) <= 0 {
		return nil
	}
	return errors.New("invalid mix digest")
}
