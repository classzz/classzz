// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"math/big"
	"time"

	"github.com/classzz/classzz/chaincfg/chainhash"
)

var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	big30 = big.NewInt(30)

	bigMinus99 = big.NewInt(-99)

	// oneLsh256 is 1 shifted left 256 bits.  It is defined here to avoid
	// the overhead of creating it multiple times.
	oneLsh256 = new(big.Int).Lsh(bigOne, 256)

	DifficultyBoundDivisor = big.NewInt(128)
	//DifficultyBoundDivisor = big.NewInt(1024) // The bound divisor of the difficulty, used in the update calculations.

)

// DifficultyAdjustmentWindow is the size of the window used by the DAA adjustment
// algorithm when calculating the current difficulty. The algorithm requires fetching
// a 'suitable' block out of blocks n-144, n-145, and n-146. We set this value equal
// to n-144 as that is the first of the three candidate blocks and we will use it
// to fetch the previous two.
const DifficultyAdjustmentWindow = 144

// The DifficultyAlgorithm specifies which algorithm to use and is passed into
// the calcNextRequiredDifficulty function.
//
// Bitcoin Cash has had three different difficulty adjustment algorithms during
// its life. What this means for us is our node needs to select which algorithm
// to use when calculating difficulty based on where it is in the chain.
type DifficultyAlgorithm uint32

const (
	// DifficultyLegacy was in effect from genesis through August 1st, 2017.
	DifficultyLegacy DifficultyAlgorithm = 0
)

// SelectDifficultyAdjustmentAlgorithm returns the difficulty adjustment algorithm that
// should be used when validating a block at the given height.
func (b *BlockChain) SelectDifficultyAdjustmentAlgorithm(height int32) DifficultyAlgorithm {
	return DifficultyLegacy
}

// HashToBig converts a chainhash.Hash into a big.Int that can be used to
// perform math comparisons.
func HashToBig(hash *chainhash.Hash) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

// CompactToBig converts a compact representation of a whole number N to an
// unsigned 32-bit number.  The representation is similar to IEEE754 floating
// point numbers.
//
// Like IEEE754 floating point, there are three basic components: the sign,
// the exponent, and the mantissa.  They are broken out as follows:
//
//	* the most significant 8 bits represent the unsigned base 256 exponent
// 	* bit 23 (the 24th bit) represents the sign bit
//	* the least significant 23 bits represent the mantissa
//
//	-------------------------------------------------
//	|   Exponent     |    Sign    |    Mantissa     |
//	-------------------------------------------------
//	| 8 bits [31-24] | 1 bit [23] | 23 bits [22-00] |
//	-------------------------------------------------
//
// The formula to calculate N is:
// 	N = (-1^sign) * mantissa * 256^(exponent-3)
//
// This compact form is only used in bitcoin to encode unsigned 256-bit numbers
// which represent difficulty targets, thus there really is not a need for a
// sign bit, but it is implemented here to stay consistent with bitcoind.
func CompactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

// BigToCompact converts a whole number N to a compact representation using
// an unsigned 32-bit number.  The compact representation only provides 23 bits
// of precision, so values larger than (2^23 - 1) only encode the most
// significant digits of the number.  See CompactToBig for details.
func BigToCompact(n *big.Int) uint32 {
	// No need to do any work if it's zero.
	if n.Sign() == 0 {
		return 0
	}

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes.  So, shift the number right or left
	// accordingly.  This is equivalent to:
	// mantissa = mantissa / 256^(exponent-3)
	var mantissa uint32
	exponent := uint(len(n.Bytes()))
	if exponent <= 3 {
		mantissa = uint32(n.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		// Use a copy to avoid modifying the caller's original number.
		tn := new(big.Int).Set(n)
		mantissa = uint32(tn.Rsh(tn, 8*(exponent-3)).Bits()[0])
	}

	// When the mantissa already has the sign bit set, the number is too
	// large to fit into the available 23-bits, so divide the number by 256
	// and increment the exponent accordingly.
	if mantissa&0x00800000 != 0 {
		mantissa >>= 8
		exponent++
	}

	// Pack the exponent, sign bit, and mantissa into an unsigned 32-bit
	// int and return it.
	compact := uint32(exponent<<24) | mantissa
	if n.Sign() < 0 {
		compact |= 0x00800000
	}
	return compact
}

// CalcWork calculates a work value from difficulty bits.  Bitcoin increases
// the difficulty for generating a block by decreasing the value which the
// generated hash must be less than.  This difficulty target is stored in each
// block header using a compact representation as described in the documentation
// for CompactToBig.  The main chain is selected by choosing the chain that has
// the most proof of work (highest difficulty).  Since a lower target difficulty
// value equates to higher actual difficulty, the work value which will be
// accumulated must be the inverse of the difficulty.  Also, in order to avoid
// potential division by zero and really small floating point numbers, the
// result adds 1 to the denominator and multiplies the numerator by 2^256.
func CalcWork(bits uint32) *big.Int {
	// Return a work value of zero if the passed difficulty bits represent
	// a negative number. Note this should not happen in practice with valid
	// blocks, but an invalid block could trigger it.
	difficultyNum := CompactToBig(bits)
	if difficultyNum.Sign() <= 0 {
		return big.NewInt(0)
	}

	// (1 << 256) / (difficultyNum + 1)
	denominator := new(big.Int).Add(difficultyNum, bigOne)
	return new(big.Int).Div(oneLsh256, denominator)
}

// findPrevTestNetDifficulty returns the difficulty of the previous block which
// did not have the special testnet minimum difficulty rule applied.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) findPrevTestNetDifficulty(startNode *blockNode) uint32 {
	// Search backwards through the chain for the last block without
	// the special rule applied.
	iterNode := startNode
	for iterNode != nil && iterNode.height%b.blocksPerRetarget != 0 &&
		iterNode.bits == b.chainParams.PowLimitBits {

		iterNode = iterNode.parent
	}

	// Return the found difficulty or the minimum difficulty if no
	// appropriate block was found.
	lastBits := b.chainParams.PowLimitBits
	if iterNode != nil {
		lastBits = iterNode.bits
	}
	return lastBits
}

// getSuitableBlock locates the two parents of passed in block, sorts the three
// blocks by timestamp and returns the median.
func (b *BlockChain) getSuitableBlock(node0 *blockNode) (*blockNode, error) {
	node1 := node0.RelativeAncestor(1)
	if node1 == nil {
		return nil, AssertError("unable to obtain relative ancestor")
	}
	node2 := node1.RelativeAncestor(1)
	if node2 == nil {
		return nil, AssertError("unable to obtain relative ancestor")
	}
	blocks := []*blockNode{node2, node1, node0}
	if blocks[0].timestamp > blocks[2].timestamp {
		blocks[0], blocks[2] = blocks[2], blocks[0]
	}
	if blocks[0].timestamp > blocks[1].timestamp {
		blocks[0], blocks[1] = blocks[1], blocks[0]
	}
	if blocks[1].timestamp > blocks[2].timestamp {
		blocks[1], blocks[2] = blocks[2], blocks[1]
	}
	return blocks[1], nil
}

// calcNextRequiredDifficulty calculates the required difficulty for the block
// after the passed previous block node based on the difficulty retarget rules.
// This function differs from the exported CalcNextRequiredDifficulty in that
// the exported version uses the current best chain as the previous block node
// while this function accepts any block node.
func (b *BlockChain) calcNextRequiredDifficulty(lastNode *blockNode, newBlockTime time.Time) (uint32, error) {

	// Genesis block.
	if lastNode == nil {
		return b.chainParams.PowLimitBits, nil
	}

	// If regest or simnet we don't adjust the difficulty
	if b.chainParams.NoDifficultyAdjustment {
		return lastNode.bits, nil
	}

	bigTime := new(big.Int).SetInt64(newBlockTime.Unix())
	bigParentTime := new(big.Int).SetInt64(lastNode.timestamp)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)
	difficulty := new(big.Int)

	// 1 - ((timestamp - parent.timestamp) // 30
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big30)
	x.Sub(bigOne, x)

	// max( 1 - (block_timestamp - parent_timestamp) // 9, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}

	if lastNode.height != 0 {
		difficulty.Sub(lastNode.workSum, lastNode.RelativeAncestor(1).workSum)
	} else {
		difficulty = lastNode.workSum
	}

	// parent_diff + (parent_diff * max( 1 - ((timestamp - parent.timestamp) // 30), -99) // 1024 )
	y.Mul(difficulty, x)
	x.Div(y, DifficultyBoundDivisor)
	newDifficulty := new(big.Int).Add(difficulty, x)
	//log.Info("Difficulty ", "number", lastNode.height, "difficulty", lastNode.workSum, "newDifficulty", newDifficulty)

	e := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	nt := new(big.Int).Sub(e, newDifficulty)
	newTarget := new(big.Int).Div(nt, newDifficulty)

	// clip again if above minimum target (too easy)
	if newTarget.Cmp(b.chainParams.PowLimit) > 0 {
		newTarget.Set(b.chainParams.PowLimit)
	}
	return BigToCompact(newTarget), nil
}

// CalcNextRequiredDifficulty calculates the required difficulty for the block
// after the end of the current best chain based on the difficulty retarget
// rules.
//
// This function is safe for concurrent access.
func (b *BlockChain) CalcNextRequiredDifficulty(timestamp time.Time) (uint32, error) {
	b.chainLock.Lock()
	tip := b.bestChain.Tip()
	difficulty, err := b.calcNextRequiredDifficulty(tip, timestamp)
	b.chainLock.Unlock()
	return difficulty, err
}
