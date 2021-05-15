package cross

import (
	"bytes"
	"encoding/hex"
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
	"strings"
)

var (
	ethPoolAddr  = "0x9ac88c5136240312f8817dbb99497ace62b03f12|0xB2451147c6154659c350EaC39ED37599bff4d32e|0xF0f50ce5054289a178fb45Ab2E373899580d12bf"
	hecoPoolAddr = "0x711d839cd1e6e81b971f5b6bbb4a6bd7c4b60ac6|0xdc3013FcF6A748c6b468de21b8A1680dbcb979ca|0x93E00a89F5CBF9c66a50aF7206c9c6f54601EC15|0x30d0e3F30D527373a27A2177fAcb4bdCc046DC1C"
	bscPoolAddr  = "0x007c98F9f2c70746a64572E67FBCc41a2b8bba18|0x711D839CD1E6E81B971F5b6bBB4a6BD7C4B60Ac6|0xdf10e0Caa2BBe67f7a1E91A3e6660cC1e34e81B9|0xa5D17B93f4156afd96be9f5B40888ffb47fA4bc1"

	burnTopics = "0x86f32d6c7a935bd338ee00610630fcfb6f043a6ad755db62064ce2ad92c45caa"
	mintTopics = "0x8fb5c7bffbb272c541556c455c74269997b816df24f56dd255c2391d92d4f1e9"
	CoinPools  = map[uint8][]byte{
		ExpandedTxConvert_ECzz: {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 101},
		ExpandedTxConvert_HCzz: {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 102},
		ExpandedTxConvert_BCzz: {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 103},
	}
)

type CommitteeVerify struct {
	EthRPC  []*rpc.Client
	HecoRPC []*rpc.Client
	BscRPC  []*rpc.Client
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

	if br.StakingAmount.Cmp(ev.Params.MinStakingAmount) < 0 {
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

	if bp.StakingAmount.Cmp(ev.Params.MinAddStakingAmount) < 0 {
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

	if br.StakingAmount.Cmp(ev.Params.MinStakingAmount) < 0 {
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
		e := fmt.Sprintf("AddMortgage in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
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

	if am.StakingAmount.Cmp(ev.Params.MinAddStakingAmount) < 0 {
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

func (ev *CommitteeVerify) VerifyConvertTx(tx *wire.MsgTx, cState *CommitteeState, info *ConvertTxInfo) (*TuplePubIndex, error) {
	if pub, err := ev.verifyConvertTx(tx, cState, info); err != nil {
		return nil, err
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

func (ev *CommitteeVerify) verifyConvertTx(tx *wire.MsgTx, cState *CommitteeState, eInfo *ConvertTxInfo) ([]byte, error) {

	switch eInfo.AssetType {
	case ExpandedTxConvert_ECzz:
		client := ev.EthRPC[rand.Intn(len(ev.EthRPC))]
		return ev.verifyConvertEthereumTypeTx("ETH", client, cState, eInfo, tx)
	case ExpandedTxConvert_HCzz:
		client := ev.HecoRPC[rand.Intn(len(ev.HecoRPC))]
		return ev.verifyConvertEthereumTypeTx("HECO", client, cState, eInfo, tx)
	case ExpandedTxConvert_BCzz:
		client := ev.BscRPC[rand.Intn(len(ev.BscRPC))]
		return ev.verifyConvertEthereumTypeTx("BSC", client, cState, eInfo, tx)
	}
	return nil, fmt.Errorf("verifyConvertTx AssetType not %v", eInfo.AssetType)
}

func (ev *CommitteeVerify) verifyConvertEthereumTypeTx(netName string, client *rpc.Client, cState *CommitteeState, eInfo *ConvertTxInfo, tx *wire.MsgTx) ([]byte, error) {

	if eInfo.AssetType == eInfo.ConvertType {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) AssetType = ConvertType = [%d]", netName, eInfo.ConvertType)
	}

	if _, ok := CoinPools[eInfo.ConvertType]; !ok && eInfo.ConvertType != 0 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ConvertType is [%d] CoinPools not find", netName, eInfo.ConvertType)
	}

	if ok := cState.ConvertExistExtTx(eInfo); ok {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) txid has already convert [txid:%s]", netName, eInfo.ExtTxHash)
	}

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", eInfo.ExtTxHash); err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) getTransactionReceipt [txid:%s] err: %s", netName, eInfo.ExtTxHash, err)
	}

	if receipt == nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) [txid:%s] not find", netName, eInfo.ExtTxHash)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) [txid:%s] Status [%d]", netName, eInfo.ExtTxHash, receipt.Status)
	}

	if len(receipt.Logs) < 1 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s)  receipt Logs length is 0 ", netName)
	}

	var txLog *types.Log
	for _, log := range receipt.Logs {
		if log.Topics[0].String() == burnTopics {
			txLog = log
			break
		}
	}

	if txLog == nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) txLog is nil ", netName)
	}

	amount := txLog.Data[:32]
	ntype := txLog.Data[32:64]
	//toToken := txLog.Data[64:]
	Amount := big.NewInt(0).SetBytes(amount)
	if Amount.Cmp(eInfo.Amount) != 0 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) amount [%d] not [%d]", netName, Amount, eInfo.Amount)
	}

	pool1 := CoinPools[eInfo.AssetType]
	add, _ := czzutil.NewAddressPubKeyHash(pool1, ev.Params)
	utxos := cState.NoCostUtxos[add.String()]
	amountPool := big.NewInt(0)
	if utxos != nil {
		for k1, _ := range utxos.POut {
			amountPool = big.NewInt(0).Add(amountPool, utxos.Amount[k1])
		}
	}

	if Amount.Cmp(amountPool) > 0 {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) tx amount [%d] > pool [%d]", netName, Amount, amountPool)
	}

	if big.NewInt(0).SetBytes(ntype).Uint64() != uint64(eInfo.ConvertType) {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s)  ntype [%d] not [%d]", netName, big.NewInt(0).SetBytes(ntype), eInfo.ConvertType)
	}

	var txjson *rpcTransaction
	// Get the current block count.
	if err := client.Call(&txjson, "eth_getTransactionByHash", eInfo.ExtTxHash); err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) getTransactionByHash [txid:%s] err: %s", netName, eInfo.ExtTxHash, err)
	}

	if eInfo.AssetType == ExpandedTxConvert_ECzz {
		if !strings.Contains(ethPoolAddr, txjson.tx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ETh [ToAddress: %s] != [%s]", netName, txjson.tx.To().String(), ethPoolAddr)
		}
	} else if eInfo.AssetType == ExpandedTxConvert_HCzz {
		if !strings.Contains(hecoPoolAddr, txjson.tx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Heco [ToAddress: %s] != [%s]", netName, txjson.tx.To().String(), ethPoolAddr)
		}
	} else if eInfo.AssetType == ExpandedTxConvert_BCzz {
		if !strings.Contains(bscPoolAddr, txjson.tx.To().String()) {
			return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Bsc [ToAddress: %s] != [%s]", netName, txjson.tx.To().String(), ethPoolAddr)
		}
	}

	extTx := txjson.tx
	Vb, R, S := extTx.RawSignatureValues()
	var V byte

	var chainID *big.Int
	if isProtectedV(Vb) {
		chainID = deriveChainId(Vb)
		V = byte(Vb.Uint64() - 35 - 2*chainID.Uint64())
	} else {
		V = byte(Vb.Uint64() - 27)
	}

	if chainID == nil {
		chainID = getChainID(eInfo.AssetType, ev.Params)
	}

	if !crypto.ValidateSignatureValues(V, R, S, false) {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) ValidateSignatureValues err", netName)
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	a := types.NewEIP155Signer(chainID)
	pk, err := crypto.Ecrecover(a.Hash(extTx).Bytes(), sig)
	if err != nil {
		return nil, fmt.Errorf("verifyConvertEthereumTypeTx (%s) Ecrecover err: %s", netName, err)
	}
	return pk, nil
}

func (ev *CommitteeVerify) VerifyConvertConfirmTx(cState *CommitteeState, info *ConvertConfirmTxInfo) error {
	switch info.ConvertType {
	case ExpandedTxConvert_ECzz:
		client := ev.EthRPC[rand.Intn(len(ev.EthRPC))]
		return ev.verifyConvertConfirmEthereumTypeTx("ETH", client, cState, info)
	case ExpandedTxConvert_HCzz:
		client := ev.HecoRPC[rand.Intn(len(ev.HecoRPC))]
		return ev.verifyConvertConfirmEthereumTypeTx("HECO", client, cState, info)
	case ExpandedTxConvert_BCzz:
		client := ev.BscRPC[rand.Intn(len(ev.BscRPC))]
		return ev.verifyConvertConfirmEthereumTypeTx("BSC", client, cState, info)
	}
	return fmt.Errorf("VerifyConvertConfirmTx AssetType not %v", info.AssetType)
}

func (ev *CommitteeVerify) verifyConvertConfirmEthereumTypeTx(netName string, client *rpc.Client, cState *CommitteeState, eInfo *ConvertConfirmTxInfo) error {

	if eInfo.AssetType == eInfo.ConvertType {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) AssetType = ConvertType = [%d]", netName, eInfo.ConvertType)
	}

	if _, ok := CoinPools[eInfo.AssetType]; !ok {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) AssetType is [%d] CoinPools not find", netName, eInfo.AssetType)
	}

	if ok := cState.ConvertConfirmExistExtTx(eInfo); ok {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) txid has already convert [txid:%s]", netName, eInfo.ExtTxHash)
	}

	var receipt *types.Receipt
	if err := client.Call(&receipt, "eth_getTransactionReceipt", eInfo.ExtTxHash); err != nil {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) getTransactionReceipt [txid:%s] err: %s", netName, eInfo.ExtTxHash, err)
	}

	if receipt == nil {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) [txid:%s] not find", netName, eInfo.ExtTxHash)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) [txid:%s] Status [%d]", netName, eInfo.ExtTxHash, receipt.Status)
	}

	if len(receipt.Logs) < 1 {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s)  receipt Logs length is 0 ", netName)
	}

	var txLog *types.Log
	for _, log := range receipt.Logs {
		if log.Topics[0].String() == mintTopics {
			txLog = log
			break
		}
	}

	if txLog == nil {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) txLog is nil ", netName)
	}

	address := txLog.Topics[1]
	mid := txLog.Data[32:64]
	amount := txLog.Data[64:]
	if big.NewInt(0).SetBytes(mid).Uint64() != eInfo.ID.Uint64() {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) mid %d not %d", netName, big.NewInt(0).SetBytes(mid), eInfo.ID.Uint64())
	}

	var hinfo *ConvertItem
	items := cState.ConvertItems[eInfo.AssetType][eInfo.ConvertType]
	for _, v := range items {
		if v.ID.Cmp(eInfo.ID) == 0 {
			hinfo = v
			break
		}
	}

	if hinfo == nil {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) ConvertItems [id:%d] is null", netName, eInfo.ID)
	}

	toaddresspuk, err := crypto.DecompressPubkey(hinfo.PubKey)
	if err != nil || toaddresspuk == nil {
		toaddresspuk, err = crypto.UnmarshalPubkey(hinfo.PubKey)
		if err != nil || toaddresspuk == nil {
			return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) toaddresspuk [puk:%s] is err: %s", netName, hex.EncodeToString(hinfo.PubKey), err)
		}
	}

	toaddress := common.Address{0}
	toaddress.SetBytes(address.Bytes())
	toaddress2 := crypto.PubkeyToAddress(*toaddresspuk)

	if toaddress.String() != toaddress2.String() {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) [toaddresspukaddress : %s] not [toaddress : %s]", netName, toaddress.String(), toaddress2.String())
	}

	amount2 := big.NewInt(0).Sub(hinfo.Amount, hinfo.FeeAmount)
	if big.NewInt(0).SetBytes(amount).Cmp(amount2) != 0 {
		return fmt.Errorf("verifyConvertConfirmEthereumTypeTx (%s) amount %d not %d", netName, big.NewInt(0).SetBytes(amount), amount2)
	}

	var txjson *rpcTransaction
	// Get the current block count.
	if err := client.Call(&txjson, "eth_getTransactionByHash", eInfo.ExtTxHash); err != nil {
		return err
	}

	// toaddress
	if eInfo.ConvertType == ExpandedTxConvert_ECzz {
		if !strings.Contains(ethPoolAddr, txjson.tx.To().String()) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) ETh [ToAddress: %s] != [%s]", netName, txjson.tx.To().String(), ethPoolAddr)
		}
	} else if eInfo.ConvertType == ExpandedTxConvert_HCzz {
		if !strings.Contains(hecoPoolAddr, txjson.tx.To().String()) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) Heco [ToAddress: %s] != [%s]", netName, txjson.tx.To().String(), hecoPoolAddr)
		}
	} else if eInfo.ConvertType == ExpandedTxConvert_BCzz {
		if !strings.Contains(bscPoolAddr, txjson.tx.To().String()) {
			return fmt.Errorf("verifyConvertEthereumTypeTx (%s) Bsc [ToAddress: %s] != [%s]", netName, txjson.tx.To().String(), bscPoolAddr)
		}
	}

	return nil
}

func (ev *CommitteeVerify) VerifyCastingTx(tx *wire.MsgTx, cState *CommitteeState, height int32) (*CastingTxInfo, error) {

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

	if tx.TxOut[1].Value != ct.Amount.Int64() && height > 1203000 {
		return nil, fmt.Errorf("Casting Amount err %d ", ct.Amount.Int64())
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
