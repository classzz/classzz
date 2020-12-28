package cross

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/rlp"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

const (
	// Entangle Transcation type
	ExpandedTxEntangle_Doge uint8 = iota
	ExpandedTxEntangle_Ltc
	ExpandedTxConvert_ECzz
	ExpandedTxConvert_TCzz
	ExpandedTxConvert_Czz
)

var (
	NoEntangle              = errors.New("no NoEntangle info in transcation")
	NoExChange              = errors.New("no NoExChange info in transcation")
	NoConvert               = errors.New("no NoConvert info in transcation")
	NoFastExChange          = errors.New("no NoFastExChange info in transcation")
	NoBeaconRegistration    = errors.New("no BeaconRegistration info in transcation")
	NoBurnTx                = errors.New("no BurnTx info in transcation")
	NoBurnProofTx           = errors.New("no BurnProofTx info in transcation")
	NoBurnReportWhiteListTx = errors.New("no WhiteListProofTx info in transcation")
	NoAddBeaconPledge       = errors.New("no AddBeaconPledge info in transcation")
	NoUpdateBeaconCoinbase  = errors.New("no UpdateBeaconCoinbase info in transcation")
	NoUpdateBeaconFreeQuota = errors.New("no UpdateBeaconFreeQuota info in transcation")

	baseUnit    = new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil)
	baseUnit1   = new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
	dogeUnit    = new(big.Int).Mul(big.NewInt(int64(12500000)), baseUnit)
	dogeUnit1   = new(big.Int).Mul(big.NewInt(int64(12500000)), baseUnit1)
	MinPunished = new(big.Int).Mul(big.NewInt(int64(20)), baseUnit)
	ZeroAddrsss = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	//ZeroAddrsss, _ = czzutil.NewLegacyAddressPubKeyHash([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, &chaincfg.TestNetParams)
)

type EntangleItem struct {
	AssetType uint8
	Value     *big.Int
	Addr      czzutil.Address
}

func (ii *EntangleItem) Clone() *EntangleItem {
	item := &EntangleItem{
		AssetType: ii.AssetType,
		Value:     new(big.Int).Set(ii.Value),
		Addr:      ii.Addr,
	}
	return item
}

type ExChangeItem struct {
	ExtTxHash   string          `json:"ext_tx_hash"`
	Height      *big.Int        `json:"height"`
	AssetType   uint8           `json:"asset_type"`
	ConvertType uint8           `json:"convert_type"`
	Address     czzutil.Address `json:"address"`
	Amount      *big.Int        `json:"amount"` // czz asset amount
}

//func (ii *ExChangeItem) Clone() *ExChangeItem {
//	item := &ExChangeItem{
//		AssetType: ii.AssetType,
//		Value:     new(big.Int).Set(ii.Value),
//		Addr:      ii.Addr,
//		BeaconID:  ii.BeaconID,
//	}
//	return item
//}

// entangle tx Sequence infomation
type EtsInfo struct {
	FeePerKB int64
	Tx       *wire.MsgTx
}

type TuplePubIndex struct {
	AssetType   uint8
	ConvertType uint8
	Index       uint32
	Pub         []byte
}

type PoolAddrItem struct {
	POut   []wire.OutPoint
	Script [][]byte
	Amount []*big.Int
}

type PunishedRewardItem struct {
	POut         wire.OutPoint
	Script       []byte
	OriginAmount *big.Int
	Addr1        czzutil.Address
	Addr2        czzutil.Address
	Addr3        czzutil.Address
	Amount       *big.Int
}

func (p *PunishedRewardItem) PkScript(pos int) []byte {
	addr := p.Addr3
	if pos == 0 {
		addr = p.Addr1
	} else if pos == 1 {
		addr = p.Addr2
	}
	b, e := txscript.PayToAddrScript(addr)
	if e != nil {
		return nil
	}
	return b
}
func (p *PunishedRewardItem) EqualPkScript(pb []byte, pos int) bool {
	b := p.PkScript(pos)
	if b == nil {
		return false
	}
	return bytes.Equal(b, pb)
}
func (p *PunishedRewardItem) Change() *big.Int {
	return new(big.Int).Sub(p.OriginAmount, new(big.Int).Mul(big.NewInt(2), p.Amount))
}

type BeaconMergeItem struct {
	POut      wire.OutPoint
	Script    []byte
	Amount    *big.Int
	ToAddress czzutil.Address
}

type ExtTxInfo interface {
	ToBytes() []byte
	GetAssetType() uint32
	GetExtTxHash() string
}

type ExChangeTxInfo struct {
	AssetType uint32
	Address   string
	Index     uint32
	Height    uint64
	Amount    *big.Int
	ExtTxHash string
	BeaconID  uint64
}

////////////////////////////////////////////////////////////////////////////
func (ec *ExChangeTxInfo) ToBytes() []byte {
	// maybe rlp encode
	data, err := rlp.EncodeToBytes(ec)
	if err != nil {
		log.Fatal("Failed to RLP encode ExChangeTxInfo: ", err)
	}
	return data
}

func (ec *ExChangeTxInfo) GetAssetType() uint32 {
	return ec.AssetType
}

func (ec *ExChangeTxInfo) GetExtTxHash() string {
	return ec.ExtTxHash
}

type ConvertTxInfo struct {
	AssetType   uint8
	ConvertType uint8
	Address     string
	Height      uint64
	ExtTxHash   string
	Index       uint32
	Amount      *big.Int
}

func (es *ConvertTxInfo) ToBytes() []byte {
	// maybe rlp encode
	data, err := rlp.EncodeToBytes(es)
	if err != nil {
		log.Fatal("Failed to RLP encode BurnTxInfo: ", "err", err)
	}
	return data
}

func (ec *ConvertTxInfo) GetAssetType() uint8 {
	return ec.AssetType
}

func (ec *ConvertTxInfo) GetExtTxHash() string {
	return ec.ExtTxHash
}

type EntangleTxInfo struct {
	AssetType uint32
	Index     uint32
	Height    uint64
	Amount    *big.Int
	ExtTxHash []byte
}

func (es *EntangleTxInfo) ToBytes() []byte {
	// maybe rlp encode
	data, err := rlp.EncodeToBytes(es)
	if err != nil {
		log.Fatal("Failed to RLP encode BurnTxInfo: ", "err", err)
	}
	return data
}

type BurnTxInfo struct {
	AssetType uint32
	Address   string
	ToAddress string
	BeaconID  uint64
	Amount    *big.Int
	Height    uint32
}

func (es *BurnTxInfo) ToBytes() []byte {
	// maybe rlp encode
	data, err := rlp.EncodeToBytes(es)
	if err != nil {
		log.Fatal("Failed to RLP encode BurnTxInfo: ", "err", err)
	}
	return data
}

type KeepedItem struct {
	AssetType uint8
	Amount    *big.Int
}

type KeepedAmount struct {
	Count byte
	Items []KeepedItem
}

func (info *KeepedAmount) Serialize() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(info.Count)
	for _, v := range info.Items {
		buf.WriteByte(byte(v.AssetType))
		b1 := v.Amount.Bytes()
		len := uint8(len(b1))
		buf.WriteByte(len)
		buf.Write(b1)
	}
	return buf.Bytes()
}

func (info *KeepedAmount) Parse(data []byte) error {
	if data == nil {
		return nil
	}
	info.Count = data[0]
	buf := bytes.NewBuffer(data[1:])

	for i := 0; i < int(info.Count); i++ {
		itype, _ := buf.ReadByte()
		b0 := make([]byte, itype)
		_, _ = buf.Read(b0)
		item := KeepedItem{
			AssetType: itype,
			Amount:    new(big.Int).SetBytes(b0),
		}
		info.Items = append(info.Items, item)
	}
	return nil
}

func (info *KeepedAmount) Add(item KeepedItem) {
	for _, v := range info.Items {
		if v.AssetType == item.AssetType {
			v.Amount.Add(v.Amount, item.Amount)
			return
		}
	}
	info.Count++
	info.Items = append(info.Items, item)
}

func (info *KeepedAmount) GetValue(t uint8) *big.Int {
	for _, v := range info.Items {
		if v.AssetType == t {
			return v.Amount
		}
	}
	return nil
}

//func IsEntangleTx(tx *wire.MsgTx) (map[uint32]*EntangleTxInfo, error) {
//	// make sure at least one txout in OUTPUT
//	einfos := make(map[uint32]*EntangleTxInfo)
//	for i, v := range tx.TxOut {
//		info, err := EntangleTxFromScript(v.PkScript)
//		if err == nil {
//			if v.Value != 0 {
//				return nil, errors.New("the output value must be 0 in entangle tx.")
//			}
//			einfos[uint32(i)] = info
//		}
//	}
//	if len(einfos) > 0 {
//		return einfos, nil
//	}
//	return nil, NoEntangle
//}

func IsBeaconRegistrationTx(tx *wire.MsgTx, params *chaincfg.Params) (*BeaconAddressInfo, error) {
	// make sure at least one txout in OUTPUT
	if len(tx.TxOut) > 0 {
		txout := tx.TxOut[0]
		if !txscript.IsBeaconRegistrationTy(txout.PkScript) {
			return nil, NoBeaconRegistration
		}
	} else {
		return nil, NoBeaconRegistration
	}

	if len(tx.TxOut) > 3 || len(tx.TxOut) < 2 || len(tx.TxIn) > 1 {
		e := fmt.Sprintf("not BeaconRegistration tx TxOut >3 or TxIn >1")
		return nil, errors.New(e)
	}

	var es *BeaconAddressInfo
	txout := tx.TxOut[0]
	info, err := BeaconRegistrationTxFromScript(txout.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txout.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		es = info
	}

	var pk []byte
	if tx.TxIn[0].Witness == nil {
		pk, err = txscript.ComputePk(tx.TxIn[0].SignatureScript)
		if err != nil {
			e := fmt.Sprintf("ComputePk err %s", err)
			return nil, errors.New(e)
		}
	} else {
		pk, err = txscript.ComputeWitnessPk(tx.TxIn[0].Witness)
		if err != nil {
			e := fmt.Sprintf("ComputeWitnessPk err %s", err)
			return nil, errors.New(e)
		}
	}

	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), params)
	if err != nil {
		e := fmt.Sprintf("NewAddressPubKeyHash err %s", err)
		return nil, errors.New(e)
	}

	info.StakingAmount = big.NewInt(tx.TxOut[1].Value)
	info.Address = address.String()

	if es != nil {
		return es, nil
	}
	return nil, NoBeaconRegistration
}

func IsAddBeaconPledgeTx(tx *wire.MsgTx, params *chaincfg.Params) (*AddBeaconPledge, error) {
	// make sure at least one txout in OUTPUT
	if len(tx.TxOut) > 0 {
		txout := tx.TxOut[0]
		if !txscript.IsAddBeaconPledgeTy(txout.PkScript) {
			return nil, NoAddBeaconPledge
		}
	} else {
		return nil, NoAddBeaconPledge
	}

	if len(tx.TxOut) > 3 || len(tx.TxOut) < 2 || len(tx.TxIn) > 1 {
		e := fmt.Sprintf("not BeaconRegistration tx TxOut >3 or TxIn >1")
		return nil, errors.New(e)
	}

	// make sure at least one txout in OUTPUT
	var bp *AddBeaconPledge

	txout := tx.TxOut[0]
	info, err := AddBeaconPledgeTxFromScript(txout.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txout.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		bp = info
	}

	var pk []byte
	if tx.TxIn[0].Witness == nil {
		pk, err = txscript.ComputePk(tx.TxIn[0].SignatureScript)
		if err != nil {
			e := fmt.Sprintf("ComputePk err %s", err)
			return nil, errors.New(e)
		}
	} else {
		pk, err = txscript.ComputeWitnessPk(tx.TxIn[0].Witness)
		if err != nil {
			e := fmt.Sprintf("ComputeWitnessPk err %s", err)
			return nil, errors.New(e)
		}
	}

	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), params)
	if err != nil {
		e := fmt.Sprintf("NewAddressPubKeyHash err %s", err)
		return nil, errors.New(e)
	}

	info.StakingAmount = big.NewInt(tx.TxOut[1].Value)
	info.Address = address.String()

	if bp != nil {
		return bp, nil
	}
	return nil, NoAddBeaconPledge
}

///////////////////////////////////////////////////
func IsMortgageTx(tx *wire.MsgTx, params *chaincfg.Params) (*BeaconAddressInfo, error) {
	// make sure at least one txout in OUTPUT
	if len(tx.TxOut) > 0 {
		txout := tx.TxOut[0]
		if !txscript.IsBeaconRegistrationTy(txout.PkScript) {
			return nil, NoBeaconRegistration
		}
	} else {
		return nil, NoBeaconRegistration
	}

	if len(tx.TxOut) > 3 || len(tx.TxOut) < 2 || len(tx.TxIn) > 1 {
		e := fmt.Sprintf("not BeaconRegistration tx TxOut >3 or TxIn >1")
		return nil, errors.New(e)
	}

	var es *BeaconAddressInfo
	txout := tx.TxOut[0]
	info, err := BeaconRegistrationTxFromScript(txout.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txout.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		es = info
	}

	var pk []byte
	if tx.TxIn[0].Witness == nil {
		pk, err = txscript.ComputePk(tx.TxIn[0].SignatureScript)
		if err != nil {
			e := fmt.Sprintf("ComputePk err %s", err)
			return nil, errors.New(e)
		}
	} else {
		pk, err = txscript.ComputeWitnessPk(tx.TxIn[0].Witness)
		if err != nil {
			e := fmt.Sprintf("ComputeWitnessPk err %s", err)
			return nil, errors.New(e)
		}
	}

	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), params)
	if err != nil {
		e := fmt.Sprintf("NewAddressPubKeyHash err %s", err)
		return nil, errors.New(e)
	}

	info.StakingAmount = big.NewInt(tx.TxOut[1].Value)
	info.Address = address.String()

	if es != nil {
		return es, nil
	}
	return nil, NoBeaconRegistration
}

func IsConvertTx(tx *wire.MsgTx) (map[uint32]*ConvertTxInfo, error) {
	// make sure at least one txout in OUTPUT

	cons := make(map[uint32]*ConvertTxInfo)
	for i, txout := range tx.TxOut {
		if !txscript.IsConvertTy(txout.PkScript) {
			continue
		}
		info, err := ConvertTxFromScript(tx.TxOut[0].PkScript)
		if err != nil {
			e := fmt.Sprintf("ConverTxFromScript err %s", err)
			return nil, errors.New(e)
		}
		cons[uint32(i)] = info
	}

	if len(cons) == 0 {
		return nil, NoConvert
	}

	return cons, nil
}

// Only txOut[0] is valid
//func IsFastExChangeTx(tx *wire.MsgTx, params *chaincfg.Params) (*ExChangeTxInfo, *BurnTxInfo, error) {
//
//	var err error
//
//	if len(tx.TxOut) > 1 {
//		txout1 := tx.TxOut[0]
//		txout2 := tx.TxOut[1]
//		if !(txscript.IsExChangeTy(txout1.PkScript) && txscript.IsBurnTy(txout2.PkScript)) {
//			return nil, nil, NoFastExChange
//		}
//	} else {
//		return nil, nil, NoFastExChange
//	}
//
//	if len(tx.TxIn) > 1 || len(tx.TxIn) < 1 || len(tx.TxOut) > 3 || len(tx.TxOut) < 2 {
//		e := fmt.Sprintf("FastExChangeTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
//		return nil, nil, errors.New(e)
//	}
//
//	exInfo, err := ExChangeTxFromScript(tx.TxOut[0].PkScript)
//	if err != nil {
//		e := fmt.Sprintf("ExChangeTxFromScript err %s", err)
//		return nil, nil, errors.New(e)
//	}
//
//	var pk []byte
//	// get from address
//	if tx.TxIn[0].Witness == nil {
//		pk, err = txscript.ComputePk(tx.TxIn[0].SignatureScript)
//		if err != nil {
//			e := fmt.Sprintf("ComputePk err %s", err)
//			return nil, nil, errors.New(e)
//		}
//	} else {
//		pk, err = txscript.ComputeWitnessPk(tx.TxIn[0].Witness)
//		if err != nil {
//			e := fmt.Sprintf("ComputeWitnessPk err %s", err)
//			return nil, nil, errors.New(e)
//		}
//	}
//
//	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), params)
//	if err != nil {
//		e := fmt.Sprintf("NewAddressPubKeyHash err %s", err)
//		return nil, nil, errors.New(e)
//	}
//
//	txout := tx.TxOut[1]
//	info, err := BurnInfoFromScript(txout.PkScript)
//	if err != nil {
//		return nil, nil, errors.New("BurnInfoFromScript the output tx.")
//	}
//
//	info.Address = address.String()
//
//	return exInfo, info, nil
//}
//
//// Only txOut[0] is valid
//func IsFastExChangeTxToStorage(tx *wire.MsgTx) (*ExChangeTxInfo, *BurnTxInfo, error) {
//
//	var err error
//
//	if len(tx.TxOut) > 1 {
//		txout1 := tx.TxOut[0]
//		txout2 := tx.TxOut[1]
//		if !(txscript.IsExChangeTy(txout1.PkScript) && txscript.IsBurnTy(txout2.PkScript)) {
//			return nil, nil, NoFastExChange
//		}
//	} else {
//		return nil, nil, NoFastExChange
//	}
//
//	if len(tx.TxIn) > 1 || len(tx.TxIn) < 1 || len(tx.TxOut) > 3 || len(tx.TxOut) < 2 {
//		e := fmt.Sprintf("FastExChangeTx in or out err  in : %v , out : %v", len(tx.TxIn), len(tx.TxOut))
//		return nil, nil, errors.New(e)
//	}
//
//	exInfo, err := ExChangeTxFromScript(tx.TxOut[0].PkScript)
//	if err != nil {
//		e := fmt.Sprintf("ExChangeTxFromScript err %s", err)
//		return nil, nil, errors.New(e)
//	}
//
//	txout := tx.TxOut[1]
//	info, err := BurnInfoFromScript(txout.PkScript)
//	if err != nil {
//		return nil, nil, errors.New("BurnInfoFromScript the output tx.")
//	}
//
//	return exInfo, info, nil
//}

//func IsBurnTx(tx *wire.MsgTx, params *chaincfg.Params) (*BurnTxInfo, error) {
//	// make sure at least one txout in OUTPUT
//	var es *BurnTxInfo
//
//	var pk []byte
//	var err error
//
//	if len(tx.TxOut) > 0 {
//		txout := tx.TxOut[0]
//		if !txscript.IsBurnTy(txout.PkScript) {
//			return nil, NoBurnTx
//		}
//	} else {
//		return nil, NoBurnTx
//	}
//
//	// get to address
//	if len(tx.TxOut) < 2 {
//		return nil, errors.New("BurnTx Must be at least two TxOut")
//	}
//	if r, err := IsSendToZeroAddress(tx.TxOut[1].PkScript); err != nil {
//		return nil, err
//	} else {
//		if !r {
//			return nil, errors.New("not send to burn address")
//		}
//	}
//	// get from address
//	if tx.TxIn[0].Witness == nil {
//		pk, err = txscript.ComputePk(tx.TxIn[0].SignatureScript)
//		if err != nil {
//			e := fmt.Sprintf("ComputePk err %s", err)
//			return nil, errors.New(e)
//		}
//	} else {
//		pk, err = txscript.ComputeWitnessPk(tx.TxIn[0].Witness)
//		if err != nil {
//			e := fmt.Sprintf("ComputeWitnessPk err %s", err)
//			return nil, errors.New(e)
//		}
//	}
//
//	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), params)
//	if err != nil {
//		e := fmt.Sprintf("NewAddressPubKeyHash err %s", err)
//		return nil, errors.New(e)
//	}
//
//	txout := tx.TxOut[0]
//	info, err := BurnInfoFromScript(txout.PkScript)
//	if err != nil {
//		return nil, errors.New("BurnInfoFromScript the output tx " + err.Error())
//	} else {
//		if txout.Value != 0 {
//			return nil, errors.New("the output value must be 0 in tx.")
//		}
//		es = info
//	}
//
//	info.Amount = big.NewInt(tx.TxOut[1].Value)
//	info.Address = address.String()
//
//	if es != nil {
//		return es, nil
//	}
//	return nil, NoBurnTx
//}

func IsSendToZeroAddress(PkScript []byte) (bool, error) {
	if pks, err := txscript.ParsePkScript(PkScript); err != nil {
		return false, err
	} else {
		if pks.Class() != txscript.PubKeyHashTy {
			return false, errors.New("Burn tx only support PubKeyHashTy")
		}
		if t, err := pks.Address(&chaincfg.MainNetParams); err != nil {
			return false, err
		} else {
			toAddress := new(big.Int).SetBytes(t.ScriptAddress()).Uint64()
			if toAddress != 0 {
				return false, nil
			}
		}
	}
	return true, nil
}

//func IsBurnProofTx(tx *wire.MsgTx) (*BurnProofInfo, error) {
//	// make sure at least one txout in OUTPUT
//	var es *BurnProofInfo
//	var err error
//
//	if len(tx.TxOut) > 0 {
//		txout := tx.TxOut[0]
//		if !txscript.IsBurnProofTy(txout.PkScript) {
//			return nil, NoBurnProofTx
//		}
//	} else {
//		return nil, NoBurnProofTx
//	}
//
//	if len(tx.TxOut) < 1 {
//		return nil, errors.New("BurnProofInfo Must be at least two TxOut")
//	}
//
//	txout := tx.TxOut[0]
//	info, err := BurnProofInfoFromScript(txout.PkScript)
//	if err != nil {
//		return nil, errors.New("BurnProofInfoFromScript the output tx.")
//	} else {
//		if txout.Value != 0 {
//			return nil, errors.New("the output value must be 0 in tx.")
//		}
//		es = info
//	}
//
//	if es != nil {
//		return es, nil
//	}
//	return nil, NoBurnProofTx
//}
//
//func IsBurnReportWhiteListTx(tx *wire.MsgTx) (*WhiteListProof, error) {
//	// make sure at least one txout in OUTPUT
//	var es *WhiteListProof
//	var err error
//
//	if len(tx.TxOut) > 0 {
//		txout := tx.TxOut[0]
//		if !txscript.IsBurnReportWhiteListTy(txout.PkScript) {
//			return nil, NoBurnReportWhiteListTx
//		}
//	} else {
//		return nil, NoBurnReportWhiteListTx
//	}
//
//	txout := tx.TxOut[0]
//	info, err := WhiteListProofFromScript(txout.PkScript)
//	if err != nil {
//		return nil, errors.New("WhiteListProofFromScript the output tx.")
//	} else {
//		if txout.Value != 0 {
//			return nil, errors.New("the output value must be 0 in tx.")
//		}
//		es = info
//	}
//	if es != nil {
//		return es, nil
//	}
//	return nil, NoBurnReportWhiteListTx
//}

//func EntangleTxFromScript(script []byte) (*EntangleTxInfo, error) {
//	data, err := txscript.GetEntangleInfoData(script)
//	if err != nil {
//		return nil, err
//	}
//	info := &EntangleTxInfo{}
//	err = info.Parse(data)
//	return info, err
//}

//func ExChangeTxFromScript(script []byte) (*ExChangeTxInfo, error) {
//	data, err := txscript.GetExChangeInfoData(script)
//	if err != nil {
//		return nil, err
//	}
//	info := &ExChangeTxInfo{}
//	err = rlp.DecodeBytes(data, info)
//	return info, err
//}

func ConvertTxFromScript(script []byte) (*ConvertTxInfo, error) {
	data, err := txscript.GetConvertInfoData(script)
	if err != nil {
		return nil, err
	}
	info := &ConvertTxInfo{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

//  Beacon
func BeaconRegistrationTxFromScript(script []byte) (*BeaconAddressInfo, error) {
	data, err := txscript.GetBeaconRegistrationData(script)
	if err != nil {
		return nil, err
	}
	info := &BeaconAddressInfo{}
	err = rlp.DecodeBytes(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

//  Beacon
func BeaconRegistrationTxFromScript2(script []byte) (*BeaconAddressInfo, error) {
	data, err := txscript.GetBeaconRegistrationData(script)
	if err != nil {
		return nil, err
	}
	info := &BeaconAddressInfo{}
	err = rlp.DecodeBytes(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func AddBeaconPledgeTxFromScript(script []byte) (*AddBeaconPledge, error) {
	data, err := txscript.GetAddBeaconPledgeData(script)
	if err != nil {
		return nil, err
	}
	info := &AddBeaconPledge{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

//func UpdateBeaconCoinbaseTxFromScript(script []byte) (*UpdateBeaconCoinbase, error) {
//	data, err := txscript.GetUpdateBeaconCoinbaseData(script)
//	if err != nil {
//		return nil, err
//	}
//	info := &UpdateBeaconCoinbase{}
//	err = rlp.DecodeBytes(data, info)
//	return info, err
//}

//func UpdateBeaconFreeQuotaTxFromScript(script []byte) (*UpdateBeaconFreeQuota, error) {
//	data, err := txscript.GetUpdateBeaconFreeQuotaData(script)
//	if err != nil {
//		return nil, err
//	}
//	info := &UpdateBeaconFreeQuota{}
//	err = rlp.DecodeBytes(data, info)
//	return info, err
//}

//func BurnInfoFromScript(script []byte) (*BurnTxInfo, error) {
//	data, err := txscript.GetBurnInfoData(script)
//	if err != nil {
//		return nil, err
//	}
//	info := &BurnTxInfo{}
//	err = rlp.DecodeBytes(data, info)
//	return info, err
//}
//func BurnProofInfoFromScript(script []byte) (*BurnProofInfo, error) {
//	data, err := txscript.GetBurnProofInfoData(script)
//	if err != nil {
//		return nil, err
//	}
//	info := &BurnProofInfo{}
//	err = rlp.DecodeBytes(data, info)
//	return info, err
//}
//func WhiteListProofFromScript(script []byte) (*WhiteListProof, error) {
//	data, err := txscript.GetWhiteListProofData(script)
//	if err != nil {
//		return nil, err
//	}
//	info := &WhiteListProof{}
//	err = rlp.DecodeBytes(data, info)
//	return info, err
//}

/////////////////////////////////////////////////////////////////////////////

func GetMaxHeight(items map[uint32]*ConvertTxInfo) uint64 {
	h := uint64(0)
	for _, v := range items {
		if h < v.Height {
			h = v.Height
		}
	}
	return h
}

func MakeMergerCoinbaseTx(tx *wire.MsgTx, pool *PoolAddrItem, items []*ExChangeItem, rewards []*PunishedRewardItem, mergeItem map[uint64][]*BeaconMergeItem) error {

	if pool == nil || len(pool.POut) == 0 {
		return nil
	}
	// make sure have enough Value to exchange
	poolIn1 := &wire.TxIn{
		PreviousOutPoint: pool.POut[0],
		SignatureScript:  pool.Script[0],
		Sequence:         wire.MaxTxInSequenceNum,
	}
	poolIn2 := &wire.TxIn{
		PreviousOutPoint: pool.POut[1],
		SignatureScript:  pool.Script[1],
		Sequence:         wire.MaxTxInSequenceNum,
	}
	// merge pool tx
	tx.TxIn[1], tx.TxIn[2] = poolIn1, poolIn2

	// reward the proof ,txin>3
	for _, v := range rewards {
		tx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: v.POut,
			SignatureScript:  v.Script,
			Sequence:         wire.MaxTxInSequenceNum,
		})
		pkScript1, err1 := txscript.PayToAddrScript(v.Addr1) // reward to robot
		pkScript2, err2 := txscript.PayToAddrScript(v.Addr2) // punished to zero address
		pkScript3, err3 := txscript.PayToAddrScript(v.Addr3) // change address
		if err1 != nil || err2 != nil || err3 != nil {
			return errors.New("PayToAddrScript failed, in reward the proof")
		}
		tx.AddTxOut(&wire.TxOut{
			Value:    new(big.Int).Set(v.Amount).Int64(),
			PkScript: pkScript1,
		})
		tx.AddTxOut(&wire.TxOut{
			Value:    new(big.Int).Set(v.Amount).Int64(),
			PkScript: pkScript2,
		})
		tx.AddTxOut(&wire.TxOut{
			Value:    new(big.Int).Sub(v.OriginAmount, new(big.Int).Mul(big.NewInt(2), v.Amount)).Int64(),
			PkScript: pkScript3,
		})
	}

	reserve1, reserve2 := pool.Amount[0].Int64()+tx.TxOut[1].Value, pool.Amount[1].Int64()
	updateTxOutValue(tx.TxOut[2], reserve2)
	allEntangle := int64(0)

	for i := range items {
		pkScript, err := txscript.PayToAddrScript(items[i].Address)
		if err != nil {
			return errors.New("Make Meger tx failed,err: " + err.Error())
		}
		out := &wire.TxOut{
			Value:    items[i].Amount.Int64(),
			PkScript: pkScript,
		}
		allEntangle += out.Value
		tx.AddTxOut(out)
	}
	tx.TxOut[1].Value = reserve1 - allEntangle
	if tx.TxOut[1].Value < 0 {
		return errors.New("pool1 amount < 0")
	}

	// merge beacon utxo
	var to czzutil.Address
	for _, Items := range mergeItem {
		allAmount := big.NewInt(0)
		for _, v := range Items {
			to = v.ToAddress
			tx.AddTxIn(&wire.TxIn{
				PreviousOutPoint: v.POut,
				SignatureScript:  v.Script,
				Sequence:         wire.MaxTxInSequenceNum,
			})
			allAmount = new(big.Int).Add(allAmount, v.Amount)
		}
		pkScript1, err1 := txscript.PayToAddrScript(to) // change address
		if err1 != nil {
			return errors.New("PayToAddrScript failed, in merge beacon utxo")
		}
		tx.AddTxOut(&wire.TxOut{
			Value:    new(big.Int).Set(allAmount).Int64(),
			PkScript: pkScript1,
		})
	}
	return nil
}

func MakeMergerCoinbaseTx2(tx *wire.MsgTx, pool *PoolAddrItem, items []*EntangleItem,
	lastScriptInfo []byte, fork bool) error {

	if pool == nil || len(pool.POut) == 0 {
		return nil
	}
	keepInfo, err := KeepedAmountFromScript(lastScriptInfo)
	if err != nil {
		return err
	}
	// make sure have enough Value to exchange
	poolIn1 := &wire.TxIn{
		PreviousOutPoint: pool.POut[0],
		SignatureScript:  pool.Script[0],
		Sequence:         wire.MaxTxInSequenceNum,
	}
	poolIn2 := &wire.TxIn{
		PreviousOutPoint: pool.POut[1],
		SignatureScript:  pool.Script[1],
		Sequence:         wire.MaxTxInSequenceNum,
	}
	// merge pool tx
	tx.TxIn[1], tx.TxIn[2] = poolIn1, poolIn2

	reserve1, reserve2 := pool.Amount[0].Int64()+tx.TxOut[1].Value, pool.Amount[1].Int64()
	updateTxOutValue(tx.TxOut[2], reserve2)
	if ok := EnoughAmount(reserve1, items, keepInfo, fork); !ok {
		return errors.New("not enough amount to be entangle...")
	}

	for i := range items {
		calcExchange(items[i], &reserve1, keepInfo, true, fork)
		pkScript, err := txscript.PayToAddrScript(items[i].Addr)
		if err != nil {
			return errors.New("Make Meger tx failed,err: " + err.Error())
		}
		out := &wire.TxOut{
			Value:    items[i].Value.Int64(),
			PkScript: pkScript,
		}
		tx.AddTxOut(out)
	}
	keepEntangleAmount(keepInfo, tx)
	tx.TxOut[1].Value = reserve1
	if reserve1 < reserve2 {
		fmt.Println("as")
	}
	return nil
}

func EnoughAmount(reserve int64, items []*EntangleItem, keepInfo *KeepedAmount, fork bool) bool {
	amount := reserve
	for _, v := range items {
		calcExchange(v.Clone(), &amount, keepInfo, false, fork)
	}
	return amount > 0
}

func calcExchange(item *EntangleItem, reserve *int64, keepInfo *KeepedAmount, change, fork bool) {
	amount := big.NewInt(0)
	cur := keepInfo.GetValue(item.AssetType)
	if cur != nil {
		amount = new(big.Int).Set(cur)
	}
	if change {
		kk := KeepedItem{
			AssetType: item.AssetType,
			Amount:    new(big.Int).Set(item.Value),
		}
		keepInfo.Add(kk)
	}
	if item.AssetType == ExpandedTxEntangle_Doge {
		if fork {
			item.Value = toDoge2(amount, item.Value)
		} else {
			item.Value = toDoge(amount, item.Value, fork)
		}
	} else if item.AssetType == ExpandedTxEntangle_Ltc {
		if fork {
			item.Value = toLtc2(amount, item.Value)
		} else {
			item.Value = toLtc(amount, item.Value, fork)
		}
	}
	*reserve = *reserve - item.Value.Int64()
}

func keepEntangleAmount(info *KeepedAmount, tx *wire.MsgTx) error {
	var scriptInfo []byte
	var err error

	scriptInfo, err = txscript.KeepedAmountScript(info.Serialize())
	if err != nil {
		return err
	}
	txout := &wire.TxOut{
		Value:    0,
		PkScript: scriptInfo,
	}
	tx.TxOut[3] = txout
	return nil
}

//func MakeMergerCoinbaseTx2(tx *wire.MsgTx, pool *PoolAddrItem, items []*EntangleItem,
//	lastScriptInfo []byte, fork bool) error {
//	if pool == nil || len(pool.POut) == 0 {
//		return nil
//	}
//	keepInfo, err := KeepedAmountFromScript(lastScriptInfo)
//	if err != nil {
//		return err
//	}
//	// make sure have enough Value to exchange
//	poolIn1 := &wire.TxIn{
//		PreviousOutPoint: pool.POut[0],
//		SignatureScript:  pool.Script[0],
//		Sequence:         wire.MaxTxInSequenceNum,
//	}
//	poolIn2 := &wire.TxIn{
//		PreviousOutPoint: pool.POut[1],
//		SignatureScript:  pool.Script[1],
//		Sequence:         wire.MaxTxInSequenceNum,
//	}
//	// merge pool tx
//	tx.TxIn[1], tx.TxIn[2] = poolIn1, poolIn2
//
//	reserve1, reserve2 := pool.Amount[0].Int64()+tx.TxOut[1].Value, pool.Amount[1].Int64()
//	updateTxOutValue(tx.TxOut[2], reserve2)
//	if ok := EnoughAmount2(reserve1, items, keepInfo, fork); !ok {
//		return errors.New("not enough amount to be entangle...")
//	}
//
//	for i := range items {
//		calcExchange2(items[i], &reserve1, keepInfo, true, fork)
//		pkScript, err := txscript.PayToAddrScript(items[i].Addr)
//		if err != nil {
//			return errors.New("Make Meger tx failed,err: " + err.Error())
//		}
//		out := &wire.TxOut{
//			Value:    items[i].Value.Int64(),
//			PkScript: pkScript,
//		}
//		tx.AddTxOut(out)
//	}
//	keepEntangleAmount(keepInfo, tx)
//	tx.TxOut[1].Value = reserve1
//	if reserve1 < reserve2 {
//		fmt.Println("as")
//	}
//	return nil
//}

func updateTxOutValue(out *wire.TxOut, value int64) error {
	out.Value += value
	return nil
}

//func calcExchange(item *ExChangeItem, reserve *int64, keepInfo *KeepedAmount, change, fork bool) {
//	amount := big.NewInt(0)
//	cur := keepInfo.GetValue(item.AssetType)
//	if cur != nil {
//		amount = new(big.Int).Set(cur)
//	}
//	if change {
//		kk := KeepedItem{
//			AssetType: item.AssetType,
//			Amount:    new(big.Int).Set(item.Value),
//		}
//		keepInfo.Add(kk)
//	}
//	if item.AssetType == ExpandedTxEntangle_Doge {
//		if fork {
//			item.Value = toDoge2(amount, item.Value)
//		} else {
//			item.Value = toDoge(amount, item.Value, fork)
//		}
//	} else if item.AssetType == ExpandedTxEntangle_Ltc {
//		if fork {
//			item.Value = toLtc2(amount, item.Value)
//		} else {
//			item.Value = toLtc(amount, item.Value, fork)
//		}
//	} else if item.AssetType == ExpandedTxEntangle_Btc {
//		item.Value = toBtc(amount, item.Value)
//	} else if item.AssetType == ExpandedTxEntangle_Bch {
//		item.Value = toBchOrBsv(amount, item.Value)
//	} else if item.AssetType == ExpandedTxEntangle_Bsv {
//		item.Value = toBchOrBsv(amount, item.Value)
//	}
//	*reserve = *reserve - item.Value.Int64()
//}
//
//func calcExchange2(item *EntangleItem, reserve *int64, keepInfo *KeepedAmount, change, fork bool) {
//	amount := big.NewInt(0)
//	cur := keepInfo.GetValue(item.AssetType)
//	if cur != nil {
//		amount = new(big.Int).Set(cur)
//	}
//	if change {
//		kk := KeepedItem{
//			AssetType: item.AssetType,
//			Amount:    new(big.Int).Set(item.Value),
//		}
//		keepInfo.Add(kk)
//	}
//	if item.AssetType == ExpandedTxEntangle_Doge {
//		if fork {
//			item.Value = toDoge2(amount, item.Value)
//		} else {
//			item.Value = toDoge(amount, item.Value, fork)
//		}
//	} else if item.AssetType == ExpandedTxEntangle_Ltc {
//		if fork {
//			item.Value = toLtc2(amount, item.Value)
//		} else {
//			item.Value = toLtc(amount, item.Value, fork)
//		}
//	} else if item.AssetType == ExpandedTxEntangle_Btc {
//		item.Value = toBtc(amount, item.Value)
//	} else if item.AssetType == ExpandedTxEntangle_Bch {
//		item.Value = toBchOrBsv(amount, item.Value)
//	} else if item.AssetType == ExpandedTxEntangle_Bsv {
//		item.Value = toBchOrBsv(amount, item.Value)
//	} else if item.AssetType == ExpandedTxEntangle_Usdt {
//		item.Value = toUSDT(amount, item.Value)
//	} else if item.AssetType == ExpandedTxEntangle_Eth {
//		item.Value = toETH(amount, item.Value)
//	} else if item.AssetType == ExpandedTxEntangle_Trx {
//		item.Value = toTRX(amount, item.Value)
//	}
//	*reserve = *reserve - item.Value.Int64()
//}

//func PreCalcEntangleAmount(item *ExChangeItem, keepInfo *KeepedAmount, fork bool) {
//	var vv int64
//	calcExchange(item, &vv, keepInfo, true, fork)
//}
//
//func EnoughAmount(reserve int64, items []*ExChangeItem, keepInfo *KeepedAmount, fork bool) bool {
//	amount := reserve
//	for _, v := range items {
//		calcExchange(v.Clone(), &amount, keepInfo, false, fork)
//	}
//	return amount > 0
//}
//
//func EnoughAmount2(reserve int64, items []*EntangleItem, keepInfo *KeepedAmount, fork bool) bool {
//	amount := reserve
//	for _, v := range items {
//		calcExchange2(v.Clone(), &amount, keepInfo, false, fork)
//	}
//	return amount > 0
//}
//
//func keepEntangleAmount(info *KeepedAmount, tx *wire.MsgTx) error {
//	var scriptInfo []byte
//	var err error
//
//	scriptInfo, err = txscript.KeepedAmountScript(info.Serialize())
//	if err != nil {
//		return err
//	}
//	txout := &wire.TxOut{
//		Value:    0,
//		PkScript: scriptInfo,
//	}
//	tx.TxOut[3] = txout
//	return nil
//}
func KeepedAmountFromScript(script []byte) (*KeepedAmount, error) {
	if script == nil {
		return &KeepedAmount{Items: []KeepedItem{}}, nil
	}
	data, err1 := txscript.GetKeepedAmountData(script)
	if err1 != nil {
		return nil, err1
	}
	keepInfo := &KeepedAmount{Items: []KeepedItem{}}
	err := keepInfo.Parse(data)
	return keepInfo, err
}

// the tool function for entangle tx
type TmpAddressPair struct {
	index   uint32
	Address czzutil.Address
}

// only pairs[0] is valid
//func ToAddressFromExChange(tx *czzutil.Tx, ev *ExChangeVerify, eState *EntangleState) ([]*TmpAddressPair, error) {
//
//	einfo, _ := IsExChangeTx(tx.MsgTx())
//	if einfo != nil {
//		// verify the entangle tx
//		pairs := make([]*TmpAddressPair, 0)
//		tt, err := ev.VerifyExChangeTx(tx.MsgTx(), eState)
//		if err != nil {
//			return nil, err
//		}
//		for _, v := range tt {
//			pub, err := RecoverPublicFromBytes(v.Pub, v.AssetType)
//			if err != nil {
//				return nil, err
//			}
//			addr, err := MakeAddress(*pub, ev.Params)
//			if err != nil {
//				return nil, err
//			}
//			pairs = append(pairs, &TmpAddressPair{
//				index:   v.Index,
//				Address: addr,
//			})
//		}
//
//		return pairs, nil
//	}
//
//	return nil, nil
//}

func ToAddressFromConverts(eState *CommitteeState, cinfo map[uint32]*ConvertTxInfo, ev *ExChangeVerify, nextBlockHeight int64) ([]*ExChangeItem, error) {

	cis := make([]*ExChangeItem, 0)
	for _, info := range cinfo {
		// verify the entangle tx
		tpi, err := ev.VerifyConvertTx(info, eState)
		if err != nil {
			return nil, err
		}

		pub, err := RecoverPublicFromBytes(tpi.Pub, tpi.AssetType)
		if err != nil {
			return nil, err
		}

		addr, err := MakeAddress(*pub, ev.Params)
		if err != nil {
			return nil, err
		}

		citem := &ExChangeItem{
			ExtTxHash:   info.ExtTxHash,
			Height:      big.NewInt(nextBlockHeight),
			AssetType:   info.AssetType,
			ConvertType: info.ConvertType,
			Address:     addr,
			Amount:      info.Amount,
		}

		if err := eState.AddConvertItem(citem); err != nil {
			return nil, err
		}

		cis = append(cis, citem)
	}

	return cis, nil
}

//func OverEntangleAmount(tx *wire.MsgTx, pool *PoolAddrItem, items []*ExChangeItem,
//	lastScriptInfo []byte, fork bool, state *EntangleState) bool {
//	if items == nil || len(items) == 0 {
//		return false
//	}
//
//	var keepInfo *KeepedAmount
//	var err error
//	if fork {
//		types := []uint8{}
//		for _, v := range items {
//			types = append(types, v.AssetType)
//		}
//		keepInfo = getKeepInfosFromState(state, types)
//	} else {
//		keepInfo, err = KeepedAmountFromScript(lastScriptInfo)
//	}
//	if err != nil || keepInfo == nil {
//		return false
//	}
//	all := pool.Amount[0].Int64() + tx.TxOut[1].Value
//	return !EnoughAmount(all, items, keepInfo, fork)
//}

//func getKeepInfosFromState(state *EntangleState, types []uint32) *KeepedAmount {
//	if state == nil {
//		return nil
//	}
//	keepinfo := &KeepedAmount{Items: []KeepedItem{}}
//	for _, v := range types {
//		keepinfo.Add(KeepedItem{
//			AssetType: v,
//			Amount:    state.getAllEntangleAmount(v),
//		})
//	}
//	return keepinfo
//}

// the return value is beacon's balance of it was staking amount
//func VerifyWhiteListProof(info *WhiteListProof, ev *ExChangeVerify, state *EntangleState) error {
//	if err := state.VerifyWhiteListProof(info); err != nil {
//		return err
//	}
//	cur := state.GetOutSideAsset(info.LightID, info.Atype)
//	if cur == nil {
//		return ErrNoRegister
//	}
//	return ev.VerifyWhiteList(cur, info, state)
//}

//func FinishWhiteListProof(info *WhiteListProof, state *EntangleState) error {
//	return state.FinishWhiteListProof(info)
//}
//func CloseProofForPunished(info *BurnProofInfo, item *BurnItem, state *EntangleState) error {
//	return state.CloseProofForPunished(info, item)
//}
//
////////////////////////////////////////////////////////////////////////////////
//func ScanTxForBeaconOnSpecHeight(beacon map[uint64][]byte) {
//
//}
//
//// just fetch the outpoint info for beacon address's regsiter and append tx
//func fetchOutPointFromTxs(txs []*czzutil.Tx, beacon map[uint64][]byte, state *EntangleState,
//	params *chaincfg.Params) map[uint64][]*wire.OutPoint {
//	res := make(map[uint64][]*wire.OutPoint)
//	for _, v := range txs {
//		_, e1 := IsAddBeaconPledgeTx(v.MsgTx(), params)
//		_, e2 := IsBeaconRegistrationTx(v.MsgTx(), params)
//		if e1 != nil || e2 != nil {
//			_, addrs, _, err := txscript.ExtractPkScriptAddrs(v.MsgTx().TxOut[1].PkScript, params)
//			if err != nil {
//				to := addrs[0].ScriptAddress()
//				id := state.GetBeaconIdByTo(to)
//				if id != 0 {
//					toAddress := big.NewInt(0).SetBytes(to).Uint64()
//					if toAddress >= 10 && toAddress <= 99 {
//						res[id] = append(res[id], wire.NewOutPoint(v.Hash(), 1))
//					}
//				}
//			}
//		}
//	}
//	return res
//}
//func SameHeightTxForBurn(tx *czzutil.Tx, txs []*czzutil.Tx, params *chaincfg.Params) bool {
//	info, e := IsBurnTx(tx.MsgTx(), params)
//	if e != nil || info == nil {
//		return false
//	}
//	for _, v := range txs {
//		if info1, err := IsBurnTx(v.MsgTx(), params); err == nil {
//			if info1 != nil && info1.Address == info.Address {
//				return true
//			}
//		}
//	}
//	return false
//}
//func GetAddressFromProofTx(tx *czzutil.Tx, params *chaincfg.Params) (czzutil.Address, error) {
//
//	var pk []byte
//	var err error
//	if tx.MsgTx().TxIn[0].Witness == nil {
//		pk, err = txscript.ComputePk(tx.MsgTx().TxIn[0].SignatureScript)
//		if err != nil {
//			return nil, err
//		}
//	} else {
//		pk, err = txscript.ComputeWitnessPk(tx.MsgTx().TxIn[0].Witness)
//		if err != nil {
//			return nil, err
//		}
//	}
//
//	addrs, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), params)
//	if err != nil {
//		return nil, err
//	}
//
//	//_, addrs, _, err := txscript.ExtractPkScriptAddrs(tx.MsgTx().TxOut[0].PkScript, params)
//	//if err == nil {
//	//	return nil
//	//}
//	return addrs, nil
//}

//////////////////////////////////////////////////////////////////////////////
func toDoge1(entangled, needed int64) int64 {
	if needed <= 0 {
		return 0
	}
	var kk, rate int64 = 0, 25
	rate = rate + int64(entangled/int64(12500000))
	p := entangled % int64(12500000)

	if (int64(12500000) - p) >= needed {
		f1 := big.NewFloat(float64(needed))
		f1 = f1.Quo(f1, big.NewFloat(float64(rate)))
		kk = toCzz(f1).Int64()
	} else {
		v1 := big.NewFloat(float64(int64(12500000) - p))
		v2 := big.NewFloat(float64(needed - p))
		r1 := big.NewFloat(float64(rate))
		v1 = v1.Quo(v1, r1)
		kk = toCzz(v1).Int64()
		rate += 1
		r2 := big.NewFloat(float64(rate))
		v2 = v2.Quo(v2, r2)
		kk = kk + toCzz(v2).Int64()
	}
	return kk
}
func reverseToDoge(keeped *big.Int) (*big.Int, *big.Int) {
	base, divisor := big.NewInt(int64(25)), big.NewInt(1)
	loopUnit := new(big.Int).Mul(big.NewInt(1150), baseUnit)
	divisor0, _ := new(big.Int).DivMod(keeped, loopUnit, new(big.Int).Set(loopUnit))
	return base.Add(base, divisor0), divisor
}

// doge has same precision with czz
func toDoge2(entangled, needed *big.Int) *big.Int {
	if needed == nil || needed.Int64() <= 0 {
		return big.NewInt(0)
	}
	keep, change := new(big.Int).Set(entangled), new(big.Int).Set(needed)
	base := big.NewInt(int64(25))
	loopUnit := new(big.Int).Mul(big.NewInt(12500000), baseUnit)
	res := big.NewInt(0)
	for {
		if change.Sign() <= 0 {
			break
		}
		divisor, remainder := new(big.Int).DivMod(keep, loopUnit, new(big.Int).Set(loopUnit))
		rate := new(big.Int).Mul(new(big.Int).Add(base, divisor), baseUnit)
		l := new(big.Int).Sub(loopUnit, remainder)
		if l.Cmp(change) >= 0 {
			res0 := new(big.Int).Quo(new(big.Int).Mul(change, baseUnit), rate)
			res = res.Add(res, res0)
			break
		} else {
			change = change.Sub(change, l)
			res0 := new(big.Int).Quo(new(big.Int).Mul(l, baseUnit), rate)
			res = res.Add(res, res0)
			keep = keep.Add(keep, l)
		}
	}
	return res
}

func toDoge(entangled, needed *big.Int, fork bool) *big.Int {
	if needed == nil || needed.Int64() <= 0 {
		return big.NewInt(0)
	}
	var du *big.Int = nil
	if fork {
		du = new(big.Int).Set(dogeUnit)
	} else {
		du = new(big.Int).Set(dogeUnit1)
	}
	var rate int64 = 25
	z, m := new(big.Int).DivMod(entangled, du, new(big.Int).Set(du))
	rate = rate + z.Int64()
	l := new(big.Int).Sub(du, m)
	base := new(big.Float).SetFloat64(float64(baseUnit.Int64()))

	if l.Cmp(needed) >= 1 {
		f1 := new(big.Float).Quo(new(big.Float).SetInt(needed), base)
		f1 = f1.Quo(f1, big.NewFloat(float64(rate)))
		return toCzz(f1)
	} else {
		v1 := new(big.Float).Quo(new(big.Float).SetInt(l), base)
		v2 := new(big.Float).Sub(new(big.Float).SetInt(needed), new(big.Float).SetInt(l))
		v2 = v2.Quo(v2, base)
		v1 = v1.Quo(v1, big.NewFloat(float64(rate)))
		rate += 1
		v2 = v2.Quo(v2, big.NewFloat(float64(rate)))
		return new(big.Int).Add(toCzz(v1), toCzz(v2))
	}
}

func toLtc1(entangled, needed int64) int64 {
	if needed <= 0 {
		return 0
	}
	var ret int64 = 0
	rate := big.NewFloat(0.0008)
	base := big.NewFloat(0.0001)

	fixed := int64(1150)
	divisor := entangled / fixed
	remainder := entangled % fixed

	base1 := base.Mul(base, big.NewFloat(float64(divisor)))
	rate = rate.Add(rate, base1)

	if fixed-remainder >= needed {
		f1 := big.NewFloat(float64(needed))
		f1 = f1.Quo(f1, rate)
		ret = toCzz(f1).Int64()
	} else {
		v1 := fixed - remainder
		v2 := needed - remainder
		f1, f2 := big.NewFloat(float64(v1)), big.NewFloat(float64(v2))
		f1 = f1.Quo(f1, rate)
		rate = rate.Add(rate, base)
		f2 = f2.Quo(f2, rate)
		ret = toCzz(f1).Int64() + toCzz(f2).Int64()
	}
	return ret
}
func reverseToLtc(keeped *big.Int) (base, divisor *big.Int) {
	base, divisor = big.NewInt(int64(80000)), big.NewInt(1)
	loopUnit := new(big.Int).Mul(big.NewInt(1150), baseUnit)
	divisor0, _ := new(big.Int).DivMod(keeped, loopUnit, new(big.Int).Set(loopUnit))
	return base.Add(base, divisor0), divisor
}

// ltc has same precision with czz
func toLtc2(entangled, needed *big.Int) *big.Int {
	if needed == nil || needed.Int64() <= 0 {
		return big.NewInt(0)
	}
	keep, change := new(big.Int).Set(entangled), new(big.Int).Set(needed)
	base := big.NewInt(int64(80000))
	loopUnit := new(big.Int).Mul(big.NewInt(1150), baseUnit)
	res := big.NewInt(0)
	for {
		if change.Sign() <= 0 {
			break
		}
		divisor, remainder := new(big.Int).DivMod(keep, loopUnit, new(big.Int).Set(loopUnit))
		rate := new(big.Int).Add(base, divisor)
		l := new(big.Int).Sub(loopUnit, remainder)
		if l.Cmp(change) >= 0 {
			res0 := new(big.Int).Quo(new(big.Int).Mul(change, baseUnit), rate)
			res = res.Add(res, res0)
			break
		} else {
			change = change.Sub(change, l)
			res0 := new(big.Int).Quo(new(big.Int).Mul(l, baseUnit), rate)
			res = res.Add(res, res0)
			keep = keep.Add(keep, l)
		}
	}
	return res
}
func toLtc(entangled, needed *big.Int, fork bool) *big.Int {
	if needed == nil || needed.Int64() <= 0 {
		return big.NewInt(0)
	}
	rate := big.NewFloat(0.0008)
	base := big.NewFloat(0.0001)
	var du *big.Int = nil
	if fork {
		du = new(big.Int).Set(baseUnit)
	} else {
		du = new(big.Int).Set(baseUnit1)
	}
	u := new(big.Float).SetFloat64(float64(du.Int64()))
	fixed := new(big.Int).Mul(big.NewInt(int64(1150)), du)
	divisor, remainder := new(big.Int).DivMod(entangled, fixed, new(big.Int).Set(fixed))

	base1 := new(big.Float).Mul(base, big.NewFloat(float64(divisor.Int64())))
	rate = rate.Add(rate, base1)
	l := new(big.Int).Sub(fixed, remainder)

	if l.Cmp(needed) >= 1 {
		// f1 := new(big.Float).Quo(new(big.Float).SetInt(needed), u)
		f1 := new(big.Float).Quo(new(big.Float).SetFloat64(float64(needed.Int64())), u)
		f1 = f1.Quo(f1, rate)
		return toCzz(f1)
	} else {
		f1 := new(big.Float).Quo(new(big.Float).SetFloat64(float64(l.Int64())), u)
		f2 := new(big.Float).Quo(new(big.Float).SetFloat64(float64(new(big.Int).Sub(needed, l).Int64())), u)
		f1 = f1.Quo(f1, rate)
		rate = rate.Add(rate, base)
		f2 = f2.Quo(f2, rate)
		return new(big.Int).Add(toCzz(f1), toCzz(f2))
	}
}

func reverseToTRX(keeped *big.Int) (*big.Int, *big.Int) {
	unit, base := big.NewInt(int64(10)), big.NewInt(int64(200))
	divisor, _ := new(big.Int).DivMod(keeped, baseUnit, new(big.Int).Set(baseUnit))
	rate := new(big.Int).Add(base, new(big.Int).Mul(unit, divisor))
	return rate, big.NewInt(1)
}

func toCzz(val *big.Float) *big.Int {
	val = val.Mul(val, big.NewFloat(float64(baseUnit.Int64())))
	ii, _ := val.Int64()
	return big.NewInt(ii)
}

func fromCzz(val int64) *big.Float {
	v := new(big.Float).Quo(big.NewFloat(float64(val)), big.NewFloat(float64(baseUnit.Int64())))
	return v
}

func fromCzz1(val *big.Int) *big.Float {
	fval := new(big.Float).SetInt(val)
	fval = fval.Quo(fval, new(big.Float).SetInt(baseUnit))
	return fval
}

func countMant(value *big.Float, prec int) int {
	if !value.Signbit() {
		str := value.Text('f', prec)
		return len(strings.Split(fmt.Sprintf("%v", str), ".")[1])
	}
	return 0
}

func makeExp(exp int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}

func makeMant(value *big.Float, prec int) *big.Int {
	base := new(big.Float).SetFloat64(float64(makeExp(countMant(value, prec)).Uint64()))
	v := new(big.Float).Mul(value, base)
	val, _ := v.Int64()
	return big.NewInt(val)
}
