package cross

import (
	// "bytes"
	// "encoding/binary"
	// "errors"
	"math/big"
	"fmt"

	"github.com/classzz/classzz/chaincfg"
	// "github.com/classzz/classzz/czzec"
	// "github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
	"testing"
)


func TestCalcModel(t *testing.T){
	sum := 10
	reserve := []int64{20,100}
	doge := int64(10000)
	itc := int64(100)
	for i:=0;i<sum;i++ {
		v1 := toDoge(reserve[0],doge)
		v2 := toLtc(reserve[1],itc)

		fmt.Println("entangle index:",i,"[doge:",doge," to czz:",v1,"][itc:",itc," to czz:",v2,"]")
		reserve[0] += doge
		reserve[1] += itc
		doge = doge * int64(i+1)
		itc = itc * int64(i+1)
	}
}

func TestStruct(t *testing.T) {
	d1 := EntangleTxInfo{
		ExTxType:	ExpandedTxEntangle_Doge,
		Index:		10,
		Height:		200,
		Amount:		big.NewInt(333311),
		ExtTxHash:	nil,
	}
	sByte := d1.Serialize()
	fmt.Println("d1.Serialize():",sByte)
	d2 := EntangleTxInfo{}
	d2.Parse(sByte)
	fmt.Println("ExTxType:",d2.ExTxType," Index:",d2.Index," Height:",d2.Height,
	"Amount:",d2.Amount,"ExtTxHash:",d2.ExtTxHash)

	Sum := byte(10)
	items := KeepedAmount{
		Count:	0,
		Items:	make([]KeepedItem, 0),
	}
	for i:=0;i<int(Sum);i++ {
		v := KeepedItem{
			ExTxType: 	ExpandedTxEntangle_Doge,
			Amount:		big.NewInt(int64(100*i)),
		}
		items.add(v)
	}
	fmt.Println("Count:",items.Count,"items:",items.Items)
	sByte2 := items.Serialize()
	fmt.Println("sByte2:",sByte2)
	items2 := KeepedAmount{}
	items2.Parse(sByte2)
	fmt.Println("Count:",items2.Count,"items:",items2.Items)
}

func TestEntangleTx(t *testing.T) {

}

func makeTxIncludeEntx() *czzutil.Tx {
	targetTx := czzutil.NewTx(&wire.MsgTx{
		TxOut: []*wire.TxOut{{
			PkScript: nil,
			Value:    10,
		}},
	})
	info := EntangleTxInfo{}
	mstx,e := MakeEntangleTx(&chaincfg.MainNetParams,targetTx.MsgTx().TxIn,10,1000,nil,&info)
	if e != nil {
		return nil
	}
	return czzutil.NewTx(mstx)
}