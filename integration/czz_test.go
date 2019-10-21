package integration

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/classzz/classzz/blockchain"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/mining"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

var (
	minAddress, _ = czzutil.DecodeAddress("czp5g27p3lz02astuyrnzd0sm90gh4280g3hgr2l0t", &chaincfg.MainNetParams)
	pubKey        = "02cd77593671ecaac86f942ac99cccaa53810bb23d7b8dd38610b068d388cbd899"
	privKey       = "bcd7220fae4f1fcff9bb6d9fd7861c880e0c522abfaa3a37ab17dad512a54885"
	CoinbaseFlags = "/classzz/"
)

func standardCoinbaseScript(nextBlockHeight int32, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(nextBlockHeight)).
		AddInt64(int64(extraNonce)).AddData([]byte(CoinbaseFlags)).
		Script()
}

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
	tx.AddTxIn(&wire.TxIn{}) // for pool1 address
	tx.AddTxIn(&wire.TxIn{})

	//Calculation incentive
	reward := blockchain.CalcBlockSubsidy(nextBlockHeight, params)

	// reward2 = reward * 19%
	reward1 := reward * 19 / 100

	// Calculate 1% of the reward.
	reward2 := reward / 100

	// Calculate 1 -20% = 80% of the reward. Coinbase
	reward3 := reward - reward1 - reward2

	// Coinbase reward
	tx.AddTxOut(&wire.TxOut{
		Value:    reward3,
		PkScript: pkScript,
	})

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

	// Make sure the coinbase is above the minimum size threshold.
	if tx.SerializeSize() < blockchain.MinTransactionSize {
		tx.TxIn[0].SignatureScript = append(tx.TxIn[0].SignatureScript,
			make([]byte, blockchain.MinTransactionSize-tx.SerializeSize()-1)...)
	}
	return czzutil.NewTx(tx), nil
}

func makeBlock(prevBlock *czzutil.Block, inclusionTxs []*czzutil.Tx,
	blockVersion int32, blockTime time.Time, miningAddr czzutil.Address,
	mineTo []wire.TxOut, net *chaincfg.Params) (*czzutil.Block, error) {

	var (
		prevHash      *chainhash.Hash
		blockHeight   int32
		prevBlockTime time.Time
	)

	// If the previous block isn't specified, then we'll construct a block
	// that builds off of the genesis block for the chain.
	if prevBlock == nil {
		prevHash = net.GenesisHash
		blockHeight = 1
		prevBlockTime = net.GenesisBlock.Header.Timestamp.Add(time.Minute)
	} else {
		prevHash = prevBlock.Hash()
		blockHeight = prevBlock.Height() + 1
		prevBlockTime = prevBlock.MsgBlock().Header.Timestamp
	}

	// If a target block time was specified, then use that as the header's
	// timestamp. Otherwise, add one second to the previous block unless
	// it's the genesis block in which case use the current time.
	var ts time.Time
	switch {
	case !blockTime.IsZero():
		ts = blockTime
	default:
		ts = prevBlockTime.Add(time.Second)
	}

	extraNonce := uint64(0)
	coinbaseScript, err := standardCoinbaseScript(blockHeight, extraNonce)
	if err != nil {
		return nil, err
	}
	coinbaseTx, err := createCoinbaseTx(net, coinbaseScript, blockHeight, miningAddr)
	if err != nil {
		return nil, err
	}

	// Create a new block ready to be solved.
	var blockTxns []*czzutil.Tx
	if inclusionTxs != nil {
		blockTxns = append(blockTxns, inclusionTxs...)
	}
	// If magnetic anomaly is enabled ally CTOR sorting
	sort.Sort(mining.TxSorter(blockTxns))

	blockTxns = append([]*czzutil.Tx{coinbaseTx}, blockTxns...)
	merkles := blockchain.BuildMerkleTreeStore(blockTxns)
	var block wire.MsgBlock
	block.Header = wire.BlockHeader{
		Version:    blockVersion,
		PrevBlock:  *prevHash,
		MerkleRoot: *merkles[len(merkles)-1],
		Timestamp:  ts,
		Bits:       net.PowLimitBits,
	}
	for _, tx := range blockTxns {
		if err := block.AddTransaction(tx.MsgTx()); err != nil {
			return nil, err
		}
	}

	// found := solveBlock(&block.Header, net.PowLimit)
	// if !found {
	// 	return nil, errors.New("Unable to solve block")
	// }

	utilBlock := czzutil.NewBlock(&block)
	utilBlock.SetHeight(blockHeight)
	return utilBlock, nil
}

func MakeBlocks(count int, address czzutil.Address) ([]*czzutil.Block, error) {
	if count <= 0 {
		return nil, nil
	}
	chainParams := chaincfg.MainNetParams
	genesisBlock := czzutil.NewBlock(chainParams.GenesisBlock)
	var nullTime time.Time

	blocks := make([]*czzutil.Block, 0, count)
	blockVersion := int32(2)
	prevBlock := genesisBlock
	for i := 0; i < count; i++ {
		block, err := makeBlock(prevBlock, nil, blockVersion,
			nullTime, address, []wire.TxOut{}, &chainParams)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
		prevBlock = block
	}
	return blocks, nil
}

func TestCZZ1(t *testing.T) {
	address := minAddress
	_, err := MakeBlocks(5, address)
	if err != nil {
		fmt.Println(err)
	}

}

