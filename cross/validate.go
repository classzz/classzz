package cross

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"math/rand"
)

var (
	ErrStakingAmount = errors.New("StakingAmount Less than minimum 1000000 czz")
	ethPoolAddr      = "0xB6475DAF416efAB1D70c893a53D7799be015Ed03"
)

type CommitteeVerify struct {
	EthRPC []*rpc.Client
	TrxRPC []string
	Cache  *CacheCommitteeState
	Params *chaincfg.Params
}

func (ev *CommitteeVerify) VerifyBeaconRegistrationTx(tx *wire.MsgTx, eState *EntangleState) (*BeaconAddressInfo, error) {

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

	if br.StakingAmount.Cmp(MinStakingAmount) < 0 {
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

func (ev *CommitteeVerify) VerifyAddBeaconPledgeTx(tx *wire.MsgTx, eState *EntangleState) (*AddBeaconPledge, error) {

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

	if bp.StakingAmount.Cmp(MinStakingAmount) < 0 {
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

func (ev *CommitteeVerify) VerifyMortgageTx(tx *wire.MsgTx, cState *CommitteeState) (*PledgeInfo, error) {

	br, _ := IsMortgageTx(tx, ev.Params)
	if br == nil {
		return nil, NoMortgage
	}

	if len(tx.TxIn) > 1 || len(tx.TxOut) > 3 || len(tx.TxOut) < 2 {
		e := fmt.Sprintf("MortgageTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
		return nil, errors.New(e)
	}

	if ok := cState.GetPledgeInfoByAddress(br.Address); ok != nil {
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

	if br.StakingAmount.Cmp(MinStakingAmount) < 0 {
		e := fmt.Sprintf("StakingAmount err")
		return nil, errors.New(e)
	}

	if len(br.CoinBaseAddress) > MaxCoinBase {
		e := fmt.Sprintf("whiteAddress.AssetType > MaxCoinBase err")
		return nil, errors.New(e)
	}

	for _, coinBaseAddress := range br.CoinBaseAddress {
		if _, err := czzutil.DecodeAddress(coinBaseAddress, ev.Params); err != nil {
			return nil, fmt.Errorf("DecodeCashAddress.AssetType err")
		}
	}

	for _, v := range cState.PledgeInfos {
		if bytes.Equal(v.ToAddress, br.ToAddress) {
			return nil, fmt.Errorf("ToAddress err")
		}
	}

	return br, nil
}

func (ev *CommitteeVerify) VerifyAddMortgageTx(tx *wire.MsgTx, cState *CommitteeState) (*AddMortgage, error) {

	am, _ := IsAddMortgageTx(tx, ev.Params)
	if am == nil {
		return nil, NoAddMortgage
	}

	if len(tx.TxIn) > 1 || len(tx.TxOut) > 3 || len(tx.TxOut) < 2 {
		e := fmt.Sprintf("BeaconRegistrationTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
		return nil, errors.New(e)
	}

	var pinfo *PledgeInfo
	if pinfo = cState.GetPledgeInfoByAddress(am.Address); pinfo == nil {
		return nil, ErrNoRegister
	}

	addr, err := czzutil.NewLegacyAddressPubKeyHash(pinfo.ToAddress, ev.Params)
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

	if tx.TxOut[1].Value != am.StakingAmount.Int64() {
		e := fmt.Sprintf("tx.TxOut[1].Value err")
		return nil, errors.New(e)
	}

	if am.StakingAmount.Cmp(MinAddStakingAmount) < 0 {
		e := fmt.Sprintf("StakingAmount err")
		return nil, errors.New(e)
	}

	return am, nil
}

func (ev *CommitteeVerify) VerifyUpdateCoinbaseAllTx(tx *wire.MsgTx, cState *CommitteeState) (*UpdateCoinbaseAll, error) {

	uc, _ := IsUpdateCoinbaseAllTx(tx, ev.Params)
	if uc == nil {
		return nil, NoUpdateCoinbaseAll
	}

	if pinfo := cState.GetPledgeInfoByAddress(uc.Address); pinfo == nil {
		return nil, ErrNoRegister
	}

	for _, coinBaseAddress := range uc.CoinBaseAddress {
		if _, err := czzutil.DecodeAddress(coinBaseAddress, ev.Params); err != nil {
			return nil, fmt.Errorf("DecodeCashAddress.AssetType err")
		}
	}

	return uc, nil
}

func (ev *CommitteeVerify) VerifyConvertTx(info *ConvertTxInfo, cState *CommitteeState) (*TuplePubIndex, error) {

	if ev.Cache != nil {
		if ok := ev.Cache.FetchExtUtxoView(info); ok {
			err := fmt.Sprintf("[txid:%s]", info.ExtTxHash)
			return nil, errors.New("txid has already entangle: " + err)
		}
	}

	if pub, err := ev.verifyConvertTx(info, cState); err != nil {
		errStr := fmt.Sprintf("[txid:%s]", info.ExtTxHash)
		return nil, errors.New("txid verify failed:" + errStr + " err: " + err.Error())
	} else {
		pair := &TuplePubIndex{
			AssetType:   info.AssetType,
			ConvertType: info.ConvertType,
			Index:       0,
			Pub:         pub,
		}
		return pair, nil
	}
}

func (ev *CommitteeVerify) verifyConvertTx(eInfo *ConvertTxInfo, eState *CommitteeState) ([]byte, error) {
	switch eInfo.AssetType {
	case ExpandedTxConvert_ECzz:
		return ev.verifyConvertEthTx(eInfo, eState)
	case ExpandedTxConvert_TCzz:
		return ev.verifyConvertTrxTx(eInfo, eState)
	}
	return nil, fmt.Errorf("verifyConvertTx AssetType is %v", eInfo.AssetType)
}

func (ev *CommitteeVerify) verifyConvertEthTx(eInfo *ConvertTxInfo, cState *CommitteeState) ([]byte, error) {

	client := ev.EthRPC[rand.Intn(len(ev.EthRPC))]

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", eInfo.ExtTxHash); err != nil {
		return nil, err
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("eth Status err")
	}

	var txjson *rpcTransaction
	// Get the current block count.
	if err := client.Call(&txjson, "eth_getTransactionByHash", eInfo.ExtTxHash); err != nil {
		return nil, err
	}

	ethtx := txjson.tx
	Vb, R, S := ethtx.RawSignatureValues()

	var V byte
	if isProtectedV(Vb) {
		chainID := deriveChainId(Vb).Uint64()
		V = byte(Vb.Uint64() - 35 - 2*chainID)
	} else {
		V = byte(Vb.Uint64() - 27)
	}

	if !crypto.ValidateSignatureValues(V, R, S, false) {
		return nil, fmt.Errorf("eth ValidateSignatureValues err")
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	a := types.NewEIP155Signer(big.NewInt(1))

	pk, err := crypto.Ecrecover(a.Hash(ethtx).Bytes(), sig)
	if err != nil {
		return nil, fmt.Errorf("Ecrecover err")
	}

	// height
	if receipt.BlockNumber.Uint64() != eInfo.Height {
		return nil, fmt.Errorf("ETh BlockNumber > Height")
	}

	// toaddress
	if txjson.tx.To().String() != ethPoolAddr {
		return nil, fmt.Errorf("ETh To != %s", ethPoolAddr)
	}

	if len(receipt.Logs) < 1 {
		return nil, fmt.Errorf("ETh receipt.Logs ")
	}

	txLog := receipt.Logs[0]
	// amount
	if big.NewInt(0).SetBytes(txLog.Data).Cmp(eInfo.Amount) != 0 {
		return nil, fmt.Errorf("ETh amount %d not %d", big.NewInt(0).SetBytes(txLog.Data), eInfo.Amount)
	}

	return pk, nil
}

func (ev *CommitteeVerify) verifyConvertTrxTx(eInfo *ConvertTxInfo, cState *CommitteeState) ([]byte, error) {
	return nil, nil
}
