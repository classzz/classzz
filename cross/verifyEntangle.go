package cross

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/classzz/wire"
)

const (
	dogePoolPub = ""
	ltcPoolPub  = ""
)
var (
	ErrHeightTooClose 	= 	errors.New("the block heigth to close for entangling")
)

type EntangleVerify struct {
	DogeCoinRPC []string
}

func (ev *EntangleVerify) VerifyEntangleTx(tx *wire.MsgTx, cache *CacheEntangleInfo) (error,[]*TuplePubIndex) {
	/*
		1. check entangle tx struct
		2. check the repeat tx
		3. check the correct tx
		4. check the pool reserve enough reward
	*/
	ok, einfo := IsEntangleTx(tx)
	if ok != nil {
		return errors.New("not entangle tx"),nil
	}
	pairs := make([]*TuplePubIndex,0)
	amount := int64(0)
	if cache != nil {
		for i, v := range einfo {
			if ok := cache.TxExist(v); !ok {
				errStr := fmt.Sprintf("[txid:%v, height:%v]", v.ExtTxHash, v.Height)
				return errors.New("txid has already entangle:" + errStr),nil
			}
			amount += tx.TxOut[i].Value
		}
	}

	for i, v := range einfo {
		if err,pub := ev.verifyTx(v.ExTxType, v.ExtTxHash, v.Index, v.Height, v.Amount); err != nil {
			errStr := fmt.Sprintf("[txid:%v, height:%v]", v.ExtTxHash, v.Index)
			return errors.New("txid verify failed:" + errStr + " err:" + err.Error()),nil
		} else {
			pairs = append(pairs,&TuplePubIndex{
				EType:		v.ExTxType,
				Index:		i,
				Pub:		pub,
			})
		}
	}

	// find the pool addrees
	// reserve := GetPoolAmount()
	// if amount >= reserve {
	// 	e := fmt.Sprintf("amount not enough,[request:%v,reserve:%v]", amount, reserve)
	// 	return errors.New(e),nil
	// }
	return nil,pairs
}

func (ev *EntangleVerify) verifyTx(ExTxType ExpandedTxType, ExtTxHash []byte, Vout uint32, 
	height uint64, amount *big.Int) (error,[]byte) {
	switch ExTxType {
	case ExpandedTxEntangle_Doge:
		return ev.verifyDogeTx(ExtTxHash, Vout, amount)
	}
	return nil,nil
}

func (ev *EntangleVerify) verifyDogeTx(ExtTxHash []byte, Vout uint32, Amount *big.Int) (error,[]byte) {

	connCfg := &rpcclient.ConnConfig{
		Host:       "localhost:8334",
		Endpoint:   "ws",
		DisableTLS: true,
	}

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return err,nil
	}
	defer client.Shutdown()

	// Get the current block count.
	if tx, err := client.GetRawTransaction(string(ExtTxHash)); err != nil {
		return err,nil
	} else {
		if len(tx.MsgTx().TxOut) < int(Vout) {
			return errors.New("doge TxOut index err"),nil
		}
		if tx.MsgTx().TxOut[Vout].Value != Amount.Int64() {
			e := fmt.Sprintf("amount err ,[request:%v,doge:%v]", Amount, tx.MsgTx().TxOut[Vout].Value)
			return errors.New(e),nil
		}
	}

	return nil,nil
}
