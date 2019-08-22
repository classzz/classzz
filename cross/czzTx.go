package cross

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

type ExpandedTxType uint8

const (
	// Entangle Transcation type
	ExpandedTxEntangle_Doge = 0xF0
	ExpandedTxEntangle_Ltc  = 0xF1
)

var (
	infoFixed = map[ExpandedTxType]uint32{
		ExpandedTxEntangle_Doge: 32,
		ExpandedTxEntangle_Ltc:  32,
	}
)

type EntangleTxInfo struct {
	ExTxType  ExpandedTxType
	ExtTxHash []byte
	Vout      uint64
}

type EntangleVerify interface {
	VerifyTx(chainType uint8, txID []byte, vout uint64) error
	GetPubByteFromTx(chainType uint8, txID []byte) (error, []byte)
}

func (info *EntangleTxInfo) Serialize() []byte {
	buf := new(bytes.Buffer)

	b0 := byte(info.ExTxType)
	buf.WriteByte(b0)
	binary.Write(buf, binary.LittleEndian, info.Vout)
	buf.Write(info.ExtTxHash)
	return buf.Bytes()
}

func (info *EntangleTxInfo) Parse(data []byte) error {
	if len(data) <= 5 {
		return errors.New("wrong lenght!")
	}
	info.ExTxType = ExpandedTxType(uint8(data[0]))
	switch info.ExTxType {
	case ExpandedTxEntangle_Doge, ExpandedTxEntangle_Ltc:
		break
	default:
		return errors.New("Parse failed,not entangle tx")
	}
	buf := bytes.NewBuffer(data[1:5])
	binary.Read(buf, binary.LittleEndian, &info.Vout)
	info.ExtTxHash = data[5:]
	if len(info.ExtTxHash) != int(infoFixed[info.ExTxType]) {
		e := fmt.Sprintf("lenght not match,[request:%v,exist:%v]", infoFixed[info.ExTxType], len(info.ExtTxHash))
		return errors.New(e)
	}
	return nil
}

func MakeEntangleTx(params *chaincfg.Params, inputs []*wire.TxIn, feeRate, inAmount czzutil.Amount,
	changeAddr czzutil.Address, info *EntangleTxInfo) (*wire.MsgTx, error) {
	// make pay script info include txHash and height
	scriptInfo, err := txscript.EntangleScript(info.Serialize())
	if err != nil {
		return nil, err
	}
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(&wire.TxOut{
		Value:    0,
		PkScript: scriptInfo,
	})
	var outputAmt czzutil.Amount = 0
	const (
		// spendSize is the largest number of bytes of a sigScript
		// which spends a p2pkh output: OP_DATA_73 <sig> OP_DATA_33 <pubkey>
		spendSize = 1 + 73 + 1 + 33
	)

	var (
		amtSelected czzutil.Amount
		txSize      int
	)
	for _, in := range inputs {
		tx.AddTxIn(in)
		txSize = tx.SerializeSize() + spendSize*len(tx.TxIn)
	}
	reqFee := czzutil.Amount(txSize * int(feeRate))
	changeVal := amtSelected - outputAmt - reqFee

	if changeVal > 0 {
		pkScript, err := txscript.PayToAddrScript(changeAddr)
		if err != nil {
			return nil, err
		}
		changeOutput := &wire.TxOut{
			Value:    int64(changeVal),
			PkScript: pkScript,
		}
		tx.AddTxOut(changeOutput)
	}

	return tx, nil
}

func SignEntangleTx(tx *wire.MsgTx, inputAmount []czzutil.Amount,
	priv *czzec.PrivateKey) error {

	for i, txIn := range tx.TxIn {
		sigScript, err := txscript.SignatureScript(tx, i,
			int64(inputAmount[i].ToUnit(czzutil.AmountSatoshi)), nil,
			txscript.SigHashAll, priv, true)
		if err != nil {
			return err
		}
		txIn.SignatureScript = sigScript
	}

	return nil
}

func IsEntangleTx(tx *wire.MsgTx) (bool, map[uint32]*EntangleTxInfo) {
	// make sure at least one txout in OUTPUT
	einfo := make(map[uint32]*EntangleTxInfo)
	for i, v := range tx.TxOut {
		info := &EntangleTxInfo{}
		if err := info.Parse(v.PkScript); err == nil {
			einfo[uint32(i)] = info
		}
	}
	return len(einfo) > 0, einfo
}

func GetPoolAmount() int64 {
	return 0
}

func MakeMegerTx(tx *wire.MsgTx) {
	/*
		1. get utxo from pool
		2. make the pool address reward
		3. make coin base reward
		4. make entangle reward(make entangle txid and output index as input's outPoint)
	*/
}
