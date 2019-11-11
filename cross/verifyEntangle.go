package cross

import (
	"errors"
	"fmt"
	"github.com/classzz/classzz/txscript"
	"math/big"
	"math/rand"

	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/classzz/wire"
)

var (
	dogePoolPub       = []byte{1, 2}
	ltcPoolPub        = []byte{1, 2}
	ErrHeightTooClose = errors.New("the block heigth to close for entangling")
)

const (
	dogeMaturity = 14
	ltcMaturity  = 14
)

type EntangleVerify struct {
	DogeCoinRPC []*rpcclient.Client
	LtcCoinRPC  []*rpcclient.Client
	Cache       *CacheEntangleInfo
}

func (ev *EntangleVerify) VerifyEntangleTx(tx *wire.MsgTx) (error, []*TuplePubIndex) {
	/*
		1. check entangle tx struct
		2. check the repeat tx
		3. check the correct tx
		4. check the pool reserve enough reward
	*/
	ok, einfo := IsEntangleTx(tx)
	if ok != nil {
		return errors.New("not entangle tx"), nil
	}
	pairs := make([]*TuplePubIndex, 0)
	amount := int64(0)
	if ev.Cache != nil {
		for i, v := range einfo {
			if ok := ev.Cache.FetchEntangleUtxoView(v); ok {
				errStr := fmt.Sprintf("[txid:%v, height:%v]", v.ExtTxHash, v.Height)
				return errors.New("txid has already entangle:" + errStr), nil
			}
			amount += tx.TxOut[i].Value
		}
	}

	for i, v := range einfo {
		if err, pub := ev.verifyTx(v.ExTxType, v.ExtTxHash, v.Index, v.Height, v.Amount); err != nil {
			errStr := fmt.Sprintf("[txid:%v, height:%v]", v.ExtTxHash, v.Index)
			return errors.New("txid verify failed:" + errStr + " err:" + err.Error()), nil
		} else {
			pairs = append(pairs, &TuplePubIndex{
				EType: v.ExTxType,
				Index: i,
				Pub:   pub,
			})
		}
	}

	// find the pool addrees
	// reserve := GetPoolAmount()
	// if amount >= reserve {
	// 	e := fmt.Sprintf("amount not enough,[request:%v,reserve:%v]", amount, reserve)
	// 	return errors.New(e),nil
	// }
	return nil, pairs
}

func (ev *EntangleVerify) verifyTx(ExTxType ExpandedTxType, ExtTxHash []byte, Vout uint32,
	height uint64, amount *big.Int) (error, []byte) {
	switch ExTxType {
	case ExpandedTxEntangle_Doge:
		return ev.verifyDogeTx(ExtTxHash, Vout, amount, height)
	case ExpandedTxEntangle_Ltc:
		return ev.verifyLtcTx(ExtTxHash, Vout, amount, height)
	}
	return nil, nil
}

func (ev *EntangleVerify) verifyDogeTx(ExtTxHash []byte, Vout uint32, Amount *big.Int, height uint64) (error, []byte) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.DogeCoinRPC[rand.Intn(len(ev.DogeCoinRPC))]

	// Get the current block count.
	fmt.Println("tran: ", string(ExtTxHash))
	if tx, err := client.GetRawTransaction(string(ExtTxHash)); err != nil {
		return err, nil
	} else {
		if len(tx.MsgTx().TxOut) < int(Vout) {
			return errors.New("doge TxOut index err"), nil
		}
		if tx.MsgTx().TxOut[Vout].Value != Amount.Int64() {
			e := fmt.Sprintf("amount err ,[request:%v,doge:%v]", Amount, tx.MsgTx().TxOut[Vout].Value)
			return errors.New(e), nil
		}
		if txscript.GetScriptClass(tx.MsgTx().TxOut[Vout].PkScript) != 2 {
			e := fmt.Sprintf("doge PkScript err ")
			return errors.New(e), nil
		}

		//pool := tx.MsgTx().TxOut[Vout].PkScript[2:22]
		//if !bytes.Equal(pool, dogePoolPub) {
		//	e := fmt.Sprintf("doge dogePoolPub err")
		//	return errors.New(e), nil
		//}

		if pk, err := txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript); err != nil {
			e := fmt.Sprintf("doge PkScript err %s", err)
			return errors.New(e), nil
		} else {

			if count, err := client.GetBlockCount(); err != nil {
				return err, nil
			} else {
				fmt.Println("pk.Script()", pk)
				if count-int64(height) > dogeMaturity {
					//return nil, pk.Script()[3:23]
					return nil, pk
				} else {
					e := fmt.Sprintf("dogeMaturity err ")
					return errors.New(e), nil
				}
			}
		}
	}
}

func (ev *EntangleVerify) verifyLtcTx(ExtTxHash []byte, Vout uint32, Amount *big.Int, height uint64) (error, []byte) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.LtcCoinRPC[rand.Intn(len(ev.LtcCoinRPC))]

	// Get the current block count.
	fmt.Println("tran: ", string(ExtTxHash))
	if tx, err := client.GetRawTransaction(string(ExtTxHash)); err != nil {
		return err, nil
	} else {
		if len(tx.MsgTx().TxOut) < int(Vout) {
			return errors.New("doge TxOut index err"), nil
		}
		if tx.MsgTx().TxOut[Vout].Value != Amount.Int64() {
			e := fmt.Sprintf("amount err ,[request:%v,doge:%v]", Amount, tx.MsgTx().TxOut[Vout].Value)
			return errors.New(e), nil
		}
		if txscript.GetScriptClass(tx.MsgTx().TxOut[Vout].PkScript) != 2 {
			e := fmt.Sprintf("doge PkScript err ")
			return errors.New(e), nil
		}

		//pool := tx.MsgTx().TxOut[Vout].PkScript[2:22]
		//if !bytes.Equal(pool, dogePoolPub) {
		//	e := fmt.Sprintf("doge dogePoolPub err")
		//	return errors.New(e), nil
		//}

		if pk, err := txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript); err != nil {
			e := fmt.Sprintf("doge PkScript err %s", err)
			return errors.New(e), nil
		} else {

			if count, err := client.GetBlockCount(); err != nil {
				return err, nil
			} else {
				fmt.Println("pk.Script()", pk)
				if count-int64(height) > ltcMaturity {
					//return nil, pk.Script()[3:23]
					return nil, pk
				} else {
					e := fmt.Sprintf("dogeMaturity err ")
					return errors.New(e), nil
				}
			}
		}
	}
}
