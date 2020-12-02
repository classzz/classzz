// Copyright (c) 2015-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/cross"
	"github.com/classzz/classzz/rlp"
	"math/big"
	"sync"
	"time"

	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/database"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

const (
	// blockHdrSize is the size of a block header.  This is simply the
	// constant from wire and is only provided here for convenience since
	// wire.BlockHeaderSize is quite long.
	blockHdrSize = wire.BlockHeaderSize

	// latestUtxoSetBucketVersion is the current version of the utxo set
	// bucket that is used to track all unspent outputs.
	latestUtxoSetBucketVersion = 2

	// latestSpendJournalBucketVersion is the current version of the spend
	// journal bucket that is used to track all spent transactions for use
	// in reorgs.
	latestSpendJournalBucketVersion = 1
)

var (
	// blockIndexBucketName is the name of the db bucket used to house to the
	// block headers and contextual information.
	blockIndexBucketName = []byte("blockheaderidx")

	// hashIndexBucketName is the name of the db bucket used to house to the
	// block hash -> block height index.
	hashIndexBucketName = []byte("hashidx")

	// heightIndexBucketName is the name of the db bucket used to house to
	// the block height -> block hash index.
	heightIndexBucketName = []byte("heightidx")

	// chainStateKeyName is the name of the db key used to store the best
	// chain state.
	chainStateKeyName = []byte("chainstate")

	// spendJournalVersionKeyName is the name of the db key used to store
	// the version of the spend journal currently in the database.
	spendJournalVersionKeyName = []byte("spendjournalversion")

	// spendJournalBucketName is the name of the db bucket used to house
	// transactions outputs that are spent in each block.
	spendJournalBucketName = []byte("spendjournal")

	// utxoStateConsistencyKeyName is the name of the db key used to store the
	// consistency status of the utxo state.
	utxoStateConsistencyKeyName = []byte("utxostateconsistency")

	// blockchainTypeKeyName is the name of the db key used to store the
	// prune status of the blockchain. If it was ever run in prune mode
	// or fastsync mode then it should be treated as a pruned chain.
	blockchainTypeKeyName = []byte("blockchaintype")

	// prunedBlockchainEntryValue is the value the corresponds to a
	// pruned blockchain.
	prunedBlockchainEntryValue = []byte("prunedblockchain")

	// utxoSetVersionKeyName is the name of the db key used to store the
	// version of the utxo set currently in the database.
	utxoSetVersionKeyName = []byte("utxosetversion")

	// utxoSetBucketName is the name of the db bucket used to house the
	// unspent transaction output set.
	utxoSetBucketName = []byte("utxosetv2")

	// pruneHeightKeyName is the name of the db key used to store the
	// height at which the blockchain is pruned.
	pruneHeightKeyName = []byte("pruneheight")

	// byteOrder is the preferred byte order used for serializing numeric
	// fields for storage in the database.
	byteOrder = binary.LittleEndian

	NetParams *chaincfg.Params
)

// errNotInMainChain signifies that a block hash or height that is not in the
// main chain was requested.
type errNotInMainChain string

// Error implements the error interface.
func (e errNotInMainChain) Error() string {
	return string(e)
}

// isNotInMainChainErr returns whether or not the passed error is an
// errNotInMainChain error.
func isNotInMainChainErr(err error) bool {
	_, ok := err.(errNotInMainChain)
	return ok
}

// errDeserialize signifies that a problem was encountered when deserializing
// data.
type errDeserialize string

// Error implements the error interface.
func (e errDeserialize) Error() string {
	return string(e)
}

// isDeserializeErr returns whether or not the passed error is an errDeserialize
// error.
func isDeserializeErr(err error) bool {
	_, ok := err.(errDeserialize)
	return ok
}

// isDbBucketNotFoundErr returns whether or not the passed error is a
// database.Error with an error code of database.ErrBucketNotFound.
func isDbBucketNotFoundErr(err error) bool {
	dbErr, ok := err.(database.Error)
	return ok && dbErr.ErrorCode == database.ErrBucketNotFound
}

// dbFetchVersion fetches an individual version with the given key from the
// metadata bucket.  It is primarily used to track versions on entities such as
// buckets.  It returns zero if the provided key does not exist.
func dbFetchVersion(dbTx database.Tx, key []byte) uint32 {
	serialized := dbTx.Metadata().Get(key)
	if serialized == nil {
		return 0
	}

	return byteOrder.Uint32(serialized[:])
}

// dbPutVersion uses an existing database transaction to update the provided
// key in the metadata bucket to the given version.  It is primarily used to
// track versions on entities such as buckets.
func dbPutVersion(dbTx database.Tx, key []byte, version uint32) error {
	var serialized [4]byte
	byteOrder.PutUint32(serialized[:], version)
	return dbTx.Metadata().Put(key, serialized[:])
}

// dbFetchOrCreateVersion uses an existing database transaction to attempt to
// fetch the provided key from the metadata bucket as a version and in the case
// it doesn't exist, it adds the entry with the provided default version and
// returns that.  This is useful during upgrades to automatically handle loading
// and adding version keys as necessary.
func dbFetchOrCreateVersion(dbTx database.Tx, key []byte, defaultVersion uint32) (uint32, error) {
	version := dbFetchVersion(dbTx, key)
	if version == 0 {
		version = defaultVersion
		err := dbPutVersion(dbTx, key, version)
		if err != nil {
			return 0, err
		}
	}

	return version, nil
}

// -----------------------------------------------------------------------------
// The transaction spend journal consists of an entry for each block connected
// to the main chain which contains the transaction outputs the block spends
// serialized such that the order is the reverse of the order they were spent.
//
// This is required because reorganizing the chain necessarily entails
// disconnecting blocks to get back to the point of the fork which implies
// unspending all of the transaction outputs that each block previously spent.
// Since the utxo set, by definition, only contains unspent transaction outputs,
// the spent transaction outputs must be resurrected from somewhere.  There is
// more than one way this could be done, however this is the most straight
// forward method that does not require having a transaction index and unpruned
// blockchain.
//
// NOTE: This format is NOT self describing.  The additional details such as
// the number of entries (transaction inputs) are expected to come from the
// block itself and the utxo set (for legacy entries).  The rationale in doing
// this is to save space.  This is also the reason the spent outputs are
// serialized in the reverse order they are spent because later transactions are
// allowed to spend outputs from earlier ones in the same block.
//
// The reserved field below used to keep track of the version of the containing
// transaction when the height in the header code was non-zero, however the
// height is always non-zero now, but keeping the extra reserved field allows
// backwards compatibility.
//
// The serialized format is:
//
//   [<header code><reserved><compressed txout>],...
//
//   Field                Type     Size
//   header code          VLQ      variable
//   reserved             byte     1
//   compressed txout
//     compressed amount  VLQ      variable
//     compressed script  []byte   variable
//
// The serialized header code format is:
//   bit 0 - containing transaction is a coinbase
//   bits 1-x - height of the block that contains the spent txout
//
// Example 1:
// From block 170 in main blockchain.
//
//    1300320511db93e1dcdb8a016b49840f8c53bc1eb68a382e97b1482ecad7b148a6909a5c
//    <><><------------------------------------------------------------------>
//     | |                                  |
//     | reserved                  compressed txout
//    header code
//
//  - header code: 0x13 (coinbase, height 9)
//  - reserved: 0x00
//  - compressed txout 0:
//    - 0x32: VLQ-encoded compressed amount for 5000000000 (50 CZZ)
//    - 0x05: special script type pay-to-pubkey
//    - 0x11...5c: x-coordinate of the pubkey
//
// Example 2:
// Adapted from block 100025 in main blockchain.
//
//    8b99700091f20f006edbc6c4d31bae9f1ccc38538a114bf42de65e868b99700086c64700b2fb57eadf61e106a100a7445a8c3f67898841ec
//    <----><><----------------------------------------------><----><><---------------------------------------------->
//     |    |                         |                        |    |                         |
//     |    reserved         compressed txout                  |    reserved         compressed txout
//    header code                                          header code
//
//  - Last spent output:
//    - header code: 0x8b9970 (not coinbase, height 100024)
//    - reserved: 0x00
//    - compressed txout:
//      - 0x91f20f: VLQ-encoded compressed amount for 34405000000 (344.05 CZZ)
//      - 0x00: special script type pay-to-pubkey-hash
//      - 0x6e...86: pubkey hash
//  - Second to last spent output:
//    - header code: 0x8b9970 (not coinbase, height 100024)
//    - reserved: 0x00
//    - compressed txout:
//      - 0x86c647: VLQ-encoded compressed amount for 13761000000 (137.61 CZZ)
//      - 0x00: special script type pay-to-pubkey-hash
//      - 0xb2...ec: pubkey hash
// -----------------------------------------------------------------------------

// SpentTxOut contains a spent transaction output and potentially additional
// contextual information such as whether or not it was contained in a coinbase
// transaction, the version of the transaction it was contained in, and which
// block height the containing transaction was included in.  As described in
// the comments above, the additional contextual information will only be valid
// when this spent txout is spending the last unspent output of the containing
// transaction.
type SpentTxOut struct {
	// Amount is the amount of the output.
	Amount int64

	// PkScipt is the the public key script for the output.
	PkScript []byte

	// Height is the height of the the block containing the creating tx.
	Height int32

	// Denotes if the creating tx is a coinbase.
	IsCoinBase bool
}

// FetchSpendJournal attempts to retrieve the spend journal, or the set of
// outputs spent for the target block. This provides a view of all the outputs
// that will be consumed once the target block is connected to the end of the
// main chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) FetchSpendJournal(targetBlock *czzutil.Block) ([]SpentTxOut, error) {
	b.chainLock.RLock()
	defer b.chainLock.RUnlock()

	var spendEntries []SpentTxOut
	err := b.db.View(func(dbTx database.Tx) error {
		var err error

		spendEntries, err = dbFetchSpendJournalEntry(dbTx, targetBlock)
		return err
	})
	if err != nil {
		return nil, err
	}

	return spendEntries, nil
}

// spentTxOutHeaderCode returns the calculated header code to be used when
// serializing the provided stxo entry.
func spentTxOutHeaderCode(stxo *SpentTxOut) uint64 {
	// As described in the serialization format comments, the header code
	// encodes the height shifted over one bit and the coinbase flag in the
	// lowest bit.
	headerCode := uint64(stxo.Height) << 1
	if stxo.IsCoinBase {
		headerCode |= 0x01
	}

	return headerCode
}

// spentTxOutSerializeSize returns the number of bytes it would take to
// serialize the passed stxo according to the format described above.
func spentTxOutSerializeSize(stxo *SpentTxOut) int {
	size := serializeSizeVLQ(spentTxOutHeaderCode(stxo))
	if stxo.Height > 0 {
		// The legacy v1 spend journal format conditionally tracked the
		// containing transaction version when the height was non-zero,
		// so this is required for backwards compat.
		size += serializeSizeVLQ(0)
	}
	return size + compressedTxOutSize(uint64(stxo.Amount), stxo.PkScript)
}

// putSpentTxOut serializes the passed stxo according to the format described
// above directly into the passed target byte slice.  The target byte slice must
// be at least large enough to handle the number of bytes returned by the
// SpentTxOutSerializeSize function or it will panic.
func putSpentTxOut(target []byte, stxo *SpentTxOut) int {
	headerCode := spentTxOutHeaderCode(stxo)
	offset := putVLQ(target, headerCode)
	if stxo.Height > 0 {
		// The legacy v1 spend journal format conditionally tracked the
		// containing transaction version when the height was non-zero,
		// so this is required for backwards compat.
		offset += putVLQ(target[offset:], 0)
	}
	return offset + putCompressedTxOut(target[offset:], uint64(stxo.Amount),
		stxo.PkScript)
}

// decodeSpentTxOut decodes the passed serialized stxo entry, possibly followed
// by other data, into the passed stxo struct.  It returns the number of bytes
// read.
func decodeSpentTxOut(serialized []byte, stxo *SpentTxOut) (int, error) {
	// Ensure there are bytes to decode.
	if len(serialized) == 0 {
		return 0, errDeserialize("no serialized bytes")
	}

	// Deserialize the header code.
	code, offset := deserializeVLQ(serialized)
	if offset >= len(serialized) {
		return offset, errDeserialize("unexpected end of data after " +
			"header code")
	}

	// Decode the header code.
	//
	// Bit 0 indicates containing transaction is a coinbase.
	// Bits 1-x encode height of containing transaction.
	stxo.IsCoinBase = code&0x01 != 0
	stxo.Height = int32(code >> 1)
	if stxo.Height > 0 {
		// The legacy v1 spend journal format conditionally tracked the
		// containing transaction version when the height was non-zero,
		// so this is required for backwards compat.
		_, bytesRead := deserializeVLQ(serialized[offset:])
		offset += bytesRead
		if offset >= len(serialized) {
			return offset, errDeserialize("unexpected end of data " +
				"after reserved")
		}
	}

	// Decode the compressed txout.
	amount, pkScript, bytesRead, err := decodeCompressedTxOut(
		serialized[offset:])
	offset += bytesRead
	if err != nil {
		return offset, errDeserialize(fmt.Sprintf("unable to decode "+
			"txout: %v", err))
	}
	stxo.Amount = int64(amount)
	stxo.PkScript = pkScript
	return offset, nil
}

// deserializeSpendJournalEntry decodes the passed serialized byte slice into a
// slice of spent txouts according to the format described in detail above.
//
// Since the serialization format is not self describing, as noted in the
// format comments, this function also requires the transactions that spend the
// txouts.
func deserializeSpendJournalEntry(serialized []byte, txns []*wire.MsgTx) ([]SpentTxOut, error) {
	// Calculate the total number of stxos.
	var numStxos int
	for _, tx := range txns {
		numStxos += len(tx.TxIn)
	}

	// When a block has no spent txouts there is nothing to serialize.
	if len(serialized) == 0 {
		// Ensure the block actually has no stxos.  This should never
		// happen unless there is database corruption or an empty entry
		// erroneously made its way into the database.
		if numStxos != 0 {
			return nil, AssertError(fmt.Sprintf("mismatched spend "+
				"journal serialization - no serialization for "+
				"expected %d stxos", numStxos))
		}

		return nil, nil
	}

	// Loop backwards through all transactions so everything is read in
	// reverse order to match the serialization order.
	stxoIdx := numStxos - 1
	offset := 0
	stxos := make([]SpentTxOut, numStxos)
	for txIdx := len(txns) - 1; txIdx > -1; txIdx-- {
		tx := txns[txIdx]

		// Loop backwards through all of the transaction inputs and read
		// the associated stxo.
		for txInIdx := len(tx.TxIn) - 1; txInIdx > -1; txInIdx-- {
			txIn := tx.TxIn[txInIdx]
			stxo := &stxos[stxoIdx]
			stxoIdx--

			n, err := decodeSpentTxOut(serialized[offset:], stxo)
			offset += n
			if err != nil {
				return nil, errDeserialize(fmt.Sprintf("unable "+
					"to decode stxo for %v: %v",
					txIn.PreviousOutPoint, err))
			}
		}
	}

	return stxos, nil
}

// serializeSpendJournalEntry serializes all of the passed spent txouts into a
// single byte slice according to the format described in detail above.
func serializeSpendJournalEntry(stxos []SpentTxOut) []byte {
	if len(stxos) == 0 {
		return nil
	}

	// Calculate the size needed to serialize the entire journal entry.
	var size int
	for i := range stxos {
		size += spentTxOutSerializeSize(&stxos[i])
	}
	serialized := make([]byte, size)

	// Serialize each individual stxo directly into the slice in reverse
	// order one after the other.
	var offset int
	for i := len(stxos) - 1; i > -1; i-- {
		offset += putSpentTxOut(serialized[offset:], &stxos[i])
	}

	return serialized
}

// dbFetchSpendJournalEntry fetches the spend journal entry for the passed block
// and deserializes it into a slice of spent txout entries.
//
// NOTE: Legacy entries will not have the coinbase flag or height set unless it
// was the final output spend in the containing transaction.  It is up to the
// caller to handle this properly by looking the information up in the utxo set.
func dbFetchSpendJournalEntry(dbTx database.Tx, block *czzutil.Block) ([]SpentTxOut, error) {
	// Exclude the coinbase transaction since it can't spend anything.
	spendBucket := dbTx.Metadata().Bucket(spendJournalBucketName)
	serialized := spendBucket.Get(block.Hash()[:])
	blockTxns := block.MsgBlock().Transactions[1:]
	stxos, err := deserializeSpendJournalEntry(serialized, blockTxns)
	if err != nil {
		// Ensure any deserialization errors are returned as database
		// corruption errors.
		if isDeserializeErr(err) {
			return nil, database.Error{
				ErrorCode: database.ErrCorruption,
				Description: fmt.Sprintf("corrupt spend "+
					"information for %v: %v", block.Hash(),
					err),
			}
		}

		return nil, err
	}

	return stxos, nil
}

// dbPutSpendJournalEntry uses an existing database transaction to update the
// spend journal entry for the given block hash using the provided slice of
// spent txouts.   The spent txouts slice must contain an entry for every txout
// the transactions in the block spend in the order they are spent.
func dbPutSpendJournalEntry(dbTx database.Tx, blockHash *chainhash.Hash, stxos []SpentTxOut) error {
	spendBucket := dbTx.Metadata().Bucket(spendJournalBucketName)
	serialized := serializeSpendJournalEntry(stxos)
	return spendBucket.Put(blockHash[:], serialized)
}

// dbRemoveSpendJournalEntry uses an existing database transaction to remove the
// spend journal entry for the passed block hash.
func dbRemoveSpendJournalEntry(dbTx database.Tx, blockHash *chainhash.Hash) error {
	spendBucket := dbTx.Metadata().Bucket(spendJournalBucketName)
	return spendBucket.Delete(blockHash[:])
}

func dbBeaconTx2(dbTx database.Tx, block *czzutil.Block) error {

	pHeight := block.Height() - 1
	pHash := block.MsgBlock().Header.PrevBlock
	eState := dbFetchEntangleState2(dbTx, pHeight, pHash)

	if block.Height() == NetParams.BeaconHeight {
		eState = cross.NewEntangleState2()
	}

	for _, tx := range block.Transactions() {
		var err error
		// BeaconRegistration
		br, _ := cross.IsBeaconRegistrationTx2(tx.MsgTx(), NetParams)
		if br != nil {
			if eState != nil {
				err = eState.RegisterBeaconAddress(br.Address, br.ToAddress, br.StakingAmount, br.Fee, br.KeepTime, br.AssetFlag, br.WhiteList, br.CoinBaseAddress)
			}
			if err != nil {
				return err
			}
		}

		// AddBeaconPledge
		bp, _ := cross.IsAddBeaconPledgeTx(tx.MsgTx(), NetParams)
		if bp != nil {
			if eState != nil {
				err = eState.AppendAmountForBeaconAddress(bp.Address, bp.StakingAmount)
			}
			if err != nil {
				return err
			}
		}
	}

	if eState != nil {
		var err error
		err = dbPutEntangleState2(dbTx, block, eState)
		if err != nil {
			return err
		}

	}
	return nil
}

func dbBeaconTx(dbTx database.Tx, block *czzutil.Block) error {

	pHeight := block.Height() - 1
	pHash := block.MsgBlock().Header.PrevBlock
	eState := dbFetchEntangleState(dbTx, pHeight, pHash)
	if block.Height() == NetParams.ExChangeHeight {
		eState = cross.NewEntangleState()
	}

	reportWhiteList := make([]uint64, 0, 0)
	for _, tx := range block.Transactions() {
		var err error

		// BeaconRegistration
		br, _ := cross.IsBeaconRegistrationTx(tx.MsgTx(), NetParams)
		if br != nil {
			if err := RegisterBeaconTxStore(eState, br, tx); err != nil {
				return err
			}
		}

		// AddBeaconPledge
		bp, _ := cross.IsAddBeaconPledgeTx(tx.MsgTx(), NetParams)
		if bp != nil {
			if err := AddBeaconPledgeTxStore(eState, bp); err != nil {
				return err
			}
		}

		// IsConvertTx
		if cinfo, _ := cross.IsConvertTx(tx.MsgTx()); cinfo != nil {
			objs, err := cross.ToAddressFromConverts(cinfo, g.chain.GetExChangeVerify(), eState, pHeight)
			if err != nil {

			}
		}

	}

	if eState != nil {
		burnTimeout := eState.TourAllUserBurnInfo(uint64(pHeight + 1))
		index := uint32(5)
		for k, _ := range burnTimeout {
			if exInfos := eState.GetBaExInfoByID(k); exInfos != nil {
				exInfos.EnItems = []*wire.OutPoint{&wire.OutPoint{
					Hash:  *block.Transactions()[0].Hash(),
					Index: index,
				}}
				eState.SetBaExInfo(k, exInfos)
			} else {
				return fmt.Errorf("beacon merge failed,exInfo not nil,id: %v", k)
			}
			index = index + 3
		}
		eState.UpdateStateToPunished(burnTimeout)

		for _, v := range reportWhiteList {
			if exInfos := eState.GetBaExInfoByID(v); exInfos != nil {
				exInfos.EnItems = []*wire.OutPoint{&wire.OutPoint{
					Hash:  *block.Transactions()[0].Hash(),
					Index: index,
				}}
				eState.SetBaExInfo(v, exInfos)
			} else {
				return fmt.Errorf("beacon merge failed,exInfo not nil,id: %v", v)
			}
			index = index + 3
		}

		if err := eState.UpdateQuotaOnBlock(uint64(block.Height())); err != nil {
			return err
		}

		if err := dbPutEntangleState(dbTx, block, eState); err != nil {
			return err
		}
	}
	return nil
}

func RegisterBeaconTxStore(state *cross.EntangleState, bai *cross.BeaconAddressInfo, tx *czzutil.Tx) error {

	var err error
	if state != nil {
		err = state.RegisterBeaconAddress(bai.Address, bai.ToAddress, bai.PubKey, bai.StakingAmount, bai.Fee, bai.KeepBlock, bai.AssetFlag, bai.WhiteList, bai.CoinBaseAddress)
	}
	if err != nil {
		return err
	}
	beaconID := state.GetBeaconIdByTo(bai.ToAddress)
	if exInfos := state.GetBaExInfoByID(beaconID); exInfos != nil {
		ex := cross.NewExBeaconInfo()
		ex.EnItems = []*wire.OutPoint{&wire.OutPoint{
			Hash:  *tx.Hash(),
			Index: 1,
		}}

		state.SetBaExInfo(beaconID, ex)
	} else {
		return errors.New(fmt.Sprintf("beacon merge failed,exInfo not nil,id:%v", beaconID))
	}
	return nil
}

func AddBeaconPledgeTxStore(state *cross.EntangleState, abp *cross.AddBeaconPledge) error {
	var err error
	if state != nil {
		err = state.AppendAmountForBeaconAddress(abp.Address, abp.StakingAmount)
	}
	if err != nil {
		return err
	}
	return nil
}

//load
func dbFetchEntangleState(dbTx database.Tx, height int32, hash chainhash.Hash) *cross.EntangleState {

	es := cross.NewEntangleState()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, height)
	buf.Write(hash.CloneBytes())
	var err error
	entangleBucket := dbTx.Metadata().Bucket(cross.EntangleStateKey)
	if entangleBucket == nil {
		if entangleBucket, err = dbTx.Metadata().CreateBucketIfNotExists(cross.EntangleStateKey); err != nil {
			return nil
		}
	}
	value := entangleBucket.Get(buf.Bytes())
	if value != nil {
		err := rlp.DecodeBytes(value, es)
		if err != nil {
			return nil
		}
		return es
	}
	return nil
}

//load
func dbFetchEntangleState2(dbTx database.Tx, height int32, hash chainhash.Hash) *cross.EntangleState2 {

	es := cross.NewEntangleState2()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, height)
	buf.Write(hash.CloneBytes())
	var err error
	entangleBucket := dbTx.Metadata().Bucket(cross.EntangleStateKey)
	if entangleBucket == nil {
		if entangleBucket, err = dbTx.Metadata().CreateBucketIfNotExists(cross.EntangleStateKey); err != nil {
			return nil
		}
	}
	value := entangleBucket.Get(buf.Bytes())
	if value != nil {
		err := rlp.DecodeBytes(value, es)
		if err != nil {
			return nil
		}
		return es
	}
	return nil
}

func dbPutEntangleState(dbTx database.Tx, block *czzutil.Block, eState *cross.EntangleState) error {
	var err error
	entangleBucket := dbTx.Metadata().Bucket(cross.EntangleStateKey)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, block.Height())
	buf.Write(block.Hash().CloneBytes())
	err = entangleBucket.Put(buf.Bytes(), eState.ToBytes())
	return err
}

func dbPutEntangleState2(dbTx database.Tx, block *czzutil.Block, eState *cross.EntangleState2) error {
	var err error
	entangleBucket := dbTx.Metadata().Bucket(cross.EntangleStateKey)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, block.Height())
	buf.Write(block.Hash().CloneBytes())
	err = entangleBucket.Put(buf.Bytes(), eState.ToBytes())
	return err
}

func dbPutBurnTxInfoEntry(dbTx database.Tx, info *cross.BurnTxInfo) error {
	BurnTxInfoBucket := dbTx.Metadata().Bucket(cross.BurnTxInfoKey)
	buf, err := hex.DecodeString(info.Address)
	if err != nil {
		return err
	}
	err = BurnTxInfoBucket.Put(buf, info.ToBytes())
	return err
}

func dbRemoveBurnTxInfoEntry(dbTx database.Tx, info *cross.BurnProofInfo) error {
	BurnTxInfoBucket := dbTx.Metadata().Bucket(cross.BurnTxInfoKey)
	buf, err := hex.DecodeString(info.Address)
	if err != nil {
		return err
	}
	err = BurnTxInfoBucket.Delete(buf)
	return err
}

// -----------------------------------------------------------------------------
// The unspent transaction output (utxo) set consists of an entry for each
// unspent output using a format that is optimized to reduce space using domain
// specific compression algorithms.  This format is a slightly modified version
// of the format used in Bitcoin Core.
//
// Each entry is keyed by an outpoint as specified below.  It is important to
// note that the key encoding uses a VLQ, which employs an MSB encoding so
// iteration of utxos when doing byte-wise comparisons will produce them in
// order.
//
// The serialized key format is:
//   <hash><output index>
//
//   Field                Type             Size
//   hash                 chainhash.Hash   chainhash.HashSize
//   output index         VLQ              variable
//
// The serialized value format is:
//
//   <header code><compressed txout>
//
//   Field                Type     Size
//   header code          VLQ      variable
//   compressed txout
//     compressed amount  VLQ      variable
//     compressed script  []byte   variable
//
// The serialized header code format is:
//   bit 0 - containing transaction is a coinbase
//   bits 1-x - height of the block that contains the unspent txout
//
// Example 1:
// From tx in main blockchain:
// Blk 1, 0e3e2357e806b6cdb1f70b54c3a3a17b6714ee1f0e68bebb44a74b1efd512098:0
//
//    03320496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52
//    <><------------------------------------------------------------------>
//     |                                          |
//   header code                         compressed txout
//
//  - header code: 0x03 (coinbase, height 1)
//  - compressed txout:
//    - 0x32: VLQ-encoded compressed amount for 5000000000 (50 CZZ)
//    - 0x04: special script type pay-to-pubkey
//    - 0x96...52: x-coordinate of the pubkey
//
// Example 2:
// From tx in main blockchain:
// Blk 113931, 4a16969aa4764dd7507fc1de7f0baa4850a246de90c45e59a3207f9a26b5036f:2
//
//    8cf316800900b8025be1b3efc63b0ad48e7f9f10e87544528d58
//    <----><------------------------------------------>
//      |                             |
//   header code             compressed txout
//
//  - header code: 0x8cf316 (not coinbase, height 113931)
//  - compressed txout:
//    - 0x8009: VLQ-encoded compressed amount for 15000000 (0.15 CZZ)
//    - 0x00: special script type pay-to-pubkey-hash
//    - 0xb8...58: pubkey hash
//
// Example 3:
// From tx in main blockchain:
// Blk 338156, 1b02d1c8cfef60a189017b9a420c682cf4a0028175f2f563209e4ff61c8c3620:22
//
//    a8a2588ba5b9e763011dd46a006572d820e448e12d2bbb38640bc718e6
//    <----><-------------------------------------------------->
//      |                             |
//   header code             compressed txout
//
//  - header code: 0xa8a258 (not coinbase, height 338156)
//  - compressed txout:
//    - 0x8ba5b9e763: VLQ-encoded compressed amount for 366875659 (3.66875659 CZZ)
//    - 0x01: special script type pay-to-script-hash
//    - 0x1d...e6: script hash
// -----------------------------------------------------------------------------

// maxUint32VLQSerializeSize is the maximum number of bytes a max uint32 takes
// to serialize as a VLQ.
var maxUint32VLQSerializeSize = serializeSizeVLQ(1<<32 - 1)

// outpointKeyPool defines a concurrent safe free list of byte slices used to
// provide temporary buffers for outpoint database keys.
var outpointKeyPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, chainhash.HashSize+maxUint32VLQSerializeSize)
		return &b // Pointer to slice to avoid boxing alloc.
	},
}

// outpointKey returns a key suitable for use as a database key in the utxo set
// while making use of a free list.  A new buffer is allocated if there are not
// already any available on the free list.  The returned byte slice should be
// returned to the free list by using the recycleOutpointKey function when the
// caller is done with it _unless_ the slice will need to live for longer than
// the caller can calculate such as when used to write to the database.
func outpointKey(outpoint wire.OutPoint) *[]byte {
	// A VLQ employs an MSB encoding, so they are useful not only to reduce
	// the amount of storage space, but also so iteration of utxos when
	// doing byte-wise comparisons will produce them in order.
	key := outpointKeyPool.Get().(*[]byte)
	idx := uint64(outpoint.Index)
	*key = (*key)[:chainhash.HashSize+serializeSizeVLQ(idx)]
	copy(*key, outpoint.Hash[:])
	putVLQ((*key)[chainhash.HashSize:], idx)
	return key
}

// DeserializeOutpointKey takes in a Utxo database key and deserializes
// it into an outpoint.
func DeserializeOutpointKey(key []byte) *wire.OutPoint {
	h, _ := chainhash.NewHash(key[:chainhash.HashSize])
	idx, _ := deserializeVLQ(key[chainhash.HashSize:])
	return wire.NewOutPoint(h, uint32(idx))
}

// recycleOutpointKey puts the provided byte slice, which should have been
// obtained via the outpointKey function, back on the free list.
func recycleOutpointKey(key *[]byte) {
	outpointKeyPool.Put(key)
}

// utxoEntryHeaderCode returns the calculated header code to be used when
// serializing the provided utxo entry.
func utxoEntryHeaderCode(entry *UtxoEntry) (uint64, error) {
	if entry.IsSpent() {
		return 0, AssertError("attempt to serialize spent utxo header")
	}

	// As described in the serialization format comments, the header code
	// encodes the height shifted over one bit and the coinbase flag in the
	// lowest bit.
	headerCode := uint64(entry.BlockHeight()) << 1
	if entry.IsCoinBase() {
		headerCode |= 0x01
	}

	return headerCode, nil
}

// serializeUtxoEntry returns the entry serialized to a format that is suitable
// for long-term storage.  The format is described in detail above.
func serializeUtxoEntry(entry *UtxoEntry) ([]byte, error) {
	// Spent outputs have no serialization.
	if entry.IsSpent() {
		return nil, nil
	}

	// Encode the header code.
	headerCode, err := utxoEntryHeaderCode(entry)
	if err != nil {
		return nil, err
	}

	// Calculate the size needed to serialize the entry.
	size := serializeSizeVLQ(headerCode) +
		compressedTxOutSize(uint64(entry.Amount()), entry.PkScript())

	// Serialize the header code followed by the compressed unspent
	// transaction output.
	serialized := make([]byte, size)
	offset := putVLQ(serialized, headerCode)
	putCompressedTxOut(serialized[offset:], uint64(entry.Amount()),
		entry.PkScript())

	return serialized, nil
}

// DeserializeUtxoEntry decodes a utxo entry from the passed serialized byte
// slice into a new UtxoEntry using a format that is suitable for long-term
// storage.  The format is described in detail above.
func DeserializeUtxoEntry(serialized []byte) (*UtxoEntry, error) {
	// Deserialize the header code.
	code, offset := deserializeVLQ(serialized)
	if offset >= len(serialized) {
		return nil, errDeserialize("unexpected end of data after header")
	}

	// Decode the header code.
	//
	// Bit 0 indicates whether the containing transaction is a coinbase.
	// Bits 1-x encode height of containing transaction.
	isCoinBase := code&0x01 != 0
	blockHeight := int32(code >> 1)

	// Decode the compressed unspent transaction output.
	amount, pkScript, _, err := decodeCompressedTxOut(serialized[offset:])
	if err != nil {
		return nil, errDeserialize(fmt.Sprintf("unable to decode "+
			"utxo: %v", err))
	}

	entry := &UtxoEntry{
		amount:      int64(amount),
		pkScript:    pkScript,
		blockHeight: blockHeight,
		packedFlags: 0,
	}
	if isCoinBase {
		entry.packedFlags |= tfCoinBase
	}

	return entry, nil
}

// deserializeUtxoCommitmentFormat takes a Utxo serialized in the commitment format and
// deserializes it into an OutPoint and UtxoEntry.
func deserializeUtxoCommitmentFormat(serialized []byte) (*wire.OutPoint, *UtxoEntry, error) {
	txid, err := chainhash.NewHash(serialized[:32])
	if err != nil {
		return nil, nil, err
	}
	index := binary.LittleEndian.Uint32(serialized[32:36])

	serializedHeight := serialized[36:40]
	coinbaseFlag := serializedHeight[3] & 0x01

	serializedHeight[3] &= 0xFE

	height := binary.LittleEndian.Uint32(serializedHeight)

	value := binary.LittleEndian.Uint64(serialized[40:48])

	pkScript := serialized[52:]

	op := wire.NewOutPoint(txid, index)
	entry := &UtxoEntry{
		amount:      int64(value),
		blockHeight: int32(height),
		pkScript:    pkScript,
		packedFlags: 0,
	}
	if coinbaseFlag > 0 {
		entry.packedFlags |= tfCoinBase
	}
	return op, entry, nil
}

// dbFetchUtxoEntryByHash attempts to find and fetch a utxo for the given hash.
// It uses a cursor and seek to try and do this as efficiently as possible.
//
// When there are no entries for the provided hash, nil will be returned for the
// both the entry and the error.
func dbFetchUtxoEntryByHash(dbTx database.Tx, hash *chainhash.Hash) (*UtxoEntry, error) {
	// Attempt to find an entry by seeking for the hash along with a zero
	// index.  Due to the fact the keys are serialized as <hash><index>,
	// where the index uses an MSB encoding, if there are any entries for
	// the hash at all, one will be found.
	cursor := dbTx.Metadata().Bucket(utxoSetBucketName).Cursor()
	key := outpointKey(wire.OutPoint{Hash: *hash, Index: 0})
	ok := cursor.Seek(*key)
	recycleOutpointKey(key)
	if !ok {
		return nil, nil
	}

	// An entry was found, but it could just be an entry with the next
	// highest hash after the requested one, so make sure the hashes
	// actually match.
	cursorKey := cursor.Key()
	if len(cursorKey) < chainhash.HashSize {
		return nil, nil
	}
	if !bytes.Equal(hash[:], cursorKey[:chainhash.HashSize]) {
		return nil, nil
	}

	return DeserializeUtxoEntry(cursor.Value())
}

// dbFetchUtxoEntry uses an existing database transaction to fetch the specified
// transaction output from the utxo set.
//
// When there is no entry for the provided output, nil will be returned for both
// the entry and the error.
func dbFetchUtxoEntry(dbTx database.Tx, outpoint wire.OutPoint) (*UtxoEntry, error) {
	// Fetch the unspent transaction output information for the passed
	// transaction output.  Return now when there is no entry.
	key := outpointKey(outpoint)
	utxoBucket := dbTx.Metadata().Bucket(utxoSetBucketName)
	serializedUtxo := utxoBucket.Get(*key)
	recycleOutpointKey(key)
	if serializedUtxo == nil {
		return nil, nil
	}

	// A non-nil zero-length entry means there is an entry in the database
	// for a spent transaction output which should never be the case.
	if len(serializedUtxo) == 0 {
		return nil, AssertError(fmt.Sprintf("database contains entry "+
			"for spent tx output %v", outpoint))
	}

	// Deserialize the utxo entry and return it.
	entry, err := DeserializeUtxoEntry(serializedUtxo)
	if err != nil {
		// Ensure any deserialization errors are returned as database
		// corruption errors.
		if isDeserializeErr(err) {
			return nil, database.Error{
				ErrorCode: database.ErrCorruption,
				Description: fmt.Sprintf("corrupt utxo entry "+
					"for %v: %v", outpoint, err),
			}
		}

		return nil, err
	}

	return entry, nil
}

// dbPutUtxoEntries uses an existing database transaction to update the utxo
// entries in the database.
func dbPutUtxoEntries(dbTx database.Tx, entries map[wire.OutPoint]*UtxoEntry) error {
	utxoBucket := dbTx.Metadata().Bucket(utxoSetBucketName)
	for outpoint, entry := range entries {
		if entry == nil || entry.IsSpent() {
			return AssertError("trying to store nil or spent entry")
		}

		// Serialize and store the utxo entry.
		serialized, err := serializeUtxoEntry(entry)
		if err != nil {
			return err
		}
		// NOTE: The key is intentionally not recycled here since the
		// database interface contract prohibits modifications.  It will
		// be garbage collected normally when the database is done with
		// it.
		key := outpointKey(outpoint)
		if err := utxoBucket.Put(*key, serialized); err != nil {
			return err
		}
	}
	return nil
}

// dbDeleteUtxoEntries uses an existing database transaction to delete the utxo
// entries from the database.
func dbDeleteUtxoEntries(dbTx database.Tx, outpoints []wire.OutPoint) error {
	utxoBucket := dbTx.Metadata().Bucket(utxoSetBucketName)
	for _, outpoint := range outpoints {
		key := outpointKey(outpoint)
		err := utxoBucket.Delete(*key)
		recycleOutpointKey(key)
		if err != nil {
			return err
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// The block index consists of two buckets with an entry for every block in the
// main chain.  One bucket is for the hash to height mapping and the other is
// for the height to hash mapping.
//
// The serialized format for values in the hash to height bucket is:
//   <height>
//
//   Field      Type     Size
//   height     uint32   4 bytes
//
// The serialized format for values in the height to hash bucket is:
//   <hash>
//
//   Field      Type             Size
//   hash       chainhash.Hash   chainhash.HashSize
// -----------------------------------------------------------------------------

// dbPutBlockIndex uses an existing database transaction to update or add the
// block index entries for the hash to height and height to hash mappings for
// the provided values.
func dbPutBlockIndex(dbTx database.Tx, hash *chainhash.Hash, height int32) error {
	// Serialize the height for use in the index entries.
	var serializedHeight [4]byte
	byteOrder.PutUint32(serializedHeight[:], uint32(height))

	// Add the block hash to height mapping to the index.
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(hashIndexBucketName)
	if err := hashIndex.Put(hash[:], serializedHeight[:]); err != nil {
		return err
	}

	// Add the block height to hash mapping to the index.
	heightIndex := meta.Bucket(heightIndexBucketName)
	return heightIndex.Put(serializedHeight[:], hash[:])
}

// dbRemoveBlockIndex uses an existing database transaction remove block index
// entries from the hash to height and height to hash mappings for the provided
// values.
func dbRemoveBlockIndex(dbTx database.Tx, hash *chainhash.Hash, height int32) error {
	// Remove the block hash to height mapping.
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(hashIndexBucketName)
	if err := hashIndex.Delete(hash[:]); err != nil {
		return err
	}

	// Remove the block height to hash mapping.
	var serializedHeight [4]byte
	byteOrder.PutUint32(serializedHeight[:], uint32(height))
	heightIndex := meta.Bucket(heightIndexBucketName)
	return heightIndex.Delete(serializedHeight[:])
}

// dbFetchHeightByHash uses an existing database transaction to retrieve the
// height for the provided hash from the index.
func dbFetchHeightByHash(dbTx database.Tx, hash *chainhash.Hash) (int32, error) {
	meta := dbTx.Metadata()
	hashIndex := meta.Bucket(hashIndexBucketName)
	serializedHeight := hashIndex.Get(hash[:])
	if serializedHeight == nil {
		str := fmt.Sprintf("block %s is not in the main chain", hash)
		return 0, errNotInMainChain(str)
	}

	return int32(byteOrder.Uint32(serializedHeight)), nil
}

// dbFetchHashByHeight uses an existing database transaction to retrieve the
// hash for the provided height from the index.
func dbFetchHashByHeight(dbTx database.Tx, height int32) (*chainhash.Hash, error) {
	var serializedHeight [4]byte
	byteOrder.PutUint32(serializedHeight[:], uint32(height))

	meta := dbTx.Metadata()
	heightIndex := meta.Bucket(heightIndexBucketName)
	hashBytes := heightIndex.Get(serializedHeight[:])
	if hashBytes == nil {
		str := fmt.Sprintf("no block at height %d exists", height)
		return nil, errNotInMainChain(str)
	}

	var hash chainhash.Hash
	copy(hash[:], hashBytes)
	return &hash, nil
}

// -----------------------------------------------------------------------------
// The best chain state consists of the best block hash and height, the total
// number of transactions up to and including those in the best block, and the
// accumulated work sum up to and including the best block.
//
// The serialized format is:
//
//   <block hash><block height><total txns><work sum length><work sum>
//
//   Field             Type             Size
//   block hash        chainhash.Hash   chainhash.HashSize
//   block height      uint32           4 bytes
//   total txns        uint64           8 bytes
//   work sum length   uint32           4 bytes
//   work sum          big.Int          work sum length
// -----------------------------------------------------------------------------

// bestChainState represents the data to be stored the database for the current
// best chain state.
type bestChainState struct {
	hash      chainhash.Hash
	height    uint32
	totalTxns uint64
	workSum   *big.Int
}

// serializeBestChainState returns the serialization of the passed block best
// chain state.  This is data to be stored in the chain state bucket.
func serializeBestChainState(state bestChainState) []byte {
	// Calculate the full size needed to serialize the chain state.
	workSumBytes := state.workSum.Bytes()
	workSumBytesLen := uint32(len(workSumBytes))
	serializedLen := chainhash.HashSize + 4 + 8 + 4 + workSumBytesLen

	// Serialize the chain state.
	serializedData := make([]byte, serializedLen)
	copy(serializedData[0:chainhash.HashSize], state.hash[:])
	offset := uint32(chainhash.HashSize)
	byteOrder.PutUint32(serializedData[offset:], state.height)
	offset += 4
	byteOrder.PutUint64(serializedData[offset:], state.totalTxns)
	offset += 8
	byteOrder.PutUint32(serializedData[offset:], workSumBytesLen)
	offset += 4
	copy(serializedData[offset:], workSumBytes)
	return serializedData[:]
}

// deserializeBestChainState deserializes the passed serialized best chain
// state.  This is data stored in the chain state bucket and is updated after
// every block is connected or disconnected form the main chain.
// block.
func deserializeBestChainState(serializedData []byte) (bestChainState, error) {
	// Ensure the serialized data has enough bytes to properly deserialize
	// the hash, height, total transactions, and work sum length.
	if len(serializedData) < chainhash.HashSize+16 {
		return bestChainState{}, database.Error{
			ErrorCode:   database.ErrCorruption,
			Description: "corrupt best chain state",
		}
	}

	state := bestChainState{}
	copy(state.hash[:], serializedData[0:chainhash.HashSize])
	offset := uint32(chainhash.HashSize)
	state.height = byteOrder.Uint32(serializedData[offset : offset+4])
	offset += 4
	state.totalTxns = byteOrder.Uint64(serializedData[offset : offset+8])
	offset += 8
	workSumBytesLen := byteOrder.Uint32(serializedData[offset : offset+4])
	offset += 4

	// Ensure the serialized data has enough bytes to deserialize the work
	// sum.
	if uint32(len(serializedData[offset:])) < workSumBytesLen {
		return bestChainState{}, database.Error{
			ErrorCode:   database.ErrCorruption,
			Description: "corrupt best chain state",
		}
	}
	workSumBytes := serializedData[offset : offset+workSumBytesLen]
	state.workSum = new(big.Int).SetBytes(workSumBytes)

	return state, nil
}

// dbPutBestState uses an existing database transaction to update the best chain
// state with the given parameters.
func dbPutBestState(dbTx database.Tx, snapshot *BestState, workSum *big.Int) error {
	// Serialize the current best chain state.
	serializedData := serializeBestChainState(bestChainState{
		hash:      snapshot.Hash,
		height:    uint32(snapshot.Height),
		totalTxns: snapshot.TotalTxns,
		workSum:   workSum,
	})

	// Store the current best chain state into the database.
	return dbTx.Metadata().Put(chainStateKeyName, serializedData)
}

// -----------------------------------------------------------------------------
// The utxo state consistency status is stored as the last hash at which the
// state finished a flush and an indicator whether it was left in the middle
// of a flush.
//
// The serialized format is:
//
//   <status code><consistent hash>
//
//   Field             Type             Size
//   status code       byte             1
//   consistent hash   chainhash.Hash   chainhash.HashSize
//
// The possible values for the state code are:
//
//   1: consistent with the given hash, no flush ongoing
//   2: flush ongoing from the stored consistent hash to best state (see best
//      state bucket)
// -----------------------------------------------------------------------------
const (
	// UTXO consistency status (UCS) codes are used to indicate the
	// consistency status of the utxo state in the database.
	// ucsEmpty is used as a return value to indicate that no status was
	// stored.  The zero value should not be stored in the database.
	ucsEmpty        byte = 0
	ucsConsistent        = 1
	ucsFlushOngoing      = 2
	// ucsNbCodes is the number of valid utxo consistency status codes.
	ucsNbCodes = 3
)

// serializeUtxoStateConsistency serializes the utxo state consistency status
// based on the given status code and hash.
func serializeUtxoStateConsistency(code byte, hash *chainhash.Hash) []byte {
	// Serialize the data as code + hash.
	serialized := make([]byte, 1+chainhash.HashSize)
	serialized[0] = code
	copy(serialized[1:], hash[:])
	return serialized
}

// deserializeUtxoStateConsistency deserializes the bytes representing the utxo
// state consistency status into the status code and hash.
func deserializeUtxoStateConsistency(serialized []byte) (byte, *chainhash.Hash, error) {
	// Size must be status code byte + single hash.
	if len(serialized) != chainhash.HashSize+1 {
		return 0, nil, database.Error{
			ErrorCode:   database.ErrCorruption,
			Description: "corrupt utxo state consistency status",
		}
	}
	code := serialized[0]
	if code >= ucsNbCodes {
		return 0, nil, database.Error{
			ErrorCode:   database.ErrCorruption,
			Description: "corrupt utxo state consistency status: unknown code",
		}
	}
	// Ignore error as this only fails on incorrect hash size.
	hash, _ := chainhash.NewHash(serialized[1 : 1+chainhash.HashSize])
	return code, hash, nil
}

// dbPutUtxoStateConsistency uses an existing database transaction to
// update the utxo state consistency status with the given parameters.
func dbPutUtxoStateConsistency(dbTx database.Tx, code byte, hash *chainhash.Hash) error {
	// Seralize the utxo state consistency status.
	serialized := serializeUtxoStateConsistency(code, hash)
	// Store the utxo state consistency status into the database.
	return dbTx.Metadata().Put(utxoStateConsistencyKeyName, serialized)
}

// dbFetchUtxoStateConsistency uses an existing database transaction to retrieve
// the utxo state consistency status from the database.  The code is 0 when
// nothing was found.
func dbFetchUtxoStateConsistency(dbTx database.Tx) (byte, *chainhash.Hash, error) {
	// Fetch the serialized data from the database.
	serialized := dbTx.Metadata().Get(utxoStateConsistencyKeyName)
	if serialized == nil {
		// Not stored, so return empty state.
		return ucsEmpty, nil, nil
	}
	// Deserialize to the consistency status.
	return deserializeUtxoStateConsistency(serialized)
}

// dbPutBlockchainType uses an existing database transaction to
// update the blockchain type entry with the provided type.
func dbPutBlockchainType(dbTx database.Tx, chainType []byte) error {
	// Store the chain type into the database.
	return dbTx.Metadata().Put(blockchainTypeKeyName, chainType)
}

// dbFetchBlockchainType uses an existing database transaction to retrieve
// the blockchain type from the database.  The returned bytes is nil if
// nothing is found.
func dbFetchBlockchainType(dbTx database.Tx) []byte {
	// Fetch the chain type from the database.
	return dbTx.Metadata().Get(blockchainTypeKeyName)
}

// dbPutPruneHeight uses an existing database transaction to
// update the prune height.
func dbPutPruneHeight(dbTx database.Tx, height uint32) error {
	buf := make([]byte, 4)
	byteOrder.PutUint32(buf, height)
	// Store the height into the database.
	return dbTx.Metadata().Put(pruneHeightKeyName, buf)
}

// dbFetchPruneHeight uses an existing database transaction to retrieve
// the prune height from the database. If the height comes back nil we
// return zero.
func dbFetchPruneHeight(dbTx database.Tx) uint32 {
	// Fetch the serialized data from the database.
	serialized := dbTx.Metadata().Get(pruneHeightKeyName)
	if serialized == nil {
		return 0
	}
	// Deserialize to the height.
	return byteOrder.Uint32(serialized)
}

// createChainState initializes both the database and the chain state to the
// genesis block.  This includes creating the necessary buckets and inserting
// the genesis block, so it must only be called on an uninitialized database.
func (b *BlockChain) createChainState() error {
	// Create a new node from the genesis block and set it as the best node.
	genesisBlock := czzutil.NewBlock(b.chainParams.GenesisBlock)
	genesisBlock.SetHeight(0)
	header := &genesisBlock.MsgBlock().Header
	node := newBlockNode(header, nil)
	node.status = statusDataStored | statusValid
	b.bestChain.SetTip(node)

	// Add the new node to the index which is used for faster lookups.
	b.index.addNode(node)

	// Initialize the state related to the best block.  Since it is the
	// genesis block, use its timestamp for the median time.
	numTxns := uint64(len(genesisBlock.MsgBlock().Transactions))
	blockSize := uint64(genesisBlock.MsgBlock().SerializeSize())
	b.stateSnapshot = newBestState(node, blockSize, numTxns,
		numTxns, time.Unix(node.timestamp, 0))

	// Create the initial the database chain state including creating the
	// necessary index buckets and inserting the genesis block.
	err := b.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()

		// Create the bucket that houses the block index data.
		_, err := meta.CreateBucket(blockIndexBucketName)
		if err != nil {
			return err
		}

		// Create the bucket that houses the chain block hash to height
		// index.
		_, err = meta.CreateBucket(hashIndexBucketName)
		if err != nil {
			return err
		}

		// Create the bucket that houses the chain block height to hash
		// index.
		_, err = meta.CreateBucket(heightIndexBucketName)
		if err != nil {
			return err
		}

		// Create the bucket that houses the spend journal data and
		// store its version.
		_, err = meta.CreateBucket(spendJournalBucketName)
		if err != nil {
			return err
		}

		err = dbPutVersion(dbTx, utxoSetVersionKeyName,
			latestUtxoSetBucketVersion)
		if err != nil {
			return err
		}

		// Create the bucket that houses the utxo set and store its
		// version.  Note that the genesis block coinbase transaction is
		// intentionally not inserted here since it is not spendable by
		// consensus rules.
		_, err = meta.CreateBucket(utxoSetBucketName)
		if err != nil {
			return err
		}
		err = dbPutVersion(dbTx, spendJournalVersionKeyName,
			latestSpendJournalBucketVersion)
		if err != nil {
			return err
		}

		// Save the genesis block to the block index database.
		err = dbStoreBlockNode(dbTx, node)
		if err != nil {
			return err
		}

		// Add the genesis block hash to height and height to hash
		// mappings to the index.
		err = dbPutBlockIndex(dbTx, &node.hash, node.height)
		if err != nil {
			return err
		}

		// Store the current best chain state into the database.
		err = dbPutBestState(dbTx, b.stateSnapshot, node.workSum)
		if err != nil {
			return err
		}

		// Store the genesis block into the database.
		return dbStoreBlock(dbTx, genesisBlock)
	})
	return err
}

// initChainState attempts to load and initialize the chain state from the
// database.  When the db does not yet contain any chain state, both it and the
// chain state are initialized to the genesis block.
func (b *BlockChain) initChainState(fastSync bool) error {
	// Determine the state of the chain database. We may need to initialize
	// everything from scratch or upgrade certain buckets.
	var initialized, hasBlockIndex bool
	err := b.db.View(func(dbTx database.Tx) error {
		initialized = dbTx.Metadata().Get(chainStateKeyName) != nil
		hasBlockIndex = dbTx.Metadata().Bucket(blockIndexBucketName) != nil
		return nil
	})
	if err != nil {
		return err
	}

	if !initialized {
		// At this point the database has not already been initialized, so
		// initialize both it and the chain state to the genesis block.
		return b.createChainState()
	}

	if !hasBlockIndex {
		err := migrateBlockIndex(b.db)
		if err != nil {
			return nil
		}
	}

	// Attempt to load the chain state from the database.
	err = b.db.View(func(dbTx database.Tx) error {
		// Fetch the stored chain state from the database metadata.
		// When it doesn't exist, it means the database hasn't been
		// initialized for use with chain yet, so break out now to allow
		// that to happen under a writable database transaction.
		serializedData := dbTx.Metadata().Get(chainStateKeyName)
		log.Tracef("Serialized chain state: %x", serializedData)
		state, err := deserializeBestChainState(serializedData)
		if err != nil {
			return err
		}

		// Load all of the headers from the data for the known best
		// chain and construct the block index accordingly.  Since the
		// number of nodes are already known, perform a single alloc
		// for them versus a whole bunch of little ones to reduce
		// pressure on the GC.
		log.Infof("Loading block index...")

		blockIndexBucket := dbTx.Metadata().Bucket(blockIndexBucketName)

		// Determine how many blocks will be loaded into the index so we can
		// allocate the right amount.
		var blockCount int32
		cursor := blockIndexBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			blockCount++
		}
		blockNodes := make([]blockNode, blockCount)

		var i int32
		var lastNode *blockNode
		cursor = blockIndexBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			header, status, err := deserializeBlockRow(cursor.Value())
			if err != nil {
				return err
			}

			// Determine the parent block node. Since we iterate block headers
			// in order of height, if the blocks are mostly linear there is a
			// very good chance the previous header processed is the parent.
			var parent *blockNode

			if lastNode == nil {
				blockHash := header.BlockHash()
				if !blockHash.IsEqual(b.chainParams.GenesisHash) {
					return AssertError(fmt.Sprintf("initChainState: Expected "+
						"first entry in block index to be genesis block, "+
						"found %s", blockHash))
				}
			} else if header.PrevBlock == lastNode.hash {
				// Since we iterate block headers in order of height, if the
				// blocks are mostly linear there is a very good chance the
				// previous header processed is the parent.
				parent = lastNode
			} else {
				parent = b.index.LookupNode(&header.PrevBlock)
				if parent == nil {
					return AssertError(fmt.Sprintf("initChainState: Could "+
						"not find parent for block %s %s", header.BlockHash(), &header.PrevBlock))
				}
			}

			// Initialize the block node for the block, connect it,
			// and add it to the block index.
			node := &blockNodes[i]
			initBlockNode(node, header, parent)
			node.status = status
			b.index.addNode(node)
			lastNode = node
			i++
		}
		log.Debug("Done loading block index")

		// Set the best chain view to the stored best state.
		tip := b.index.LookupNode(&state.hash)
		if tip == nil {
			return AssertError(fmt.Sprintf("initChainState: cannot find "+
				"chain tip %s in block index", state.hash))
		}
		b.bestChain.SetTip(tip)

		// Load the raw block bytes for the best block.
		var block wire.MsgBlock
		var blockBytes []byte
		lastCheckpoint := b.LatestCheckpoint()
		if !fastSync || (lastCheckpoint != nil && tip.height > lastCheckpoint.Height) {
			blockBytes, err = dbTx.FetchBlock(&state.hash)
			if err != nil {
				return err
			}
			err = block.Deserialize(bytes.NewReader(blockBytes))
			if err != nil {
				return err
			}
		}

		// As a final consistency check, we'll run through all the
		// nodes which are ancestors of the current chain tip, and mark
		// them as valid if they aren't already marked as such.  This
		// is a safe assumption as all the block before the current tip
		// are valid by definition.
		for iterNode := tip; iterNode != nil; iterNode = iterNode.parent {
			// If this isn't already marked as valid in the index, then
			// we'll mark it as valid now to ensure consistency once
			// we're up and running.
			if !iterNode.status.KnownValid() {
				log.Infof("Block %v (height=%v) ancestor of "+
					"chain tip not marked as valid, "+
					"upgrading to valid for consistency",
					iterNode.hash, iterNode.height)

				b.index.SetStatusFlags(iterNode, statusValid)
			}
		}

		// Initialize the state related to the best block.
		blockSize := uint64(len(blockBytes))
		numTxns := uint64(len(block.Transactions))
		b.stateSnapshot = newBestState(tip, blockSize, numTxns,
			state.totalTxns, tip.CalcPastMedianTime())

		return nil
	})
	if err != nil {
		return err
	}

	// As we might have updated the index after it was loaded, we'll
	// attempt to flush the index to the DB. This will only result in a
	// write if the elements are dirty, so it'll usually be a noop.
	return b.index.flushToDB()
}

// deserializeBlockRow parses a value in the block index bucket into a block
// header and block status bitfield.
func deserializeBlockRow(blockRow []byte) (*wire.BlockHeader, blockStatus, error) {
	buffer := bytes.NewReader(blockRow)

	var header wire.BlockHeader
	err := header.Deserialize(buffer)
	if err != nil {
		return nil, statusNone, err
	}

	statusByte, err := buffer.ReadByte()
	if err != nil {
		return nil, statusNone, err
	}

	return &header, blockStatus(statusByte), nil
}

// dbFetchHeaderByHash uses an existing database transaction to retrieve the
// block header for the provided hash.
func dbFetchHeaderByHash(dbTx database.Tx, hash *chainhash.Hash) (*wire.BlockHeader, error) {
	headerBytes, err := dbTx.FetchBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	var header wire.BlockHeader
	err = header.Deserialize(bytes.NewReader(headerBytes))
	if err != nil {
		return nil, err
	}

	return &header, nil
}

// dbFetchHeaderByHeight uses an existing database transaction to retrieve the
// block header for the provided height.
func dbFetchHeaderByHeight(dbTx database.Tx, height int32) (*wire.BlockHeader, error) {
	hash, err := dbFetchHashByHeight(dbTx, height)
	if err != nil {
		return nil, err
	}

	return dbFetchHeaderByHash(dbTx, hash)
}

// dbFetchBlockByNode uses an existing database transaction to retrieve the
// raw block for the provided node, deserialize it, and return a czzutil.Block
// with the height set.
func dbFetchBlockByNode(dbTx database.Tx, node *blockNode) (*czzutil.Block, error) {
	// Load the raw block bytes from the database.
	blockBytes, err := dbTx.FetchBlock(&node.hash)
	if err != nil {
		return nil, err
	}

	// Create the encapsulated block and set the height appropriately.
	block, err := czzutil.NewBlockFromBytes(blockBytes)
	if err != nil {
		return nil, err
	}
	block.SetHeight(node.height)

	return block, nil
}

// dbStoreBlockNode stores the block header and validation status to the block
// index bucket. This overwrites the current entry if there exists one.
func dbStoreBlockNode(dbTx database.Tx, node *blockNode) error {
	// Serialize block data to be stored.
	w := bytes.NewBuffer(make([]byte, 0, blockHdrSize+1))
	header := node.Header()
	err := header.Serialize(w)
	if err != nil {
		return err
	}
	err = w.WriteByte(byte(node.status))
	if err != nil {
		return err
	}
	value := w.Bytes()

	// Write block header data to block index bucket.
	blockIndexBucket := dbTx.Metadata().Bucket(blockIndexBucketName)
	key := blockIndexKey(&node.hash, uint32(node.height))
	return blockIndexBucket.Put(key, value)
}

// dbStoreBlock stores the provided block in the database if it is not already
// there. The full block data is written to ffldb.
func dbStoreBlock(dbTx database.Tx, block *czzutil.Block) error {
	hasBlock, err := dbTx.HasBlock(block.Hash())
	if err != nil {
		return err
	}
	if hasBlock {
		return nil
	}
	return dbTx.StoreBlock(block)
}

// blockIndexKey generates the binary key for an entry in the block index
// bucket. The key is composed of the block height encoded as a big-endian
// 32-bit unsigned int followed by the 32 byte block hash.
func blockIndexKey(blockHash *chainhash.Hash, blockHeight uint32) []byte {
	indexKey := make([]byte, chainhash.HashSize+4)
	binary.BigEndian.PutUint32(indexKey[0:4], blockHeight)
	copy(indexKey[4:chainhash.HashSize+4], blockHash[:])
	return indexKey
}

func (b *BlockChain) CurrentEstate() *cross.EntangleState {
	hash := b.bestChain.tip().hash
	height := b.bestChain.tip().height
	eState := b.exChangeVerify.Cache.LoadEntangleState(height, hash)
	return eState
}

func (b *BlockChain) CurrentEstate2() *cross.EntangleState2 {
	hash := b.bestChain.tip().hash
	height := b.bestChain.tip().height
	eState := b.exChangeVerify.Cache.LoadEntangleState2(height, hash)
	return eState
}

func (b *BlockChain) GetEstateByHashAndHeight(hash chainhash.Hash, height int32) *cross.EntangleState {
	eState := b.exChangeVerify.Cache.LoadEntangleState(height, hash)
	return eState
}

func (b *BlockChain) GetEstateByHashAndHeight2(hash chainhash.Hash, height int32) *cross.EntangleState2 {
	eState := b.exChangeVerify.Cache.LoadEntangleState2(height, hash)
	return eState
}

// BlockByHeight returns the block at the given height in the main chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) BlockByHeight(blockHeight int32) (*czzutil.Block, error) {
	// Lookup the block height in the best chain.
	node := b.bestChain.NodeByHeight(blockHeight)
	if node == nil {
		str := fmt.Sprintf("no block at height %d exists", blockHeight)
		return nil, errNotInMainChain(str)
	}

	// Load the block from the database and return it.
	var block *czzutil.Block
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		block, err = dbFetchBlockByNode(dbTx, node)
		return err
	})
	return block, err
}

// BlockByHash returns the block from the main chain with the given hash with
// the appropriate chain height set.
//
// This function is safe for concurrent access.
func (b *BlockChain) BlockByHash(hash *chainhash.Hash) (*czzutil.Block, error) {
	// Lookup the block hash in block index and ensure it is in the best
	// chain.
	node := b.index.LookupNode(hash)
	if node == nil || !b.bestChain.Contains(node) {
		str := fmt.Sprintf("block %s is not in the main chain", hash)
		return nil, errNotInMainChain(str)
	}

	// Load the block from the database and return it.
	var block *czzutil.Block
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		block, err = dbFetchBlockByNode(dbTx, node)
		return err
	})
	return block, err
}

func (b *BlockChain) blockByHashAndHeight(hash *chainhash.Hash, height int32) (*czzutil.Block, error) {
	node := &blockNode{
		hash:   *hash,
		height: height,
	}
	// Load the block from the database and return it.
	var block *czzutil.Block
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		var exists bool
		exists, err = dbTx.HasBlock(hash)
		if err != nil || !exists {
			return err
		}
		block, err = dbFetchBlockByNode(dbTx, node)
		return err
	})
	return block, err
}
