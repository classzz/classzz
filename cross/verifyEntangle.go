package cross

import (
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/classzz/wire"
)

const (
	dogePoolPub = ""
	ltcPoolPub  = ""
)

func VerifyEntangleTx(tx *wire.MsgTx, cache *CacheEntangleInfo) error {
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
		if err := verifyTx(v.ExTxType, v.ExtTxHash, v.Index, v.Height, v.Amount); err != nil {
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

func verifyTx(ExTxType ExpandedTxType, ExtTxHash []byte, Vout uint32, height uint64, amount *big.Int) error {
	switch ExTxType {
	case ExpandedTxEntangle_Doge:
		return verifyDogeTx(ExtTxHash, Vout)
	}
	return nil
}

func verifyDogeTx(ExtTxHash []byte, Vout uint32) error {

	connCfg := &rpcclient.ConnConfig{
		Host:       "localhost:8334",
		Endpoint:   "ws",
		DisableTLS: true,
	}

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	// Get the current block count.
	tx, err := client.GetRawTransaction("c6c28ffee56883a8ce71d60d059d245a28a9e3a