package cross

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/czzutil"
	"math/big"
	"math/rand"

	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/classzz/wire"
)

var (
	ErrHeightTooClose = errors.New("the block heigth to close for entangling")
)

const (
	dogePoolAddr = "DNGzkoZbnVMihLTMq8M1m7L62XvN3d2cN2"
	ltcPoolAddr  = "MUy9qiaLQtaqmKBSk27FXrEEfUkRBeddCZ"
	dogeMaturity = 14
	ltcMaturity  = 12
)

type EntangleVerify struct {
	DogeCoinRPC []*rpcclient.Client
	LtcCoinRPC  []*rpcclient.Client
	Cache       *CacheEntangleInfo
}

func (ev *EntangleVerify) VerifyEntangleTx(tx *wire.MsgTx) ([]*TuplePubIndex, error) {
	/*
		1. check entangle tx struct
		2. check the repeat tx
		3. check the correct tx
		4. check the pool reserve enough reward
	*/
	einfos, _ := IsEntangleTx(tx)
	if einfos == nil {
		return nil, errors.New("not entangle tx")
	}
	pairs := make([]*TuplePubIndex, 0)
	amount := int64(0)
	if ev.Cache != nil {
		for i, v := range einfos {
			if ok := ev.Cache.FetchEntangleUtxoView(v); ok {
				errStr := fmt.Sprintf("[txid:%s, height:%v]", hex.EncodeToString(v.ExtTxHash), v.Height)
				return nil, errors.New("txid has already entangle:" + errStr)
			}
			amount += tx.TxOut[i].Value
		}
	}

	for i, v := range einfos {
		if pub, err := ev.verifyTx(v.ExTxType, v.ExtTxHash, v.Index, v.Height, v.Amount); err != nil {
			errStr := fmt.Sprintf("[txid:%s, height:%v]", hex.EncodeToString(v.ExtTxHash), v.Index)
			return nil, errors.New("txid verify failed:" + errStr + " err:" + err.Error())
		} else {
			pairs = append(pairs, &TuplePubIndex{
				EType: v.ExTxType,
				Index: i,
				Pub:   pub,
			})
		}
	}

	return pairs, nil
}

func (ev *EntangleVerify) verifyTx(ExTxType ExpandedTxType, ExtTxHash []byte, Vout uint32,
	height uint64, amount *big.Int) ([]byte, error) {
	switch ExTxType {
	case ExpandedTxEntangle_Doge:
		return ev.verifyDogeTx(ExtTxHash, Vout, amount, height)
	case ExpandedTxEntangle_Ltc:
		return ev.verifyLtcTx(ExtTxHash, Vout, amount, height)
	}
	return nil, nil
}

func (ev *EntangleVerify) verifyDogeTx(ExtTxHash []byte, Vout uint32, Amount *big.Int, height uint64) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.DogeCoinRPC[rand.Intn(len(ev.DogeCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(string(ExtTxHash)); err != nil {
		return nil, err
	} else {

		if len(tx.MsgTx().TxIn) < 1 || len(tx.MsgTx().TxOut) < 1 {
			e := fmt.Sprintf("doge Transactionis in or out len < 0  in : %v , out : %v", len(tx.MsgTx().TxIn), len(tx.MsgTx().TxOut))
			return nil, errors.New(e)
		}

		if len(tx.MsgTx().TxOut) < int(Vout) {
			return nil, errors.New("doge TxOut index err")
		}

		var pk []byte
		if tx.MsgTx().TxIn[0].Witness == nil {
			pk, err = txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript)
			if err != nil {
				e := fmt.Sprintf("doge ComputePk err %s", err)
				return nil, errors.New(e)
			}
		} else {
			pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[0].Witness)
			if err != nil {
				e := fmt.Sprintf("doge ComputeWitnessPk err %s", err)
				return nil, errors.New(e)
			}
		}

		if bhash, err := client.GetBlockHash(int64(height)); err == nil {
			fmt.Println(bhash.String())
			if dblock, err := client.GetDogecoinBlock(bhash.String()); err == nil {
				if !CheckTransactionisBlock(string(ExtTxHash), dblock) {
					e := fmt.Sprintf("doge Transactionis %s not in BlockHeight %v", string(ExtTxHash), height)
					return nil, errors.New(e)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}

		if Amount.Int64() < 0 || tx.MsgTx().TxOut[Vout].Value != Amount.Int64() {
			e := fmt.Sprintf("doge amount err ,[request:%v,doge:%v]", Amount, tx.MsgTx().TxOut[Vout].Value)
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[Vout].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("doge PkScript err")
			return nil, errors.New(e)
		}

		dogeparams := &chaincfg.Params{
			LegacyScriptHashAddrID: 0x1e,
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[Vout].PkScript)
		if err != nil {
			return nil, err
		}

		addr, err := czzutil.NewLegacyAddressScriptHashFromHash(pub, dogeparams)
		if err != nil {
			e := fmt.Sprintf("doge addr err")
			return nil, errors.New(e)
		}

		if addr.String() != dogePoolAddr {
			e := fmt.Sprintf("doge dogePoolPub err")
			return nil, errors.New(e)
		}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(height) > dogeMaturity {
				return pk, nil
			} else {
				e := fmt.Sprintf("doge Maturity err")
				return nil, errors.New(e)
			}
		}

	}
}

func (ev *EntangleVerify) verifyLtcTx(ExtTxHash []byte, Vout uint32, Amount *big.Int, height uint64) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.LtcCoinRPC[rand.Intn(len(ev.LtcCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(string(ExtTxHash)); err != nil {
		return nil, err
	} else {

		if len(tx.MsgTx().TxIn) < 1 || len(tx.MsgTx().TxOut) < 1 {
			e := fmt.Sprintf("ltc Transactionis in or out len < 0  in : %v , out : %v", len(tx.MsgTx().TxIn), len(tx.MsgTx().TxOut))
			return nil, errors.New(e)
		}

		if len(tx.MsgTx().TxOut) < int(Vout) {
			return nil, errors.New("ltc TxOut index err")
		}

		var pk []byte
		if tx.MsgTx().TxIn[0].Witness == nil {
			pk, err = txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript)
			if err != nil {
				e := fmt.Sprintf("ltc ComputePk err %s", err)
				return nil, errors.New(e)
			}
		} else {
			pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[0].Witness)
			if err != nil {
				e := fmt.Sprintf("ltc ComputeWitnessPk err %s", err)
				return nil, errors.New(e)
			}
		}

		if bhash, err := client.GetBlockHash(int64(height)); err == nil {
			fmt.Println(bhash.String())
			if dblock, err := client.GetDogecoinBlock(bhash.String()); err == nil {
				if !CheckTransactionisBlock(string(ExtTxHash), dblock) {
					e := fmt.Sprintf("ltc Transactionis %s not in BlockHeight %v", string(ExtTxHash), height)
					return nil, errors.New(e)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}

		if Amount.Int64() < 0 || tx.MsgTx().TxOut[Vout].Value != Amount.Int64() {
			e := fmt.Sprintf("ltc amount err ,[request:%v,ltc:%v]", Amount, tx.MsgTx().TxOut[Vout].Value)
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[Vout].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("ltc PkScript err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[Vout].PkScript)
		if err != nil {
			return nil, err
		}

		ltcparams := &chaincfg.Params{
			LegacyScriptHashAddrID: 0x32,
		}

		addr, err := czzutil.NewLegacyAddressScriptHashFromHash(pub, ltcparams)
		if err != nil {
			e := fmt.Sprintf("ltc addr err")
			return nil, errors.New(e)
		}

		if addr.String() != ltcPoolAddr {
			e := fmt.Sprintf("ltc PoolAddr err")
			return nil, errors.New(e)
		}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(height) > ltcMaturity {
				return pk, nil
			} else {
				e := fmt.Sprintf("ltc Maturity err")
				return nil, errors.New(e)
			}
		}
	}
}

func CheckTransactionisBlock(txhash string, block *rpcclient.DogecoinMsgBlock) bool {
	for _, dtx := range block.Txs {
		if dtx == txhash {
			return true
		}
	}
	return false
}
