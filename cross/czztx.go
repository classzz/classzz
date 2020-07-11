package cross

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/classzz/rlp"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

type ExpandedTxType uint8

const (
	// Entangle Transcation type
	ExpandedTxEntangle_Doge = 0xF0
	ExpandedTxEntangle_Ltc  = 0xF1
	ExpandedTxEntangle_Btc  = 0xF2
	ExpandedTxEntangle_Bsv  = 0xF3
	ExpandedTxEntangle_Bch  = 0xF4
)

func (et ExpandedTxType) ExpandedTxTypeToAssetType() uint32 {
	switch et {
	case ExpandedTxEntangle_Doge:
		return LhAssetDOGE
	case ExpandedTxEntangle_Ltc:
		return LhAssetLTC
	case ExpandedTxEntangle_Btc:
		return LhAssetBTC
	case ExpandedTxEntangle_Bch:
		return LhAssetBCH
	case ExpandedTxEntangle_Bsv:
		return LhAssetBSV
	}
	return 0
}

var (
	NoEntangle           = errors.New("no entangle info in transcation")
	NoExChange           = errors.New("no NoExChange info in transcation")
	NoBeaconRegistration = errors.New("no BeaconRegistration info in transcation")
	NoBurnTx             = errors.New("no BurnTx info in transcation")
	NoBurnProofTx        = errors.New("no BurnProofTx info in transcation")
	NoWhiteListProofTx   = errors.New("no WhiteListProofTx info in transcation")
	NoAddBeaconPledge    = errors.New("no AddBeaconPledge info in transcation")
	NoAddBeaconCoinbase  = errors.New("no AddBeaconCoinbase info in transcation")

	infoFixed = map[ExpandedTxType]uint32{
		ExpandedTxEntangle_Doge: 64,
		ExpandedTxEntangle_Ltc:  64,
		ExpandedTxEntangle_Btc:  64,
		ExpandedTxEntangle_Bsv:  64,
		ExpandedTxEntangle_Bch:  64,
	}
	baseUnit       = new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil)
	baseUnit1      = new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
	dogeUnit       = new(big.Int).Mul(big.NewInt(int64(12500000)), baseUnit)
	dogeUnit1      = new(big.Int).Mul(big.NewInt(int64(12500000)), baseUnit1)
	MinPunished    = new(big.Int).Mul(big.NewInt(int64(20)), baseUnit)
	ZeroAddrsss, _ = czzutil.NewLegacyAddressPubKeyHash([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, &chaincfg.MainNetParams)
)

type EntangleItem struct {
	EType ExpandedTxType
	Value *big.Int
	Addr  czzutil.Address
}

func (ii *EntangleItem) Clone() *EntangleItem {
	item := &EntangleItem{
		EType: ii.EType,
		Value: new(big.Int).Set(ii.Value),
		Addr:  ii.Addr,
	}
	return item
}

type ExChangeItem struct {
	EType ExpandedTxType
	Value *big.Int
	Addr  czzutil.Address
	BID   uint64
}

func (ii *ExChangeItem) Clone() *ExChangeItem {
	item := &ExChangeItem{
		EType: ii.EType,
		Value: new(big.Int).Set(ii.Value),
		Addr:  ii.Addr,
		BID:   ii.BID,
	}
	return item
}

// entangle tx Sequence infomation
type EtsInfo struct {
	FeePerKB int64
	Tx       *wire.MsgTx
}

type TuplePubIndex struct {
	EType ExpandedTxType
	Index uint32
	Pub   []byte
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

type ExChangeTxInfo struct {
	Address   string
	ExTxType  ExpandedTxType
	Index     uint32
	Height    uint64
	Amount    *big.Int
	ExtTxHash string
	BID       uint64
}

type EntangleTxInfo struct {
	ExTxType  ExpandedTxType
	Index     uint32
	Height    uint64
	Amount    *big.Int
	ExtTxHash []byte
}

func (info *EntangleTxInfo) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.WriteByte(byte(info.ExTxType))
	binary.Write(buf, binary.LittleEndian, info.Index)
	binary.Write(buf, binary.LittleEndian, info.Height)
	b1 := info.Amount.Bytes()
	len := uint8(len(b1))
	buf.WriteByte(byte(len))

	buf.Write(b1)
	buf.Write(info.ExtTxHash)
	return buf.Bytes()
}

func (info *EntangleTxInfo) Parse(data []byte) error {
	if len(data) <= 14 {
		return errors.New("wrong lenght!")
	}
	//data = data[4:]
	info.ExTxType = ExpandedTxType(data[0])
	switch info.ExTxType {
	case ExpandedTxEntangle_Doge, ExpandedTxEntangle_Ltc, ExpandedTxEntangle_Btc, ExpandedTxEntangle_Bsv, ExpandedTxEntangle_Bch:
		break
	default:
		return errors.New("Parse failed,not entangle tx")
	}
	buf := bytes.NewBuffer(data[1:])
	binary.Read(buf, binary.LittleEndian, &info.Index)
	binary.Read(buf, binary.LittleEndian, &info.Height)
	l, _ := buf.ReadByte()
	b0 := make([]byte, int(uint32(l)))
	n, _ := buf.Read(b0)
	if int(uint32(l)) != n {
		return errors.New("b0 not equal n")
	}
	amount := big.NewInt(0)
	amount.SetBytes(b0)
	info.Amount = amount
	info.ExtTxHash = make([]byte, int(infoFixed[info.ExTxType]))
	n2, _ := buf.Read(info.ExtTxHash)

	if len(info.ExtTxHash) != n2 {
		return errors.New("len(info.ExtTxHash) not equal n2")
	}

	// if len(info.ExtTxHash) != int(infoFixed[info.ExTxType]) {
	// 	e := fmt.Sprintf("lenght not match,[request:%v,exist:%v]", infoFixed[info.ExTxType], len(info.ExtTxHash))
	// 	return errors.New(e)
	// }
	return nil
}

type BurnTxInfo struct {
	ExTxType ExpandedTxType
	Address  string
	LightID  uint64
	Amount   *big.Int
}

//func (es *BurnTxInfo) DecodeRLP(s *rlp.Stream) error {
//	type Store1 struct {
//		LightID uint64
//	}
//	var eb Store1
//	if err := s.Decode(&eb); err != nil {
//		return err
//	}
//	es.LightID = eb.LightID
//	return nil
//}
//func (es *BurnTxInfo) EncodeRLP(w io.Writer) error {
//	type Store1 struct {
//		LightID uint64
//	}
//	return rlp.Encode(w, &Store1{
//		LightID: es.LightID,
//	})
//}

type KeepedItem struct {
	ExTxType ExpandedTxType
	Amount   *big.Int
}
type KeepedAmount struct {
	Count byte
	Items []KeepedItem
}

func (info *KeepedAmount) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.WriteByte(info.Count)
	for _, v := range info.Items {
		buf.WriteByte(byte(v.ExTxType))
		b1 := v.Amount.Bytes()
		len := uint8(len(b1))
		buf.WriteByte(byte(len))
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
		l, _ := buf.ReadByte()
		b0 := make([]byte, int(uint32(l)))
		_, _ = buf.Read(b0)
		item := KeepedItem{
			ExTxType: ExpandedTxType(itype),
			Amount:   new(big.Int).SetBytes(b0),
		}
		info.Items = append(info.Items, item)
	}
	return nil
}
func (info *KeepedAmount) Add(item KeepedItem) {
	for _, v := range info.Items {
		if v.ExTxType == item.ExTxType {
			v.Amount.Add(v.Amount, item.Amount)
			return
		}
	}
	info.Count++
	info.Items = append(info.Items, item)
}
func (info *KeepedAmount) GetValue(t ExpandedTxType) *big.Int {
	for _, v := range info.Items {
		if v.ExTxType == t {
			return v.Amount
		}
	}
	return nil
}

func MakeEntangleTx(params *chaincfg.Params, inputs []*wire.TxIn, feeRate, inAmount czzutil.Amount,
	changeAddr czzutil.Address, info *EntangleTxInfo) (*wire.MsgTx, error) {
	// make pay script info include txHash and height
	scriptInfo, err := txscript.EntangleScript(info.Serialize())
	if err != nil {
		return nil, err
	}
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(&wire.TxOut{
		Value:    0,
		PkScript: scriptInfo,
	})
	var outputAmt czzutil.Amount = 0
	const (
		// spendSize is the largest number of bytes of a sigScript
		// which spends a p2pkh output: OP_DATA_73 <sig> OP_DATA_33 <pubkey>
		spendSize = 1 + 73 + 1 + 33
	)

	var (
		amtSelected czzutil.Amount
		txSize      int
	)
	for _, in := range inputs {
		tx.AddTxIn(in)
		txSize = tx.SerializeSize() + spendSize*len(tx.TxIn)
	}
	reqFee := czzutil.Amount(txSize * int(feeRate))
	changeVal := amtSelected - outputAmt - reqFee

	if changeVal > 0 {
		pkScript, err := txscript.PayToAddrScript(changeAddr)
		if err != nil {
			return nil, err
		}
		changeOutput := &wire.TxOut{
			Value:    int64(changeVal),
			PkScript: pkScript,
		}
		tx.AddTxOut(changeOutput)
	}

	return tx, nil
}

func SignEntangleTx(tx *wire.MsgTx, inputAmount []czzutil.Amount,
	priv *czzec.PrivateKey) error {

	for i, txIn := range tx.TxIn {
		sigScript, err := txscript.SignatureScript(tx, i,
			int64(inputAmount[i].ToUnit(czzutil.AmountSatoshi)), nil,
			txscript.SigHashAll, priv, true)
		if err != nil {
			return err
		}
		txIn.SignatureScript = sigScript
	}

	return nil
}

func IsEntangleTx(tx *wire.MsgTx) (map[uint32]*EntangleTxInfo, error) {
	// make sure at least one txout in OUTPUT
	einfos := make(map[uint32]*EntangleTxInfo)
	for i, v := range tx.TxOut {
		info, err := EntangleTxFromScript(v.PkScript)
		if err == nil {
			if v.Value != 0 {
				return nil, errors.New("the output value must be 0 in entangle tx.")
			}
			einfos[uint32(i)] = info
		}
	}
	if len(einfos) > 0 {
		return einfos, nil
	}
	return nil, NoEntangle
}

// Only txOut[0] is valid
func IsExChangeTx(tx *wire.MsgTx) (map[uint32]*ExChangeTxInfo, error) {

	// make sure at least one txout in OUTPUT
	einfos := make(map[uint32]*ExChangeTxInfo)
	for i, v := range tx.TxOut {
		info, err := ExChangeTxFromScript(v.PkScript)
		if err == nil && i == 0 {
			if v.Value != 0 {
				return nil, errors.New("the output value must be 0 in ExChange tx.")
			}
			einfos[uint32(i)] = info
		}
	}
	if len(einfos) > 0 {
		return einfos, nil
	}
	return nil, NoExChange
}

// BeaconRegistration
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
	info.PubKey = pk

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

func IsAddBeaconCoinbaseTx(tx *wire.MsgTx, params *chaincfg.Params) (*AddBeaconCoinbase, error) {
	// make sure at least one txout in OUTPUT
	if len(tx.TxOut) > 0 {
		txout := tx.TxOut[0]
		if !txscript.IsAddBeaconCoinbaseTy(txout.PkScript) {
			return nil, NoAddBeaconCoinbase
		}
	} else {
		return nil, NoAddBeaconCoinbase
	}

	if len(tx.TxOut) > 3 || len(tx.TxOut) < 2 || len(tx.TxIn) > 1 {
		e := fmt.Sprintf("not BeaconRegistration tx TxOut >3 or TxIn >1")
		return nil, errors.New(e)
	}

	// make sure at least one txout in OUTPUT
	var bp *AddBeaconCoinbase

	txout := tx.TxOut[0]
	info, err := AddBeaconCoinbaseTxFromScript(txout.PkScript)
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

	info.Address = address.String()

	if bp != nil {
		return bp, nil
	}
	return nil, NoAddBeaconCoinbase
}

func IsBurnTx(tx *wire.MsgTx, params *chaincfg.Params) (*BurnTxInfo, error) {
	// make sure at least one txout in OUTPUT
	var es *BurnTxInfo

	var pk []byte
	var err error
	// get to address
	if len(tx.TxOut) < 2 {
		return nil, errors.New("BurnTx Must be at least two TxOut")
	}
	if r, err := IsSendToZeroAddress(tx.TxOut[1].PkScript); err != nil {
		return nil, err
	} else {
		if !r {
			return nil, errors.New("not send to burn address")
		}
	}
	// get from address
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

	txout := tx.TxOut[0]
	info, err := BurnInfoFromScript(txout.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txout.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		es = info
	}

	info.Amount = big.NewInt(tx.TxOut[1].Value)
	info.Address = address.String()

	if es != nil {
		return es, nil
	}
	return nil, NoBurnTx
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

func IsBurnProofTx(tx *wire.MsgTx) (*BurnProofInfo, error) {
	// make sure at least one txout in OUTPUT
	var es *BurnProofInfo
	var err error

	if len(tx.TxOut) < 1 {
		return nil, errors.New("BurnProofInfo Must be at least two TxOut")
	}

	txout := tx.TxOut[0]
	info, err := BurnProofInfoFromScript(txout.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txout.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		es = info
	}

	if es != nil {
		return es, nil
	}
	return nil, NoBurnProofTx
}

func IsWhiteListProofTx(tx *wire.MsgTx) (*WhiteListProof, error) {
	// make sure at least one txout in OUTPUT
	var es *WhiteListProof
	var err error
	if len(tx.TxOut) < 2 {
		return nil, errors.New("WhiteListProof Must be at least two TxOut")
	}
	txout := tx.TxOut[0]
	info, err := WhiteListProofFromScript(txout.PkScript)
	if err != nil {
		return nil, errors.New("the output tx.")
	} else {
		if txout.Value != 0 {
			return nil, errors.New("the output value must be 0 in tx.")
		}
		es = info
	}
	if es != nil {
		return es, nil
	}
	return nil, NoWhiteListProofTx
}

func EntangleTxFromScript(script []byte) (*EntangleTxInfo, error) {
	data, err := txscript.GetEntangleInfoData(script)
	if err != nil {
		return nil, err
	}
	info := &EntangleTxInfo{}
	err = info.Parse(data)
	return info, err
}

func ExChangeTxFromScript(script []byte) (*ExChangeTxInfo, error) {
	data, err := txscript.GetExChangeInfoData(script)
	if err != nil {
		return nil, err
	}
	info := &ExChangeTxInfo{}
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
	return info, err
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

func AddBeaconCoinbaseTxFromScript(script []byte) (*AddBeaconCoinbase, error) {
	data, err := txscript.GetAddBeaconCoinbaseData(script)
	if err != nil {
		return nil, err
	}
	info := &AddBeaconCoinbase{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

func BurnInfoFromScript(script []byte) (*BurnTxInfo, error) {
	data, err := txscript.GetBurnInfoData(script)
	if err != nil {
		return nil, err
	}
	info := &BurnTxInfo{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}
func BurnProofInfoFromScript(script []byte) (*BurnProofInfo, error) {
	data, err := txscript.GetBurnProofInfoData(script)
	if err != nil {
		return nil, err
	}
	info := &BurnProofInfo{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}
func WhiteListProofFromScript(script []byte) (*WhiteListProof, error) {
	data, err := txscript.GetWhiteListProofData(script)
	if err != nil {
		return nil, err
	}
	info := &WhiteListProof{}
	err = rlp.DecodeBytes(data, info)
	return info, err
}

/////////////////////////////////////////////////////////////////////////////

func GetMaxHeight(items map[uint32]*EntangleTxInfo) uint64 {
	h := uint64(0)
	for _, v := range items {
		if h < v.Height {
			h = v.Height
		}
	}
	return h
}

func VerifyTxsSequence(infos []*EtsInfo) error {
	if infos == nil {
		return nil
	}
	pre, pos := uint64(0), 0
	for i, v := range infos {
		einfos, _ := IsEntangleTx(v.Tx)
		if einfos != nil {
			h := GetMaxHeight(einfos)
			if pre > h && infos[pos].FeePerKB <= infos[i].FeePerKB {
				return errors.New(fmt.Sprintf("tx sequence wrong,[i=%d,h=%v,f=%v][i=%d,h=%v,f=%v]",
					pos, pre, infos[pos].FeePerKB, i, h, infos[i].FeePerKB))
			} else {
				pre, pos = h, i
			}
		}
	}

	return nil
}

func MakeMergeCoinbaseTx(tx *wire.MsgTx, pool *PoolAddrItem, items []*ExChangeItem, rewards []*PunishedRewardItem,
	mergeItem map[uint64][]*BeaconMergeItem) error {

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

	reserve1, reserve2 := pool.Amount[0].Int64()+tx.TxOut[1].Value, pool.Amount[1].Int64()
	updateTxOutValue(tx.TxOut[2], reserve2)
	allEntangle := int64(0)

	for i := range items {
		pkScript, err := txscript.PayToAddrScript(items[i].Addr)
		if err != nil {
			return errors.New("Make Meger tx failed,err: " + err.Error())
		}
		out := &wire.TxOut{
			Value:    items[i].Value.Int64(),
			PkScript: pkScript,
		}
		allEntangle += out.Value
		tx.AddTxOut(out)
	}
	tx.TxOut[1].Value = reserve1 - allEntangle
	if tx.TxOut[1].Value < 0 {
		panic(errors.New("pool1 amount < 0"))
	}
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

func MakeMergerCoinbaseTx2(height *big.Int, tx *wire.MsgTx, items []*ExChangeItem, state *EntangleState) error {
	if len(items) == 0 || state == nil {
		return nil
	}
	for i := range items {
		amount, err := state.AddEntangleItem(items[i].Addr.EncodeAddress(), uint8(items[i].EType),
			items[i].BID, height, items[i].Value)
		if err != nil {
			return errors.New(fmt.Sprintf("MakeMergerCoinbaseTx2 failed,i=%d,bid=%v,type=%d,amount=%v,err=%s",
				i, items[i].BID, items[i].EType, items[i].Value.String(), err.Error()))
		}
		pkScript, err1 := txscript.PayToAddrScript(items[i].Addr)
		if err1 != nil {
			return errors.New("Make Meger tx failed,err: " + err1.Error())
		}
		out := &wire.TxOut{
			Value:    amount.Int64(),
			PkScript: pkScript,
		}
		tx.AddTxOut(out)
	}
	return nil
}
func updateTxOutValue(out *wire.TxOut, value int64) error {
	out.Value += value
	return nil
}

func calcExchange(item *ExChangeItem, reserve *int64, keepInfo *KeepedAmount, change, fork bool) {
	amount := big.NewInt(0)
	cur := keepInfo.GetValue(item.EType)
	if cur != nil {
		amount = new(big.Int).Set(cur)
	}
	if change {
		kk := KeepedItem{
			ExTxType: item.EType,
			Amount:   new(big.Int).Set(item.Value),
		}
		keepInfo.Add(kk)
	}
	if item.EType == ExpandedTxEntangle_Doge {
		if fork {
			item.Value = toDoge2(amount, item.Value)
		} else {
			item.Value = toDoge(amount, item.Value, fork)
		}
	} else if item.EType == ExpandedTxEntangle_Ltc {
		if fork {
			item.Value = toLtc2(amount, item.Value)
		} else {
			item.Value = toLtc(amount, item.Value, fork)
		}
	} else if item.EType == ExpandedTxEntangle_Btc {
		item.Value = toBtc(amount, item.Value)
	} else if item.EType == ExpandedTxEntangle_Bch {
		item.Value = toBchOrBsv(amount, item.Value)
	} else if item.EType == ExpandedTxEntangle_Bsv {
		item.Value = toBchOrBsv(amount, item.Value)
	}
	*reserve = *reserve - item.Value.Int64()
}

func PreCalcEntangleAmount(item *ExChangeItem, keepInfo *KeepedAmount, fork bool) {
	var vv int64
	calcExchange(item, &vv, keepInfo, true, fork)
}

func EnoughAmount(reserve int64, items []*ExChangeItem, keepInfo *KeepedAmount, fork bool) bool {
	amount := reserve
	for _, v := range items {
		calcExchange(v.Clone(), &amount, keepInfo, false, fork)
	}
	return amount > 0
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
func ToAddressFromExChange(tx *czzutil.Tx, ev *ExChangeVerify, eState *EntangleState) ([]*TmpAddressPair, error) {
	// txhash := tx.Hash()
	einfo, _ := IsExChangeTx(tx.MsgTx())
	if einfo != nil {
		// verify the entangle tx

		pairs := make([]*TmpAddressPair, 0)
		tt, err := ev.VerifyExChangeTx(tx.MsgTx(), eState)
		if err != nil {
			return nil, err
		}
		for _, v := range tt {
			pub, err1 := RecoverPublicFromBytes(v.Pub, v.EType)
			if err1 != nil {
				return nil, err1
			}
			err2, addr := MakeAddress(*pub)
			if err2 != nil {
				return nil, err2
			}
			pairs = append(pairs, &TmpAddressPair{
				index:   v.Index,
				Address: addr,
			})
		}

		return pairs, nil
	}

	return nil, nil
}
func OverEntangleAmount(tx *wire.MsgTx, pool *PoolAddrItem, items []*ExChangeItem,
	lastScriptInfo []byte, fork bool, state *EntangleState) bool {
	if items == nil || len(items) == 0 {
		return false
	}

	var keepInfo *KeepedAmount
	var err error
	if fork {
		types := []uint8{}
		for _, v := range items {
			types = append(types, uint8(v.EType))
		}
		keepInfo = getKeepInfosFromState(state, types)
	} else {
		keepInfo, err = KeepedAmountFromScript(lastScriptInfo)
	}
	if err != nil || keepInfo == nil {
		return false
	}
	all := pool.Amount[0].Int64() + tx.TxOut[1].Value
	return !EnoughAmount(all, items, keepInfo, fork)
}

func getKeepInfosFromState(state *EntangleState, types []uint8) *KeepedAmount {
	if state == nil {
		return nil
	}
	keepinfo := &KeepedAmount{Items: []KeepedItem{}}
	for _, v := range types {
		keepinfo.Add(KeepedItem{
			ExTxType: ExpandedTxType(v),
			Amount:   state.getAllEntangleAmount(v),
		})
	}
	return keepinfo
}

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

func FinishWhiteListProof(info *WhiteListProof, state *EntangleState) error {
	return state.FinishWhiteListProof(info)
}
func CloseProofForPunished(info *BurnProofInfo, item *BurnItem, state *EntangleState) error {
	return state.CloseProofForPunished(info, item)
}

//////////////////////////////////////////////////////////////////////////////
func ScanTxForBeaconOnSpecHeight(beacon map[uint64][]byte) {

}

// just fetch the outpoint info for beacon address's regsiter and append tx
func fetchOutPointFromTxs(txs []*czzutil.Tx, beacon map[uint64][]byte, state *EntangleState,
	params *chaincfg.Params) map[uint64][]*wire.OutPoint {
	res := make(map[uint64][]*wire.OutPoint)
	for _, v := range txs {
		_, e1 := IsAddBeaconPledgeTx(v.MsgTx(), params)
		_, e2 := IsBeaconRegistrationTx(v.MsgTx(), params)
		if e1 != nil || e2 != nil {
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(v.MsgTx().TxOut[1].PkScript, params)
			if err != nil {
				to := addrs[0].ScriptAddress()
				id := state.GetBeaconIdByTo(to)
				if id != 0 {
					toAddress := big.NewInt(0).SetBytes(to).Uint64()
					if toAddress >= 10 && toAddress <= 99 {
						res[id] = append(res[id], wire.NewOutPoint(v.Hash(), 1))
					}
				}
			}
		}
	}
	return res
}
func SameHeightTxForBurn(tx *czzutil.Tx, txs []*czzutil.Tx, params *chaincfg.Params) bool {
	info, e := IsBurnTx(tx.MsgTx(), params)
	if e != nil || info == nil {
		return false
	}
	for _, v := range txs {
		if info1, err := IsBurnTx(v.MsgTx(), params); err == nil {
			if info1 != nil && info1.Address == info.Address {
				return true
			}
		}
	}
	return false
}
func GetAddressFromProofTx(tx *czzutil.Tx, params *chaincfg.Params) czzutil.Address {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(tx.MsgTx().TxOut[1].PkScript, params)
	if err == nil {
		return nil
	}
	return addrs[0]
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
func reverseToBtc(keeped *big.Int) (*big.Int, *big.Int) {
	unit, base := big.NewInt(int64(10)), big.NewInt(int64(200))
	divisor, _ := new(big.Int).DivMod(keeped, baseUnit, new(big.Int).Set(baseUnit))
	rate := new(big.Int).Add(base, new(big.Int).Mul(unit, divisor))
	return rate, big.NewInt(1)
}
func toBtc(entangled, needed *big.Int) *big.Int {
	if needed == nil || needed.Int64() <= 0 {
		return big.NewInt(0)
	}
	keep, change := new(big.Int).Set(entangled), new(big.Int).Set(needed)
	unit, base := big.NewInt(int64(10)), big.NewInt(int64(200))
	res := big.NewInt(0)
	for {
		if change.Sign() <= 0 {
			break
		}
		divisor, remainder := new(big.Int).DivMod(keep, baseUnit, new(big.Int).Set(baseUnit))
		rate := new(big.Int).Add(base, new(big.Int).Mul(unit, divisor))
		l := new(big.Int).Sub(baseUnit, remainder)
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
func reverseToBchOrBsv(keeped *big.Int) (*big.Int, *big.Int) {
	unit, base := big.NewInt(int64(1000)), big.NewInt(int64(10000))
	loopUnit := new(big.Int).Mul(big.NewInt(300), baseUnit)
	divisor, _ := new(big.Int).DivMod(keeped, loopUnit, new(big.Int).Set(loopUnit))
	rate := new(big.Int).Add(base, new(big.Int).Mul(unit, divisor))
	return rate, big.NewInt(1)
}
func toBchOrBsv(entangled, needed *big.Int) *big.Int {
	if needed == nil || needed.Int64() <= 0 {
		return big.NewInt(0)
	}
	keep, change := new(big.Int).Set(entangled), new(big.Int).Set(needed)
	unit, base := big.NewInt(int64(1000)), big.NewInt(int64(10000))
	loopUnit := new(big.Int).Mul(big.NewInt(300), baseUnit)
	res := big.NewInt(0)
	for {
		if change.Sign() <= 0 {
			break
		}
		divisor, remainder := new(big.Int).DivMod(keep, loopUnit, new(big.Int).Set(loopUnit))
		rate := new(big.Int).Add(base, new(big.Int).Mul(unit, divisor))
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
