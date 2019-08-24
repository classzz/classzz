package cross

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"

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

type EntangleItem struct {
	EType ExpandedTxType
	Value *big.Int
	Addr  czzutil.Address
}

type PoolAddrItem struct {
	POut   []*wire.OutPoint
	Script [][]byte
	Amount []*big.Int
}

type EntangleTxInfo struct {
	ExTxType  ExpandedTxType
	Index     uint32
	Height    uint64
	Amount    *big.Int
	ExtTxHash []byte
}

func (info *EntangleTxInfo) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.WriteByte(byte(info.ExTxType))
	binary.Write(buf, binary.LittleEndian, info.Index)
	binary.Write(buf, binary.LittleEndian, info.Height)
	b1 := info.Amount.Bytes()
	len := uint8(len(b1))
	buf.WriteByte(byte(len))

	buf.Write(b1)
	buf.Write(info.ExtTxHash)
	return buf.Bytes()
}

func (info *EntangleTxInfo) Parse(data []byte) error {
	if len(data) <= 14 {
		return errors.New("wrong lenght!")
	}
	info.ExTxType = ExpandedTxType(uint8(data[0]))
	switch info.ExTxType {
	case ExpandedTxEntangle_Doge, ExpandedTxEntangle_Ltc:
		break
	default:
		return errors.New("Parse failed,not entangle tx")
	}
	buf := bytes.NewBuffer(data[1:])
	binary.Read(buf, binary.LittleEndian, &info.Index)
	binary.Read(buf, binary.LittleEndian, &info.Height)
	l, _ := buf.ReadByte()
	b0 := make([]byte, int(uint32(l)))
	n, _ := buf.Read(b0)
	if int(uint32(l)) != n {
		panic("b0 not equal n")
	}
	info.Amount.SetBytes(b0)
	info.ExtTxHash = make([]byte, int(infoFixed[info.ExTxType]))
	n2, _ := buf.Read(info.ExtTxHash)

	if len(info.ExtTxHash) != n2 {
		panic("len(info.ExtTxHash) not equal n2")
	}

	// if len(info.ExtTxHash) != int(infoFixed[info.ExTxType]) {
	// 	e := fmt.Sprintf("lenght not match,[request:%v,exist:%v]", infoFixed[info.ExTxType], len(info.ExtTxHash))
	// 	return errors.New(e)
	// }
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

/*
MakeMegerTx
	tx (coinbase tx):
		in:
		1 empty hash of coinbase txin
		2 pooladdr1 of txin
		3 pooladdr2 of txin
			''''''''''''''''
			entangle tx1
				  .
				  .
				  .
			entangle txn
			''''''''''''''''
		out:
			1. coinbase txout
			2. pooladdr1 txout
			3. pooladdr2 txout
			   '''''''''''''''
				entangle txout1
						.
						.
						.
				entangle txoutn
			   '''''''''''''''
*/
func MakeMegerTx(tx *wire.MsgTx, pool *PoolAddrItem, amount []*big.Int, items []*EntangleItem) error {

	if pool == nil || len(pool.POut) == 0 {
		return nil
	}
	// make sure have enough Value to exchange
	poolIn1 := &wire.TxIn{
		PreviousOutPoint: *pool.POut[0],
		SignatureScript:  pool.Script[0],
		Sequence:         wire.MaxTxInSequenceNum,
	}
	poolIn2 := &wire.TxIn{
		PreviousOutPoint: *pool.POut[1],
		SignatureScript:  pool.Script[1],
		Sequence:         wire.MaxTxInSequenceNum,
	}
	reserve1, reserve2 := pool.Amount[0].Int64()+tx.TxOut[1].Value, pool.Amount[1].Int64()
	updateTxOutValue(tx.TxOut[2], reserve2)
	// merge pool tx
	tx.TxIn[1], tx.TxIn[2] = poolIn1, poolIn2
	for i, v := range tx.TxIn {
		if i > 2 {
			calcExchange(items[i-2], &reserve1)
			pkScript, err := txscript.PayToAddrScript(items[i-2].Addr)
			if err != nil {
				return err
			}
			out := &wire.TxOut{
				Value:    items[i-2].Value.Int64(),
				PkScript: pkScript,
			}
			tx.AddTxOut(out)
		}
	}

	tx.TxOut[1].Value = reserve1
	return nil
}

func updateTxOutValue(out *wire.TxOut, value int64) error {
	out.Value += value
	return nil
}

func calcExchange(item *EntangleItem, reserve *int64) {
	if item.EType == ExpandedTxEntangle_Doge {
		item.Value = new(big.Int).SetInt64(int64(toDoge(item.Value).Int64() / 25))
	} else if item.EType == ExpandedTxEntangle_Ltc {
		item.Value = new(big.Int).SetInt64(int64(toLtc(item.Value).Int64() / 5))
	}
	*reserve = *reserve - item.Value.Int64()
}

func toDoge(v *big.Int) *big.Int {
	return new(big.Int).Set(v)
}
func toLtc(v *big.Int) *big.Int {
	return new(big.Int).Set(v)
}

