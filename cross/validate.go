package cross

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/classzz/classzz/btcjson"
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

type ExChangeVerify struct {
	DogeCoinRPC []*rpcclient.Client
	LtcCoinRPC  []*rpcclient.Client
	BtcCoinRPC  []*rpcclient.Client
	BchCoinRPC  []*rpcclient.Client
	BsvCoinRPC  []*rpcclient.Client
	Cache       *CacheEntangleInfo
	Params      *chaincfg.Params
}

func (ev *ExChangeVerify) VerifyExChangeTx(tx *wire.MsgTx, eState *EntangleState) ([]*TuplePubIndex, error) {
	/*
		1. check entangle tx struct
		2. check the repeat tx
		3. check the correct tx
		4. check the pool reserve enough reward
	*/
	einfos, _ := IsExChangeTx(tx)
	if einfos == nil {
		return nil, errors.New("not entangle tx")
	}
	pairs := make([]*TuplePubIndex, 0)
	amount := int64(0)
	if ev.Cache != nil {
		for i, v := range einfos {
			if ok := ev.Cache.FetchExChangeUtxoView(v); ok {
				errStr := fmt.Sprintf("[txid:%s, height:%v]", hex.EncodeToString(v.ExtTxHash), v.Height)
				return nil, errors.New("txid has already entangle:" + errStr)
			}
			amount += tx.TxOut[i].Value
		}
	}

	for i, v := range einfos {
		if pub, err := ev.verifyTx(v, eState); err != nil {
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

func (ev *ExChangeVerify) verifyTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {
	switch eInfo.ExTxType {
	case ExpandedTxEntangle_Doge:
		return ev.verifyDogeTx(eInfo, eState)
	case ExpandedTxEntangle_Ltc:
		return ev.verifyLtcTx(eInfo, eState)
	case ExpandedTxEntangle_Btc:
		return ev.verifyBtcTx(eInfo, eState)
	case ExpandedTxEntangle_Bsv:
		return ev.verifyBsvTx(eInfo, eState)
	case ExpandedTxEntangle_Bch:
		return ev.verifyBchTx(eInfo, eState)
	}
	return nil, nil
}

func (ev *ExChangeVerify) verifyDogeTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.DogeCoinRPC[rand.Intn(len(ev.DogeCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(string(eInfo.ExtTxHash)); err != nil {
		return nil, err
	} else {

		if len(tx.MsgTx().TxIn) < 1 || len(tx.MsgTx().TxOut) < 1 {
			e := fmt.Sprintf("doge Transactionis in or out len < 0  in : %v , out : %v", len(tx.MsgTx().TxIn), len(tx.MsgTx().TxOut))
			return nil, errors.New(e)
		}

		if len(tx.MsgTx().TxOut) < int(eInfo.Index) {
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

		if bhash, err := client.GetBlockHash(int64(eInfo.Height)); err == nil {
			if dblock, err := client.GetDogecoinBlock(bhash.String()); err == nil {
				if !CheckTransactionisBlock(string(eInfo.ExtTxHash), dblock) {
					e := fmt.Sprintf("doge Transactionis %s not in BlockHeight %v", hex.EncodeToString(eInfo.ExtTxHash), eInfo.Height)
					return nil, errors.New(e)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}

		if eInfo.Amount.Int64() < 0 || tx.MsgTx().TxOut[eInfo.Index].Value != eInfo.Amount.Int64() {
			e := fmt.Sprintf("doge amount err ,[request:%v,doge:%v]", eInfo.Amount, tx.MsgTx().TxOut[eInfo.Index].Value)
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("doge PkScript err")
			return nil, errors.New(e)
		}

		dogeparams := &chaincfg.Params{
			LegacyScriptHashAddrID: 0x1e,
		}

		bai := eState.getBeaconAddress(eInfo.BID)
		if bai == nil {
			e := fmt.Sprintf("doge PkScript err")
			return nil, errors.New(e)
		}

		addr, err := czzutil.DecodeAddress(bai.Address, ev.Params)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Invalid address or key: " + err.Error(),
			}
		}

		addr2, err := czzutil.NewLegacyAddressScriptHashFromHash(addr.ScriptAddress(), dogeparams)
		if err != nil {
			e := fmt.Sprintf("doge addr err")
			return nil, errors.New(e)
		}

		_, pub2, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		addr3, err := czzutil.NewLegacyAddressScriptHashFromHash(pub2, dogeparams)
		if err != nil {
			e := fmt.Sprintf("doge addr err")
			return nil, errors.New(e)
		}

		addr2Str := addr2.String()
		addr3Str := addr3.String()
		fmt.Println("addr2Str", addr2Str, "addr3Str", addr3Str)

		//if addr3.String() != addr2.String() {
		//	e := fmt.Sprintf("doge dogePoolPub err")
		//	return nil, errors.New(e)
		//}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(eInfo.Height) > dogeMaturity {
				return pk, nil
			} else {
				e := fmt.Sprintf("doge Maturity err")
				return nil, errors.New(e)
			}
		}

	}
}

func (ev *ExChangeVerify) verifyLtcTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.LtcCoinRPC[rand.Intn(len(ev.LtcCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(string(eInfo.ExtTxHash)); err != nil {
		return nil, err
	} else {

		if len(tx.MsgTx().TxIn) < 1 || len(tx.MsgTx().TxOut) < 1 {
			e := fmt.Sprintf("ltc Transactionis in or out len < 0  in : %v , out : %v", len(tx.MsgTx().TxIn), len(tx.MsgTx().TxOut))
			return nil, errors.New(e)
		}

		if len(tx.MsgTx().TxOut) < int(eInfo.Index) {
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

		if bhash, err := client.GetBlockHash(int64(eInfo.Height)); err == nil {
			if dblock, err := client.GetDogecoinBlock(bhash.String()); err == nil {
				if !CheckTransactionisBlock(string(eInfo.ExtTxHash), dblock) {
					e := fmt.Sprintf("ltc Transactionis %s not in BlockHeight %v", hex.EncodeToString(eInfo.ExtTxHash), eInfo.Height)
					return nil, errors.New(e)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}

		if eInfo.Amount.Int64() < 0 || tx.MsgTx().TxOut[eInfo.Index].Value != eInfo.Amount.Int64() {
			e := fmt.Sprintf("ltc amount err ,[request:%v,ltc:%v]", eInfo.Amount, tx.MsgTx().TxOut[eInfo.Index].Value)
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("ltc PkScript err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
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

		bai := eState.getBeaconAddress(eInfo.BID)
		if bai == nil {
			e := fmt.Sprintf("ltc PkScript err")
			return nil, errors.New(e)
		}

		addr2, err := czzutil.DecodeAddress(bai.Address, ev.Params)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "ltc Invalid address or key: " + err.Error(),
			}
		}

		addr3, err := czzutil.NewLegacyAddressScriptHashFromHash(addr2.ScriptAddress(), ltcparams)
		if err != nil {
			e := fmt.Sprintf("ltc addr err")
			return nil, errors.New(e)
		}

		addr2Str := addr2.String()
		addr3Str := addr3.String()
		fmt.Println("addr2Str", addr2Str, "addr3Str", addr3Str)

		//if addr3.String() != addr2.String() {
		//	e := fmt.Sprintf("doge dogePoolPub err")
		//	return nil, errors.New(e)
		//}

		if addr.String() != ltcPoolAddr {
			e := fmt.Sprintf("ltc PoolAddr err")
			return nil, errors.New(e)
		}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(eInfo.Height) > ltcMaturity {
				return pk, nil
			} else {
				e := fmt.Sprintf("ltc Maturity err")
				return nil, errors.New(e)
			}
		}
	}
}

func (ev *ExChangeVerify) verifyBtcTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.LtcCoinRPC[rand.Intn(len(ev.LtcCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(string(eInfo.ExtTxHash)); err != nil {
		return nil, err
	} else {

		if len(tx.MsgTx().TxIn) < 1 || len(tx.MsgTx().TxOut) < 1 {
			e := fmt.Sprintf("btc Transactionis in or out len < 0  in : %v , out : %v", len(tx.MsgTx().TxIn), len(tx.MsgTx().TxOut))
			return nil, errors.New(e)
		}

		if len(tx.MsgTx().TxOut) < int(eInfo.Index) {
			return nil, errors.New("btc TxOut index err")
		}

		var pk []byte
		if tx.MsgTx().TxIn[0].Witness == nil {
			pk, err = txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript)
			if err != nil {
				e := fmt.Sprintf("btc ComputePk err %s", err)
				return nil, errors.New(e)
			}
		} else {
			pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[0].Witness)
			if err != nil {
				e := fmt.Sprintf("btc ComputeWitnessPk err %s", err)
				return nil, errors.New(e)
			}
		}

		if bhash, err := client.GetBlockHash(int64(eInfo.Height)); err == nil {
			if dblock, err := client.GetDogecoinBlock(bhash.String()); err == nil {
				if !CheckTransactionisBlock(string(eInfo.ExtTxHash), dblock) {
					e := fmt.Sprintf("btc Transactionis %s not in BlockHeight %v", hex.EncodeToString(eInfo.ExtTxHash), eInfo.Height)
					return nil, errors.New(e)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}

		if eInfo.Amount.Int64() < 0 || tx.MsgTx().TxOut[eInfo.Index].Value != eInfo.Amount.Int64() {
			e := fmt.Sprintf("btc amount err ,[request:%v,ltc:%v]", eInfo.Amount, tx.MsgTx().TxOut[eInfo.Index].Value)
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("btc PkScript err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		ltcparams := &chaincfg.Params{
			LegacyScriptHashAddrID: 0x32,
		}

		addr, err := czzutil.NewLegacyAddressScriptHashFromHash(pub, ltcparams)
		if err != nil {
			e := fmt.Sprintf("btc addr err")
			return nil, errors.New(e)
		}

		bai := eState.getBeaconAddress(eInfo.BID)
		if bai == nil {
			e := fmt.Sprintf("btc PkScript err")
			return nil, errors.New(e)
		}

		addr2, err := czzutil.DecodeAddress(bai.Address, ev.Params)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Invalid address or key: " + err.Error(),
			}
		}

		addr3, err := czzutil.NewLegacyAddressScriptHashFromHash(addr2.ScriptAddress(), ltcparams)
		if err != nil {
			e := fmt.Sprintf("btc addr err")
			return nil, errors.New(e)
		}

		addr2Str := addr2.String()
		addr3Str := addr3.String()
		fmt.Println("addr2Str", addr2Str, "addr3Str", addr3Str)

		//if addr3.String() != addr2.String() {
		//	e := fmt.Sprintf("doge dogePoolPub err")
		//	return nil, errors.New(e)
		//}

		if addr.String() != ltcPoolAddr {
			e := fmt.Sprintf("btc PoolAddr err")
			return nil, errors.New(e)
		}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(eInfo.Height) > ltcMaturity {
				return pk, nil
			} else {
				e := fmt.Sprintf("btc Maturity err")
				return nil, errors.New(e)
			}
		}
	}
}

func (ev *ExChangeVerify) verifyBchTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.LtcCoinRPC[rand.Intn(len(ev.LtcCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(string(eInfo.ExtTxHash)); err != nil {
		return nil, err
	} else {

		if len(tx.MsgTx().TxIn) < 1 || len(tx.MsgTx().TxOut) < 1 {
			e := fmt.Sprintf("Bch Transactionis in or out len < 0  in : %v , out : %v", len(tx.MsgTx().TxIn), len(tx.MsgTx().TxOut))
			return nil, errors.New(e)
		}

		if len(tx.MsgTx().TxOut) < int(eInfo.Index) {
			return nil, errors.New("Bch TxOut index err")
		}

		var pk []byte
		if tx.MsgTx().TxIn[0].Witness == nil {
			pk, err = txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript)
			if err != nil {
				e := fmt.Sprintf("Bch ComputePk err %s", err)
				return nil, errors.New(e)
			}
		} else {
			pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[0].Witness)
			if err != nil {
				e := fmt.Sprintf("Bch ComputeWitnessPk err %s", err)
				return nil, errors.New(e)
			}
		}

		if bhash, err := client.GetBlockHash(int64(eInfo.Height)); err == nil {
			if dblock, err := client.GetDogecoinBlock(bhash.String()); err == nil {
				if !CheckTransactionisBlock(string(eInfo.ExtTxHash), dblock) {
					e := fmt.Sprintf("Bch Transactionis %s not in BlockHeight %v", hex.EncodeToString(eInfo.ExtTxHash), eInfo.Height)
					return nil, errors.New(e)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}

		if eInfo.Amount.Int64() < 0 || tx.MsgTx().TxOut[eInfo.Index].Value != eInfo.Amount.Int64() {
			e := fmt.Sprintf("Bch amount err ,[request:%v,ltc:%v]", eInfo.Amount, tx.MsgTx().TxOut[eInfo.Index].Value)
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("Bch PkScript err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		ltcparams := &chaincfg.Params{
			LegacyScriptHashAddrID: 0x32,
		}

		addr, err := czzutil.NewLegacyAddressScriptHashFromHash(pub, ltcparams)
		if err != nil {
			e := fmt.Sprintf("Bch addr err")
			return nil, errors.New(e)
		}

		bai := eState.getBeaconAddress(eInfo.BID)
		if bai == nil {
			e := fmt.Sprintf("Bch PkScript err")
			return nil, errors.New(e)
		}

		addr2, err := czzutil.DecodeAddress(bai.Address, ev.Params)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Bch Invalid address or key: " + err.Error(),
			}
		}

		addr3, err := czzutil.NewLegacyAddressScriptHashFromHash(addr2.ScriptAddress(), ltcparams)
		if err != nil {
			e := fmt.Sprintf("Bch addr err")
			return nil, errors.New(e)
		}

		addr2Str := addr2.String()
		addr3Str := addr3.String()
		fmt.Println("addr2Str", addr2Str, "addr3Str", addr3Str)

		//if addr3.String() != addr2.String() {
		//	e := fmt.Sprintf("doge dogePoolPub err")
		//	return nil, errors.New(e)
		//}

		if addr.String() != ltcPoolAddr {
			e := fmt.Sprintf("Bch PoolAddr err")
			return nil, errors.New(e)
		}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(eInfo.Height) > ltcMaturity {
				return pk, nil
			} else {
				e := fmt.Sprintf("Bch Maturity err")
				return nil, errors.New(e)
			}
		}
	}
}

func (ev *ExChangeVerify) verifyBsvTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.LtcCoinRPC[rand.Intn(len(ev.LtcCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(string(eInfo.ExtTxHash)); err != nil {
		return nil, err
	} else {

		if len(tx.MsgTx().TxIn) < 1 || len(tx.MsgTx().TxOut) < 1 {
			e := fmt.Sprintf("Bsv Transactionis in or out len < 0  in : %v , out : %v", len(tx.MsgTx().TxIn), len(tx.MsgTx().TxOut))
			return nil, errors.New(e)
		}

		if len(tx.MsgTx().TxOut) < int(eInfo.Index) {
			return nil, errors.New("Bsv TxOut index err")
		}

		var pk []byte
		if tx.MsgTx().TxIn[0].Witness == nil {
			pk, err = txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript)
			if err != nil {
				e := fmt.Sprintf("Bsv ComputePk err %s", err)
				return nil, errors.New(e)
			}
		} else {
			pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[0].Witness)
			if err != nil {
				e := fmt.Sprintf("Bsv ComputeWitnessPk err %s", err)
				return nil, errors.New(e)
			}
		}

		if bhash, err := client.GetBlockHash(int64(eInfo.Height)); err == nil {
			if dblock, err := client.GetDogecoinBlock(bhash.String()); err == nil {
				if !CheckTransactionisBlock(string(eInfo.ExtTxHash), dblock) {
					e := fmt.Sprintf("Bsv Transactionis %s not in BlockHeight %v", hex.EncodeToString(eInfo.ExtTxHash), eInfo.Height)
					return nil, errors.New(e)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}

		if eInfo.Amount.Int64() < 0 || tx.MsgTx().TxOut[eInfo.Index].Value != eInfo.Amount.Int64() {
			e := fmt.Sprintf("Bsv amount err ,[request:%v,ltc:%v]", eInfo.Amount, tx.MsgTx().TxOut[eInfo.Index].Value)
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("Bsv PkScript err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		ltcparams := &chaincfg.Params{
			LegacyScriptHashAddrID: 0x32,
		}

		addr, err := czzutil.NewLegacyAddressScriptHashFromHash(pub, ltcparams)
		if err != nil {
			e := fmt.Sprintf("Bsv addr err")
			return nil, errors.New(e)
		}

		bai := eState.getBeaconAddress(eInfo.BID)
		if bai == nil {
			e := fmt.Sprintf("Bsv PkScript err")
			return nil, errors.New(e)
		}

		addr2, err := czzutil.DecodeAddress(bai.Address, ev.Params)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Bsv Invalid address or key: " + err.Error(),
			}
		}

		addr3, err := czzutil.NewLegacyAddressScriptHashFromHash(addr2.ScriptAddress(), ltcparams)
		if err != nil {
			e := fmt.Sprintf("Bsv addr err")
			return nil, errors.New(e)
		}

		addr2Str := addr2.String()
		addr3Str := addr3.String()
		fmt.Println("addr2Str", addr2Str, "addr3Str", addr3Str)

		//if addr3.String() != addr2.String() {
		//	e := fmt.Sprintf("doge dogePoolPub err")
		//	return nil, errors.New(e)
		//}

		if addr.String() != ltcPoolAddr {
			e := fmt.Sprintf("Bsv PoolAddr err")
			return nil, errors.New(e)
		}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(eInfo.Height) > ltcMaturity {
				return pk, nil
			} else {
				e := fmt.Sprintf("Bsv Maturity err")
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

func (ev *ExChangeVerify) VerifyBeaconRegistrationTx(tx *wire.MsgTx, eState *EntangleState) (*BeaconAddressInfo, error) {

	br, _ := IsBeaconRegistrationTx(tx, ev.Params)
	if br == nil {
		return nil, NoBeaconRegistration
	}

	if len(tx.TxIn) > 1 || len(tx.TxOut) > 3 || len(tx.TxOut) < 2 {
		e := fmt.Sprintf("BeaconRegistrationTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
		return nil, errors.New(e)
	}

	if _, ok := eState.EnInfos[br.Address]; ok {
		return nil, ErrRepeatRegister
	}

	addr, err := czzutil.NewLegacyAddressPubKeyHash(br.ToAddress, ev.Params)
	if err != nil {
		return nil, err
	}

	// Create a new script which pays to the provided address.
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(tx.TxOut[1].PkScript, pkScript) {
		e := fmt.Sprintf("tx.TxOut[1].PkScript err")
		return nil, errors.New(e)
	}

	if tx.TxOut[1].Value != br.StakingAmount.Int64() {
		e := fmt.Sprintf("tx.TxOut[1].Value err")
		return nil, errors.New(e)
	}

	toAddress := big.NewInt(0).SetBytes(br.ToAddress).Uint64()
	if toAddress < 10 || toAddress > 99 {
		e := fmt.Sprintf("toAddress err")
		return nil, errors.New(e)
	}

	if !validFee(big.NewInt(int64(br.Fee))) {
		e := fmt.Sprintf("Fee err")
		return nil, errors.New(e)
	}

	if !validKeepTime(big.NewInt(int64(br.KeepTime))) {
		e := fmt.Sprintf("KeepTime err")
		return nil, errors.New(e)
	}

	if br.StakingAmount.Cmp(MinStakingAmountForBeaconAddress) < 0 {
		e := fmt.Sprintf("StakingAmount err")
		return nil, errors.New(e)
	}

	if !ValidAssetType(br.AssetFlag) {
		e := fmt.Sprintf("AssetFlag err")
		return nil, errors.New(e)
	}

	for _, whiteAddress := range br.WhiteList {
		if !ValidPK(whiteAddress.Pk) {
			e := fmt.Sprintf("whiteAddress.Pk err")
			return nil, errors.New(e)
		}
		if !ValidAssetType(whiteAddress.AssetType) {
			e := fmt.Sprintf("whiteAddress.AssetType err")
			return nil, errors.New(e)
		}
	}

	if len(br.CoinBaseAddress) > MaxCoinBase {
		e := fmt.Sprintf("whiteAddress.AssetType > MaxCoinBase err")
		return nil, errors.New(e)
	}

	for _, coinBaseAddress := range br.CoinBaseAddress {
		if _, err := czzutil.DecodeAddress(coinBaseAddress, ev.Params); err != nil {
			e := fmt.Sprintf("DecodeCashAddress.AssetType err")
			return nil, errors.New(e)
		}
	}

	for _, v := range eState.EnInfos {
		if bytes.Equal(v.ToAddress, br.ToAddress) {
			e := fmt.Sprintf("ToAddress err")
			return nil, errors.New(e)
		}
	}

	return br, nil
}

func (ev *ExChangeVerify) VerifyAddBeaconPledgeTx(tx *wire.MsgTx, eState *EntangleState) (*AddBeaconPledge, error) {

	bp, _ := IsAddBeaconPledgeTx(tx, ev.Params)
	if bp == nil {
		return nil, NoAddBeaconPledge
	}

	if len(tx.TxIn) > 1 || len(tx.TxOut) > 3 || len(tx.TxOut) < 2 {
		e := fmt.Sprintf("BeaconRegistrationTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
		return nil, errors.New(e)
	}

	if _, ok := eState.EnInfos[bp.Address]; ok {
		return nil, ErrRepeatRegister
	}

	addr, err := czzutil.NewLegacyAddressPubKeyHash(bp.ToAddress, ev.Params)
	if err != nil {
		return nil, err
	}

	// Create a new script which pays to the provided address.
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(tx.TxOut[1].PkScript, pkScript) {
		e := fmt.Sprintf("tx.TxOut[1].PkScript err")
		return nil, errors.New(e)
	}

	if tx.TxOut[1].Value != bp.StakingAmount.Int64() {
		e := fmt.Sprintf("tx.TxOut[1].Value err")
		return nil, errors.New(e)
	}

	toAddress := big.NewInt(0).SetBytes(bp.ToAddress).Uint64()
	if toAddress < 10 || toAddress > 99 {
		e := fmt.Sprintf("toAddress err")
		return nil, errors.New(e)
	}

	if bp.StakingAmount.Cmp(MinStakingAmountForBeaconAddress) < 0 {
		e := fmt.Sprintf("StakingAmount err")
		return nil, errors.New(e)
	}

	for _, v := range eState.EnInfos {
		if bytes.Equal(v.ToAddress, bp.ToAddress) {
			e := fmt.Sprintf("ToAddress err")
			return nil, errors.New(e)
		}
	}

	return bp, nil
}

func (ev *ExChangeVerify) VerifyAddBeaconCoinbaseTx(tx *wire.MsgTx, eState *EntangleState) (*AddBeaconCoinbase, error) {

	bp, _ := IsAddBeaconCoinbaseTx(tx, ev.Params)
	if bp == nil {
		return nil, NoAddBeaconPledge
	}

	if len(tx.TxIn) > 1 || len(tx.TxOut) > 2 || len(tx.TxOut) < 1 {
		e := fmt.Sprintf("BeaconRegistrationTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
		return nil, errors.New(e)
	}

	if _, ok := eState.EnInfos[bp.Address]; ok {
		return nil, ErrRepeatRegister
	}

	addr, err := czzutil.NewLegacyAddressPubKeyHash(bp.ToAddress, ev.Params)
	if err != nil {
		return nil, err
	}

	// Create a new script which pays to the provided address.
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(tx.TxOut[1].PkScript, pkScript) {
		e := fmt.Sprintf("tx.TxOut[1].PkScript err")
		return nil, errors.New(e)
	}

	toAddress := big.NewInt(0).SetBytes(bp.ToAddress).Uint64()
	if toAddress < 10 || toAddress > 99 {
		e := fmt.Sprintf("toAddress err")
		return nil, errors.New(e)
	}

	for _, v := range eState.EnInfos {
		if bytes.Equal(v.ToAddress, bp.ToAddress) {
			e := fmt.Sprintf("ToAddress err")
			return nil, errors.New(e)
		}
	}

	return bp, nil
}

func (ev *ExChangeVerify) VerifyBurn(info *BurnTxInfo, eState *EntangleState) error {
	// 1. check the from address is equal beacon address
	// 2. check the to address is equal the user's address within the info obj
	// 3. check the amount from the tx(outsize tx) eq the amount(in info)
	return nil
}

func (ev *ExChangeVerify) VerifyBurnProof(info *BurnProofInfo, eState *EntangleState) error {
	// 1. check the from address is equal beacon address
	// 2. check the to address is equal the user's address within the info obj
	// 3. check the amount from the tx(outsize tx) eq the amount(in info)
	return nil
}
