package cross

import (
	// "github.com/classzz/classzz/rlp"
	// "bytes"
	// "encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/classzz/classzz/chaincfg"
	// "github.com/classzz/classzz/chaincfg/chainhash"

	// "github.com/classzz/classzz/czzec"
	"github.com/classzz/classzz/txscript"
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
		v1 := toDoge1(reserve[0], doge)
		v2 := toLtc1(reserve[1], itc)

		fmt.Println("entangle index:", i, "[doge:", doge, " to czz:", fromCzz(v1).String(), "][itc:", itc, " to czz:", fromCzz(v2).String(), "]")
		reserve[0] += doge
		reserve[1] += itc
		doge = doge * int64(i+1)
		itc = itc * int64(i+1)
	}
	fmt.Println("finish")
}

func TestCaclMode2(t *testing.T) {
	sum := 10
	reserve := []*big.Int{new(big.Int).Mul(big.NewInt(0), baseUnit), new(big.Int).Mul(big.NewInt(0), baseUnit)}
	doge, itc := new(big.Int).Mul(big.NewInt(10000), baseUnit), new(big.Int).Mul(big.NewInt(100), baseUnit)

	for i := 0; i < sum; i++ {
		v1 := toDoge2(reserve[0], doge)
		v2 := toLtc2(reserve[1], itc)
		fmt.Println("==============================================entangle index:", i, "==============================================")
		fmt.Printf("entangle doge:%v []\n", doge)
		fmt.Printf("entangle doge:%v [doge]\n", fromCzz1(doge).Text('f', 4))
		fmt.Printf("to czz:%v []\n", v1)
		fmt.Printf("to czz:%v [czz]\n", fromCzz1(v1).Text('f', 4))

		fmt.Printf("entangle itc:%v []\n", itc)
		fmt.Printf("entangle itc:%v [itc]\n", fromCzz1(itc).Text('f', 4))
		fmt.Printf("to czz:%v []\n", v2)
		fmt.Printf("to czz:%v [czz]\n", fromCzz1(v2).Text('f', 4))
		fmt.Println("==============================================entangle index:", i, "==============================================")
		reserve[0] = reserve[0].Add(reserve[0], doge)
		reserve[1] = reserve[1].Add(reserve[1], itc)
		doge = doge.Mul(doge, big.NewInt(int64(i+1)))
		itc = itc.Mul(itc, big.NewInt(int64(i+1)))
	}
	fmt.Println("finish")
}

func TestCaclMode3(t *testing.T) {
	rate := big.NewFloat(0.0008)
	f1 := big.NewFloat(float64(100))
	f1 = f1.Quo(f1, rate)
	ret := toCzz(f1).Int64()
	fmt.Println(ret)

	itc := big.NewInt(100)
	f2 := new(big.Float).SetInt(itc)
	// f2 := big.NewFloat(float64(itc.Int64()))
	// f2 := new(big.Float).SetFloat64(float64(itc.Int64()))
	fmt.Println(f2)
	f2 = f2.Quo(f2, rate)
	fmt.Println(toCzz(f2).Int64())
	itc = itc.Mul(itc, baseUnit)
	f2 = new(big.Float).Quo(big.NewFloat(float64(itc.Int64())), rate)
	fmt.Println(toCzz(f2).Int64())

	fmt.Println("finish")
}
func TestStruct(t *testing.T) {
	d1 := EntangleTxInfo{
		AssetType: ExpandedTxEntangle_Doge,
		Index:     10,
		Height:    200,
		Amount:    big.NewInt(333311),
		ExtTxHash: nil,
	}
	sByte := d1.Serialize()
	fmt.Println("d1.Serialize():", sByte)
	d2 := EntangleTxInfo{}
	d2.Parse(sByte)
	fmt.Println("AssetType:", d2.AssetType, " Index:", d2.Index, " Height:", d2.Height,
		"Amount:", d2.Amount, "ExtTxHash:", d2.ExtTxHash)

	Sum := byte(10)
	items := KeepedAmount{
		Count: 0,
		Items: make([]KeepedItem, 0),
	}
	for i := 0; i < int(Sum); i++ {
		v := KeepedItem{
			AssetType: ExpandedTxEntangle_Doge,
			Amount:    big.NewInt(int64(100 * i)),
		}
		items.Add(v)
	}
	fmt.Println("Count:", items.Count, "items:", items.Items)
	sByte2 := items.Serialize()
	fmt.Println("sByte2:", sByte2)
	scriptInfo, err := txscript.KeepedAmountScript(sByte2)
	if err != nil {
		fmt.Println(err)
	}
	itme3, err2 := KeepedAmountFromScript(scriptInfo)
	if err2 != nil {
		fmt.Println(err2)
	}
	fmt.Println("Count:", itme3.Count, "items:", itme3.Items)
	items2 := KeepedAmount{}
	items2.Parse(sByte2)
	fmt.Println("Count:", items2.Count, "items:", items2.Items)
	fmt.Println("finish")
}

func makeTxIncludeEntx() *czzutil.Tx {
	targetTx := czzutil.NewTx(&wire.MsgTx{
		TxOut: []*wire.TxOut{{
			PkScript: nil,
			Value:    10,
		}},
	})

	info := EntangleTxInfo{
		AssetType: ExpandedTxEntangle_Doge,
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
	// entangleAddress := make(map[chainhash.Hash][]*TmpAddressPair)
	// tx := makeTxIncludeEntx()
	// if tx == nil {
	// 	fmt.Println("make tx include entangle info failed")
	// 	return
	// }
	// txs := []*czzutil.Tx{tx}
	// infos := ToEntangleItems(txs, entangleAddress)
	// if infos != nil {
	// 	for i, v := range infos {
	// 		fmt.Println(i, v)
	// 	}
	// }
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
func TestFloat(t *testing.T) {
	rate := big.NewFloat(-0.0008)
	fmt.Println(rate.Signbit())
	r1, a1 := rate.Float64()
	s := rate.String()
	length := len(strings.Split(fmt.Sprintf("%v", s), ".")[1])
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(0)), nil)
	fmt.Println(r1, a1, rate.Prec(), s, length, base)
	fmt.Println("finish")
}

func TestFloat2(t *testing.T) {
	rate := big.NewFloat(0.0008)
	base := big.NewFloat(0.0001)
	base1 := new(big.Float).Mul(base, big.NewFloat(float64(75)))
	fmt.Println("base1", base1)
	rate = rate.Add(rate, base1)
	fmt.Println("rate", rate)

	str := rate.Text('f', 4)
	fmt.Println(rate.Signbit(), str)
	pos := countMant(rate, 4)
	fmt.Println(pos)
	fmt.Println(makeMant(rate, 4))
	fmt.Println("finish")
}

func TestBigInt(t *testing.T) {
	change, rate := big.NewInt(399), big.NewInt(200)
	c1 := new(big.Int).Mul(change, baseUnit)
	res0 := new(big.Int).Quo(c1, rate)
	fmt.Println("res0:", res0.String(), "czz:", fromCzz1(res0).Text('f', 6))
	fmt.Println("finish")
}
func TestStateRlp(t *testing.T) {
	ValidAssetType(10)
	state := NewEntangleState()
	l := state.ToBytes()
	fmt.Println("state len:", l)

	fmt.Println("finish")
}
