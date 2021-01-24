package cross

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"io/ioutil"
	"math/big"
	"math/rand"
	"net/http"
)

var (
	ErrStakingAmount = errors.New("StakingAmount Less than minimum 1000000 czz")
	ethPoolAddr      = "0x1fA2FcD19ae095cc82AdBd0906BD3E4b68Be5832"
	trxMaturity      = uint64(0)
	CoinPools        = map[uint8][]byte{
		ExpandedTxConvert_ECzz: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 101},
		ExpandedTxConvert_TCzz: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 102},
		ExpandedTxConvert_HCzz: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 103},
	}
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
	case ExpandedTxConvert_HCzz:
		return ev.verifyConvertHecoTx(eInfo, eState)
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

	txLog := receipt.Logs[1]
	amount := txLog.Data[:32]
	ntype := txLog.Data[32:]
	if big.NewInt(0).SetBytes(amount).Cmp(eInfo.Amount) != 0 {
		return nil, fmt.Errorf("ETh amount %d not %d", big.NewInt(0).SetBytes(amount), eInfo.Amount)
	}

	if big.NewInt(0).SetBytes(ntype).Uint64() != uint64(eInfo.ConvertType) {
		return nil, fmt.Errorf("ETh ntype %d not %d", big.NewInt(0).SetBytes(ntype), eInfo.ConvertType)
	}

	return pk, nil
}

func (ev *CommitteeVerify) verifyConvertTrxTx(eInfo *ConvertTxInfo, cState *CommitteeState) ([]byte, error) {
	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client := ev.TrxRPC[rand.Intn(len(ev.TrxRPC))]

	data := make(map[string]interface{})
	data["value"] = eInfo.ExtTxHash
	data["visible"] = true
	bytesData, _ := json.Marshal(data)

	// Get the current block count.
	bytes.NewReader(bytesData)
	resp, err := http.Post("https://"+client+"/wallet/gettransactionbyid", "application/json", bytes.NewReader(bytesData))
	if err != nil {
		return nil, fmt.Errorf("trxTx ContractRet err")
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("trxTx ContractRet err")
	}

	trxTx := &TrxTx{}
	err = json.Unmarshal(body, trxTx)
	if err != nil {
		return nil, fmt.Errorf("trxTx ContractRet err")
	}

	if err := json.Unmarshal(body, trxTx); err != nil {
		return nil, err
	}

	if nil == trxTx.Ret || len(trxTx.Ret) < 1 || trxTx.Ret[0].ContractRet != "SUCCESS" {
		return nil, fmt.Errorf("trxTx ContractRet err")
	}

	hash_1, _ := hex.DecodeString(trxTx.RawDataHex)
	hash := sha256.Sum256(hash_1)
	fmt.Println(len(trxTx.Signature[0]))
	r, _ := hex.DecodeString(trxTx.Signature[0][:65])
	s, _ := hex.DecodeString(trxTx.Signature[0][65:130])

	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	pk, err := crypto.Ecrecover(hash[:], sig)

	//if ExChangeAmount.Cmp(ExChangeStakingAmount) > 0 {
	//	e := fmt.Sprintf("usdt ExChangeAmount > ExChangeStakingAmount")
	//	return nil, errors.New(e)
	//}

	data = make(map[string]interface{})
	data["num"] = 1
	bytesData, _ = json.Marshal(data)

	// Get the current block count.
	bytes.NewReader(bytesData)
	resp, err = http.Post("https://"+client+"/wallet/getblockbylatestnum", "application/json", bytes.NewReader(bytesData))
	if err != nil {
		panic(err)
	}

	body, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	trxBlock := &TrxBlock{}
	err = json.Unmarshal(body, trxTx)
	if err != nil {
		panic(err)
	}

	if trxBlock.block[0].BlockHeader.RawData.Number-eInfo.Height > trxMaturity {
		return pk, nil
	} else {
		e := fmt.Sprintf("trx Maturity err")
		return nil, errors.New(e)
	}

	return nil, nil

}

func (ev *CommitteeVerify) verifyConvertHecoTx(eInfo *ConvertTxInfo, cState *CommitteeState) ([]byte, error) {

	client := ev.HecoRPC[rand.Intn(len(ev.HecoRPC))]

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", eInfo.ExtTxHash); err != nil {
		return nil, err
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("heco Status err")
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
