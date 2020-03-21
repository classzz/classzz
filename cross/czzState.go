package cross

import (
	"bytes"
	"io"
	"log"
	// "encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sort"

	// "github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"
	// "github.com/classzz/classzz/czzec"
	// "github.com/classzz/classzz/txscript"
	// "github.com/classzz/classzz/wire"
	"github.com/classzz/classzz/rlp"
	"github.com/classzz/czzutil"
)

type EntangleState struct {
	EnInfos       map[czzutil.Address]*BeaconAddressInfo
	EnEntitys     map[uint64]UserEntangleInfos
	CurExchangeID uint64
}
type StoreBeaconAddress struct {
	Address czzutil.Address
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
	return bytes.Compare(vs[i].Address.ScriptAddress(),vs[j].Address.ScriptAddress()) == -1
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
	return nil, nil
}
func (es *EntangleState) fromSlice(v1 SortStoreBeaconAddress, v2 SortStoreUserInfos) {
	enInfos := make(map[czzutil.Address]*BeaconAddressInfo)
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

/////////////////////////////////////////////////////////////////
// keep staking enough amount asset
func (es *EntangleState) RegisterBeaconAddress(addr czzutil.Address, amount *big.Int,
	fee, keeptime uint64, assetType uint32) error {
	if amount.Cmp(MinStakingAmountForBeaconAddress) < 0 {
		return ErrLessThanMin
	}
	if _, ok := es.EnInfos[addr]; ok {
		return ErrRepeatRegister
	}
	info := &BeaconAddressInfo{
		ExchangeID:     es.CurExchangeID + 1,
		Address:        addr,
		StakingAmount:  new(big.Int).Set(amount),
		AssetFlag:      assetType,
		Fee:            fee,
		KeepTime:       keeptime,
		EnAssets:       make([]*EnAssetItem, 0, 0),
		EntangleAmount: big.NewInt(0),
		WhiteList:      make([]*WhiteUnit, 0, 0),
	}
	es.EnInfos[addr] = info
	return nil
}
func (es *EntangleState) AppendWhiteList(addr czzutil.Address, wlist []*WhiteUnit) error {
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

// UnregisterBeaconAddress need to check all the proves and handle all the user's burn coins
func (es *EntangleState) UnregisterBeaconAddress(addr czzutil.Address) error {
	if val, ok := es.EnInfos[addr]; ok {
		last := new(big.Int).Sub(val.StakingAmount, val.EntangleAmount)
		redeemAmount(addr, last)
	} else {
		return ErrNoRegister
	}
	return nil
}

// AddEntangleItem add item in the state, keep BeaconAddress have enough amount to entangle,
func (es *EntangleState) AddEntangleItem(addr czzutil.Address, aType uint32, lightID uint64,
	height, amount *big.Int) (*big.Int, error) {
	if es.AddressInWhiteList(addr,true) {
		return nil,ErrAddressInWhiteList
	}
	lh := es.getBeaconAddress(lightID)
	if lh == nil {
		return nil, ErrNoRegister
	}
	if !isValidAsset(aType, lh.AssetFlag) {
		return nil, ErrNoUserAsset
	}
	sendAmount := big.NewInt(0)
	var err error
	lhEntitys, ok := es.EnEntitys[lightID]
	if !ok {
		lhEntitys = UserEntangleInfos(make(map[czzutil.Address]EntangleEntitys))
	}
	if lhEntitys != nil {
		userEntitys, ok1 := lhEntitys[addr]
		if !ok1 {
			userEntitys = EntangleEntitys(make([]*EntangleEntity, 0, 0))
		}
		found := false
		var userEntity *EntangleEntity
		for _, v := range userEntitys {
			if aType == v.AssetType {
				found = true
				v.EnOutsideAmount = new(big.Int).Add(v.EnOutsideAmount, amount)
				userEntity = v
				break
			}
		}
		if !found {
			userEntity = &EntangleEntity{
				ExchangeID:     lightID,
				Address:        addr,
				AssetType:      aType,
				Height:         new(big.Int).Set(height),
				EnOutsideAmount: new(big.Int).Set(amount),
				BurnAmount:     newBurnInfos(),
				MaxRedeem:		big.NewInt(0),
				OriginAmount:	big.NewInt(0),
			}
			userEntitys = append(userEntitys, userEntity)
		}

		// calc the send amount
		reserve := es.getEntangledAmount(lightID, aType)
		sendAmount, err = calcEntangleAmount(reserve, amount, aType)
		if err != nil {
			return nil, err
		}
		userEntity.increaseOriginAmount(sendAmount)
		userEntity.updateFreeQuotaOfHeight(height,amount)
		lh.addEnAsset(aType, amount)
		lh.recordEntangleAmount(sendAmount)
		lhEntitys[addr] = userEntitys
		es.EnEntitys[lightID] = lhEntitys
	}
	return sendAmount, nil
}

// BurnAsset user burn the czz asset to exchange the outside asset,the caller keep the burn was true.
// verify the txid,keep equal amount czz
func (es *EntangleState) BurnAsset(addr czzutil.Address, aType uint32, lightID,height uint64,
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
	userEntity.BurnAmount.addBurnItem(height,amount)
	res := new(big.Int).Div(new(big.Int).Mul(amount, big.NewInt(int64(light.Fee))), big.NewInt(int64(light.Fee)))

	return res, nil
}
func (es *EntangleState) ConfiscateAsset() error {
	return nil
}

//////////////////////////////////////////////////////////////////////
func redeemAmount(addr czzutil.Address, amount *big.Int) error {
	if amount.Sign() > 0 {

	}
	return nil
}
func calcEntangleAmount(reserve, reqAmount *big.Int, atype uint32) (*big.Int, error) {
	return nil, nil
}
func (es *EntangleState) AddressInWhiteList(addr czzutil.Address,self bool) bool {
	for k,val := range es.EnInfos {
		if self && equalAddress(k,addr) {
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
// 最低质押额度＝ 100 万 CZZ ＋（累计跨链买入 CZZ －累计跨链卖出 CZZ）x 汇率比
func (es *EntangleState) LimitStakingAmount(eid uint64, atype uint32) *big.Int {
	lh := es.getBeaconAddress(eid)
	if lh != nil {
		l := new(big.Int).Sub(lh.StakingAmount, lh.EntangleAmount)
		if l.Sign() > 0 {
			l = new(big.Int).Sub(l, MinStakingAmountForBeaconAddress)
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
				res := userEntity.updateFreeQuotaForAllType(big.NewInt(int64(height)),big.NewInt(int64(lh.KeepTime)))
				lh.updateFreeQuota(res)
			}
		}
	}
	return nil
}
// TourAllUserBurnInfo Tours all user's burned asset and check which is timeout to redeem
func (es *EntangleState) TourAllUserBurnInfo(height uint64) map[uint64]UserTimeOutBurnInfo  {
	// maybe get cache for recently burned user
	res := make(map[uint64]UserTimeOutBurnInfo)
	for k,users :=range es.EnEntitys {
		userItems := make(map[czzutil.Address]TypeTimeOutBurnInfo)
		for k1,entitys := range users {
			items := entitys.getBurnTimeout(height,true)
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
	for eid,items := range infos {
		userEntitys, ok := es.EnEntitys[eid]
		if ok {
			// set state=3 after be punished by system consensus
			userEntitys.updateBurnState(3,items)
		}
	}
}
func SummayPunishedInfos(infos map[uint64]UserTimeOutBurnInfo) map[uint64]LHPunishedItems {
	res := make(map[uint64]LHPunishedItems)
	for k,userInfos := range infos {
		items := make([]*LHPunishedItem,0,0)
		for addr,val := range userInfos {
			items = append(items,&LHPunishedItem{
				User:	addr,
				All:	val.getAll(),
			})
		}
		res[k] = LHPunishedItems(items)
	}
	return res
}
// FinishHandleUserBurn the BeaconAddress finish the burn item
func (es *EntangleState) FinishHandleUserBurn(lightID,height uint64,addr czzutil.Address,atype uint32,amount  *big.Int) error {
	userEntitys, ok := es.EnEntitys[lightID]
	if !ok {
		fmt.Println("FinishHandleUserBurn:cann't found the BeaconAddress id:", lightID)
	} else {
		for addr1, userEntity := range userEntitys {
			if bytes.Equal(addr.ScriptAddress(),addr1.ScriptAddress()) {
				userEntity.finishBurnState(height,amount,atype)
			}
		}
	} 
	return nil
}
func (es *EntangleState) toBytes() []byte {
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
func Hash(es *EntangleState) chainhash.Hash {
	return chainhash.HashH(es.toBytes())
}
func NewEntangleState() *EntangleState {
	return &EntangleState{
		EnInfos:       make(map[czzutil.Address]*BeaconAddressInfo),
		EnEntitys:     make(map[uint64]UserEntangleInfos),
		CurExchangeID: 0,
	}
}
