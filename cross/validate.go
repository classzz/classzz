package cross

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
	"github.com/classzz/czzutil/base58"
)

var (
	ErrHeightTooClose = errors.New("the block heigth to close for entangling")
	ErrStakingAmount  = errors.New("StakingAmount Less than minimum 1000000 czz")

	ltcparams = &chaincfg.Params{
		LegacyPubKeyHashAddrID: 0x30,
		LegacyScriptHashAddrID: 0x32,
	}

	dogeparams = &chaincfg.Params{
		LegacyPubKeyHashAddrID: 0x1e,
		LegacyScriptHashAddrID: 0x16,
	}
)

const (
	dogeMaturity = 2
	ltcMaturity  = 0
	btcMaturity  = 0
	bchMaturity  = 12
	bsvMaturity  = 12
	usdtMaturity = 12
)

type ExChangeVerify struct {
	DogeCoinRPC []*rpcclient.Client
	LtcCoinRPC  []*rpcclient.Client
	BtcCoinRPC  []*rpcclient.Client
	BchCoinRPC  []*rpcclient.Client
	BsvCoinRPC  []*rpcclient.Client
	UsdtCoinRPC []*rpcclient.Client
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
	einfo, _ := IsExChangeTx(tx)
	if einfo == nil {
		return nil, errors.New("not entangle tx")
	}
	pairs := make([]*TuplePubIndex, 0)
	amount := int64(0)
	if ev.Cache != nil {
		if ok := ev.Cache.FetchExChangeUtxoView(einfo); ok {
			errStr := fmt.Sprintf("[txid:%s, height:%v]", einfo.ExtTxHash, einfo.Height)
			return nil, errors.New("txid has already entangle:" + errStr)
		}
		amount += tx.TxOut[0].Value
	}

	if pub, err := ev.verifyTx(einfo, eState); err != nil {
		errStr := fmt.Sprintf("[txid:%s, height:%v]", einfo.ExtTxHash, einfo.Index)
		return nil, errors.New("txid verify failed:" + errStr + " err:" + err.Error())
	} else {
		pairs = append(pairs, &TuplePubIndex{
			AssetType: einfo.AssetType,
			Index:     0,
			Pub:       pub,
		})
	}

	return pairs, nil
}

func (ev *ExChangeVerify) VerifyFastExChangeTx(tx *wire.MsgTx, eState *EntangleState) error {
	/*
		1. check entangle tx struct
		2. check the repeat tx
		3. check the correct tx
		4. check the pool reserve enough reward
	*/
	einfo, _, _ := IsFastExChangeTx(tx, ev.Params)
	if einfo == nil {
		return errors.New("not entangle tx")
	}

	if _, err := ev.verifyTx(einfo, eState); err != nil {
		errStr := fmt.Sprintf("[txid:%s, height:%v]", einfo.ExtTxHash, einfo.Index)
		return errors.New("txid verify failed:" + errStr + " err:" + err.Error())
	}

	if err := ev.VerifyBurn(tx, eState); err != nil {
		errStr := fmt.Sprintf("[txid:%s, height:%v]", einfo.ExtTxHash, einfo.Index)
		return errors.New("txid verify failed:" + errStr + " err:" + err.Error())
	}

	return nil
}

func (ev *ExChangeVerify) verifyTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {
	switch eInfo.AssetType {
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
	return nil, fmt.Errorf("verifyTx AssetType is %v", eInfo.AssetType)
}

func (ev *ExChangeVerify) verifyDogeTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.DogeCoinRPC[rand.Intn(len(ev.DogeCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(eInfo.ExtTxHash); err != nil {
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
				if !CheckTransactionisBlock(eInfo.ExtTxHash, dblock) {
					e := fmt.Sprintf("doge Transactionis %s not in BlockHeight %v", eInfo.ExtTxHash, eInfo.Height)
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

		reserve := eState.GetEntangleAmountByAll(uint8(ExpandedTxEntangle_Doge))
		sendAmount, err := calcEntangleAmount(reserve, eInfo.Amount, uint8(ExpandedTxEntangle_Doge))

		bai := eState.getBeaconAddress(eInfo.BeaconID)
		if bai == nil {
			e := fmt.Sprintf("doge PkScript err")
			return nil, errors.New(e)
		}

		uinfos := eState.EnUserExChangeInfos[bai.BeaconID]

		RedeemableAmount := uinfos.GetRedeemableAmountAll()
		BurnAmount := uinfos.GetBurnAmount()
		BurnAmount = big.NewInt(0).Mul(BurnAmount, big.NewInt(2))

		ExChangeAmount := big.NewInt(0).Add(RedeemableAmount, BurnAmount)
		ExChangeAmount = big.NewInt(0).Add(ExChangeAmount, sendAmount)
		ExChangeStakingAmount := big.NewInt(0).Sub(bai.StakingAmount, MinStakingAmountForBeaconAddress)

		if ExChangeAmount.Cmp(ExChangeStakingAmount) > 0 {
			e := fmt.Sprintf("doge ExChangeAmount > ExChangeStakingAmount")
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("doge PkScript err")
			return nil, errors.New(e)
		}

		addr, err := czzutil.NewLegacyAddressPubKeyHash(czzutil.Hash160(bai.PubKey), dogeparams)
		if err != nil {
			e := fmt.Sprintf("doge addr err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		addr2, err := czzutil.NewLegacyAddressPubKeyHash(pub, dogeparams)
		if err != nil {
			e := fmt.Sprintf("doge addr err")
			return nil, errors.New(e)
		}

		addrStr := addr.String()
		addr2Str := addr2.String()

		if addr.String() != addr2.String() {
			return nil, fmt.Errorf("doge PoolPub err add1 %s add2 %s", addrStr, addr2Str)
		}

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
	if tx, err := client.GetWitnessRawTransaction(eInfo.ExtTxHash); err != nil {
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
				if !CheckTransactionisBlock(eInfo.ExtTxHash, dblock) {
					e := fmt.Sprintf("ltc Transactionis %s not in BlockHeight %v", eInfo.ExtTxHash, eInfo.Height)
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

		reserve := eState.GetEntangleAmountByAll(uint8(ExpandedTxEntangle_Ltc))
		sendAmount, err := calcEntangleAmount(reserve, eInfo.Amount, uint8(ExpandedTxEntangle_Ltc))

		bai := eState.getBeaconAddress(eInfo.BeaconID)
		if bai == nil {
			e := fmt.Sprintf("ltc PkScript err")
			return nil, errors.New(e)
		}

		ExChangeAmount := big.NewInt(0).Add(bai.EntangleAmount, sendAmount)
		ExChangeStakingAmount := big.NewInt(0).Sub(bai.StakingAmount, MinStakingAmountForBeaconAddress)

		if ExChangeAmount.Cmp(ExChangeStakingAmount) > 0 {
			e := fmt.Sprintf("ltc ExChangeAmount > ExChangeStakingAmount")
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("ltc PkScript err")
			return nil, errors.New(e)
		}

		addr, err := czzutil.NewLegacyAddressPubKeyHash(czzutil.Hash160(bai.PubKey), ltcparams)
		if err != nil {
			e := fmt.Sprintf("ltc addr err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		addr2, err := czzutil.NewLegacyAddressPubKeyHash(pub, ltcparams)
		if err != nil {
			e := fmt.Sprintf("ltc addr err")
			return nil, errors.New(e)
		}

		addrStr := addr.String()
		addr2Str := addr2.String()
		fmt.Println("addr2Str", addrStr, "addr3Str", addr2Str)

		if addr.String() != addr2.String() {
			return nil, fmt.Errorf("ltc ltcPoolPub err add1 %s add2 %s", addrStr, addr2Str)
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
	client := ev.BtcCoinRPC[rand.Intn(len(ev.BtcCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(eInfo.ExtTxHash); err != nil {
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
				if !CheckTransactionisBlock(eInfo.ExtTxHash, dblock) {
					e := fmt.Sprintf("btc Transactionis %s not in BlockHeight %v", eInfo.ExtTxHash, eInfo.Height)
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

		reserve := eState.GetEntangleAmountByAll(uint8(ExpandedTxEntangle_Btc))
		sendAmount, err := calcEntangleAmount(reserve, eInfo.Amount, uint8(ExpandedTxEntangle_Btc))

		bai := eState.getBeaconAddress(eInfo.BeaconID)
		if bai == nil {
			e := fmt.Sprintf("btc PkScript err")
			return nil, errors.New(e)
		}

		ExChangeAmount := big.NewInt(0).Add(bai.EntangleAmount, sendAmount)
		ExChangeStakingAmount := big.NewInt(0).Sub(bai.StakingAmount, MinStakingAmountForBeaconAddress)

		if ExChangeAmount.Cmp(ExChangeStakingAmount) > 0 {
			e := fmt.Sprintf("btc ExChangeAmount > ExChangeStakingAmount")
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("btc PkScript err")
			return nil, errors.New(e)
		}

		addr, err := czzutil.NewLegacyAddressPubKeyHash(czzutil.Hash160(bai.PubKey), ev.Params)
		if err != nil {
			e := fmt.Sprintf("btc addr err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		addr2, err := czzutil.NewLegacyAddressPubKeyHash(pub, ev.Params)
		if err != nil {
			e := fmt.Sprintf("btc addr err")
			return nil, errors.New(e)
		}

		addrStr := addr.String()
		addr2Str := addr2.String()

		if addr.String() != addr2.String() {
			return nil, fmt.Errorf("btc PoolPub err add1 %s add2 %s", addrStr, addr2Str)
		}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(eInfo.Height) > btcMaturity {
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
	client := ev.BchCoinRPC[rand.Intn(len(ev.BchCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(eInfo.ExtTxHash); err != nil {
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
				if !CheckTransactionisBlock(eInfo.ExtTxHash, dblock) {
					e := fmt.Sprintf("Bch Transactionis %s not in BlockHeight %v", eInfo.ExtTxHash, eInfo.Height)
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

		reserve := eState.GetEntangleAmountByAll(uint8(ExpandedTxEntangle_Bch))
		sendAmount, err := calcEntangleAmount(reserve, eInfo.Amount, uint8(ExpandedTxEntangle_Bch))

		bai := eState.getBeaconAddress(eInfo.BeaconID)
		if bai == nil {
			e := fmt.Sprintf("bch PkScript err")
			return nil, errors.New(e)
		}

		ExChangeAmount := big.NewInt(0).Add(bai.EntangleAmount, sendAmount)
		ExChangeStakingAmount := big.NewInt(0).Sub(bai.StakingAmount, MinStakingAmountForBeaconAddress)

		if ExChangeAmount.Cmp(ExChangeStakingAmount) > 0 {
			e := fmt.Sprintf("bch ExChangeAmount > ExChangeStakingAmount")
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("Bch PkScript err")
			return nil, errors.New(e)
		}

		addr, err := czzutil.NewLegacyAddressPubKeyHash(czzutil.Hash160(bai.PubKey), ev.Params)
		if err != nil {
			e := fmt.Sprintf("bch addr err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		addr2, err := czzutil.NewLegacyAddressPubKeyHash(pub, ev.Params)
		if err != nil {
			e := fmt.Sprintf("bch addr err")
			return nil, errors.New(e)
		}

		addrStr := addr.String()
		addr2Str := addr2.String()

		if addr.String() != addr2.String() {
			return nil, fmt.Errorf("bch PoolPub err add1 %s add2 %s", addrStr, addr2Str)
		}

		//if addr.String() != ltcPoolAddr {
		//	e := fmt.Sprintf("Bch PoolAddr err")
		//	return nil, errors.New(e)
		//}

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
	client := ev.BsvCoinRPC[rand.Intn(len(ev.BsvCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(eInfo.ExtTxHash); err != nil {
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
				if !CheckTransactionisBlock(eInfo.ExtTxHash, dblock) {
					e := fmt.Sprintf("Bsv Transactionis %s not in BlockHeight %v", eInfo.ExtTxHash, eInfo.Height)
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

		reserve := eState.GetEntangleAmountByAll(uint8(ExpandedTxEntangle_Bsv))
		sendAmount, err := calcEntangleAmount(reserve, eInfo.Amount, uint8(ExpandedTxEntangle_Bsv))

		bai := eState.getBeaconAddress(eInfo.BeaconID)
		if bai == nil {
			e := fmt.Sprintf("Bsv PkScript err")
			return nil, errors.New(e)
		}

		ExChangeAmount := big.NewInt(0).Add(bai.EntangleAmount, sendAmount)
		ExChangeStakingAmount := big.NewInt(0).Sub(bai.StakingAmount, MinStakingAmountForBeaconAddress)

		if ExChangeAmount.Cmp(ExChangeStakingAmount) > 0 {
			e := fmt.Sprintf("Bsv ExChangeAmount > ExChangeStakingAmount")
			return nil, errors.New(e)
		}

		ScriptClass := txscript.GetScriptClass(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if ScriptClass != txscript.PubKeyHashTy && ScriptClass != txscript.ScriptHashTy {
			e := fmt.Sprintf("Bsv PkScript err")
			return nil, errors.New(e)
		}

		addr, err := czzutil.NewLegacyAddressPubKeyHash(czzutil.Hash160(bai.PubKey), ev.Params)
		if err != nil {
			e := fmt.Sprintf("Bsv addr err")
			return nil, errors.New(e)
		}

		_, pub, err := txscript.ExtractPkScriptPub(tx.MsgTx().TxOut[eInfo.Index].PkScript)
		if err != nil {
			return nil, err
		}

		addr2, err := czzutil.NewLegacyAddressPubKeyHash(pub, ev.Params)
		if err != nil {
			e := fmt.Sprintf("Bsv addr err")
			return nil, errors.New(e)
		}

		addrStr := addr.String()
		addr2Str := addr2.String()

		if addr.String() != addr2.String() {
			return nil, fmt.Errorf("bsv PoolPub err add1 %s add2 %s", addrStr, addr2Str)
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

func (ev *ExChangeVerify) verifyUsdtTx(eInfo *ExChangeTxInfo, eState *EntangleState) ([]byte, error) {

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.UsdtCoinRPC[rand.Intn(len(ev.UsdtCoinRPC))]

	// Get the current block count.
	if tx, err := client.GetWitnessRawTransaction(eInfo.ExtTxHash); err != nil {
		return nil, err
	} else {

		if len(tx.MsgTx().TxIn) < 1 || len(tx.MsgTx().TxOut) < 1 {
			e := fmt.Sprintf("usdt Transactionis in or out len < 0  in : %v , out : %v", len(tx.MsgTx().TxIn), len(tx.MsgTx().TxOut))
			return nil, errors.New(e)
		}

		if len(tx.MsgTx().TxOut) < int(eInfo.Index) {
			return nil, errors.New("usdt TxOut index err")
		}

		var pk []byte
		if tx.MsgTx().TxIn[0].Witness == nil {
			pk, err = txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript)
			if err != nil {
				e := fmt.Sprintf("usdt ComputePk err %s", err)
				return nil, errors.New(e)
			}
		} else {
			pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[0].Witness)
			if err != nil {
				e := fmt.Sprintf("usdt ComputeWitnessPk err %s", err)
				return nil, errors.New(e)
			}
		}

		bai := eState.getBeaconAddress(eInfo.BeaconID)
		if bai == nil {
			e := fmt.Sprintf("usdt PkScript err")
			return nil, errors.New(e)
		}

		ExtTxHash, err := chainhash.NewHashFromStr(eInfo.ExtTxHash)
		if err != nil {
			return nil, err
		}

		if tx2, err := client.OmniGetTransactionResult(ExtTxHash); err != nil {
			return nil, err
		} else {

			if tx2.PropertyId != 31 {
				return nil, err
			}

			if tx2.TypeInt != 0 {
				return nil, err
			}

			ex_amount, err := strconv.ParseFloat(tx2.Amount, 64)
			if err != nil {
				return nil, err
			}

			if eInfo.Amount.Int64() < 0 || int64(ex_amount*100000000) != eInfo.Amount.Int64() {
				e := fmt.Sprintf("usdt amount err ,[request:%v,ltc:%v]", eInfo.Amount, tx.MsgTx().TxOut[eInfo.Index].Value)
				return nil, errors.New(e)
			}

			addr, err := czzutil.NewLegacyAddressPubKeyHash(czzutil.Hash160(bai.PubKey), ev.Params)
			if err != nil {
				e := fmt.Sprintf("usdt addr err")
				return nil, errors.New(e)
			}

			addrStr := addr.String()
			if addr.String() != tx2.ReferenceAddress {
				return nil, fmt.Errorf("usdt PoolPub err add1 %s add2 %s", addrStr, tx2.ReferenceAddress)
			}

		}

		if bhash, err := client.GetBlockHash(int64(eInfo.Height)); err == nil {
			if dblock, err := client.GetDogecoinBlock(bhash.String()); err == nil {
				if !CheckTransactionisBlock(eInfo.ExtTxHash, dblock) {
					e := fmt.Sprintf("usdt Transactionis %s not in BlockHeight %v", eInfo.ExtTxHash, eInfo.Height)
					return nil, errors.New(e)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}

		reserve := eState.GetEntangleAmountByAll(uint8(ExpandedTxEntangle_Usdt))
		sendAmount, err := calcEntangleAmount(reserve, eInfo.Amount, uint8(ExpandedTxEntangle_Usdt))

		ExChangeAmount := big.NewInt(0).Add(bai.EntangleAmount, sendAmount)
		ExChangeStakingAmount := big.NewInt(0).Sub(bai.StakingAmount, MinStakingAmountForBeaconAddress)

		if ExChangeAmount.Cmp(ExChangeStakingAmount) > 0 {
			e := fmt.Sprintf("usdt ExChangeAmount > ExChangeStakingAmount")
			return nil, errors.New(e)
		}

		if count, err := client.GetBlockCount(); err != nil {
			return nil, err
		} else {
			if count-int64(eInfo.Height) > usdtMaturity {
				return pk, nil
			} else {
				e := fmt.Sprintf("usdt Maturity err")
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

func (ev *ExChangeVerify) VerifyBeaconRegistrationTx2(tx *wire.MsgTx, eState *EntangleState2) (*BeaconAddressInfo, error) {

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

	if !validKeepTime(big.NewInt(int64(br.KeepBlock))) {
		e := fmt.Sprintf("KeepTime err")
		return nil, errors.New(e)
	}

	if br.StakingAmount.Cmp(MinStakingAmountForBeaconAddress) < 0 {
		e := fmt.Sprintf("StakingAmount err")
		return nil, errors.New(e)
	}

	if !ValidAssetType(uint8(br.AssetFlag)) {
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

func (ev *ExChangeVerify) VerifyBeaconRegistrationTx(bai *BeaconAddressInfo, eState *EntangleState) error {

	if _, ok := eState.EnInfos[bai.Address]; ok {
		return ErrRepeatRegister
	}

	toAddress := big.NewInt(0).SetBytes(bai.ToAddress).Uint64()
	if toAddress < 10 || toAddress > 99 {
		e := fmt.Sprintf("toAddress err")
		return errors.New(e)
	}

	if !validFee(big.NewInt(int64(bai.Fee))) {
		e := fmt.Sprintf("Fee err")
		return errors.New(e)
	}

	if !validKeepTime(big.NewInt(int64(bai.KeepBlock))) {
		e := fmt.Sprintf("KeepTime err")
		return errors.New(e)
	}

	if bai.StakingAmount == nil || bai.StakingAmount.Cmp(MinStakingAmountForBeaconAddress) < 0 {
		return ErrStakingAmount
	}

	if !ValidAssetFlag(bai.AssetFlag) {
		e := fmt.Sprintf("AssetFlag err")
		return errors.New(e)
	}

	for _, whiteAddress := range bai.WhiteList {
		if !ValidPK(whiteAddress.Pk) {
			e := fmt.Sprintf("whiteAddress.Pk err")
			return errors.New(e)
		}
		if !ValidAssetType(whiteAddress.AssetType) {
			e := fmt.Sprintf("whiteAddress.AssetType err")
			return errors.New(e)
		}
	}

	if len(bai.CoinBaseAddress) > MaxCoinBase {
		e := fmt.Sprintf("whiteAddress.AssetType > MaxCoinBase err")
		return errors.New(e)
	}

	for _, coinBaseAddress := range bai.CoinBaseAddress {
		if _, err := czzutil.DecodeAddress(coinBaseAddress, ev.Params); err != nil {
			e := fmt.Sprintf("DecodeCashAddress.AssetType err")
			return errors.New(e)
		}
	}

	for _, v := range eState.EnInfos {
		if bytes.Equal(v.ToAddress, bai.ToAddress) {
			e := fmt.Sprintf("ToAddress err")
			return errors.New(e)
		}
	}

	return nil
}

func (ev *ExChangeVerify) VerifyAddBeaconPledgeTx2(tx *wire.MsgTx, eState *EntangleState2) (*AddBeaconPledge, error) {

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

func (ev *ExChangeVerify) VerifyUpdateBeaconCoinbaseTx(tx *wire.MsgTx, eState *EntangleState) (*UpdateBeaconCoinbase, error) {

	bp, _ := IsUpdateBeaconCoinbaseTx(tx, ev.Params)
	if bp == nil {
		return nil, NoUpdateBeaconCoinbase
	}

	if len(tx.TxIn) > 1 || len(tx.TxOut) > 2 || len(tx.TxOut) < 1 {
		e := fmt.Sprintf("BeaconRegistrationTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
		return nil, errors.New(e)
	}

	if _, ok := eState.EnInfos[bp.Address]; ok {
		return nil, ErrRepeatRegister
	}

	if len(bp.CoinBaseAddress) > MaxCoinBase {
		e := fmt.Sprintf("whiteAddress.AssetType > MaxCoinBase err")
		return nil, errors.New(e)
	}

	for _, coinBaseAddress := range bp.CoinBaseAddress {
		if _, err := czzutil.DecodeAddress(coinBaseAddress, ev.Params); err != nil {
			e := fmt.Sprintf("DecodeCashAddress.AssetType err")
			return nil, errors.New(e)
		}
	}

	return bp, nil
}

func (ev *ExChangeVerify) VerifyUpdateBeaconFreeQuotaTx(tx *wire.MsgTx, eState *EntangleState) (*UpdateBeaconFreeQuota, error) {

	bp, _ := IsUpdateBeaconFreeQuotaTx(tx, ev.Params)
	if bp == nil {
		return nil, NoUpdateBeaconFreeQuota
	}

	if len(tx.TxIn) > 1 || len(tx.TxOut) > 2 || len(tx.TxOut) < 1 {
		e := fmt.Sprintf("BeaconRegistrationTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
		return nil, errors.New(e)
	}

	bai := eState.EnInfos[bp.Address]

	if bai == nil {
		return nil, ErrRepeatRegister
	}

	Free := eState.BaExInfo[bai.BeaconID]

	if Free == nil {
		return nil, ErrRepeatRegister
	}

	if len(bp.FreeQuota) > MaxCoinType {
		e := fmt.Sprintf("whiteAddress.AssetType > MaxCoinBase err")
		return nil, errors.New(e)
	}
	quotaSum := uint64(0)
	for _, quota := range bp.FreeQuota {
		quotaSum = quotaSum + quota
	}

	if quotaSum > 100 {
		e := fmt.Sprintf("quotaSum > 100 err")
		return nil, errors.New(e)
	}

	return bp, nil
}

func (ev *ExChangeVerify) VerifyBurn(tx *wire.MsgTx, eState *EntangleState) error {
	// 1. check the from address is equal beacon address
	// 2. check the to address is equal the user's address within the info obj
	// 3. check the amount from the tx(outsize tx) eq the amount(in info)

	info, _ := IsBurnTx(tx, ev.Params)
	if info == nil {
		return NoBurnTx
	}

	bai := eState.EnInfos[info.Address]
	if bai != nil {
		baseAmount := big.NewInt(0)
		ebInfo := eState.BaExInfo[bai.BeaconID]
		for _, fqinfo := range ebInfo.Free.Items {
			if fqinfo.AssetType == uint8(info.AssetType) {
				baseAmount = fqinfo.Amount
				break
			}
		}

		//
		if baseAmount.Cmp(info.Amount) < 0 {
			return errors.New("VerifyBurn baseAmount > Amount")
		}

		_, version, err := base58.CheckDecode(info.ToAddress)
		if err != nil {
			return fmt.Errorf("VerifyBurn info.ToAddress is err %s", err)
		}

		switch info.AssetType {

		case ExpandedTxEntangle_Doge:
			if version != dogeparams.LegacyPubKeyHashAddrID {
				return fmt.Errorf("VerifyBurn info.ToAddress is version %v", version)
			}

		case ExpandedTxEntangle_Ltc:
			if version != ltcparams.LegacyPubKeyHashAddrID {
				return fmt.Errorf("VerifyBurn info.ToAddress is version %v", version)
			}

		default:
			if version != ev.Params.LegacyPubKeyHashAddrID {
				return fmt.Errorf("VerifyBurn info.ToAddress is version %v", version)
			}
		}
		return nil
	}

	uei := eState.EnUserExChangeInfos[info.BeaconID]
	if uei == nil {
		return errors.New("EnUserExChangeInfos is nil")
	}

	es := uei[info.Address]
	if es == nil {
		return errors.New("UserEntangleInfos is nil")
	}

	if info.Amount.Cmp(es.MaxRedeem) > 0 {
		return errors.New("Amount < MaxRedeem")
	}

	return nil
}

func (ev *ExChangeVerify) VerifyBurnProofBeacon(info *BurnProofInfo, eState *EntangleState, curHeight uint64) (uint64, *BurnItem, error) {
	// 1. check the from address is equal beacon address
	// 2. check the to address is equal the user's address within the info obj
	// 3. check the amount from the tx(outsize tx) eq the amount(in info)

	uei := eState.EnUserExChangeInfos[info.BeaconID]
	if uei == nil {
		return 0, nil, errors.New("VerifyBurnProofBeacon EnUserExChangeInfos is nil")
	}
	var client *rpcclient.Client
	switch info.AssetType {
	case ExpandedTxEntangle_Doge:
		client = ev.DogeCoinRPC[rand.Intn(len(ev.DogeCoinRPC))]
	case ExpandedTxEntangle_Ltc:
		client = ev.LtcCoinRPC[rand.Intn(len(ev.LtcCoinRPC))]
	case ExpandedTxEntangle_Btc:
		client = ev.BtcCoinRPC[rand.Intn(len(ev.BtcCoinRPC))]
	case ExpandedTxEntangle_Bch:
		client = ev.BchCoinRPC[rand.Intn(len(ev.BchCoinRPC))]
	case ExpandedTxEntangle_Bsv:
		client = ev.BsvCoinRPC[rand.Intn(len(ev.BsvCoinRPC))]
	case ExpandedTxEntangle_Usdt:
		client = ev.UsdtCoinRPC[rand.Intn(len(ev.UsdtCoinRPC))]
	}

	if client == nil {
		e := fmt.Sprintf("VerifyBurnProofBeacon client is nil , AssetType = %v", info.AssetType)
		return 0, nil, errors.New(e)
	}

	bai := eState.getBeaconAddress(info.BeaconID)
	if bai == nil {
		e := fmt.Sprintf("VerifyBurnProofBeacon BeaconAddress is nil , BeaconID = %v", info.BeaconID)
		return 0, nil, errors.New(e)
	}

	etx, bAdd, err := ev.GetTxInAddress(info, client)
	if err != nil {
		return 0, nil, err
	}

	paddr, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(bai.PubKey), ev.Params)
	if err != nil {
		e := fmt.Sprintf("NewAddressPubKeyHash Address %s != BeaconAddress %s", hex.EncodeToString(bai.PubKey), bAdd.String())
		return 0, nil, errors.New(e)
	}

	if paddr.String() != bAdd.String() {
		e := fmt.Sprintf("VerifyBurnProof Address %s != BeaconAddress %s", hex.EncodeToString(bai.PubKey), bAdd.String())
		return 0, nil, errors.New(e)
	}
	if ExpandedTxEntangle_Usdt == info.AssetType {

		exHash, err := chainhash.NewHashFromStr(info.TxHash)
		if err != nil {
			e := fmt.Sprintf("VerifyBurnProof Address %s != BeaconAddress %s", hex.EncodeToString(bai.PubKey), bAdd.String())
			return 0, nil, errors.New(e)
		}

		omresult, err := client.OmniGetTransactionResult(exHash)
		if err != nil {
			e := fmt.Sprintf("VerifyBurnProof Address %s != BeaconAddress %s", hex.EncodeToString(bai.PubKey), bAdd.String())
			return 0, nil, errors.New(e)
		}

		ex_amount, err := strconv.ParseFloat(omresult.Amount, 64)
		if err != nil {
			return 0, nil, err
		}

		if int64(ex_amount*100000000) != info.Amount.Int64() {
			e := fmt.Sprintf("VerifyBurnProof Value != Amount")
			return 0, nil, errors.New(e)
		}

	} else {
		if etx.MsgTx().TxOut[info.OutIndex].Value != info.Amount.Int64() {
			e := fmt.Sprintf("VerifyBurnProof Value != Amount")
			return 0, nil, errors.New(e)
		}
	}

	outHeight := uint64(0)
	var bi *BurnItem
	for addr, userEntity := range uei {
		if info.Address == addr {
			bi, err = userEntity.verifyBurnProof(info, outHeight, curHeight)
			if err != nil {
				return 0, nil, err
			}
		} else {
			return 0, nil, ErrNotMatchUser
		}
	}

	return 0, bi, nil
}

func (ev *ExChangeVerify) GetTxInAddress(info *BurnProofInfo, client *rpcclient.Client) (*czzutil.Tx, czzutil.Address, error) {

	if tx, err := client.GetWitnessRawTransaction(info.TxHash); err != nil {
		return nil, nil, err
	} else {
		var pk []byte
		if tx.MsgTx().TxIn[0].Witness == nil {
			pk, err = txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript)
			if err != nil {
				e := fmt.Sprintf("btc ComputePk err %s", err)
				return nil, nil, errors.New(e)
			}
		} else {
			pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[0].Witness)
			if err != nil {
				e := fmt.Sprintf("btc ComputeWitnessPk err %s", err)
				return nil, nil, errors.New(e)
			}
		}

		addrs, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), ev.Params)
		if err != nil {
			e := fmt.Sprintf("addr err")
			return nil, nil, errors.New(e)
		}

		return tx, addrs, nil
	}
}

func (ev *ExChangeVerify) VerifyWhiteListProof(info *WhiteListProof, state *EntangleState) error {

	var client *rpcclient.Client
	switch info.AssetType {
	case ExpandedTxEntangle_Doge:
		client = ev.DogeCoinRPC[rand.Intn(len(ev.DogeCoinRPC))]
	case ExpandedTxEntangle_Ltc:
		client = ev.LtcCoinRPC[rand.Intn(len(ev.LtcCoinRPC))]
	case ExpandedTxEntangle_Btc:
		client = ev.BtcCoinRPC[rand.Intn(len(ev.BtcCoinRPC))]
	case ExpandedTxEntangle_Bch:
		client = ev.BchCoinRPC[rand.Intn(len(ev.BchCoinRPC))]
	case ExpandedTxEntangle_Bsv:
		client = ev.BsvCoinRPC[rand.Intn(len(ev.BsvCoinRPC))]
	case ExpandedTxEntangle_Usdt:
		client = ev.UsdtCoinRPC[rand.Intn(len(ev.UsdtCoinRPC))]
	}

	bai := state.getBeaconByID(info.BeaconID)
	if bai == nil {
		return errors.New("VerifyBurnProof EnEntitys is nil")
	}

	_, in, out, err := ev.GetTxInPk(info, client)
	if !bytes.Equal(bai.PubKey, in) {
		return fmt.Errorf("address err %s", err)
	}

	whiteList := state.GetWhiteList(info.BeaconID)
	for _, wu := range whiteList {

		addrs, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(wu.Pk), ev.Params)
		if err != nil {
			return fmt.Errorf("NewAddressPubKeyHash err")
		}

		if wu.AssetType == info.AssetType && bytes.Equal(out, addrs.ScriptAddress()) && !bytes.Equal(out, czzutil.Hash160(in)) {
			return fmt.Errorf("Illegal transfer err %s", err)
		}
	}

	infos := state.EnUserExChangeInfos[info.BeaconID]

	for k, v := range infos {
		addr, _ := czzutil.DecodeAddress(k, &chaincfg.MainNetParams)
		if bytes.Equal(out, addr.ScriptAddress()) {
			for _, burn := range v.BurnAmounts {
				for _, item := range burn.Items {
					if item.RedeemState == 0 {
						return fmt.Errorf("Illegal transfer err %s", err)
					}
				}
			}
		}
	}

	return nil
}

func (ev *ExChangeVerify) GetTxInPk(info *WhiteListProof, client *rpcclient.Client) (*czzutil.Tx, []byte, []byte, error) {

	if tx, err := client.GetWitnessRawTransaction(info.TxHash); err != nil {
		return nil, nil, nil, err
	} else {
		var pk []byte
		if tx.MsgTx().TxIn[info.InIndex].Witness == nil {
			pk, err = txscript.ComputePk(tx.MsgTx().TxIn[info.InIndex].SignatureScript)
			if err != nil {
				e := fmt.Sprintf("btc ComputePk err %s", err)
				return nil, nil, nil, errors.New(e)
			}
		} else {
			pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[info.InIndex].Witness)
			if err != nil {
				e := fmt.Sprintf("btc ComputeWitnessPk err %s", err)
				return nil, nil, nil, errors.New(e)
			}
		}
		out := tx.MsgTx().TxOut[info.OutIndex].PkScript
		return tx, pk, out, nil
	}
}
