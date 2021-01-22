package cross

import (
	"bytes"
	"encoding/hex"
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
	ExpandedTxConvert_Czz
	ExpandedTxConvert_ECzz
	ExpandedTxConvert_TCzz
	ExpandedTxConvert_HCzz
)

var (
	NoBeaconRegistration = errors.New("no BeaconRegistration info in transcation")
	NoAddBeaconPledge    = errors.New("no AddBeaconPledge info in transcation")
	NoMortgage           = errors.New("no NoMortgage info in transcation")
	NoAddMortgage        = errors.New("no NoAddMortgage info in transcation")
	NoUpdateCoinbaseAll  = errors.New("no NoUpdateCoinbaseAll info in transcation")
	NoConvert            = errors.New("no NoConvert info in transcation")
	NoCasting            = errors.New("no NoCasting info in transcation")

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

////////////////////////////////////////////////////////////////////////////
type Mortgage struct {
	ToAddress       []byte
	StakingAmount   *big.Int
	CoinBaseAddress []string
}

type AddMortgage struct {
	Address       string
	StakingAmount *big.Int
}

type UpdateCoinbaseAll struct {
	Address         string   `json:"address"`
	CoinBaseAddress []string `json:"coinbase_address"`
}

type ConvertTxInfo struct {
	AssetType   uint8
	ConvertType uint8
	PubKey      []byte
	Height      uint64
	ExtTxHash   string
	Index       uint32
	Amount      *big.Int
}

type CastingTxInfo struct {
	ConvertType uint8
	PubKey      []byte
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
func IsMortgageTx(tx *wire.MsgTx, params *chaincfg.Params) (*PledgeInfo, error) {
	// make sure at least one txout in OUTPUT
	if len(tx.TxOut) > 0 {
		txout := tx.TxOut[0]
		if !txscript.IsMortgageTy(txout.PkScript) {
			return nil, NoMortgage
		}
	} else {
		return nil, NoMortgage
	}

	if len(tx.TxOut) > 3 || len(tx.TxOut) < 2 || len(tx.TxIn) > 1 {
		return nil, fmt.Errorf("not BeaconRegistration tx TxOut >3 or TxIn >1")
	}

	var es *PledgeInfo
	txOut := tx.TxOut[0]
	info, err := MortgageFromScript(txOut.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txOut.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		es = &PledgeInfo{
			ToAddress:       info.ToAddress,
			StakingAmount:   info.StakingAmount,
			CoinBaseAddress: info.CoinBaseAddress,
		}
	}

	var pk []byte
	if tx.TxIn[0].Witness == nil {
		pk, err = txscript.ComputePk(tx.TxIn[0].SignatureScript)
		if err != nil {
			return nil, fmt.Errorf("ComputePk err %s", err)
		}
	} else {
		pk, err = txscript.ComputeWitnessPk(tx.TxIn[0].Witness)
		if err != nil {
			return nil, fmt.Errorf("ComputeWitnessPk err %s", err)
		}
	}

	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), params)
	if err != nil {
		return nil, fmt.Errorf("NewAddressPubKeyHash err %s", err)
	}

	es.StakingAmount = big.NewInt(tx.TxOut[1].Value)
	es.Address = address.String()
	es.PubKey = pk

	return es, nil
}

func IsAddMortgageTx(tx *wire.MsgTx, params *chaincfg.Params) (*AddMortgage, error) {
	// make sure at least one txOut in OUTPUT
	if len(tx.TxOut) > 0 {
		txout := tx.TxOut[0]
		if !txscript.IsAddMortgageTy(txout.PkScript) {
			return nil, NoAddMortgage
		}
	} else {
		return nil, NoAddMortgage
	}

	if len(tx.TxOut) > 3 || len(tx.TxOut) < 2 || len(tx.TxIn) > 1 {
		e := fmt.Sprintf("not BeaconRegistration tx TxOut >3 or TxIn >1")
		return nil, errors.New(e)
	}

	// make sure at least one txout in OUTPUT
	var am *AddMortgage

	txout := tx.TxOut[0]
	info, err := AddMortgageFromScript(txout.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txout.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		am = info
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

	am.StakingAmount = big.NewInt(tx.TxOut[1].Value)
	am.Address = address.String()

	if am != nil {
		return am, nil
	}
	return nil, NoAddBeaconPledge
}

func IsUpdateCoinbaseAllTx(tx *wire.MsgTx, params *chaincfg.Params) (*UpdateCoinbaseAll, error) {
	// make sure at least one txout in OUTPUT
	if len(tx.TxOut) > 0 {
		txout := tx.TxOut[0]
		if !txscript.IsUpdateCoinbaseAllTy(txout.PkScript) {
			return nil, NoUpdateCoinbaseAll
		}
	} else {
		return nil, NoUpdateCoinbaseAll
	}

	if len(tx.TxOut) > 2 || len(tx.TxOut) < 1 || len(tx.TxIn) > 1 {
		e := fmt.Sprintf("not BeaconRegistration tx TxOut >3 or TxIn >1")
		return nil, errors.New(e)
	}

	// make sure at least one txout in OUTPUT
	var uca *UpdateCoinbaseAll

	txout := tx.TxOut[0]
	info, err := UpdateBeaconCoinbaseAllTxFromScript(txout.PkScript)
	if err != nil {
		return nil, errors.New("UpdateBeaconCoinbaseTxFromScript the output tx.")
	} else {
		if txout.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		uca = info
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

	uca.Address = address.String()
	if uca != nil {
		return uca, nil
	}
	return nil, NoUpdateCoinbaseAll
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

func IsCastingTx(tx *wire.MsgTx) (*CastingTxInfo, error) {
	// make sure at least one txout in OUTPUT
	if len(tx.TxOut) > 0 {
		txout := tx.TxOut[0]
		if !txscript.IsCastingTy(txout.PkScript) {
			return nil, NoCasting
		}
	} else {
		return nil, NoCasting
	}

	if len(tx.TxOut) > 3 || len(tx.TxOut) < 2 || len(tx.TxIn) > 1 {
		return nil, fmt.Errorf("not CastingTx tx TxOut >3 or TxIn >1")
	}

	var es *CastingTxInfo
	txOut := tx.TxOut[0]
	info, err := CastingTxFromScript(txOut.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txOut.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		es = &CastingTxInfo{
			Amount:      info.Amount,
			ConvertType: info.ConvertType,
			PubKey:      info.PubKey,
		}
	}

	var pk []byte
	if tx.TxIn[0].Witness == nil {
		pk, err = txscript.ComputePk(tx.TxIn[0].SignatureScript)
		if err != nil {
			return nil, fmt.Errorf("ComputePk err %s", err)
		}
	} else {
		pk, err = txscript.ComputeWitnessPk(tx.TxIn[0].Witness)
		if err != nil {
			return nil, fmt.Errorf("ComputeWitnessPk err %s", err)
		}
	}

	es.PubKey = pk
	return es, nil
}

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

func AddBeaconPledgeTxFromScript(script []byte) (*AddBeaconPledge, error) {
	data, err := txscript.GetAddBeaconPledgeData(script)
	if err != nil {
		return nil, err
	}
	info := &AddBeaconPledge{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

//  Beacon
func MortgageFromScript(script []byte) (*Mortgage, error) {
	data, err := txscript.GetMortgageData(script)
	if err != nil {
		return nil, err
	}
	info := &Mortgage{}
	err = rlp.DecodeBytes(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func AddMortgageFromScript(script []byte) (*AddMortgage, error) {
	data, err := txscript.GetAddMortgageData(script)
	if err != nil {
		return nil, err
	}
	info := &AddMortgage{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

func UpdateBeaconCoinbaseAllTxFromScript(script []byte) (*UpdateCoinbaseAll, error) {
	data, err := txscript.GetUpdateCoinbaseAllData(script)
	if err != nil {
		return nil, err
	}
	info := &UpdateCoinbaseAll{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

func ConvertTxFromScript(script []byte) (*ConvertTxInfo, error) {
	data, err := txscript.GetConvertInfoData(script)
	if err != nil {
		return nil, err
	}
	info := &ConvertTxInfo{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

func CastingTxFromScript(script []byte) (*CastingTxInfo, error) {
	data, err := txscript.GetCastingInfoData(script)
	if err != nil {
		return nil, err
	}
	info := &CastingTxInfo{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

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

func MakeMergerCoinbaseTx(tx *wire.MsgTx, cState *CommitteeState, pool *PoolAddrItem, items []*ConvertTxInfo, rewards []*PunishedRewardItem, mergeItem map[uint64][]*BeaconMergeItem) error {

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

	// 计算池子的地址
	poolC := make(map[string]*big.Int)

	for _, v := range items {
		if v.ConvertType == ExpandedTxConvert_Czz {
			pkScript, _ := txscript.PayToPubKeyHashScript(v.PubKey)
			tx.AddTxOut(&wire.TxOut{
				Value:    v.Amount.Int64(),
				PkScript: pkScript,
			})
			continue
		}

		pool1 := CoinPools[v.AssetType]
		pool2 := CoinPools[v.ConvertType]
		poolC[hex.EncodeToString(pool1)] = big.NewInt(0)
		poolC[hex.EncodeToString(pool2)] = big.NewInt(0)
	}

	for _, v := range items {
		if v.ConvertType == ExpandedTxConvert_Czz {
			continue
		}
		pool1 := CoinPools[v.AssetType]
		pool2 := CoinPools[v.ConvertType]
		poolC[hex.EncodeToString(pool1)] = big.NewInt(0).Sub(poolC[hex.EncodeToString(pool1)], v.Amount)
		poolC[hex.EncodeToString(pool2)] = big.NewInt(0).Add(poolC[hex.EncodeToString(pool2)], v.Amount)
	}

	for k, v := range poolC {

	}

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

func updateTxOutValue(out *wire.TxOut, value int64) error {
	out.Value += value
	return nil
}

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

func ToAddressFromConverts(eState *CommitteeState, cinfo map[uint32]*ConvertTxInfo, ev *CommitteeVerify) ([]*ConvertTxInfo, error) {

	ctis := make([]*ConvertTxInfo, 0, 0)
	for _, info := range cinfo {
		// verify the entangle tx
		tpi, err := ev.VerifyConvertTx(info, eState)
		if err != nil {
			return nil, err
		}

		//pub, err := RecoverPublicFromBytes(tpi.Pub, tpi.AssetType)
		//if err != nil {
		//	return nil, err
		//}
		//
		//addr, err := MakeAddress(*pub, ev.Params)
		//if err != nil {
		//	return nil, err
		//}

		info.PubKey = tpi.Pub
		if err := eState.Convert(info); err != nil {
			return nil, err
		}
		ctis = append(ctis, info)
	}

	return ctis, nil
}

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
