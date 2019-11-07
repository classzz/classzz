// Package consensus implements the czz consensus engine.
package consensus

import (
	"encoding/binary"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"golang.org/x/crypto/sha3"
	"hash"
	"math/big"
	"math/rand"
	"time"
)

func init() {
	czzTbl = &CZZTBL{
		data:  make([]byte, TBLSize),
		bflag: 0,
	}
}

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

// hasher is a repetitive hasher allowing the same hash data structures to be
// reused between hash runs instead of requiring new ones to be created.
type hasher func(dest []byte, data []byte)

// makeHasher creates a repetitive hasher, allowing the same hash data structures
// to be reused between hash runs instead of requiring new ones to be created.
// The returned function is not thread safe!
func makeHasher(h hash.Hash) hasher {
	return func(dest []byte, data []byte) {
		h.Write(data)
		h.Sum(dest[:0])
		h.Reset()
	}
}

// CZZhashFull aggregates data from the full dataset (using the full in-memory
// dataset) in order to produce our final value for a particular header hash and
// nonce.
func CZZhashFull(hash []byte, nonce uint64) []byte {
	return HashCZZ(hash, nonce)
}

func makeDatasetHash(dataset []uint64) []byte {
	var datas []byte
	tmp := make([]byte, 8)
	for _, v := range dataset {
		binary.LittleEndian.PutUint64(tmp, v)
		datas = append(datas, tmp...)
	}
	sha256 := makeHasher(sha3.New256())
	output := make([]byte, 32)
	sha256(output, datas[:])
	return output
}

func MineBlock(conf *MiningParam) (uint64, bool) {
	var (
		nonce = conf.Begin
		found = false
	)

	rand.Seed(time.Now().Unix())
	index := rand.Int63n(5)
	for i := index; i > 0; i-- {
		time.Sleep(1 * time.Second)
	}
	found = true
	return nonce, found

}
func VerifyBlockSeal(Info *CzzConsensusParam, nonce uint64) error {
	return nil
}
