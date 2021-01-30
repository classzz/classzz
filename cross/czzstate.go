package cross

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/classzz/classzz/wire"
	"io"
	"log"
	"math/big"
	"sort"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/rlp"
	"github.com/classzz/czzutil"
)

type CommitteeInfo struct {
	Id          *big.Int
	StartHeight *big.Int
	EndHeight   *big.Int
	Members     []*CommitteeMember
	BackMembers []*CommitteeMember
}

type extCommitteeInfo struct {
	Id          *big.Int
	StartHeight *big.Int
	EndHeight   *big.Int
	Members     []*CommitteeMember
	BackMembers []*CommitteeMember
}

func (ci *CommitteeInfo) DecodeRLP(s *rlp.Stream) error {
	var eci extCommitteeInfo
	if err := s.Decode(&eci); err != nil {
		return err
	}
	ci.Id, ci.StartHeight, ci.EndHeight, ci.Members, ci.BackMembers = eci.Id, eci.StartHeight, eci.EndHeight, eci.Members, eci.BackMembers
	return nil
}

func (ci *CommitteeInfo) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extCommitteeInfo{
		Id:          ci.Id,
		StartHeight: ci.StartHeight,
		EndHeight:   ci.EndHeight,
		Members:     ci.Members,
		BackMembers: ci.BackMembers,
	})
}

type CommitteeMember struct {
	Coinbase      string
	CommitteeBase string
	Publickey     []byte
	Flag          uint32
	MType         uint32
}

type extCommitteeMember struct {
	Coinbase      string
	CommitteeBase string
	Publickey     []byte
	Flag          uint32
	MType         uint32
}

func (cm *CommitteeMember) DecodeRLP(s *rlp.Stream) error {
	var eci extCommitteeMember
	if err := s.Decode(&eci); err != nil {
		return err
	}
	cm.Coinbase, cm.CommitteeBase, cm.Publickey, cm.Flag, cm.MType = eci.Coinbase, eci.CommitteeBase, eci.Publickey, eci.Flag, eci.MType
	return nil
}

func (cm *CommitteeMember) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extCommitteeMember{
		Coinbase:      cm.Coinbase,
		CommitteeBase: cm.CommitteeBase,
		Publickey:     cm.Publickey,
		Flag:          cm.Flag,
		MType:         cm.MType,
	})
}

type PledgeInfo struct {
	ID              *big.Int `json:"id"`
	Address         string   `json:"address"`
	PubKey          []byte   `json:"pub_key"`
	ToAddress       []byte   `json:"toAddress"`
	StakingAmount   *big.Int `json:"staking_amount"`
	CoinBaseAddress []string `json:"coinbase_address"`
}

type extPledgeInfo struct {
	ID              *big.Int `json:"id"`
	Address         string   `json:"address"`
	PubKey          []byte   `json:"pub_key"`
	ToAddress       []byte   `json:"toAddress"`
	StakingAmount   *big.Int `json:"staking_amount"`
	CoinBaseAddress []string `json:"coinbase_address"`
}

func (pi *PledgeInfo) DecodeRLP(s *rlp.Stream) error {
	var epi extPledgeInfo
	if err := s.Decode(&epi); err != nil {
		return err
	}
	pi.ID, pi.Address, pi.PubKey, pi.ToAddress, pi.StakingAmount, pi.CoinBaseAddress = epi.ID, epi.Address, epi.PubKey, epi.ToAddress, epi.StakingAmount, epi.CoinBaseAddress
	return nil
}

func (pi *PledgeInfo) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extPledgeInfo{
		ID:              pi.ID,
		Address:         pi.Address,
		PubKey:          pi.PubKey,
		ToAddress:       pi.ToAddress,
		StakingAmount:   pi.StakingAmount,
		CoinBaseAddress: pi.CoinBaseAddress,
	})
}

type ConvertItem struct {
	ID        *big.Int `json:"id"`
	ExtTxHash string   `json:"ext_tx_hash"`
	PubKey    []byte   `json:"pub_key"`
	Amount    *big.Int `json:"amount"` // czz asset amount
}

type ConvertItems map[uint8][]*ConvertItem

type StoreConvertItem struct {
	Type        uint8
	ConvertItem []*ConvertItem
}

type SortStoreConvertItem []*StoreConvertItem

func (vs SortStoreConvertItem) Len() int {
	return len(vs)
}

func (vs SortStoreConvertItem) Less(i, j int) bool {
	return vs[i].Type < vs[j].Type
}

func (vs SortStoreConvertItem) Swap(i, j int) {
	it := vs[i]
	vs[i] = vs[j]
	vs[j] = it
}

func (ci *ConvertItems) toSlice() SortStoreConvertItem {
	v1 := make([]*StoreConvertItem, 0, 0)
	for k, v := range *ci {
		v1 = append(v1, &StoreConvertItem{
			Type:        k,
			ConvertItem: v,
		})
	}
	sort.Sort(SortStoreConvertItem(v1))
	return SortStoreConvertItem(v1)
}

func (ci *ConvertItems) fromSlice(vv SortStoreConvertItem) {
	convertItems := make(map[uint8][]*ConvertItem)
	for _, v := range vv {
		convertItems[v.Type] = v.ConvertItem
	}
	*ci = ConvertItems(convertItems)
}

func (ci *ConvertItems) DecodeRLP(s *rlp.Stream) error {
	type Store1 struct {
		Value SortStoreConvertItem
	}
	var eb Store1
	if err := s.Decode(&eb); err != nil {
		return err
	}
	ci.fromSlice(eb.Value)
	return nil
}

func (ci *ConvertItems) EncodeRLP(w io.Writer) error {
	type Store1 struct {
		Value SortStoreConvertItem
	}
	s1 := ci.toSlice()
	return rlp.Encode(w, &Store1{
		Value: s1,
	})
}

type CommitteeState struct {
	PledgeInfos         []*PledgeInfo
	CommitteeInfos      []*CommitteeInfo
	MaxItemID           *big.Int
	ConvertItems        map[uint8]ConvertItems
	ConvertConfirmItems map[uint8]ConvertItems
	NoCostUtxos         map[string]*PoolAddrItem
}

type StoreConvertItems struct {
	Type         uint8
	ConvertItems ConvertItems
}

type SortStoreConvertItems []*StoreConvertItems

func (ci SortStoreConvertItems) Len() int {
	return len(ci)
}

func (ci SortStoreConvertItems) Less(i, j int) bool {
	return ci[i].Type < ci[j].Type
}

func (ci SortStoreConvertItems) Swap(i, j int) {
	it := ci[i]
	ci[i] = ci[j]
	ci[j] = it
}

type StoreConvertConfirmItems struct {
	Type                uint8
	ConvertConfirmItems ConvertItems
}

type SortStoreConvertConfirmItems []*StoreConvertConfirmItems

func (ci SortStoreConvertConfirmItems) Len() int {
	return len(ci)
}

func (ci SortStoreConvertConfirmItems) Less(i, j int) bool {
	return ci[i].Type < ci[j].Type
}

func (ci SortStoreConvertConfirmItems) Swap(i, j int) {
	it := ci[i]
	ci[i] = ci[j]
	ci[j] = it
}

type StoreNoCostUtxos struct {
	Type        string
	NoCostUtxos *PoolAddrItem
}

type SortStoreNoCostUtxos []*StoreNoCostUtxos

func (ci SortStoreNoCostUtxos) Len() int {
	return len(ci)
}

func (ci SortStoreNoCostUtxos) Less(i, j int) bool {
	return ci[i].Type < ci[j].Type
}

func (ci SortStoreNoCostUtxos) Swap(i, j int) {
	it := ci[i]
	ci[i] = ci[j]
	ci[j] = it
}

func (cs *CommitteeState) toSlice() (SortStoreConvertItems, SortStoreConvertConfirmItems, SortStoreNoCostUtxos) {
	v1 := make([]*StoreConvertItems, 0, 0)
	for k, v := range cs.ConvertItems {
		v1 = append(v1, &StoreConvertItems{
			Type:         k,
			ConvertItems: v,
		})
	}

	v2 := make([]*StoreConvertConfirmItems, 0, 0)
	for k, v := range cs.ConvertConfirmItems {
		v2 = append(v2, &StoreConvertConfirmItems{
			Type:                k,
			ConvertConfirmItems: v,
		})
	}

	v3 := make([]*StoreNoCostUtxos, 0, 0)
	for k, v := range cs.NoCostUtxos {
		v3 = append(v3, &StoreNoCostUtxos{
			Type:        k,
			NoCostUtxos: v,
		})
	}

	sort.Sort(SortStoreConvertItems(v1))
	sort.Sort(SortStoreConvertConfirmItems(v2))
	sort.Sort(SortStoreNoCostUtxos(v3))
	return SortStoreConvertItems(v1), SortStoreConvertConfirmItems(v2), SortStoreNoCostUtxos(v3)
}

func (cs *CommitteeState) fromSlice(v1 SortStoreConvertItems, v2 SortStoreConvertConfirmItems, v3 SortStoreNoCostUtxos) {
	convertItems := make(map[uint8]ConvertItems)
	convertConfirmItems := make(map[uint8]ConvertItems)
	noCostUtxos := make(map[string]*PoolAddrItem)
	for _, v := range v1 {
		convertItems[v.Type] = v.ConvertItems
	}
	for _, v := range v2 {
		convertConfirmItems[v.Type] = v.ConvertConfirmItems
	}
	for _, v := range v3 {
		noCostUtxos[v.Type] = v.NoCostUtxos
	}
	cs.ConvertItems = convertItems
	cs.ConvertConfirmItems = convertConfirmItems
	cs.NoCostUtxos = noCostUtxos
}

type extCommitteeState struct {
	PledgeInfos         []*PledgeInfo
	CommitteeInfos      []*CommitteeInfo
	MaxItemID           *big.Int
	ConvertItems        SortStoreConvertItems
	ConvertConfirmItems SortStoreConvertConfirmItems
	NoCostUtxos         SortStoreNoCostUtxos
}

func (cs *CommitteeState) DecodeRLP(s *rlp.Stream) error {
	var ecs extCommitteeState
	if err := s.Decode(&ecs); err != nil {
		return err
	}
	cs.PledgeInfos, cs.CommitteeInfos, cs.MaxItemID = ecs.PledgeInfos, ecs.CommitteeInfos, ecs.MaxItemID
	cs.fromSlice(ecs.ConvertItems, ecs.ConvertConfirmItems, ecs.NoCostUtxos)
	return nil
}

func (cs *CommitteeState) EncodeRLP(w io.Writer) error {
	s1, s2, s3 := cs.toSlice()
	return rlp.Encode(w, extCommitteeState{
		PledgeInfos:         cs.PledgeInfos,
		CommitteeInfos:      cs.CommitteeInfos,
		MaxItemID:           cs.MaxItemID,
		ConvertItems:        s1,
		ConvertConfirmItems: s2,
		NoCostUtxos:         s3,
	})
}

/////////////////////////////////////////////////////////////////
func (cs *CommitteeState) GetPledgeInfoByID(id *big.Int) *PledgeInfo {
	for _, v := range cs.PledgeInfos {
		if v.ID.Cmp(id) == 0 {
			return v
		}
	}
	return nil
}

func (cs *CommitteeState) GetPledgeInfoByAddress(address string) *PledgeInfo {
	for _, v := range cs.PledgeInfos {
		if v.Address == address {
			return v
		}
	}
	return nil
}

func (cs *CommitteeState) GetPledgeInfoByAddressTo(to []byte) *PledgeInfo {
	for _, v := range cs.PledgeInfos {
		if bytes.Equal(v.ToAddress, to) {
			return v
		}
	}
	return nil
}

func (cs *CommitteeState) GetPledgeInfoToAddrByID(id *big.Int, params *chaincfg.Params) string {
	if b := cs.GetPledgeInfoByID(id); b != nil {
		addr, err := czzutil.NewLegacyAddressPubKeyHash(b.ToAddress, params)
		if err == nil {
			return addr.String()
		}
	}
	return ""
}

func (cs *CommitteeState) GetPledgeInfoAddressByID(id *big.Int) string {
	pi := cs.GetPledgeInfoByID(id)
	if pi == nil {
		return ""
	}
	return pi.Address
}

func (cs *CommitteeState) GetPledgeInfoMaxId() *big.Int {
	maxId := big.NewInt(0)
	for _, v := range cs.PledgeInfos {
		if maxId.Cmp(v.ID) < 0 {
			maxId = v.ID
		}
	}
	return maxId
}

/////////////////////////////////////////////////////////////////
// keep staking enough amount asset
func (cs *CommitteeState) Mortgage(address string, to []byte, pubKey []byte, amount *big.Int, cba []string) error {

	if amount.Cmp(MinStakingAmount) < 0 {
		return ErrLessThanMin
	}

	if info := cs.GetPledgeInfoByAddress(address); info != nil {
		return ErrRepeatRegister
	}

	if info := cs.GetPledgeInfoByAddressTo(to); info != nil {
		return ErrRepeatToAddress
	}

	maxId := cs.GetPledgeInfoMaxId()
	info := &PledgeInfo{
		ID:              big.NewInt(0).Add(maxId, big.NewInt(1)),
		Address:         address,
		PubKey:          pubKey,
		ToAddress:       to,
		StakingAmount:   new(big.Int).Set(amount),
		CoinBaseAddress: cba,
	}

	cs.PledgeInfos = append(cs.PledgeInfos, info)
	return nil
}

func (cs *CommitteeState) AddMortgage(address string, amount *big.Int) error {
	if info := cs.GetPledgeInfoByAddress(address); info == nil {
		return ErrRepeatRegister
	} else {
		info.StakingAmount = new(big.Int).Add(info.StakingAmount, amount)
	}
	return nil
}

func (cs *CommitteeState) UpdateCoinbaseAll(address string, coinBases []string) error {
	if info := cs.GetPledgeInfoByAddress(address); info == nil {
		return ErrNoRegister
	} else {
		if len(coinBases) >= MaxCoinBase {
			return errors.New("more than max coinbase")
		}
		info.CoinBaseAddress = coinBases
	}
	return nil
}

// AddEntangleItem add item in the state, keep BeaconAddress have enough amount to entangle,
func (cs *CommitteeState) Convert(info *ConvertTxInfo) error {

	convertItem := &ConvertItem{
		ID:        big.NewInt(0).Add(cs.MaxItemID, big.NewInt(1)),
		ExtTxHash: info.ExtTxHash,
		PubKey:    info.PubKey,
		Amount:    info.Amount,
	}
	cs.MaxItemID = convertItem.ID

	if info.ConvertType == ExpandedTxConvert_Czz {
		if _, ok := cs.ConvertConfirmItems[info.AssetType]; !ok {
			item := ConvertItems{}
			items := make([]*ConvertItem, 0, 0)
			items = append(items, convertItem)
			item[info.ConvertType] = items
			cs.ConvertConfirmItems[info.AssetType] = item
		} else {
			items := cs.ConvertConfirmItems[info.AssetType][info.ConvertType]
			items = append(items, convertItem)
			cs.ConvertConfirmItems[info.AssetType][info.ConvertType] = items
		}
	} else {
		if _, ok := cs.ConvertItems[info.AssetType]; !ok {
			item := ConvertItems{}
			items := make([]*ConvertItem, 0, 0)
			items = append(items, convertItem)
			item[info.ConvertType] = items
			cs.ConvertItems[info.AssetType] = item
		} else {
			items := cs.ConvertItems[info.AssetType][info.ConvertType]
			items = append(items, convertItem)
			cs.ConvertItems[info.AssetType][info.ConvertType] = items
		}
	}

	return nil
}

// AddEntangleItem add item in the state, keep BeaconAddress have enough amount to entangle,
func (cs *CommitteeState) Casting(info *CastingTxInfo) error {

	convertItem := &ConvertItem{
		ID:     big.NewInt(0).Add(cs.MaxItemID, big.NewInt(1)),
		PubKey: info.PubKey,
		Amount: info.Amount,
	}
	cs.MaxItemID = convertItem.ID

	if _, ok := cs.ConvertItems[ExpandedTxConvert_Czz]; !ok {
		item := ConvertItems{}
		items := make([]*ConvertItem, 0, 0)
		items = append(items, convertItem)
		item[info.ConvertType] = items
		cs.ConvertItems[ExpandedTxConvert_Czz] = item
	} else {
		items := cs.ConvertItems[ExpandedTxConvert_Czz][info.ConvertType]
		items = append(items, convertItem)
		cs.ConvertItems[ExpandedTxConvert_Czz][info.ConvertType] = items
	}
	return nil
}

// AddEntangleItem add item in the state, keep BeaconAddress have enough amount to entangle,
func (cs *CommitteeState) ConvertConfirm(info *ConvertConfirmTxInfo) error {

	var hinfo *ConvertItem
	var index int
	items := cs.ConvertItems[info.AssetType][info.ConvertType]
	for i, v := range items {
		if v.ID.Cmp(info.ID) == 0 {
			hinfo = v
			index = i
		}
	}

	if hinfo == nil {
		return fmt.Errorf("")
	}

	items = append(items[:index], items[index+1:]...)
	cs.ConvertItems[info.AssetType][info.ConvertType] = items

	convertItem := &ConvertItem{
		ID:        hinfo.ID,
		ExtTxHash: hinfo.ExtTxHash,
		PubKey:    hinfo.PubKey,
		Amount:    hinfo.Amount,
	}

	if _, ok := cs.ConvertConfirmItems[info.AssetType]; !ok {
		item := ConvertItems{}
		items := make([]*ConvertItem, 0, 0)
		items = append(items, convertItem)
		item[info.ConvertType] = items
		cs.ConvertConfirmItems[info.AssetType] = item
	} else {
		items := cs.ConvertConfirmItems[info.AssetType][info.ConvertType]
		items = append(items, convertItem)
		cs.ConvertConfirmItems[info.AssetType][info.ConvertType] = items
	}

	return nil
}

func (cs *CommitteeState) PutNoCostUtxos(address string, POut wire.OutPoint, Script []byte, Amount int64) {

	if _, ok := cs.NoCostUtxos[address]; ok {
		ncu := cs.NoCostUtxos[address]
		ncu.POut = append(ncu.POut, POut)
		ncu.Script = append(ncu.Script, Script)
		ncu.Amount = append(ncu.Amount, big.NewInt(Amount))
		cs.NoCostUtxos[address] = ncu
	} else {
		items := &PoolAddrItem{
			POut:   make([]wire.OutPoint, 0),
			Script: make([][]byte, 0),
			Amount: make([]*big.Int, 0),
		}
		items.POut = append(items.POut, POut)
		items.Script = append(items.Script, Script)
		items.Amount = append(items.Amount, big.NewInt(Amount))
		cs.NoCostUtxos[address] = items
	}

}

func (cs *CommitteeState) ToBytes() []byte {
	// maybe rlp encode
	//msg, err := json.Marshal(es)
	//fmt.Println("EntangleState = ", string(msg))
	data, err := rlp.EncodeToBytes(cs)
	if err != nil {
		log.Fatal("Failed to RLP encode EntangleState: ", err)
	}
	return data
}

func (cs *CommitteeState) Hash() chainhash.Hash {
	return chainhash.HashH(cs.ToBytes())
}

func NewCommitteeState() *CommitteeState {
	return &CommitteeState{
		PledgeInfos:         make([]*PledgeInfo, 0, 0),
		CommitteeInfos:      make([]*CommitteeInfo, 0, 0),
		MaxItemID:           big.NewInt(0),
		ConvertItems:        make(map[uint8]ConvertItems),
		ConvertConfirmItems: make(map[uint8]ConvertItems),
		NoCostUtxos:         make(map[string]*PoolAddrItem),
	}
}

////////////////////////////////////////////////////////////////////////////
type EntangleState struct {
	EnInfos       map[string]*BeaconAddressInfo
	EnEntitys     map[uint64]UserEntangleInfos
	PoolAmount1   *big.Int
	PoolAmount2   *big.Int
	CurExchangeID uint64
}
type StoreBeaconAddress struct {
	Address string
	Lh      *BeaconAddressInfo
}
type StoreUserInfos struct {
	EID       uint64
	UserInfos UserEntangleInfos
}
type SortStoreBeaconAddress []*StoreBeaconAddress

func (vs SortStoreBeaconAddress) Len() int {
	return len(vs)
}
func (vs SortStoreBeaconAddress) Less(i, j int) bool {
	return bytes.Compare([]byte(vs[i].Address), []byte(vs[j].Address)) == -1
}
func (vs SortStoreBeaconAddress) Swap(i, j int) {
	it := vs[i]
	vs[i] = vs[j]
	vs[j] = it
}

type SortStoreUserInfos []*StoreUserInfos

func (vs SortStoreUserInfos) Len() int {
	return len(vs)
}
func (vs SortStoreUserInfos) Less(i, j int) bool {
	return vs[i].EID < vs[j].EID
}
func (vs SortStoreUserInfos) Swap(i, j int) {
	it := vs[i]
	vs[i] = vs[j]
	vs[j] = it
}

/////////////////////////////////////////////////////////////////

func (es *EntangleState) toSlice() (SortStoreBeaconAddress, SortStoreUserInfos) {
	v1, v2 := make([]*StoreBeaconAddress, 0, 0), make([]*StoreUserInfos, 0, 0)
	for k, v := range es.EnInfos {
		v1 = append(v1, &StoreBeaconAddress{
			Address: k,
			Lh:      v,
		})
	}
	for k, v := range es.EnEntitys {
		v2 = append(v2, &StoreUserInfos{
			EID:       k,
			UserInfos: v,
		})
	}
	sort.Sort(SortStoreBeaconAddress(v1))
	sort.Sort(SortStoreUserInfos(v2))
	return SortStoreBeaconAddress(v1), SortStoreUserInfos(v2)
}

func (es *EntangleState) fromSlice(v1 SortStoreBeaconAddress, v2 SortStoreUserInfos) {
	enInfos := make(map[string]*BeaconAddressInfo)
	entitys := make(map[uint64]UserEntangleInfos)
	for _, v := range v1 {
		enInfos[v.Address] = v.Lh
	}
	for _, v := range v2 {
		entitys[v.EID] = v.UserInfos
	}
	es.EnInfos, es.EnEntitys = enInfos, entitys
}

func (es *EntangleState) DecodeRLP(s *rlp.Stream) error {
	type Store1 struct {
		ID     uint64
		Value1 SortStoreBeaconAddress
		Value2 SortStoreUserInfos
	}
	var eb Store1
	if err := s.Decode(&eb); err != nil {
		return err
	}
	es.CurExchangeID = eb.ID
	es.fromSlice(eb.Value1, eb.Value2)
	return nil
}
func (es *EntangleState) EncodeRLP(w io.Writer) error {
	type Store1 struct {
		ID     uint64
		Value1 SortStoreBeaconAddress
		Value2 SortStoreUserInfos
	}
	s1, s2 := es.toSlice()
	return rlp.Encode(w, &Store1{
		ID:     es.CurExchangeID,
		Value1: s1,
		Value2: s2,
	})
}
func (es *EntangleState) getBeaconByID(eid uint64) *BeaconAddressInfo {
	for _, v := range es.EnInfos {
		if v.ExchangeID == eid {
			return v
		}
	}
	return nil
}
func (es *EntangleState) getBeaconAddressFromTo(to []byte) *BeaconAddressInfo {
	for _, v := range es.EnInfos {
		if bytes.Equal(v.ToAddress, to) {
			return v
		}
	}
	return nil
}

/////////////////////////////////////////////////////////////////
// keep staking enough amount asset
func (es *EntangleState) RegisterBeaconAddress(addr string, to []byte, amount *big.Int,
	fee, keeptime uint64, assetType uint32, wu []*WhiteUnit, cba []string) error {
	if !validFee(big.NewInt(int64(fee))) || !validKeepTime(big.NewInt(int64(keeptime))) ||
		!ValidAssetType(assetType) {
		return ErrInvalidParam
	}
	if amount.Cmp(MinStakingAmount) < 0 {
		return ErrLessThanMin
	}
	if _, ok := es.EnInfos[addr]; ok {
		return ErrRepeatRegister
	}
	if info := es.getBeaconAddressFromTo(to); info != nil {
		return ErrRepeatToAddress
	}
	info := &BeaconAddressInfo{
		ExchangeID:      es.CurExchangeID + 1,
		Address:         addr,
		ToAddress:       to,
		StakingAmount:   new(big.Int).Set(amount),
		AssetFlag:       assetType,
		Fee:             fee,
		KeepTime:        keeptime,
		EnAssets:        make([]*EnAssetItem, 0, 0),
		EntangleAmount:  big.NewInt(0),
		WhiteList:       wu,
		CoinBaseAddress: cba,
	}
	es.CurExchangeID = info.ExchangeID
	es.EnInfos[addr] = info
	return nil
}
func (es *EntangleState) AppendWhiteList(addr string, wlist []*WhiteUnit) error {
	if val, ok := es.EnInfos[addr]; ok {
		cnt := len(val.WhiteList)
		if cnt+len(wlist) >= MaxWhiteListCount {
			return errors.New("more than max white list")
		}
		for _, v := range wlist {
			if ValidAssetType(v.AssetType) && ValidPK(v.Pk) {
				val.WhiteList = append(val.WhiteList, v)
			}
		}
		return nil
	} else {
		return ErrNoRegister
	}
}
func (es *EntangleState) AppendCoinbase(addr string, coinbases []string) error {
	if val, ok := es.EnInfos[addr]; ok {
		cnt := len(val.CoinBaseAddress)
		if cnt+len(coinbases) >= MaxCoinBase {
			return errors.New("more than max coinbase")
		}
		for _, v := range coinbases {
			if v != "" {
				val.CoinBaseAddress = append(val.CoinBaseAddress, v)
			}
		}
		return nil
	} else {
		return ErrNoRegister
	}
}
func (es *EntangleState) AppendAmountForBeaconAddress(addr string, amount *big.Int) error {
	if info, ok := es.EnInfos[addr]; !ok {
		return ErrRepeatRegister
	} else {
		info.StakingAmount = new(big.Int).Add(info.StakingAmount, amount)
		return nil
	}
}
func (es *EntangleState) UpdateCoinbase(addr, update, newAddr string) error {
	if val, ok := es.EnInfos[addr]; ok {
		for i, v := range val.CoinBaseAddress {
			if v == update {
				val.CoinBaseAddress[i] = newAddr
			}
		}
		return nil
	} else {
		return ErrNoRegister
	}
}
func (es *EntangleState) UpdateCfgForBeaconAddress(addr string, fee, keeptime uint64, assetType uint32) error {
	if !validFee(big.NewInt(int64(fee))) || !validKeepTime(big.NewInt(int64(keeptime))) ||
		!ValidAssetType(assetType) {
		return ErrInvalidParam
	}
	if info, ok := es.EnInfos[addr]; ok {
		return ErrRepeatRegister
	} else {
		info.Fee, info.AssetFlag, info.KeepTime = fee, assetType, keeptime
	}
	return nil
}
func (es *EntangleState) GetCoinbase(addr string) []string {
	if val, ok := es.EnInfos[addr]; ok {
		res := make([]string, 0, 0)
		res = append(res, val.CoinBaseAddress[:]...)
	}
	return nil
}

// UnregisterBeaconAddress need to check all the proves and handle all the user's burn coins
func (es *EntangleState) UnregisterBeaconAddress(addr string) error {
	if val, ok := es.EnInfos[addr]; ok {
		last := new(big.Int).Sub(val.StakingAmount, val.EntangleAmount)
		redeemAmount(addr, last)
	} else {
		return ErrNoRegister
	}
	return nil
}

// BurnAsset user burn the czz asset to exchange the outside asset,the caller keep the burn was true.
// verify the txid,keep equal amount czz
// returns the amount czz by user's burnned, took out fee by beaconaddress
func (es *EntangleState) BurnAsset(addr string, aType uint32, lightID, height uint64,
	amount *big.Int) (*big.Int, error) {
	light := es.getBeaconAddress(lightID)
	if light == nil {
		return nil, ErrNoRegister
	}
	lhEntitys, ok := es.EnEntitys[lightID]
	if !ok {
		return nil, ErrNoRegister
	}
	userEntitys, ok1 := lhEntitys[addr]
	if !ok1 {
		return nil, ErrNoUserReg
	}
	// self redeem amount, maybe add the free quota in the BeaconAddress
	validAmount := userEntitys.getAllRedeemableAmount()
	if amount.Cmp(validAmount) > 0 {
		return nil, ErrNotEnouthBurn
	}

	var userEntity *EntangleEntity
	for _, v := range userEntitys {
		if aType == v.AssetType {
			userEntity = v
			break
		}
	}
	if userEntity == nil {
		return nil, ErrNoUserAsset
	}
	reserve := es.getEntangledAmount(lightID, aType)
	base, divisor, err := getRedeemRateByBurnCzz(reserve, aType)
	if err != nil {
		return nil, err
	}
	// get out asset for burn czz
	outAmount := new(big.Int).Div(new(big.Int).Mul(amount, base), divisor)
	userEntity.BurnAmount.addBurnItem(height, amount, outAmount)
	res := new(big.Int).Div(new(big.Int).Mul(amount, big.NewInt(int64(light.Fee))), big.NewInt(int64(MAXBASEFEE)))

	return new(big.Int).Sub(amount, res), nil
}
func (es *EntangleState) SetInitPoolAmount(amount1, amount2 *big.Int) {
	es.PoolAmount1, es.PoolAmount2 = new(big.Int).Set(amount1), new(big.Int).Set(amount2)
}
func (es *EntangleState) AddPoolAmount(amount1, amount2 *big.Int) {
	es.PoolAmount1 = new(big.Int).Add(es.PoolAmount1, amount1)
	es.PoolAmount2 = new(big.Int).Add(es.PoolAmount2, amount2)
}
func (es *EntangleState) SubPoolAmount1(amount *big.Int) {
	es.PoolAmount1 = new(big.Int).Sub(es.PoolAmount1, amount)
}
func (es *EntangleState) SubPoolAmount2(amount *big.Int) {
	es.PoolAmount2 = new(big.Int).Sub(es.PoolAmount2, amount)
}

//////////////////////////////////////////////////////////////////////
func redeemAmount(addr string, amount *big.Int) error {
	if amount.Sign() > 0 {
	}
	return nil
}

func calcEntangleAmount(reserve, reqAmount *big.Int, atype uint8) (*big.Int, error) {
	switch atype {
	case ExpandedTxEntangle_Doge:
		return toDoge2(reserve, reqAmount), nil
	case ExpandedTxEntangle_Ltc:
		return toLtc2(reserve, reqAmount), nil
	default:
		return nil, ErrNoUserAsset
	}
}

func getRedeemRateByBurnCzz(reserve *big.Int, atype uint32) (*big.Int, *big.Int, error) {
	return nil, nil, ErrNoUserAsset
}

func (es *EntangleState) AddressInWhiteList(addr string, self bool) bool {
	for k, val := range es.EnInfos {
		if self && equalAddress(k, addr) {
			return true
		}
		if val.addressInWhiteList(addr) {
			return true
		}
	}
	return false
}

func (es *EntangleState) getEntangledAmount(lightID uint64, atype uint32) *big.Int {
	aa := big.NewInt(0)
	if lhEntitys, ok := es.EnEntitys[lightID]; ok {
		for _, userEntitys := range lhEntitys {
			for _, vv := range userEntitys {
				if atype == vv.AssetType {
					aa = aa.Add(aa, vv.EnOutsideAmount)
					break
				}
			}
		}
	}
	return aa
}

func (es *EntangleState) getBeaconAddress(id uint64) *BeaconAddressInfo {
	for _, val := range es.EnInfos {
		if val.ExchangeID == id {
			return val
		}
	}
	return nil
}

func (es *EntangleState) getAllEntangleAmount(atype uint32) *big.Int {
	all := big.NewInt(0)
	for _, val := range es.EnInfos {
		for _, v := range val.EnAssets {
			if v.AssetType == atype {
				all = all.Add(all, v.Amount)
				break
			}
		}
	}
	return all
}

//Minimum pledge amount = 1 million CZZ + (cumulative cross-chain buying CZZ - cumulative cross-chain selling CZZ) x exchange rate ratio
func (es *EntangleState) LimitStakingAmount(eid uint64, atype uint32) *big.Int {
	lh := es.getBeaconAddress(eid)
	if lh != nil {
		l := new(big.Int).Sub(lh.StakingAmount, lh.EntangleAmount)
		if l.Sign() > 0 {
			l = new(big.Int).Sub(l, MinStakingAmount)
			if l.Sign() > 0 {
				return l
			}
		}
	}
	return nil
}

//////////////////////////////////////////////////////////////////////
// UpdateQuotaOnBlock called in insertBlock for update user's quota state
func (es *EntangleState) UpdateQuotaOnBlock(height uint64) error {
	for _, lh := range es.EnInfos {
		userEntitys, ok := es.EnEntitys[lh.ExchangeID]
		if !ok {
			fmt.Println("cann't found the BeaconAddress id:", lh.ExchangeID)
		} else {
			for _, userEntity := range userEntitys {
				res := userEntity.updateFreeQuotaForAllType(big.NewInt(int64(height)), big.NewInt(int64(lh.KeepTime)))
				lh.updateFreeQuota(res)
			}
		}
	}
	return nil
}

// TourAllUserBurnInfo Tours all user's burned asset and check which is timeout to redeem
func (es *EntangleState) TourAllUserBurnInfo(height uint64) map[uint64]UserTimeOutBurnInfo {
	// maybe get cache for recently burned user
	res := make(map[uint64]UserTimeOutBurnInfo)
	for k, users := range es.EnEntitys {
		userItems := make(map[string]TypeTimeOutBurnInfo)
		for k1, entitys := range users {
			items := entitys.getBurnTimeout(height, true)
			if len(items) > 0 {
				userItems[k1] = items
			}
		}
		if len(userItems) > 0 {
			res[k] = UserTimeOutBurnInfo(userItems)
		}
	}
	return res
}
func (es *EntangleState) UpdateStateToPunished(infos map[uint64]UserTimeOutBurnInfo) {
	for eid, items := range infos {
		userEntitys, ok := es.EnEntitys[eid]
		if ok {
			// set state=3 after be punished by system consensus
			userEntitys.updateBurnState(3, items)
		}
	}
}
func SummayPunishedInfos(infos map[uint64]UserTimeOutBurnInfo) map[uint64]LHPunishedItems {
	res := make(map[uint64]LHPunishedItems)
	for k, userInfos := range infos {
		items := make([]*LHPunishedItem, 0, 0)
		for addr, val := range userInfos {
			items = append(items, &LHPunishedItem{
				User: addr,
				All:  val.getAll(),
			})
		}
		res[k] = LHPunishedItems(items)
	}
	return res
}
func (es *EntangleState) FinishBeaconAddressPunished(eid uint64, amount *big.Int) error {
	beacon := es.getBeaconByID(eid)
	if beacon == nil {
		return ErrNoRegister
	}
	// get limit staking warnning message
	return beacon.updatePunished(amount)
}
func (es *EntangleState) verifyBurnProof(info *BurnProofInfo) error {
	userEntitys, ok := es.EnEntitys[info.LightID]
	if !ok {
		fmt.Println("verifyBurnProof:cann't found the BeaconAddress id:", info.LightID)
		return ErrNoRegister
	} else {
		for addr1, userEntity := range userEntitys {
			if bytes.Equal(info.Address.ScriptAddress(), []byte(addr1)) {
				return userEntity.verifyBurnProof(info)
			} else {
				return ErrNotMatchUser
			}
		}
	}
	return nil
}

// FinishHandleUserBurn the BeaconAddress finish the burn item
func (es *EntangleState) FinishHandleUserBurn(lightID, height uint64, addr czzutil.Address, atype uint32, amount *big.Int) error {
	userEntitys, ok := es.EnEntitys[lightID]
	if !ok {
		fmt.Println("FinishHandleUserBurn:cann't found the BeaconAddress id:", lightID)
		return ErrNoRegister
	} else {
		for addr1, userEntity := range userEntitys {
			if bytes.Equal(addr.ScriptAddress(), []byte(addr1)) {
				userEntity.finishBurnState(height, amount, atype)
			}
		}
	}
	return nil
}
func (es *EntangleState) ToBytes() []byte {
	// maybe rlp encode
	data, err := rlp.EncodeToBytes(es)
	if err != nil {
		log.Fatal("Failed to RLP encode EntangleState", "err", err)
	}
	return data
}
func (es *EntangleState) Save() error {
	return nil
}
func (es *EntangleState) Load() error {
	return nil
}

func (es *EntangleState) Hash() chainhash.Hash {
	return chainhash.HashH(es.ToBytes())
}
func NewEntangleState() *EntangleState {
	return &EntangleState{
		EnInfos:       make(map[string]*BeaconAddressInfo),
		EnEntitys:     make(map[uint64]UserEntangleInfos),
		CurExchangeID: 0,
		PoolAmount1:   big.NewInt(0),
		PoolAmount2:   big.NewInt(0),
	}
}
