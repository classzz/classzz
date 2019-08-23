package cross

import (
	"errors"
	"fmt"
	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/classzz/wire"
	"math/big"
)

const (
	dogePoolPub = ""
	ltcPoolPub  = ""
)

type EntangleVerify struct {
	DogeCoinRPC []string
}

func (ev *EntangleVerify) VerifyEntangleTx(tx *wire.MsgTx, cache *CacheEntangleInfo) error {
	/*
		1. check entangle tx struct
		2. check the repeat tx
		3. check the correct tx
		4. check the pool reserve enough reward
	*/
	ok, einfo := IsEntangleTx(tx)
	if !ok {
		return errors.New("not entangle tx")
	}
	amount := int64(0)
	for i, _ := range einfo {
		//if ok := cache.TxExist(v); !ok {
		//	errStr := fmt.Sprintf("[txid:%v, height:%v]", v.ExtTxHash, v.Vout)
		//	return errors.New("txid has already entangle:" + errStr)
		//}
		amount += tx.TxOut[i].Value
	}

	for _, v := range einfo {
		if err := ev.verifyTx(v.ExTxType, v.ExtTxHash, v.Index, v.Height, v.Amount); err != nil {
			errStr := fmt.Sprintf("[txid:%v, height:%v]", v.ExtTxHash, v.Index)
			return errors.New("txid verify failed:" + errStr + " err:" + err.Error())
		}
	}

	// find the pool addrees
	reserve := GetPoolAmount()
	if amount >= reserve {
		e := fmt.Sprintf("amount not enough,[request:%v,reserve:%v]", amount, reserve)
		return errors.New(e)
	}
	return nil
}

func (ev *EntangleVerify) verifyTx(ExTxType ExpandedTxType, ExtTxHash []byte, Vout uint32, height uint64, amount *big.Int) error {
	switch ExTxType {
	case ExpandedTxEntangle_Doge:
		return ev.verifyDogeTx(ExtTxHash, Vout, amount)
	}
	return nil
}

func (ev *EntangleVerify) verifyDogeTx(ExtTxHash []byte, Vout uint32, Amount *big.Int) error {

	connCfg := &rpcclient.ConnConfig{
		Host:       "localhost:8334",
		Endpoint:   "ws",
		DisableTLS: true,
	}

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return err
	}
	defer client.Shutdown()

	// Get the current block count.
	if tx, err := client.GetRawTransaction(string(ExtTxHash)); err != nil {
		return err
	} else {
		if len(tx.MsgTx().TxOut) < int(Vout) {
			return errors.New("doge TxOut index err")
		}
		if tx.MsgTx().TxOut[Vout].Value != Amount.Int64() {
			e := fmt.Sprintf("amount err ,[request:%v,doge:%v]", Amount, tx.MsgTx().TxOut[Vout].Value)
			return errors.New(e)
		}
	}

	return nil
}
