package cross

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
	"fmt"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/classzz/chaincfg/chainhash"
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
		ExpandedTxEntangle_Doge: 64,
		ExpandedTxEntangle_Ltc:  64,
	}
)

type EntangleItem struct {
	EType ExpandedTxType
	Value *big.Int
	Addr  czzutil.Address
}

func (ii *EntangleItem) Clone() *EntangleItem {
	item := &EntangleItem{
		EType: ii.EType,
		Value: new(big.Int).Set(ii.Value),
		Addr:  ii.Addr,
	}
	return item
}

type TuplePubIndex struct {
	EType ExpandedTxType
	Index uint32
	Pub   []byte
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
	info.ExTxType = ExpandedTxType(data[0])
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
	amount := big.NewInt(0)
	amount.SetBytes(b0)
	info.Amount = amount
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

type KeepedItem struct {
	ExTxType ExpandedTxType
	Amount   *big.Int
}
type KeepedAmount struct {
	Count byte
	Items []KeepedItem
}

func (info *KeepedAmount) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.WriteByte(info.Count)
	for _, v := range info.Items {
		buf.WriteByte(byte(v.ExTxType))
		b1 := v.Amount.Bytes()
		len := uint8(len(b1))
		buf.WriteByte(byte(len))
		buf.Write(b1)
	}
	return buf.Bytes()
}

func (info *KeepedAmount) Parse(data []byte) error {
	info.Count = data[0]
	buf := bytes.NewBuffer(data[1:])

	for i := 0; i < int(info.Count); i++ {
		itype, _ := buf.ReadByte()
		l, _ := buf.ReadByte()
		b0 := make([]byte, int(uint32(l)))
		_, _ = buf.Read(b0)
		item := KeepedItem{
			ExTxType: ExpandedTxType(itype),
			Amount:   new(big.Int).SetBytes(b0),
		}
		info.Items = append(info.Items, item)
	}
	return nil
}
func (info *KeepedAmount) add(item KeepedItem) {
	for _, v := range info.Items {
		if v.ExTxType == item.ExTxType {
			v.Amount.Add(v.Amount, item.Amount)
			return
		}
	}
	info.Count++
	info.Items = append(info.Items, item)
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

func IsEntangleTx(tx *wire.MsgTx) (error, map[uint32]*EntangleTxInfo) {
	// make sure at least one txout in OUTPUT
	einfo := make(map[uint32]*EntangleTxInfo)
	for i, v := range tx.TxOut {
		info := &EntangleTxInfo{}
		if err := info.Parse(v.PkScript[4:]); err == nil {
			if v.Value != 0 {
				return errors.New("the output value must be 0 in entangle tx."), nil
			}
			einfo[uint32(i)] = info
		}
	}
	if len(einfo) > 0 {
		return nil, einfo
	}
	return errors.New("no entangle info in transcation"), nil
}

func GetMaxHeight(items map[uint32]*EntangleTxInfo) uint64 {
	h := uint64(0)
	for _, v := range items {
		if h < v.Height {
			h = v.Height
		}
	}
	return h
}
func VerifyTxsSequence(txs []*czzutil.Tx) error {
	mix := uint64(0)
	for i, v := range txs {
		e, ii := IsEntangleTx(v.MsgTx())
		if e == nil {
			h := GetMaxHeight(ii)
			if mix > h {
				return errors.New(fmt.Sprintf("tx sequence wrong,[i=%d,h=%v][i=%d,h=%v]",i-1,mix,i,h))
			} else {
				mix = h
			}
		}
	}
	return nil
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
func MakeMergeTx(tx *wire.MsgTx, pool *PoolAddrItem, items []*EntangleItem) error {

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
	if ok := EnoughAmount(reserve1, items); !ok {
		return errors.New("not enough amount to be entangle...")
	}
	// add keeped Amount txout
	tx.AddTxOut(&wire.TxOut{
		Value:    0,
		PkScript: nil,
	})
	keepInfo := KeepedAmount{Items: []KeepedItem{}}
	// merge pool tx
	tx.TxIn[1], tx.TxIn[2] = poolIn1, poolIn2
	for i := range items {
		calcExchange(items[i], &reserve1)
		pkScript, err := txscript.PayToAddrScript(items[i].Addr)
		if err != nil {
			return errors.New("Make Meger tx failed,err: " + err.Error())
		}
		out := &wire.TxOut{
			Value:    items[i].Value.Int64(),
			PkScript: pkScript,
		}
		keepInfo.add(KeepedItem{
			ExTxType: items[i].EType,
			Amount:   new(big.Int).Set(items[i].Value),
		})
		tx.AddTxOut(out)
	}
	keepEntangleAmount(&keepInfo, tx)
	tx.TxOut[1].Value = reserve1
	return nil
}

func updateTxOutValue(out *wire.TxOut, value int64) error {
	out.Value += value
	return nil
}

func calcExchange(item *EntangleItem, reserve *int64) KeepedItem {

	if item.EType == ExpandedTxEntangle_Doge {
		item.Value = new(big.Int).SetInt64(toDoge(*reserve,item.Value.Int64()))
	} else if item.EType == ExpandedTxEntangle_Ltc {
		item.Value = new(big.Int).SetInt64(toLtc(*reserve,item.Value.Int64()))
	}
	*reserve = *reserve - item.Value.Int64()
	kk := KeepedItem{
		ExTxType: item.EType,
		Amount:   new(big.Int).Set(item.Value),
	}
	return kk
}

func EnoughAmount(reserve int64, items []*EntangleItem) bool {
	amount := reserve
	for _, v := range items {
		calcExchange(v.Clone(), &amount)
	}
	return amount > 0
}

func keepEntangleAmount(info *KeepedAmount, tx *wire.MsgTx) error {

	scriptInfo, err := txscript.KeepedAmountScript(info.Serialize())
	if err != nil {
		return err
	}
	txout := &wire.TxOut{
		Value:    0,
		PkScript: scriptInfo,
	}
	tx.TxOut[3] = txout
	return nil
}

func toDoge(entangled, needed int64) int64 {
	if needed <= 0 {
		return 0
	}
	var kk, rate int64 = 0, 25
	rate = rate + int64(entangled/int64(12500000))
	p := entangled % int64(12500000)

	if (int64(12500000) - p) >= needed {
		f1 := big.NewFloat(float64(needed))
		f1 = f1.Quo(f1,big.NewFloat(float64(rate)))
		kk = toCzz(f1).Int64()
	} else {
		v1 := big.NewFloat(float64(int64(12500000) - p))
		v2 := big.NewFloat(float64(needed - p))
		r1 := big.NewFloat(float64(rate))
		v1 = v1.Quo(v1,r1)
		kk = toCzz(v1).Int64()
		rate += 1
		r2 := big.NewFloat(float64(rate))
		v2 = v2.Quo(v2,r2)
		kk = kk + toCzz(v2).Int64()
	}
	return kk
}

func toLtc(entangled, needed int64) int64 {
	if needed <= 0 {
		return 0
	}
	var ret int64 = 0
	rate := big.NewFloat(0.0008)
	base := big.NewFloat(0.0001)

	fixed := int64(1150)
	divisor := entangled / fixed
	remainder := entangled % fixed

	base1 := base.Mul(base, big.NewFloat(float64(divisor)))
	rate = rate.Add(rate, base1)

	if fixed-remainder >= needed {
		f1 := big.NewFloat(float64(needed))
		f1 = f1.Quo(f1, rate)
		ret = toCzz(f1).Int64()
	} else {
		v1 := fixed - remainder
		v2 := needed - remainder
		f1, f2 := big.NewFloat(float64(v1)), big.NewFloat(float64(v2))
		f1 = f1.Quo(f1, rate)
		rate = rate.Add(rate, base)
		f2 = f2.Quo(f2, rate)
		ret = toCzz(f1).Int64() + toCzz(f2).Int64()
	}
	return ret
}
func toCzz(val *big.Float) *big.Int {
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
	val = val.Mul(val, big.NewFloat(float64(base.Int64())))
	ii, _ := val.Int64()
	return big.NewInt(ii)
}

// the tool function for entangle tx 
type TmpAddressPair struct {
	index   uint32
	address czzutil.Address
}

func ToEntangleItems(txs []*czzutil.Tx, addrs map[chainhash.Hash][]*TmpAddressPair) []*EntangleItem {
	items := make([]*EntangleItem, 0)
	for _, v := range txs {
		err, infos := IsEntangleTx(v.MsgTx())
		if err == nil {
			for i, out := range infos {
				item := &EntangleItem{
					EType: out.ExTxType,
					Value: new(big.Int).Set(out.Amount),
					Addr:  nil,
				}
				pairs, ok := addrs[*v.Hash()]
				if ok {
					for _, vv := range pairs {
						if i == vv.index {
							item.Addr = vv.address
						}
					}
				}
				items = append(items, item)
			}
		}
	}
	return items
}

func ToAddressFromEntangle(tx *czzutil.Tx, ev *EntangleVerify) ([]*TmpAddressPair, error) {
	// txhash := tx.Hash()
	err, _ := IsEntangleTx(tx.MsgTx())
	if err == nil {
		// verify the entangle tx

		pairs := make([]*TmpAddressPair, 0)
		err, tt := ev.VerifyEntangleTx(tx.MsgTx())
		if err != nil {
			return nil, err
		}
		for _, v := range tt {
			pub, err1 := RecoverPublicFromBytes(v.Pub, v.EType)
			if err1 != nil {
				return nil, err1
			}
			err2, addr := MakeAddress(*pub)
			if err2 != nil {
				return nil, err2
			}
			pairs = append(pairs, &TmpAddressPair{
				index:   v.Index,
				address: addr,
			})
		}

		return pairs, nil
	}

	return nil, nil
}
func OverEntangleAmount(tx *wire.MsgTx, pool *PoolAddrItem, items []*EntangleItem) bool {
	if items == nil || len(items) == 0 {
		return false
	}
	all := pool.Amount[0].Int64() + tx.TxOut[1].Value
	return !EnoughAmount(all, items)
}


