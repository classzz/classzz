package cross

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"math/rand"
)

var (
	ErrStakingAmount = errors.New("StakingAmount Less than minimum 1000000 czz")
	ethPoolAddr      = "0xaD348f004cadD3cE93f567B20e578F33b7272306"
	hecoPoolAddr     = "0xaD348f004cadD3cE93f567B20e578F33b7272306"
	trxMaturity      = uint64(0)
	CoinPools        = map[uint8][]byte{
		ExpandedTxConvert_ECzz: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 101},
		ExpandedTxConvert_HCzz: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 102},
	}
	F10E18      = big.NewFloat(100000000000000000)
	EthChainID  = big.NewInt(3)
	HecoChainID = big.NewInt(256)
)

type CommitteeVerify struct {
	EthRPC  []*rpc.Client
	TrxRPC  []string
	HecoRPC []*rpc.Client
	Cache   *CacheCommitteeState
	Params  *chaincfg.Params
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

func (ev *CommitteeVerify) VerifyConvertTx(cState *CommitteeState, info *ConvertTxInfo) (*TuplePubIndex, error) {

	if ev.Cache != nil {
		if ok := ev.Cache.FetchExtUtxoView(info); ok {
			err := fmt.Sprintf("[txid:%s]", info.ExtTxHash)
			return nil, errors.New("txid has already convert: " + err)
		}
	}

	if pub, err := ev.verifyConvertTx(cState, info); err != nil {
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

func (ev *CommitteeVerify) verifyConvertTx(cState *CommitteeState, eInfo *ConvertTxInfo) ([]byte, error) {

	switch eInfo.AssetType {
	case ExpandedTxConvert_ECzz:
		return ev.verifyConvertEthTx(cState, eInfo)
	case ExpandedTxConvert_HCzz:
		return ev.verifyConvertHecoTx(eInfo)
	}
	return nil, fmt.Errorf("verifyConvertTx AssetType is %v", eInfo.AssetType)
}

func (ev *CommitteeVerify) verifyConvertEthTx(cState *CommitteeState, eInfo *ConvertTxInfo) ([]byte, error) {

	client := ev.EthRPC[rand.Intn(len(ev.EthRPC))]

	if _, ok := CoinPools[eInfo.ConvertType]; !ok {
		return nil, fmt.Errorf("verifyConvertEthTx %d CoinPools not find ", eInfo.ConvertType)
	}

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", eInfo.ExtTxHash); err != nil {
		return nil, err
	}

	if receipt == nil {
		return nil, fmt.Errorf("verifyConvertEthTx ExtTxHash not find")
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("verifyConvertEthTx Status err")
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
		return nil, fmt.Errorf("verifyConvertEthTx ValidateSignatureValues err")
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	a := types.NewEIP155Signer(EthChainID)

	pk, err := crypto.Ecrecover(a.Hash(ethtx).Bytes(), sig)
	if err != nil {
		return nil, fmt.Errorf("verifyConvertEthTx Ecrecover err")
	}

	// toaddress
	//if txjson.tx.To().String() != ethPoolAddr {
	//	return nil, fmt.Errorf("ETh To != %s", ethPoolAddr)
	//}

	if len(receipt.Logs) < 1 {
		return nil, fmt.Errorf("verifyConvertEthTx receipt.Logs ")
	}

	var txLog *types.Log
	for _, log := range receipt.Logs {
		if log.Topics[0].String() == "0x86f32d6c7a935bd338ee00610630fcfb6f043a6ad755db62064ce2ad92c45caa" {
			txLog = log
		}
	}
	if txLog == nil {
		return nil, fmt.Errorf("verifyConvertEthTx txLog is nil ")
	}

	amount := txLog.Data[:32]
	ntype := txLog.Data[32:64]
	//toToken := txLog.Data[64:]
	Amount := big.NewInt(0).SetBytes(amount)
	Amount1, _ := big.NewFloat(0.0).Quo(new(big.Float).SetInt64(Amount.Int64()), F10E18).Float64()
	Amount2 := FloatRound(Amount1, 8)
	Amount3 := big.NewInt(int64(Amount2 * 100000000))
	if Amount3.Cmp(eInfo.Amount) != 0 {
		return nil, fmt.Errorf("verifyConvertEthTx amount %d not %d", Amount3, eInfo.Amount)
	}

	pool1 := CoinPools[eInfo.AssetType]
	add, _ := czzutil.NewAddressPubKeyHash(pool1, ev.Params)
	utxos := cState.NoCostUtxos[add.String()]
	amount_pool := big.NewInt(0)
	if utxos != nil {
		for k1, _ := range utxos.POut {
			amount_pool = big.NewInt(0).Add(amount_pool, utxos.Amount[k1])
		}
	}

	if Amount3.Cmp(amount_pool) > 0 {
		return nil, fmt.Errorf("verifyConvertEthTx tx amount %d Is greater than pool %d", Amount3, amount_pool)
	}

	if big.NewInt(0).SetBytes(ntype).Uint64() != uint64(eInfo.ConvertType) {
		return nil, fmt.Errorf("verifyConvertEthTx ntype %d not %d", big.NewInt(0).SetBytes(ntype), eInfo.ConvertType)
	}

	return pk, nil
}

func (ev *CommitteeVerify) verifyConvertHecoTx(eInfo *ConvertTxInfo) ([]byte, error) {

	client := ev.HecoRPC[rand.Intn(len(ev.HecoRPC))]

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", eInfo.ExtTxHash); err != nil {
		return nil, err
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("verifyConvertHecoTx Status err")
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
		return nil, fmt.Errorf("verifyConvertHecoTx ValidateSignatureValues err")
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	a := types.NewEIP155Signer(HecoChainID)

	pk, err := crypto.Ecrecover(a.Hash(ethtx).Bytes(), sig)
	if err != nil {
		return nil, fmt.Errorf("verifyConvertHecoTx Ecrecover err")
	}

	// toaddress
	//if txjson.tx.To().String() != ethPoolAddr {
	//	return nil, fmt.Errorf("ETh To != %s", ethPoolAddr)
	//}

	if len(receipt.Logs) < 1 {
		return nil, fmt.Errorf("verifyConvertHecoTx receipt.Logs ")
	}
	var txLog *types.Log
	for _, log := range receipt.Logs {
		if log.Topics[0].String() == "0x86f32d6c7a935bd338ee00610630fcfb6f043a6ad755db62064ce2ad92c45caa" {
			txLog = log
		}
	}
	if txLog == nil {
		return nil, fmt.Errorf("verifyConvertHecoTx txLog == nil ")
	}

	amount := txLog.Data[:32]
	ntype := txLog.Data[32:64]
	//toToken := txLog.Data[64:]
	Amount := big.NewInt(0).SetBytes(amount)
	Amount1, _ := big.NewFloat(0.0).Quo(new(big.Float).SetInt64(Amount.Int64()), F10E18).Float64()
	fmt.Println(Amount1)
	Amount2 := FloatRound(Amount1, 8)
	fmt.Println(Amount2)
	Amount3 := big.NewInt(int64(Amount2 * 100000000))
	if Amount3.Cmp(eInfo.Amount) != 0 {
		return nil, fmt.Errorf("verifyConvertHecoTx amount %d not %d", Amount3, eInfo.Amount)
	}

	if big.NewInt(0).SetBytes(ntype).Uint64() != uint64(eInfo.ConvertType) {
		return nil, fmt.Errorf("verifyConvertHecoTx ntype %d not %d", big.NewInt(0).SetBytes(ntype), eInfo.ConvertType)
	}

	return pk, nil
}

func (ev *CommitteeVerify) VerifyConvertConfirmTx(cState *CommitteeState, info *ConvertConfirmTxInfo) error {

	if ev.Cache != nil {
		if ok := ev.Cache.FetchExtUtxoView(info); ok {
			err := fmt.Sprintf("[txid:%s]", info.ExtTxHash)
			return errors.New("verifyConvertHecoTx txid has already convert: " + err)
		}
	}

	if err := ev.verifyConvertConfirmTx(info, cState); err != nil {
		errStr := fmt.Sprintf("[txid:%s]", info.ExtTxHash)
		return errors.New("verifyConvertHecoTx txid verify failed:" + errStr + " err: " + err.Error())
	}
	return nil
}

func (ev *CommitteeVerify) verifyConvertConfirmTx(eInfo *ConvertConfirmTxInfo, eState *CommitteeState) error {
	switch eInfo.ConvertType {
	case ExpandedTxConvert_ECzz:
		return ev.verifyConvertConfirmEthTx(eInfo, eState)
	case ExpandedTxConvert_HCzz:
		return ev.verifyConvertConfirmHecoTx(eInfo, eState)
	}
	return fmt.Errorf("verifyConvertTx AssetType is %v", eInfo.AssetType)
}

func (ev *CommitteeVerify) verifyConvertConfirmEthTx(eInfo *ConvertConfirmTxInfo, cState *CommitteeState) error {

	client := ev.EthRPC[rand.Intn(len(ev.EthRPC))]

	if _, ok := CoinPools[eInfo.AssetType]; !ok {
		return fmt.Errorf("%d CoinPools not find ", eInfo.ConvertType)
	}

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", eInfo.ExtTxHash); err != nil {
		return err
	}

	if receipt == nil {
		return fmt.Errorf("eth ExtTxHash not find")
	}

	if receipt.Status != 1 {
		return fmt.Errorf("eth Status err")
	}

	var txjson *rpcTransaction
	// Get the current block count.
	if err := client.Call(&txjson, "eth_getTransactionByHash", eInfo.ExtTxHash); err != nil {
		return err
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
		return fmt.Errorf("eth ValidateSignatureValues err")
	}

	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	a := types.NewEIP155Signer(EthChainID)
	pk, err := crypto.Ecrecover(a.Hash(ethtx).Bytes(), sig)
	if err != nil {
		return fmt.Errorf("Ecrecover err")
	}

	// toaddress
	//if txjson.tx.To().String() != ethPoolAddr {
	//	return fmt.Errorf("ETh To != %s", ethPoolAddr)
	//}

	if len(receipt.Logs) < 1 {
		return fmt.Errorf("ETh receipt.Logs")
	}

	var hinfo *ConvertItem
	items := cState.ConvertItems[eInfo.AssetType][eInfo.ConvertType]
	for _, v := range items {
		if v.ID.Cmp(eInfo.ID) == 0 {
			hinfo = v
		}
	}

	if hinfo == nil {
		return fmt.Errorf("hinfo is null")
	}

	txLog := receipt.Logs[1]
	amount := txLog.Data[:32]
	ntype := txLog.Data[32:]
	if big.NewInt(0).SetBytes(amount).Cmp(hinfo.Amount) != 0 {
		return fmt.Errorf("ETh amount %d not %d", big.NewInt(0).SetBytes(amount), eInfo.Amount)
	}

	if big.NewInt(0).SetBytes(ntype).Uint64() != uint64(eInfo.ConvertType) {
		return fmt.Errorf("ETh ntype %d not %d", big.NewInt(0).SetBytes(ntype), eInfo.ConvertType)
	}

	if bytes.Equal(pk, hinfo.PubKey) {
		return fmt.Errorf("ETh pk %d not %d", pk, hinfo.PubKey)
	}

	return nil
}

func (ev *CommitteeVerify) verifyConvertConfirmHecoTx(eInfo *ConvertConfirmTxInfo, cState *CommitteeState) error {

	client := ev.HecoRPC[rand.Intn(len(ev.HecoRPC))]
	if _, ok := CoinPools[eInfo.AssetType]; !ok {
		return fmt.Errorf("%d CoinPools not find ", eInfo.AssetType)
	}

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", eInfo.ExtTxHash); err != nil {
		return err
	}

	if receipt == nil {
		return fmt.Errorf("verifyConvertConfirmHecoTx ExtTxHash not find")
	}

	if receipt.Status != 1 {
		return fmt.Errorf("verifyConvertConfirmHecoTx Status err")
	}

	var txjson *rpcTransaction
	// Get the current block count.
	if err := client.Call(&txjson, "eth_getTransactionByHash", eInfo.ExtTxHash); err != nil {
		return err
	}

	hecotx := txjson.tx
	Vb, R, S := hecotx.RawSignatureValues()

	var V byte
	chainID := HecoChainID
	if isProtectedV(Vb) {
		chainID = deriveChainId(Vb)
		V = byte(Vb.Uint64() - 35 - 2*chainID.Uint64())
	} else {
		V = byte(Vb.Uint64() - 27)
	}

	if !crypto.ValidateSignatureValues(V, R, S, false) {
		return fmt.Errorf("verifyConvertConfirmHecoTx ValidateSignatureValues err")
	}

	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	EIP := types.NewEIP155Signer(chainID)
	pk, err := crypto.Ecrecover(EIP.Hash(hecotx).Bytes(), sig)
	if err != nil {
		return fmt.Errorf("verifyConvertConfirmHecoTx Ecrecover err")
	}

	// toaddress
	//if txjson.tx.To().String() != ethPoolAddr {
	//	return fmt.Errorf("ETh To != %s", ethPoolAddr)
	//}

	if len(receipt.Logs) < 1 {
		return fmt.Errorf("verifyConvertConfirmHecoTx receipt.Logs")
	}

	var txLog *types.Log
	for _, log := range receipt.Logs {
		if log.Topics[0].String() == "0x8fb5c7bffbb272c541556c455c74269997b816df24f56dd255c2391d92d4f1e9" {
			txLog = log
		}
	}

	if txLog == nil {
		return fmt.Errorf("verifyConvertConfirmHecoTx txLog == nil ")
	}

	address := txLog.Topics[1]
	mid := txLog.Data[32:64]
	amount := txLog.Data[64:]
	if big.NewInt(0).SetBytes(mid).Uint64() != eInfo.ID.Uint64() {
		return fmt.Errorf("ETh mid %d not %d", big.NewInt(0).SetBytes(mid), eInfo.ID.Uint64())
	}

	var hinfo *ConvertItem
	items := cState.ConvertItems[eInfo.AssetType][eInfo.ConvertType]
	for _, v := range items {
		if v.ID.Cmp(eInfo.ID) == 0 {
			hinfo = v
		}
	}

	if hinfo == nil {
		return fmt.Errorf("verifyConvertConfirmHecoTx hinfo is null")
	}

	toaddresspuk, err := crypto.DecompressPubkey(hinfo.PubKey)
	if err != nil || toaddresspuk == nil {
		toaddresspuk, err = crypto.UnmarshalPubkey(hinfo.PubKey)
		if err != nil || toaddresspuk == nil {
			return err
		}
	}

	toaddress := common.Address{0}
	toaddress.SetBytes(address.Bytes())
	toaddress2 := crypto.PubkeyToAddress(*toaddresspuk)

	if toaddress.String() != toaddress2.String() {
		return fmt.Errorf("verifyConvertConfirmHecoTx toaddress %d  toaddress2 %d", toaddress, toaddress2)
	}

	if big.NewInt(0).SetBytes(amount).Cmp(hinfo.Amount) <= 0 {
		return fmt.Errorf("verifyConvertConfirmHecoTx amount %d not %d", big.NewInt(0).SetBytes(amount), eInfo.Amount)
	}

	fmt.Println(pk)
	return nil
}

func (ev *CommitteeVerify) VerifyCastingTx(tx *wire.MsgTx, cState *CommitteeState) (*CastingTxInfo, error) {

	ct, _ := IsCastingTx(tx)
	if ct == nil {
		return nil, NoCasting
	}

	if _, ok := CoinPools[ct.ConvertType]; !ok {
		return nil, fmt.Errorf("Casting not find ConvertType err %v ", ct.ConvertType)
	}
	pool := CoinPools[ct.ConvertType]
	PkScript, _ := txscript.PayToPubKeyHashScript(pool)
	if !bytes.Equal(PkScript, tx.TxOut[1].PkScript) {
		return nil, fmt.Errorf("Casting PkScript err %s ", tx.TxOut[1].PkScript)
	}

	var pk []byte
	var err error
	if tx.TxIn[0].Witness == nil {
		pk, err = txscript.ComputePk(tx.TxIn[0].SignatureScript)
		if err != nil {
			e := fmt.Sprintf("Casting ComputePk err %s", err)
			return nil, errors.New(e)
		}
	} else {
		pk, err = txscript.ComputeWitnessPk(tx.TxIn[0].Witness)
		if err != nil {
			e := fmt.Sprintf("Casting ComputeWitnessPk err %s", err)
			return nil, errors.New(e)
		}
	}

	ct.PubKey = pk
	return ct, nil
}
