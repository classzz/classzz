package cross

import (
	// "bytes"
	// "encoding/binary"
	// "errors"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"

	// "github.com/classzz/classzz/czzec"
	// "github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

var (
	changeAddr, _ = czzutil.DecodeAddress("czp5g27p3lz02astuyrnzd0sm90gh4280g3hgr2l0t", &chaincfg.MainNetParams)
	pubKey        = "02cd77593671ecaac86f942ac99cccaa53810bb23d7b8dd38610b068d388cbd899"
	privKey       = "bcd7220fae4f1fcff9bb6d9fd7861c880e0c522abfaa3a37ab17dad512a54885"
)

func TestCalcModel(t *testing.T) {
	sum := 10
	reserve := []int64{20, 100}
	doge := int64(10000)
	itc := int64(100)
	for i := 0; i < sum; i++ {
		v1 := toDoge(reserve[0], doge)
		v2 := toLtc(reserve[1], itc)

		fmt.Println("entangle index:", i, "[doge:", doge, " to czz:", fromCzz(v1).String(), "][itc:", itc, " to czz:", fromCzz(v2).String(), "]")
		reserve[0] += doge
		reserve[1] += itc
		doge = doge * int64(i+1)
		itc = itc * int64(i+1)
	}
	fmt.Println("finish")
}

func TestStruct(t *testing.T) {
	d1 := EntangleTxInfo{
		ExTxType:  ExpandedTxEntangle_Doge,
		Index:     10,
		Height:    200,
		Amount:    big.NewInt(333311),
		ExtTxHash: nil,
	}
	sByte := d1.Serialize()
	fmt.Println("d1.Serialize():", sByte)
	d2 := EntangleTxInfo{}
	d2.Parse(sByte)
	fmt.Println("ExTxType:", d2.ExTxType, " Index:", d2.Index, " Height:", d2.Height,
		"Amount:", d2.Amount, "ExtTxHash:", d2.ExtTxHash)

	Sum := byte(10)
	items := KeepedAmount{
		Count: 0,
		Items: make([]KeepedItem, 0),
	}
	for i := 0; i < int(Sum); i++ {
		v := KeepedItem{
			ExTxType: ExpandedTxEntangle_Doge,
			Amount:   big.NewInt(int64(100 * i)),
		}
		items.add(v)
	}
	fmt.Println("Count:", items.Count, "items:", items.Items)
	sByte2 := items.Serialize()
	fmt.Println("sByte2:", sByte2)
	items2 := KeepedAmount{}
	items2.Parse(sByte2)
	fmt.Println("Count:", items2.Count, "items:", items2.Items)
}

func makeTxIncludeEntx() *czzutil.Tx {
	targetTx := czzutil.NewTx(&wire.MsgTx{
		TxOut: []*wire.TxOut{{
			PkScript: nil,
			Value:    10,
		}},
	})

	info := EntangleTxInfo{
		ExTxType:  ExpandedTxEntangle_Doge,
		Index:     1,
		Height:    100,
		Amount:    big.NewInt(20),
		ExtTxHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	mstx, e := MakeEntangleTx(&chaincfg.MainNetParams, targetTx.MsgTx().TxIn, 10, 1000, changeAddr, &info)
	if e != nil {
		return nil
	}
	return czzutil.NewTx(mstx)
}
func TestToolFunc1(t *testing.T) {
	entangleAddress := make(map[chainhash.Hash][]*TmpAddressPair)
	tx := makeTxIncludeEntx()
	if tx == nil {
		fmt.Println("make tx include entangle info failed")
		return
	}
	txs := []*czzutil.Tx{tx}
	infos := ToEntangleItems(txs, entangleAddress)
	if infos != nil {
		for i, v := range infos {
			fmt.Println(i, v)
		}
	}
}

func TestAddrTrans(t *testing.T) {
	pub, err := hex.DecodeString(pubKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	pp, err1 := RecoverPublicFromBytes(pub, ExpandedTxEntangle_Doge)
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	err2, addr := MakeAddress(*pp)
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	fmt.Println(addr.String())
}

