// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"container/heap"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/classzz/classzz/blockchain"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/cross"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

const (
	// MinHighPriority is the minimum priority value that allows a
	// transaction to be considered high priority.
	MinHighPriority = czzutil.SatoshiPerBitcoin * 144.0 / 250

	// blockHeaderOverhead is the max number of bytes it takes to serialize
	// a block header and max possible transaction count.
	blockHeaderOverhead = wire.BlockHeaderSize + wire.MaxVarIntPayload

	// CoinbaseFlags is added to the coinbase script of a generated block
	// and is used to monitor BIP16 support as well as blocks that are
	// generated via classzz.
	CoinbaseFlags = "/classzz/"
)

// TxDesc is a descriptor about a transaction in a transaction source along with
// additional metadata.
type TxDesc struct {
	// Tx is the transaction associated with the entry.
	Tx *czzutil.Tx

	// Added is the time when the entry was added to the source pool.
	Added time.Time

	// Height is the block height when the entry was added to the the source
	// pool.
	Height int32

	// Fee is the total fee the transaction associated with the entry pays.
	Fee int64

	// FeePerKB is the fee the transaction pays in Satoshi per 1000 bytes.
	FeePerKB int64
}

// TxSource represents a source of transactions to consider for inclusion in
// new blocks.
//
// The interface contract requires that all of these methods are safe for
// concurrent access with respect to the source.
type TxSource interface {
	// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
	LastUpdated() time.Time

	// MiningDescs returns a slice of mining descriptors for all the
	// transactions in the source pool.
	MiningDescs() []*TxDesc

	// HaveTransaction returns whether or not the passed transaction hash
	// exists in the source pool.
	HaveTransaction(hash *chainhash.Hash) bool
}

// txPrioItem houses a transaction along with extra information that allows the
// transaction to be prioritized and track dependencies on other transactions
// which have not been mined into a block yet.
type txPrioItem struct {
	tx       *czzutil.Tx
	fee      int64
	priority float64
	feePerKB int64

	// dependsOn holds a map of transaction hashes which this one depends
	// on.  It will only be set when the transaction references other
	// transactions in the source pool and hence must come after them in
	// a block.
	dependsOn map[chainhash.Hash]struct{}
}

// txPriorityQueueLessFunc describes a function that can be used as a compare
// function for a transaction priority queue (txPriorityQueue).
type txPriorityQueueLessFunc func(*txPriorityQueue, int, int) bool

// txPriorityQueue implements a priority queue of txPrioItem elements that
// supports an arbitrary compare function as defined by txPriorityQueueLessFunc.
type txPriorityQueue struct {
	lessFunc txPriorityQueueLessFunc
	items    []*txPrioItem
}

// Len returns the number of items in the priority queue.  It is part of the
// heap.Interface implementation.
func (pq *txPriorityQueue) Len() int {
	return len(pq.items)
}

// Less returns whether the item in the priority queue with index i should sort
// before the item with index j by deferring to the assigned less function.  It
// is part of the heap.Interface implementation.
func (pq *txPriorityQueue) Less(i, j int) bool {
	return pq.lessFunc(pq, i, j)
}

// Swap swaps the items at the passed indices in the priority queue.  It is
// part of the heap.Interface implementation.
func (pq *txPriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

// Push pushes the passed item onto the priority queue.  It is part of the
// heap.Interface implementation.
func (pq *txPriorityQueue) Push(x interface{}) {
	pq.items = append(pq.items, x.(*txPrioItem))
}

// Pop removes the highest priority item (according to Less) from the priority
// queue and returns it.  It is part of the heap.Interface implementation.
func (pq *txPriorityQueue) Pop() interface{} {
	n := len(pq.items)
	item := pq.items[n-1]
	pq.items[n-1] = nil
	pq.items = pq.items[0 : n-1]
	return item
}

// SetLessFunc sets the compare function for the priority queue to the provided
// function.  It also invokes heap.Init on the priority queue using the new
// function so it can immediately be used with heap.Push/Pop.
func (pq *txPriorityQueue) SetLessFunc(lessFunc txPriorityQueueLessFunc) {
	pq.lessFunc = lessFunc
	heap.Init(pq)
}

// txPQByPriority sorts a txPriorityQueue by transaction priority and then fees
// per kilobyte.
func txPQByPriority(pq *txPriorityQueue, i, j int) bool {
	// Using > here so that pop gives the highest priority item as opposed
	// to the lowest.  Sort by priority first, then fee.
	if pq.items[i].priority == pq.items[j].priority {
		return pq.items[i].feePerKB > pq.items[j].feePerKB
	}
	return pq.items[i].priority > pq.items[j].priority

}

// txPQByFee sorts a txPriorityQueue by fees per kilobyte and then transaction
// priority.
func txPQByFee(pq *txPriorityQueue, i, j int) bool {
	// Using > here so that pop gives the highest fee item as opposed
	// to the lowest.  Sort by fee first, then priority.
	if pq.items[i].feePerKB == pq.items[j].feePerKB {
		return pq.items[i].priority > pq.items[j].priority
	}
	return pq.items[i].feePerKB > pq.items[j].feePerKB
}

func txPQByFeeAndHeight(pq *txPriorityQueue, i, j int) bool {
	// Using > here so that pop gives the highest fee item as opposed
	// to the lowest.  Sort by fee first, then priority.
	if pq.items[i].feePerKB == pq.items[j].feePerKB {
		einfos1, _ := cross.IsConvertTx(pq.items[i].tx.MsgTx())
		einfos2, _ := cross.IsConvertTx(pq.items[j].tx.MsgTx())
		if einfos1 != nil && einfos2 != nil {
			return cross.GetMaxHeight(einfos1) < cross.GetMaxHeight(einfos2)
		}
		return pq.items[i].priority > pq.items[j].priority
	}
	return pq.items[i].feePerKB > pq.items[j].feePerKB
}

// newTxPriorityQueue returns a new transaction priority queue that reserves the
// passed amount of space for the elements.  The new priority queue uses either
// the txPQByPriority or the txPQByFee compare function depending on the
// sortByFee parameter and is already initialized for use with heap.Push/Pop.
// The priority queue can grow larger than the reserved space, but extra copies
// of the underlying array can be avoided by reserving a sane value.
func newTxPriorityQueue(reserve int, sortByFee bool) *txPriorityQueue {
	pq := &txPriorityQueue{
		items: make([]*txPrioItem, 0, reserve),
	}
	//fmt.Println("sortByFee", sortByFee)
	if sortByFee {
		pq.SetLessFunc(txPQByFeeAndHeight)
	} else {
		pq.SetLessFunc(txPQByPriority)
	}
	return pq
}

// BlockTemplate houses a block that has yet to be solved along with additional
// details about the fees and the number of signature operations for each
// transaction in the block.
type BlockTemplate struct {
	// Block is a block that is ready to be solved by miners.  Thus, it is
	// completely valid with the exception of satisfying the proof-of-work
	// requirement.
	Block *wire.MsgBlock

	// Fees contains the amount of fees each transaction in the generated
	// template pays in base units.  Since the first transaction is the
	// coinbase, the first entry (offset 0) will contain the negative of the
	// sum of the fees of all other transactions.
	Fees []int64

	// SigOpCosts contains the number of signature operations each
	// transaction in the generated template performs.
	SigOpCosts []int64

	// Height is the height at which the block template connects to the main
	// chain.
	Height int32

	// ValidPayAddress indicates whether or not the template coinbase pays
	// to an address or is redeemable by anyone.  See the documentation on
	// NewBlockTemplate for details on which this can be useful to generate
	// templates without a coinbase payment address.
	ValidPayAddress bool

	// MaxBlockSize is the block size consensus rule used when creating the block
	MaxBlockSize uint32

	// MaxSigOps is the block size consensus rule used when creating the block
	MaxSigOps uint32
}

// mergeUtxoView adds all of the entries in viewB to viewA.  The result is that
// viewA will contain all of its original entries plus all of the entries
// in viewB.  It will replace any entries in viewB which also exist in viewA
// if the entry in viewA is spent.
func mergeUtxoView(viewA *blockchain.UtxoViewpoint, viewB *blockchain.UtxoViewpoint) {
	viewAEntries := viewA.Entries()
	for outpoint, entryB := range viewB.Entries() {
		if entryA, exists := viewAEntries[outpoint]; !exists ||
			entryA == nil || entryA.IsSpent() {

			viewAEntries[outpoint] = entryB
		}
	}
}

// standardCoinbaseScript returns a standard script suitable for use as the
// signature script of the coinbase transaction of a new block.  In particular,
// it starts with the block height that is required by version 2 blocks and adds
// the extra nonce as well as additional coinbase flags.
func standardCoinbaseScript(nextBlockHeight int32, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(nextBlockHeight)).
		AddInt64(int64(extraNonce)).AddData([]byte(CoinbaseFlags)).
		Script()
}

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
//
// See the comment for NewBlockTemplate for more information about why the nil
// address handling is useful.
func createCoinbaseTx(params *chaincfg.Params, coinbaseScript []byte, nextBlockHeight int32, addr czzutil.Address) (*czzutil.Tx, error) {
	// Create the script to pay to the provided payment address if one was
	// specified.  Otherwise create a script that allows the coinbase to be
	// redeemable by anyone.
	var pkScript []byte
	var pkScript1 []byte
	var pkScript2 []byte

	CoinPool1 := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	CoinPool2 := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

	var err error
	pkScript1, err = txscript.PayToPubKeyHashScript(CoinPool1)
	if err != nil {
		return nil, err
	}

	pkScript2, err = txscript.PayToPubKeyHashScript(CoinPool2)
	if err != nil {
		return nil, err
	}

	if addr != nil {
		var err error
		pkScript, err = txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		scriptBuilder := txscript.NewScriptBuilder()
		pkScript, err = scriptBuilder.AddOp(txscript.OP_TRUE).Script()
		if err != nil {
			return nil, err
		}
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		SignatureScript: coinbaseScript,
		Sequence:        wire.MaxTxInSequenceNum,
	})
	if nextBlockHeight >= params.EntangleHeight {
		// utxo of coinbase in params.EntangleHeight-1 block
		tx.AddTxIn(&wire.TxIn{}) // for pool address hold this
		tx.AddTxIn(&wire.TxIn{})
	}

	//Calculation incentive
	reward := blockchain.CalcBlockSubsidy(nextBlockHeight, params)

	// reward2 = reward * 19%
	reward1 := reward * 19 / 100

	// Calculate 1% of the reward.
	reward2 := reward / 100

	// Calculate 1 - 20% = 80% of the reward. Coinbase
	reward3 := reward - reward1 - reward2

	// Coinbase reward
	tx.AddTxOut(&wire.TxOut{
		Value:    reward3,
		PkScript: pkScript,
	})

	//Sum up all the previous pool value
	if nextBlockHeight == params.EntangleHeight {
		reward1 = reward1 * int64(params.EntangleHeight-1)
		reward2 = reward2 * int64(params.EntangleHeight-1)
	}

	// CoinPool1 reward
	tx.AddTxOut(&wire.TxOut{
		Value:    reward1,
		PkScript: pkScript1,
	})

	// CoinPool2 reward
	tx.AddTxOut(&wire.TxOut{
		Value:    reward2,
		PkScript: pkScript2,
	})
	// the amount of already entangled,placeholder
	if nextBlockHeight >= params.EntangleHeight && nextBlockHeight < params.ExChangeHeight {
		keepInfo := cross.KeepedAmount{Items: []cross.KeepedItem{}}
		keepInfo.Add(cross.KeepedItem{
			AssetType: cross.ExpandedTxEntangle_Doge,
			Amount:    big.NewInt(0),
		})
		keepInfo.Add(cross.KeepedItem{
			AssetType: cross.ExpandedTxEntangle_Ltc,
			Amount:    big.NewInt(0),
		})
		scriptInfo, err := txscript.KeepedAmountScript(keepInfo.Serialize())
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(&wire.TxOut{
			Value:    0,
			PkScript: scriptInfo,
		})
	}
	// Make sure the coinbase is above the minimum size threshold.
	if tx.SerializeSize() < blockchain.MinTransactionSize {
		tx.TxIn[0].SignatureScript = append(tx.TxIn[0].SignatureScript,
			make([]byte, blockchain.MinTransactionSize-tx.SerializeSize()-1)...)
	}
	return czzutil.NewTx(tx), nil
}

// spendTransaction updates the passed view by marking the inputs to the passed
// transaction as spent.  It also adds all outputs in the passed transaction
// which are not provably unspendable as available unspent transaction outputs.
func spendTransaction(utxoView *blockchain.UtxoViewpoint, tx *czzutil.Tx, height int32) error {
	for _, txIn := range tx.MsgTx().TxIn {
		entry := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if entry != nil {
			entry.Spend()
		}
	}

	utxoView.AddTxOuts(tx, height)
	return nil
}

// logSkippedDeps logs any dependencies which are also skipped as a result of
// skipping a transaction while generating a block template at the trace level.
func logSkippedDeps(tx *czzutil.Tx, deps map[chainhash.Hash]*txPrioItem) {
	if deps == nil {
		return
	}

	for _, item := range deps {
		log.Tracef("Skipping tx %s since it depends on %s\n",
			item.tx.Hash(), tx.Hash())
	}
}

// MinimumMedianTime returns the minimum allowed timestamp for a block building
// on the end of the provided best chain.  In particular, it is one second after
// the median timestamp of the last several blocks per the chain consensus
// rules.
func MinimumMedianTime(chainState *blockchain.BestState) time.Time {
	return chainState.MedianTime.Add(time.Second)
}

// medianAdjustedTime returns the current time adjusted to ensure it is at least
// one second after the median timestamp of the last several blocks per the
// chain consensus rules.
func medianAdjustedTime(chainState *blockchain.BestState, timeSource blockchain.MedianTimeSource) time.Time {
	// The timestamp for the block must not be before the median timestamp
	// of the last several blocks.  Thus, choose the maximum between the
	// current time and one second after the past median time.  The current
	// timestamp is truncated to a second boundary before comparison since a
	// block timestamp does not supported a precision greater than one
	// second.
	newTimestamp := timeSource.AdjustedTime()
	minTimestamp := MinimumMedianTime(chainState)
	if newTimestamp.Before(minTimestamp) {
		newTimestamp = minTimestamp
	}

	return newTimestamp
}

// BlkTmplGenerator provides a type that can be used to generate block templates
// based on a given mining policy and source of transactions to choose from.
// It also houses additional state required in order to ensure the templates
// are built on top of the current best chain and adhere to the consensus rules.
type BlkTmplGenerator struct {
	policy      *Policy
	chainParams *chaincfg.Params
	txSource    TxSource
	chain       *blockchain.BlockChain
	timeSource  blockchain.MedianTimeSource
	sigCache    *txscript.SigCache
	hashCache   *txscript.HashCache
}

// NewBlkTmplGenerator returns a new block template generator for the given
// policy using transactions from the provided transaction source.
//
// The additional state-related fields are required in order to ensure the
// templates are built on top of the current best chain and adhere to the
// consensus rules.
func NewBlkTmplGenerator(policy *Policy, params *chaincfg.Params,
	txSource TxSource, chain *blockchain.BlockChain,
	timeSource blockchain.MedianTimeSource,
	sigCache *txscript.SigCache,
	hashCache *txscript.HashCache) *BlkTmplGenerator {

	return &BlkTmplGenerator{
		policy:      policy,
		chainParams: params,
		txSource:    txSource,
		chain:       chain,
		timeSource:  timeSource,
		sigCache:    sigCache,
		hashCache:   hashCache,
	}
}

// NewBlockTemplate returns a new block template that is ready to be solved
// using the transactions from the passed transaction source pool and a coinbase
// that either pays to the passed address if it is not nil, or a coinbase that
// is redeemable by anyone if the passed address is nil.  The nil address
// functionality is useful since there are cases such as the getblocktemplate
// RPC where external mining software is responsible for creating their own
// coinbase which will replace the one generated for the block template.  Thus
// the need to have configured address can be avoided.
//
// The transactions selected and included are prioritized according to several
// factors.  First, each transaction has a priority calculated based on its
// value, age of inputs, and size.  Transactions which consist of larger
// amounts, older inputs, and small sizes have the highest priority.  Second, a
// fee per kilobyte is calculated for each transaction.  Transactions with a
// higher fee per kilobyte are preferred.  Finally, the block generation related
// policy settings are all taken into account.
//
// Transactions which only spend outputs from other transactions already in the
// block chain are immediately added to a priority queue which either
// prioritizes based on the priority (then fee per kilobyte) or the fee per
// kilobyte (then priority) depending on whether or not the BlockPrioritySize
// policy setting allots space for high-priority transactions.  Transactions
// which spend outputs from other transactions in the source pool are added to a
// dependency map so they can be added to the priority queue once the
// transactions they depend on have been included.
//
// Once the high-priority area (if configured) has been filled with
// transactions, or the priority falls below what is considered high-priority,
// the priority queue is updated to prioritize by fees per kilobyte (then
// priority).
//
// When the fees per kilobyte drop below the TxMinFreeFee policy setting, the
// transaction will be skipped unless the BlockMinSize policy setting is
// nonzero, in which case the block will be filled with the low-fee/free
// transactions until the block size reaches that minimum size.
//
// Any transactions which would cause the block to exceed the BlockMaxSize
// policy setting, exceed the maximum allowed signature operations per block, or
// otherwise cause the block to be invalid are skipped.
//
// Given the above, a block generated by this function is of the following form:
//
//   -----------------------------------  --  --
//  |      Coinbase Transaction         |   |   |
//  |-----------------------------------|   |   |
//  |                                   |   |   | ----- policy.BlockPrioritySize
//  |   High-priority Transactions      |   |   |
//  |                                   |   |   |
//  |-----------------------------------|   | --
//  |                                   |   |
//  |                                   |   |
//  |                                   |   |--- policy.BlockMaxSize
//  |  Transactions prioritized by fee  |   |
//  |  until <= policy.TxMinFreeFee     |   |
//  |                                   |   |
//  |                                   |   |
//  |                                   |   |
//  |-----------------------------------|   |
//  |  Low-fee/Non high-priority (free) |   |
//  |  transactions (while block size   |   |
//  |  <= policy.BlockMinSize)          |   |
//   -----------------------------------  --
func (g *BlkTmplGenerator) NewBlockTemplate(payToAddress czzutil.Address) (*BlockTemplate, *cross.EntangleState, error) {
	// Extend the most recently known best block.
	best := g.chain.BestSnapshot()
	nextBlockHeight := best.Height + 1

	maxBlockSize := g.chain.MaxBlockSize()

	// Create a standard coinbase transaction paying to the provided
	// address.  NOTE: The coinbase value will be updated to include the
	// fees from the selected transactions later after they have actually
	// been selected.  It is created here to detect any errors early
	// before potentially doing a lot of work below.  The extra nonce helps
	// ensure the transaction is not a duplicate transaction (paying the
	// same value to the same public key address would otherwise be an
	// identical transaction for block version 1).
	extraNonce := uint64(0)
	coinbaseScript, err := standardCoinbaseScript(nextBlockHeight, extraNonce)
	if err != nil {
		return nil, nil, err
	}
	coinbaseTx, err := createCoinbaseTx(g.chainParams, coinbaseScript,
		nextBlockHeight, payToAddress)
	if err != nil {
		return nil, nil, err
	}

	// TODO: after the fork activates we obviously will need to add the
	// ScriptVerifySchnorr to the StandardVerifyFlags. We will not be adding
	// ScriptVerifyAllowSegwitRecovery because StandardVerifyFlags because
	// it's something only mining pools should be using because it allows
	// insecure spends. However, we might want to create an option to set
	// the flag in both the mempool and here for miners so they can accept
	// segwit recovery txs.
	scriptFlags := txscript.StandardVerifyFlags

	coinbaseSigOps := int64(blockchain.CountSigOps(coinbaseTx, scriptFlags))

	// Get the current source transactions and create a priority queue to
	// hold the transactions which are ready for inclusion into a block
	// along with some priority related and fee metadata.  Reserve the same
	// number of items that are available for the priority queue.  Also,
	// choose the initial sort order for the priority queue based on whether
	// or not there is an area allocated for high-priority transactions.
	sourceTxns := g.txSource.MiningDescs()
	sortedByFee := g.policy.BlockPrioritySize == 0 || true
	priorityQueue := newTxPriorityQueue(len(sourceTxns), sortedByFee)

	// Create a slice to hold the transactions to be included in the
	// generated block with reserved space.  Also create a utxo view to
	// house all of the input transactions so multiple lookups can be
	// avoided.
	blockTxns := make([]*czzutil.Tx, 0, len(sourceTxns))
	blockUtxos := blockchain.NewUtxoViewpoint()
	entangleItems := make([]*cross.ExChangeItem, 0, 0)
	convertItems := make([]*cross.ConvertItem, 0, 0)
	// dependers is used to track transactions which depend on another
	// transaction in the source pool.  This, in conjunction with the
	// dependsOn map kept with each dependent transaction helps quickly
	// determine which dependent transactions are now eligible for inclusion
	// in the block once each transaction has been included.
	dependers := make(map[chainhash.Hash]map[chainhash.Hash]*txPrioItem)

	// Create slices to hold the fees and number of signature operations
	// for each of the selected transactions and add an entry for the
	// coinbase.  This allows the code below to simply append details about
	// a transaction as it is selected for inclusion in the final block.
	// However, since the total fees aren't known yet, use a dummy value for
	// the coinbase fee which will be updated later.
	txFees := make([]int64, 0, len(sourceTxns))
	txSigOps := make([]int64, 0, len(sourceTxns))
	txFees = append(txFees, -1) // Updated once known
	txSigOps = append(txSigOps, coinbaseSigOps)

	cHash, cheight := best.Hash, best.Height
	lView, lerr := g.chain.FetchPoolUtxoView(&cHash, cheight)
	if lerr != nil {
		return nil, nil, err
	}
	poolItem := toPoolAddrItems(lView)
	rewards := make([]*cross.PunishedRewardItem, 0, 0)
	mergeItems := make(map[uint64][]*cross.BeaconMergeItem)
	var lastScriptInfo []byte
	if g.chainParams.EntangleHeight <= nextBlockHeight && g.chainParams.ExChangeHeight > nextBlockHeight {
		var err error
		lastScriptInfo, err = g.getlastScriptInfo(&cHash, cheight)
		if err != nil {
			return nil, nil, err
		}
	}
	var eState *cross.EntangleState
	if g.chainParams.ExChangeHeight < nextBlockHeight {
		eState = g.chain.CurrentEstate()
	}

	if g.chainParams.ExChangeHeight == nextBlockHeight {
		eState = cross.NewEntangleState()
	}

	fork := false
	var eState2 *cross.EntangleState2
	if g.chainParams.BeaconHeight < nextBlockHeight && g.chainParams.ExChangeHeight > nextBlockHeight {
		eState2 = g.chain.CurrentEstate2()
		fork = true
	}

	if g.chainParams.BeaconHeight == nextBlockHeight {
		eState2 = cross.NewEntangleState2()
	}

	//if g.chainParams.BeaconHeight == nextBlockHeight {
	//	eState.PoolAmount1 = big.NewInt(coinbaseTx.MsgTx().TxOut[1].Value)
	//	eState.PoolAmount2 = big.NewInt(coinbaseTx.MsgTx().TxOut[2].Value)
	//}

	log.Debugf("Considering %d transactions for inclusion to new block",
		len(sourceTxns))

	txkeys := make(map[chainhash.Hash]string)

mempoolLoop:
	for _, txDesc := range sourceTxns {
		// A block can't have more than one coinbase or contain
		// non-finalized transactions.
		tx := txDesc.Tx
		txkeys[*tx.Hash()] = ""
		if blockchain.IsCoinBase(tx) {
			if g.chainParams.EntangleHeight > nextBlockHeight && len(tx.MsgTx().TxIn) == 3 {
				log.Tracef("Skipping coinbase tx %s", tx.Hash())
				continue
			} else if len(tx.MsgTx().TxIn) == 1 {
				log.Tracef("Skipping coinbase tx %s", tx.Hash())
				continue
			}
		}
		if !blockchain.IsFinalizedTransaction(tx, nextBlockHeight,
			g.timeSource.AdjustedTime()) {

			log.Tracef("Skipping non-finalized tx %s", tx.Hash())
			continue
		}

		// Fetch all of the utxos referenced by the this transaction.
		// NOTE: This intentionally does not fetch inputs from the
		// mempool since a transaction which depends on other
		// transactions in the mempool must come after those
		// dependencies in the final generated block.
		utxos, err := g.chain.FetchUtxoView(tx)
		if err != nil {
			log.Warnf("Unable to fetch utxo view for tx %s: %v",
				tx.Hash(), err)
			continue
		}

		// Setup dependencies for any transactions which reference
		// other transactions in the mempool so they can be properly
		// ordered below.
		prioItem := &txPrioItem{tx: tx}
		for _, txIn := range tx.MsgTx().TxIn {
			originHash := &txIn.PreviousOutPoint.Hash
			entry := utxos.LookupEntry(txIn.PreviousOutPoint)
			if entry == nil || entry.IsSpent() {
				if !g.txSource.HaveTransaction(originHash) {
					log.Tracef("Skipping tx %s because it "+
						"references unspent output %s "+
						"which is not available",
						tx.Hash(), txIn.PreviousOutPoint)
					continue mempoolLoop
				}

				// The transaction is referencing another
				// transaction in the source pool, so setup an
				// ordering dependency.
				deps, exists := dependers[*originHash]
				if !exists {
					deps = make(map[chainhash.Hash]*txPrioItem)
					dependers[*originHash] = deps
				}
				deps[*prioItem.tx.Hash()] = prioItem
				if prioItem.dependsOn == nil {
					prioItem.dependsOn = make(
						map[chainhash.Hash]struct{})
				}
				prioItem.dependsOn[*originHash] = struct{}{}

				// Skip the check below. We already know the
				// referenced transaction is available.
				continue
			}
		}

		// Calculate the final transaction priority using the input
		// value age sum as well as the adjusted transaction size.  The
		// formula is: sum(inputValue * inputAge) / adjustedTxSize
		prioItem.priority = CalcPriority(tx.MsgTx(), utxos,
			nextBlockHeight)

		// Calculate the fee in Satoshi/kB.
		prioItem.feePerKB = txDesc.FeePerKB
		prioItem.fee = txDesc.Fee

		// Add the transaction to the priority queue to mark it ready
		// for inclusion in the block unless it has dependencies.
		if prioItem.dependsOn == nil {
			heap.Push(priorityQueue, prioItem)
		}

		// Merge the referenced outputs from the input transactions to
		// this transaction into the block utxo view.  This allows the
		// code below to avoid a second lookup.
		mergeUtxoView(blockUtxos, utxos)
	}

	log.Tracef("Priority queue len %d, dependers len %d",
		priorityQueue.Len(), len(dependers))

	// The starting block size is the size of the block header plus the max
	// possible transaction count size, plus the size of the coinbase
	// transaction.
	blockSize := uint32(blockHeaderOverhead * coinbaseTx.MsgTx().SerializeSize())
	blockSigOps := coinbaseSigOps
	totalFees := int64(0)
	maxSigOps := blockchain.MaxBlockSigOps(blockSize)
	//exItems := make([]*cross.ExChangeItem, 0)

	// Choose which transactions make it into the block.
	for priorityQueue.Len() > 0 {
		// Grab the highest priority (or highest fee per kilobyte
		// depending on the sort order) transaction.
		prioItem := heap.Pop(priorityQueue).(*txPrioItem)
		tx := prioItem.tx

		// Grab any transactions which depend on this one.
		deps := dependers[*tx.Hash()]

		key := false
		for _, in := range tx.MsgTx().TxIn {
			if _, ok := txkeys[in.PreviousOutPoint.Hash]; ok {
				key = true
			}
		}

		if key {
			log.Debugf("Skipping tx %s because it would exceed "+
				"the max block size", tx.Hash())
			logSkippedDeps(tx, deps)
			continue
		}

		// Enforce maximum block size.  Also check for overflow.
		txSize := uint32(tx.MsgTx().SerializeSize())
		blockPlusTxSize := blockSize + txSize
		if blockPlusTxSize < txSize ||
			blockPlusTxSize >= g.policy.BlockMaxSize {
			log.Debugf("Skipping tx %s because it would exceed "+
				"the max block size", tx.Hash())
			logSkippedDeps(tx, deps)
			continue
		}

		// Enforce maximum signature operation cost per block.  Also
		// check for overflow.
		sigOps, err := blockchain.GetSigOps(tx, false,
			blockUtxos, scriptFlags)
		maxSigOps = blockchain.MaxBlockSigOps(blockPlusTxSize)
		if err != nil {
			log.Debugf("Skipping tx %s due to error in "+
				"GetSigOpCost: %v", tx.Hash(), err)
			logSkippedDeps(tx, deps)
			continue
		}
		if blockSigOps+int64(sigOps) < blockSigOps ||
			blockSigOps+int64(sigOps) > int64(maxSigOps) {
			log.Debugf("Skipping tx %s because it would "+
				"exceed the maximum sigops per block", tx.Hash())
			logSkippedDeps(tx, deps)
			continue
		}

		// Skip free transactions once the block is larger than the
		// minimum block size.
		if sortedByFee &&
			prioItem.feePerKB < int64(g.policy.TxMinFreeFee) &&
			blockPlusTxSize >= g.policy.BlockMinSize {

			log.Debugf("Skipping tx %s with feePerKB %d "+
				"< TxMinFreeFee %d and block size %d >= "+
				"minBlockSize %d", tx.Hash(), prioItem.feePerKB,
				g.policy.TxMinFreeFee, blockPlusTxSize,
				g.policy.BlockMinSize)
			logSkippedDeps(tx, deps)
			continue
		}

		// Prioritize by fee per kilobyte once the block is larger than
		// the priority size or there are no more high-priority
		// transactions.
		if !sortedByFee && (blockPlusTxSize >= g.policy.BlockPrioritySize ||
			prioItem.priority <= MinHighPriority) {

			log.Tracef("Switching to sort by fees per "+
				"kilobyte blockSize %d >= BlockPrioritySize "+
				"%d || priority %.2f <= minHighPriority %.2f",
				blockPlusTxSize, g.policy.BlockPrioritySize,
				prioItem.priority, MinHighPriority)

			sortedByFee = true
			priorityQueue.SetLessFunc(txPQByFee)

			// Put the transaction back into the priority queue and
			// skip it so it is re-priortized by fees if it won't
			// fit into the high-priority section or the priority
			// is too low.  Otherwise this transaction will be the
			// final one in the high-priority section, so just fall
			// though to the code below so it is added now.
			if blockPlusTxSize > g.policy.BlockPrioritySize ||
				prioItem.priority < MinHighPriority {

				heap.Push(priorityQueue, prioItem)
				continue
			}
		}

		// Ensure the transaction inputs pass all of the necessary
		// preconditions before allowing it to be added to the block.
		_, err = blockchain.CheckTransactionInputs(tx, nextBlockHeight,
			blockUtxos, g.chainParams)
		if err != nil {
			log.Tracef("Skipping tx %s due to error in "+
				"CheckTransactionInputs: %v", tx.Hash(), err)
			logSkippedDeps(tx, deps)
			continue
		}

		if g.chainParams.BeaconHeight < nextBlockHeight && g.chainParams.ExChangeHeight > nextBlockHeight {

			// BeaconRegistrationTx
			if br, _ := cross.IsBeaconRegistrationTx(tx.MsgTx(), g.chainParams); br != nil {
				if err = eState2.RegisterBeaconAddress(br.Address, br.ToAddress, br.StakingAmount, br.Fee, br.KeepBlock, br.AssetFlag, br.WhiteList, br.CoinBaseAddress); err != nil {
					return nil, nil, err
				}
			}

			// AddBeaconPledgeTx
			if bp, _ := cross.IsAddBeaconPledgeTx(tx.MsgTx(), g.chainParams); bp != nil {
				if err = eState2.AppendAmountForBeaconAddress(bp.Address, bp.StakingAmount); err != nil {
					return nil, nil, err
				}
			}
		}

		if nextBlockHeight >= g.chainParams.ConverHeight {

			// IsConvertTx
			if cinfo, _ := cross.IsConvertTx(tx.MsgTx()); cinfo != nil {
				objs, err := cross.ToAddressFromConverts(cinfo, g.chain.GetExChangeVerify(), eState, nextBlockHeight)
				if err != nil {
					log.Tracef("Skipping tx %s due to error in "+
						"toAddressFromEntangle: %v", tx.Hash(), err)
					logSkippedDeps(tx, deps)
					continue
				}

				convertItems = append(convertItems, objs...)
			}

			beaconMerge, beaconID, txAmount := 0, uint64(0), big.NewInt(0)
			// BeaconRegistrationTx
			if info, _ := cross.IsBeaconRegistrationTx(tx.MsgTx(), g.chainParams); info != nil {
				if err := eState.RegisterBeaconAddress(info.Address, info.ToAddress, info.PubKey, info.StakingAmount, info.Fee,
					info.KeepBlock, info.AssetFlag, info.WhiteList, info.CoinBaseAddress); err != nil {
					log.Tracef("Skipping tx %s due to error in "+
						"IsBeaconRegistrationTx RegisterBeaconAddress: %v", tx.Hash(), err)
					logSkippedDeps(tx, deps)
					continue
				}
				beaconMerge, beaconID, txAmount = 1, eState.GetBeaconIdByTo(info.ToAddress), new(big.Int).Set(info.StakingAmount)
			}

			// AddBeaconPledgeTx
			if bp, _ := cross.IsAddBeaconPledgeTx(tx.MsgTx(), g.chainParams); bp != nil {
				if err = eState.AppendAmountForBeaconAddress(bp.Address, bp.StakingAmount); err != nil {
					log.Tracef("Skipping tx %s due to error in "+
						"IsAddBeaconPledgeTx AppendAmountForBeaconAddress: %v", tx.Hash(), err)
					logSkippedDeps(tx, deps)
					continue
				}
				beaconMerge, beaconID, txAmount = 2, eState.GetBeaconIdByTo(bp.ToAddress), new(big.Int).Set(bp.StakingAmount)
			}

			if beaconMerge > 0 && nextBlockHeight >= g.chainParams.ExChangeHeight+1 {

				if beaconMerge == 1 {
					if exInfos := eState.GetBaExInfoByID(beaconID); exInfos != nil {

						ex := cross.NewExBeaconInfo()
						ex.EnItems = []*wire.OutPoint{&wire.OutPoint{
							Hash:  *tx.Hash(),
							Index: 1,
						}}

						err := eState.SetBaExInfo(beaconID, ex)
						if err != nil {
							log.Tracef("Skipping tx %s due to error in "+
								"GetBaExInfoByID SetBaExInfo: %v", tx.Hash(), err)
							logSkippedDeps(tx, deps)
							continue
						}
					} else {
						err := fmt.Sprintf("beacon merge failed,exInfo not nil,id:%v", beaconID)
						log.Tracef("Skipping tx %s due to error in "+
							"GetBaExInfoByID : %v", tx.Hash(), err)
						logSkippedDeps(tx, deps)
						continue
					}
				} else {
					exInfos := eState.GetBaExInfoByID(beaconID)
					if exInfos == nil {
						err := fmt.Sprintf("beacon merge(in GetExInfos) failed,tx:%s,id:%v", tx.Hash(), beaconID)
						log.Tracef("Skipping tx %s due to error in "+
							"GetBaExInfoByID : %v", tx.Hash(), err)
						logSkippedDeps(tx, deps)
						continue
					}
					if view, err1 := g.chain.FetchUtxoForBeacon(exInfos.EnItems); err1 != nil {
						err := fmt.Sprintf("beacon merge(in fetch) failed,tx:%s,id:%v", tx.Hash(), beaconID)
						log.Tracef("Skipping tx %s due to error in "+
							"FetchUtxoForBeacon : %v", tx.Hash(), err)
						logSkippedDeps(tx, deps)
						continue
					} else {
						if mergeItem, to := toMergeBeaconItems(view, beaconID, eState, g.chainParams); mergeItem == nil {
							err := fmt.Sprintf("beacon merge failed,tx:%s,id:%v", tx.Hash(), beaconID)
							log.Tracef("Skipping tx %s due to error in "+
								"toMergeBeaconItems : %v", tx.Hash(), err)
							logSkippedDeps(tx, deps)
							continue
						} else {
							mergeItems[beaconID] = append(mergeItems[beaconID], mergeItem)
							mergeItems[beaconID] = append(mergeItems[beaconID], &cross.BeaconMergeItem{
								POut: wire.OutPoint{
									Hash:  *tx.Hash(),
									Index: 1,
								},
								ToAddress: to,
								Amount:    txAmount,
							})
						}
					}
				}
			}

		}

		err = blockchain.ValidateTransactionScripts(tx, blockUtxos,
			txscript.StandardVerifyFlags, g.sigCache,
			g.hashCache)
		if err != nil {
			log.Tracef("Skipping tx %s due to error in "+
				"ValidateTransactionScripts: %v", tx.Hash(), err)
			logSkippedDeps(tx, deps)
			continue
		}
		// Spend the transaction inputs in the block utxo view and add
		// an entry for it to ensure any transactions which reference
		// this one have it available as an input and can ensure they
		// aren't double spending.
		spendTransaction(blockUtxos, tx, nextBlockHeight)

		// Add the transaction to the block, increment counters, and
		// save the fees and signature operation counts to the block
		// template.
		blockTxns = append(blockTxns, tx)
		blockSize = blockPlusTxSize
		blockSigOps += int64(sigOps)
		totalFees += prioItem.fee
		txFees = append(txFees, prioItem.fee)
		txSigOps = append(txSigOps, int64(sigOps))

		log.Tracef("Adding tx %s (priority %.2f, feePerKB %.2f)",
			prioItem.tx.Hash(), prioItem.priority, prioItem.feePerKB)

		// Add transactions which depend on this one (and also do not
		// have any other unsatisified dependencies) to the priority
		// queue.
		for _, item := range deps {
			// Add the transaction to the priority queue if there
			// are no more dependencies after this one.
			delete(item.dependsOn, *tx.Hash())
			if len(item.dependsOn) == 0 {
				heap.Push(priorityQueue, item)
			}
		}
	}

	// Now that the actual transactions have been selected, update the
	// block size for the real transaction count and coinbase value with
	// the total fees accordingly.
	blockSize -= wire.MaxVarIntPayload - uint32(wire.VarIntSerializeSize(uint64(len(blockTxns))))
	coinbaseTx.MsgTx().TxOut[0].Value += totalFees
	txFees[0] = -totalFees

	// Calculate the required difficulty for the block.  The timestamp
	// is potentially adjusted to ensure it comes after the median time of
	// the last several blocks per the chain consensus rules.
	ts := medianAdjustedTime(best, g.timeSource)
	reqDifficulty, err := g.chain.CalcNextRequiredDifficulty(ts)
	if err != nil {
		return nil, nil, err
	}

	// Calculate the next expected block version based on the state of the
	// rule change deployments.
	nextBlockVersion, err := g.chain.CalcNextBlockVersion()
	if err != nil {
		return nil, nil, err
	}

	// we need to sort transactions by txid to comply with the CTOR consensus rule.
	sort.Sort(TxSorter(blockTxns))

	if g.chainParams.ExChangeHeight <= nextBlockHeight && eState != nil {
		burnTimeout := eState.TourAllUserBurnInfo(uint64(nextBlockHeight))
		for beaconID, bit := range burnTimeout {

			toAddress := eState.GetBeaconToAddrByID(beaconID, g.chainParams)
			exInfos := eState.GetBaExInfoByID(beaconID)

			if toAddress == nil || exInfos == nil {
				return nil, nil, errors.New("toAddress == nil || exInfos == nil")
			}

			var view *blockchain.UtxoViewpoint
			if view, err = g.chain.FetchUtxoForBeacon(exInfos.EnItems); err != nil {
				return nil, nil, err
			}

			AmountSum := big.NewInt(0)
			for k, v := range bit {
				fmt.Println("burnTimeout k:", k)
				AmountSum = big.NewInt(0).Add(AmountSum, v.AmountSum)
			}

			if item, err := toRewardsByPunishedBurn(view, payToAddress, toAddress, AmountSum); err != nil {
				return nil, nil, err
			} else {
				rewards = append(rewards, item)
			}
		}
		eState.UpdateStateToPunished(burnTimeout)
	}

	// make entangle tx if it exist
	if g.chainParams.EntangleHeight <= nextBlockHeight && g.chainParams.ExChangeHeight > nextBlockHeight {
		eItems := make([]*cross.EntangleItem, 0)
		err = cross.MakeMergerCoinbaseTx2(coinbaseTx.MsgTx(), poolItem, eItems, lastScriptInfo, fork)
		if err != nil {
			return nil, nil, err
		}
	}

	// make entangle tx if it exist
	if g.chainParams.ExChangeHeight <= nextBlockHeight {
		err = cross.MakeMergerCoinbaseTx(coinbaseTx.MsgTx(), poolItem, entangleItems, convertItems, rewards, mergeItems)
		if err != nil {
			return nil, nil, err
		}
	}

	CIDRoot := chainhash.Hash{}
	// Beacon
	if g.chainParams.BeaconHeight <= nextBlockHeight && g.chainParams.ExChangeHeight > nextBlockHeight && eState2 != nil {
		CIDRoot = cross.Hash2(eState2)
	}

	// ExChange
	if g.chainParams.ExChangeHeight <= nextBlockHeight && eState != nil {
		CIDRoot = cross.Hash(eState)
	}

	blockTxns = append([]*czzutil.Tx{coinbaseTx}, blockTxns...)

	// Create a new block ready to be solved.
	merkles := blockchain.BuildMerkleTreeStore(blockTxns)
	var msgBlock wire.MsgBlock
	msgBlock.Header = wire.BlockHeader{
		Version:    nextBlockVersion,
		PrevBlock:  best.Hash,
		MerkleRoot: *merkles[len(merkles)-1],
		CIDRoot:    CIDRoot,
		Timestamp:  ts,
		Bits:       reqDifficulty,
	}
	for _, tx := range blockTxns {
		if err := msgBlock.AddTransaction(tx.MsgTx()); err != nil {
			return nil, nil, err
		}
	}

	// Finally, perform a full check on the created block against the chain
	// consensus rules to ensure it properly connects to the current best
	// chain with no issues.
	block := czzutil.NewBlock(&msgBlock)
	block.SetHeight(nextBlockHeight)
	if err := g.chain.CheckConnectBlockTemplate(block); err != nil {
		return nil, nil, err
	}

	log.Debugf("Created new block template (%d transactions, %d in "+
		"fees, %d signature operations, %d size, target difficulty "+
		"%064x)", len(msgBlock.Transactions), totalFees, blockSigOps,
		blockSize, blockchain.CompactToBig(msgBlock.Header.Bits))

	var eState3 *cross.EntangleState
	if g.chainParams.BeaconHeight <= nextBlockHeight-1 && g.chainParams.ExChangeHeight > nextBlockHeight-1 {
		eState4 := g.chain.CurrentEstate2()

		bai2s := make(map[string]*cross.BeaconAddressInfo)
		for k, v := range eState4.EnInfos {
			bai2 := &cross.BeaconAddressInfo{
				BeaconID:        v.ExchangeID,
				StakingAmount:   v.StakingAmount,
				EntangleAmount:  v.EntangleAmount,
				CoinBaseAddress: v.CoinBaseAddress,
			}
			bai2s[k] = bai2
		}

		eState3 = &cross.EntangleState{
			EnInfos: bai2s,
		}

	} else if g.chainParams.ExChangeHeight <= nextBlockHeight-1 {
		eState3 = g.chain.CurrentEstate()
	}

	return &BlockTemplate{
		Block:           &msgBlock,
		Fees:            txFees,
		SigOpCosts:      txSigOps,
		Height:          nextBlockHeight,
		ValidPayAddress: payToAddress != nil,
		MaxBlockSize:    uint32(maxBlockSize),
		MaxSigOps:       uint32(maxSigOps),
	}, eState3, nil
}

// UpdateBlockTime updates the timestamp in the header of the passed block to
// the current time while taking into account the median time of the last
// several blocks to ensure the new time is after that time per the chain
// consensus rules.  Finally, it will update the target difficulty if needed
// based on the new time for the test networks since their target difficulty can
// change based upon time.
func (g *BlkTmplGenerator) UpdateBlockTime(msgBlock *wire.MsgBlock) error {
	// The new timestamp is potentially adjusted to ensure it comes after
	// the median time of the last several blocks per the chain consensus
	// rules.
	newTime := medianAdjustedTime(g.chain.BestSnapshot(), g.timeSource)
	msgBlock.Header.Timestamp = newTime

	// Recalculate the difficulty if running on a network that requires it.
	if g.chainParams.ReduceMinDifficulty {
		difficulty, err := g.chain.CalcNextRequiredDifficulty(newTime)
		if err != nil {
			return err
		}
		msgBlock.Header.Bits = difficulty
	}

	return nil
}

// UpdateExtraNonce updates the extra nonce in the coinbase script of the passed
// block by regenerating the coinbase script with the passed value and block
// height.  It also recalculates and updates the new merkle root that results
// from changing the coinbase script.
func (g *BlkTmplGenerator) UpdateExtraNonce(msgBlock *wire.MsgBlock, blockHeight int32, extraNonce uint64) error {
	coinbaseScript, err := standardCoinbaseScript(blockHeight, extraNonce)
	if err != nil {
		return err
	}
	if len(coinbaseScript) > blockchain.MaxCoinbaseScriptLen {
		return fmt.Errorf("coinbase transaction script length "+
			"of %d is out of range (min: %d, max: %d)",
			len(coinbaseScript), blockchain.MinCoinbaseScriptLen,
			blockchain.MaxCoinbaseScriptLen)
	}
	msgBlock.Transactions[0].TxIn[0].SignatureScript = coinbaseScript

	// TODO(davec): A czzutil.Block should use saved in the state to avoid
	// recalculating all of the other transaction hashes.
	// block.Transactions[0].InvalidateCache()

	// Recalculate the merkle root with the updated extra nonce.
	block := czzutil.NewBlock(msgBlock)
	merkles := blockchain.BuildMerkleTreeStore(block.Transactions())
	msgBlock.Header.MerkleRoot = *merkles[len(merkles)-1]
	return nil
}

// BestSnapshot returns information about the current best chain block and
// related state as of the current point in time using the chain instance
// associated with the block template generator.  The returned state must be
// treated as immutable since it is shared by all callers.
//
// This function is safe for concurrent access.
func (g *BlkTmplGenerator) BestSnapshot() *blockchain.BestState {
	return g.chain.BestSnapshot()
}

// TxSource returns the associated transaction source.
//
// This function is safe for concurrent access.
func (g *BlkTmplGenerator) TxSource() TxSource {
	return g.txSource
}
func (g *BlkTmplGenerator) getlastScriptInfo(hash *chainhash.Hash, height int32) ([]byte, error) {
	block, err := g.chain.BlockByHash(hash)
	if err != nil {
		return nil, err
	}
	if block.Height() != height {
		return nil, errors.New("the height not match")
	}
	tx, err := block.Tx(0)
	if err != nil {
		return nil, err
	}
	if height < g.chainParams.EntangleHeight {
		return nil, nil
	}
	txout := tx.MsgTx().TxOut[3]
	return txout.PkScript, nil
}
func toPoolAddrItems(view *blockchain.UtxoViewpoint) *cross.PoolAddrItem {
	items := &cross.PoolAddrItem{
		POut:   make([]wire.OutPoint, 2),
		Script: make([][]byte, 2),
		Amount: make([]*big.Int, 2),
	}
	if view != nil {
		m := view.Entries()
		for k, v := range m {
			items.POut[k.Index-1] = k
			items.Script[k.Index-1] = v.PkScript()
			items.Amount[k.Index-1] = new(big.Int).SetInt64(v.Amount())
		}
	}
	return items
}
func toMergeBeaconItems(view *blockchain.UtxoViewpoint, id uint64, state *cross.EntangleState, params *chaincfg.Params) (*cross.BeaconMergeItem, czzutil.Address) {
	if view == nil {
		return nil, nil
	}
	to := state.GetBeaconToAddrByID(id, params)
	if to == nil {
		return nil, to
	}
	items := &cross.BeaconMergeItem{
		ToAddress: to,
	}
	m := view.Entries()
	for k, v := range m {
		items.POut, items.Script, items.Amount = k, v.PkScript(), new(big.Int).SetInt64(v.Amount())
		break
	}
	return items, to
}

func toRewardsByPunished(info *cross.BurnProofInfo, view *blockchain.UtxoViewpoint,
	rewardAddress, changeAddress czzutil.Address) (*cross.PunishedRewardItem, error) {
	if view == nil {
		return nil, errors.New("view is nil")
	}
	res := &cross.PunishedRewardItem{
		Amount: new(big.Int).Mul(big.NewInt(1), info.Amount),
		Addr1:  rewardAddress,
		Addr2:  cross.ZeroAddrsss,
		Addr3:  changeAddress,
	}
	m := view.Entries()
	for k, v := range m {
		res.POut, res.Script, res.OriginAmount = k, v.PkScript(), new(big.Int).SetInt64(v.Amount())
		break
	}
	return res, nil
}

func toRewardsByPunishedBurn(view *blockchain.UtxoViewpoint, rewardAddress, changeAddress czzutil.Address, Amount *big.Int) (*cross.PunishedRewardItem, error) {
	if view == nil {
		return nil, errors.New("view is nil")
	}
	res := &cross.PunishedRewardItem{
		Amount: new(big.Int).Mul(big.NewInt(1), Amount),
		Addr1:  rewardAddress,
		Addr2:  cross.ZeroAddrsss,
		Addr3:  changeAddress,
	}
	m := view.Entries()
	for k, v := range m {
		res.POut, res.Script, res.OriginAmount = k, v.PkScript(), new(big.Int).SetInt64(v.Amount())
		break
	}
	return res, nil
}

func toRewardsByWhiteListPunished(info *cross.WhiteListProof, view *blockchain.UtxoViewpoint, height uint64,
	state *cross.EntangleState, rewardAddress, toAddress czzutil.Address) (*cross.PunishedRewardItem, error) {
	if view == nil {
		return nil, errors.New("view is nil")
	}
	amount := state.CalcSlashingForWhiteListProof(info.Amount, info.AssetType, info.BeaconID)
	res := &cross.PunishedRewardItem{
		Amount: new(big.Int).Mul(big.NewInt(1), amount),
		Addr1:  rewardAddress,
		Addr2:  cross.ZeroAddrsss,
		Addr3:  toAddress,
	}
	m := view.Entries()
	for k, v := range m {
		res.POut, res.Script, res.OriginAmount = k, v.PkScript(), new(big.Int).SetInt64(v.Amount())
		break
	}
	return res, nil
}
