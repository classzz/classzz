package cross

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"sort"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/rlp"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

type ExBeaconInfo struct {
	EnItems []*wire.OutPoint
	Proofs  []*WhiteListProof
}

func (e *ExBeaconInfo) EqualProof(proof *WhiteListProof) bool {
	for _, v := range e.Proofs {
		if v.Height == proof.Height {
			return true
		}
	}
	return false
}
func (e *ExBeaconInfo) AppendProof(proof *WhiteListProof) error {
	if !e.EqualProof(proof) {
		e.Proofs = append(e.Proofs, proof.Clone())
		return nil
	}
	return ErrRepeatProof
}

type EntangleState struct {
	EnInfos       map[string]*BeaconAddressInfo
	EnEntitys     map[uint64]UserEntangleInfos
	BaExInfo      map[uint64]*ExBeaconInfo // merge tx(outpoint) in every lid
	PoolAmount1   *big.Int
	PoolAmount2   *big.Int
	CurExchangeID uint64
}

/////////////////////////////////////////////////////////////////
type StoreBeaconAddress struct {
	Address string
	Lh      *BeaconAddressInfo
}
type StoreUserInfos struct {
	EID       uint64
	UserInfos UserEntangleInfos
}
type StoreBeaconExInfos struct {
	EID   uint64
	EItem *ExBeaconInfo
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

type SortStoreBeaconExInfos []*StoreBeaconExInfos

func (vs SortStoreBeaconExInfos) Len() int {
	return len(vs)
}
func (vs SortStoreBeaconExInfos) Less(i, j int) bool {
	return vs[i].EID < vs[j].EID
}
func (vs SortStoreBeaconExInfos) Swap(i, j int) {
	it := vs[i]
	vs[i] = vs[j]
	vs[j] = it
}

/////////////////////////////////////////////////////////////////
func (es *EntangleState) toSlice() (SortStoreBeaconAddress, SortStoreUserInfos, SortStoreBeaconExInfos) {
	v1, v2, v3 := make([]*StoreBeaconAddress, 0, 0), make([]*StoreUserInfos, 0, 0), make([]*StoreBeaconExInfos, 0, 0)
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
	for k, v := range es.BaExInfo {
		v3 = append(v3, &StoreBeaconExInfos{
			EID:   k,
			EItem: v,
		})
	}
	sort.Sort(SortStoreBeaconAddress(v1))
	sort.Sort(SortStoreUserInfos(v2))
	sort.Sort(SortStoreBeaconExInfos(v3))
	return SortStoreBeaconAddress(v1), SortStoreUserInfos(v2), SortStoreBeaconExInfos(v3)
}
func (es *EntangleState) fromSlice(v1 SortStoreBeaconAddress, v2 SortStoreUserInfos, v3 SortStoreBeaconExInfos) {
	enInfos := make(map[string]*BeaconAddressInfo)
	entitys := make(map[uint64]UserEntangleInfos)
	exInfos := make(map[uint64]*ExBeaconInfo)
	for _, v := range v1 {
		enInfos[v.Address] = v.Lh
	}
	for _, v := range v2 {
		entitys[v.EID] = v.UserInfos
	}
	for _, v := range v3 {
		exInfos[v.EID] = v.EItem
	}
	es.EnInfos, es.EnEntitys, es.BaExInfo = enInfos, entitys, exInfos
}
func (es *EntangleState) DecodeRLP(s *rlp.Stream) error {
	type Store1 struct {
		ID     uint64
		Value1 SortStoreBeaconAddress
		Value2 SortStoreUserInfos
		Value3 SortStoreBeaconExInfos
	}
	var eb Store1
	if err := s.Decode(&eb); err != nil {
		return err
	}
	es.CurExchangeID = eb.ID
	es.fromSlice(eb.Value1, eb.Value2, eb.Value3)
	return nil
}
func (es *EntangleState) EncodeRLP(w io.Writer) error {
	type Store1 struct {
		ID     uint64
		Value1 SortStoreBeaconAddress
		Value2 SortStoreUserInfos
		Value3 SortStoreBeaconExInfos
	}
	s1, s2, s3 := es.toSlice()
	return rlp.Encode(w, &Store1{
		ID:     es.CurExchangeID,
		Value1: s1,
		Value2: s2,
		Value3: s3,
	})
}

/////////////////////////////////////////////////////////////////
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
func (es *EntangleState) GetBeaconIdByTo(to []byte) uint64 {
	info := es.getBeaconAddressFromTo(to)
	if info != nil {
		return info.ExchangeID
	}
	return 0
}
func (es *EntangleState) getBeaconToAddressByID(i uint64) []byte {
	if info := es.getBeaconByID(i); info != nil {
		return info.getToAddress()
	}
	return nil
}
func (es *EntangleState) GetBeaconToAddrByID(i uint64) czzutil.Address {
	if b := es.getBeaconToAddressByID(i); b != nil {
		addr, err := czzutil.NewLegacyAddressPubKeyHash(b, &chaincfg.MainNetParams)
		if err == nil {
			return addr
		}
	}
	return nil
}
func (es *EntangleState) GetExInfosByID(id uint64) *ExBeaconInfo {
	if v, ok := es.BaExInfo[id]; ok {
		return v
	}
	return nil
}
func (es *EntangleState) SetExBeaconInfo(id uint64, info *ExBeaconInfo) error {
	es.BaExInfo[id] = info
	return nil
}
func (es *EntangleState) GetOutSideAsset(id uint64, atype uint8) *big.Int {
	lh := es.getBeaconByID(id)
	if lh == nil {
		return nil
	}
	return lh.getOutSideAsset(atype)
}
func (es *EntangleState) GetWhiteList(id uint64) []*WhiteUnit {
	lh := es.getBeaconByID(id)
	if lh == nil {
		return nil
	}
	return lh.getWhiteList()
}
func (es *EntangleState) getBeaconAddressByID(id uint64) string {
	lh := es.getBeaconByID(id)
	if lh == nil {
		return ""
	}
	return lh.Address
}

/////////////////////////////////////////////////////////////////
// keep staking enough amount asset
func (es *EntangleState) RegisterBeaconAddress(addr string, to []byte, pubkey []byte, amount *big.Int,
	fee, keeptime uint64, assetFlag uint32, wu []*WhiteUnit, cba []string) error {
	if !validFee(big.NewInt(int64(fee))) || !validKeepTime(big.NewInt(int64(keeptime))) ||
		!ValidAssetFlag(assetFlag) {
		return ErrInvalidParam
	}
	if amount.Cmp(MinStakingAmountForBeaconAddress) < 0 {
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
		PubKey:          pubkey,
		ToAddress:       to,
		StakingAmount:   new(big.Int).Set(amount),
		AssetFlag:       assetFlag,
		Fee:             fee,
		KeepTime:        keeptime,
		EnAssets:        make([]*EnAssetItem, 0, 0),
		EntangleAmount:  big.NewInt(0),
		WhiteList:       wu,
		CoinBaseAddress: cba,
	}

	ebi := &ExBeaconInfo{
		EnItems: make([]*wire.OutPoint, 0, 0),
		Proofs:  make([]*WhiteListProof, 0, 0),
	}
	es.CurExchangeID = info.ExchangeID
	es.EnInfos[addr] = info
	es.BaExInfo[es.CurExchangeID] = ebi
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
func (es *EntangleState) UpdateCfgForBeaconAddress(addr string, fee, keeptime uint64, AssetFlag uint32) error {
	if !validFee(big.NewInt(int64(fee))) || !validKeepTime(big.NewInt(int64(keeptime))) ||
		!ValidAssetFlag(AssetFlag) {
		return ErrInvalidParam
	}
	if info, ok := es.EnInfos[addr]; ok {
		return ErrRepeatRegister
	} else {
		info.Fee, info.AssetFlag, info.KeepTime = fee, AssetFlag, keeptime
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

// AddEntangleItem add item in the state, keep BeaconAddress have enough amount to entangle,
func (es *EntangleState) AddEntangleItem(addr string, aType uint8, lightID uint64,
	height, amount *big.Int) (*big.Int, error) {
	if es.AddressInWhiteList(addr, true) {
		return nil, ErrAddressInWhiteList
	}
	lh := es.getBeaconAddress(lightID)
	if lh == nil {
		return nil, ErrNoRegister
	}
	aType1 := ExpandedTxTypeToAssetType(aType)
	if !isValidAsset(aType1, lh.AssetFlag) {
		return nil, ErrNoUserAsset
	}
	sendAmount := big.NewInt(0)
	var err error
	// calc the send amount
	reserve := es.getEntangleAmountByAll(aType)
	sendAmount, err = calcEntangleAmount(reserve, amount, aType)
	if err != nil {
		return nil, err
	}
	if err := lh.EnoughToEntangle(sendAmount); err != nil {
		return nil, err
	}

	lhEntitys, ok := es.EnEntitys[lightID]
	if !ok {
		lhEntitys = UserEntangleInfos(make(map[string]EntangleEntitys))
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
				ExchangeID:      lightID,
				Address:         addr,
				AssetType:       aType,
				Height:          new(big.Int).Set(height),
				OldHeight:       new(big.Int).Set(height), // init same the Height
				EnOutsideAmount: new(big.Int).Set(amount),
				BurnAmount:      newBurnInfos(),
				MaxRedeem:       big.NewInt(0),
				OriginAmount:    big.NewInt(0),
			}
			userEntitys = append(userEntitys, userEntity)
		}
		userEntity.increaseOriginAmount(sendAmount)
		userEntity.updateFreeQuotaOfHeight(height, amount)
		lh.addEnAsset(aType, amount)
		lh.recordEntangleAmount(sendAmount)
		lhEntitys[addr] = userEntitys
		es.EnEntitys[lightID] = lhEntitys
	}
	return sendAmount, nil
}

// BurnAsset user burn the czz asset to exchange the outside asset,the caller keep the burn was true.
// verify the txid,keep equal amount czz
// returns the amount czz by user's burnned, took out fee by beaconaddress
func (es *EntangleState) BurnAsset(addr string, aType uint8, lightID, height uint64,
	amount *big.Int) (*big.Int, *big.Int, error) {
	light := es.getBeaconAddress(lightID)
	if light == nil {
		return nil, nil, ErrNoRegister
	}
	lhEntitys, ok := es.EnEntitys[lightID]
	if !ok {
		return nil, nil, ErrNoRegister
	}
	userEntitys, ok1 := lhEntitys[addr]
	if !ok1 {
		return nil, nil, ErrNoUserReg
	}
	// self redeem amount, maybe add the free quota in the BeaconAddress
	validAmount := userEntitys.getAllRedeemableAmount()
	if amount.Cmp(validAmount) > 0 {
		return nil, nil, ErrNotEnouthBurn
	}

	var userEntity *EntangleEntity
	for _, v := range userEntitys {
		if aType == v.AssetType {
			userEntity = v
			break
		}
	}
	if userEntity == nil {
		return nil, nil, ErrNoUserAsset
	}
	reserve := es.getEntangleAmountByAll(aType)
	base, divisor, err := getRedeemRateByBurnCzz(reserve, aType)
	if err != nil {
		return nil, nil, err
	}
	// get out asset for burn czz
	outAllAmount := new(big.Int).Div(new(big.Int).Mul(amount, base), divisor)
	fee := new(big.Int).Div(new(big.Int).Mul(amount, big.NewInt(int64(light.Fee))), big.NewInt(int64(MAXBASEFEE)))
	outFeeAmount := new(big.Int).Div(new(big.Int).Mul(fee, base), divisor)
	userEntity.BurnAmount.addBurnItem(height, amount, fee, outFeeAmount, outAllAmount)

	return new(big.Int).Sub(amount, fee), fee, nil
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
	case ExpandedTxEntangle_Btc:
		return toBtc(reserve, reqAmount), nil
	case ExpandedTxEntangle_Bsv, ExpandedTxEntangle_Bch:
		return toBchOrBsv(reserve, reqAmount), nil
	default:
		return nil, ErrNoUserAsset
	}
}
func getRedeemRateByBurnCzz(reserve *big.Int, atype uint8) (*big.Int, *big.Int, error) {
	switch atype {
	case ExpandedTxEntangle_Doge:
		base, divisor := reverseToDoge(reserve)
		return base, divisor, nil
	case ExpandedTxEntangle_Ltc:
		base, divisor := reverseToLtc(reserve)
		return base, divisor, nil
	case ExpandedTxEntangle_Btc:
		base, divisor := reverseToBtc(reserve)
		return base, divisor, nil
	case ExpandedTxEntangle_Bsv, ExpandedTxEntangle_Bch:
		base, divisor := reverseToBchOrBsv(reserve)
		return base, divisor, nil
	default:
		return nil, nil, ErrNoUserAsset
	}
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
func (es *EntangleState) getEntangledAmount(lightID uint64, atype uint8) *big.Int {
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
func (es *EntangleState) getEntangleAmountByAll(atype uint8) *big.Int {
	aa := big.NewInt(0)
	for _, lhEntitys := range es.EnEntitys {
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
func (es *EntangleState) getAllEntangleAmount(atype uint8) *big.Int {
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
	slashingAmount := new(big.Int).Mul(big.NewInt(2), amount)
	return beacon.updatePunished(slashingAmount)
}

func (es *EntangleState) verifyBurnProof(info *BurnProofInfo, outHeight, curHeight uint64) (*BurnItem, error) {
	userEntitys, ok := es.EnEntitys[info.LightID]
	if !ok {
		fmt.Println("verifyBurnProof:cann't found the BeaconAddress id:", info.LightID)
		return nil, ErrNoRegister
	} else {
		for addr1, userEntity := range userEntitys {
			if info.Address == addr1 {
				return userEntity.verifyBurnProof(info, outHeight, curHeight)
			} else {
				return nil, ErrNotMatchUser
			}
		}
	}
	return nil, nil
}

func (es *EntangleState) CloseProofForPunished(info *BurnProofInfo, item *BurnItem) error {
	es.FinishBeaconAddressPunished(info.LightID, info.Amount)
	userEntitys, ok := es.EnEntitys[info.LightID]
	if !ok {
		fmt.Println("CloseProofForPunished:cann't found the BeaconAddress id:", info.LightID)
		return ErrNoRegister
	} else {
		for addr1, userEntity := range userEntitys {
			if info.Address == addr1 {
				return userEntity.closeProofForPunished(item, info.Atype)
			} else {
				return ErrNotMatchUser
			}
		}
	}
	return nil
}

// FinishHandleUserBurn the BeaconAddress finish the burn item
func (es *EntangleState) FinishHandleUserBurn(info *BurnProofInfo, proof *BurnProofItem) error {
	userEntitys, ok := es.EnEntitys[info.LightID]
	if !ok {
		fmt.Println("FinishHandleUserBurn:cann't found the BeaconAddress id:", info.LightID)
		return ErrNoRegister
	} else {
		for addr1, userEntity := range userEntitys {
			if info.Address == addr1 {
				userEntity.finishBurnState(info.Height, info.Amount, info.Atype, proof)
			}
		}
	}
	return nil
}

// FinishHandleUserBurn the BeaconAddress finish the burn item
func (es *EntangleState) UpdateHandleUserBurn(info *BurnProofInfo, proof *BurnProofItem) error {
	userEntitys, ok := es.EnEntitys[info.LightID]
	if !ok {
		fmt.Println("FinishHandleUserBurn:cann't found the BeaconAddress id:", info.LightID)
		return ErrNoRegister
	} else {
		for addr1, userEntity := range userEntitys {
			if info.Address == addr1 {
				userEntity.updateBurnState2(info.Height, info.Amount, info.Atype, proof)
			}
		}
	}
	return nil
}

//func (es *EntangleState) VerifyWhiteListProof(proof *WhiteListProof) error {
//	if info := es.GetExInfosByID(proof.LightID); info != nil {
//		if !info.EqualProof(proof) {
//			return ErrRepeatProof
//		} else {
//			return nil
//		}
//	}
//	return ErrNoRegister
//}
func (es *EntangleState) FinishWhiteListProof(proof *WhiteListProof) error {
	if info := es.GetExInfosByID(proof.LightID); info != nil {
		info.AppendProof(proof)
		es.SetExBeaconInfo(proof.LightID, info)
		return nil
	}
	return ErrNoRegister
}

////////////////////////////////////////////////////////////////////////////
// calc the punished amount by outside asset in the height
// the return value(flag by czz) will be mul * 2
func (es *EntangleState) CalcSlashingForWhiteListProof(outAmount *big.Int, atype uint8, lightID uint64) *big.Int {
	// get current rate with czz and outside asset in heigth
	reserve := es.getEntangleAmountByAll(atype)
	sendAmount, err := calcEntangleAmount(reserve, outAmount, atype)
	if err != nil {
		return nil
	}
	return sendAmount
}

////////////////////////////////////////////////////////////////////////////
func (es *EntangleState) ToBytes() []byte {
	// maybe rlp encode
	data, err := rlp.EncodeToBytes(es)
	if err != nil {
		log.Fatal("Failed to RLP encode EntangleState: ", "err", err)
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
	return chainhash.HashH(es.ToBytes())
}
func NewEntangleState() *EntangleState {
	return &EntangleState{
		EnInfos:       make(map[string]*BeaconAddressInfo),
		EnEntitys:     make(map[uint64]UserEntangleInfos),
		BaExInfo:      make(map[uint64]*ExBeaconInfo), // merge tx(outpoint) in every lid
		CurExchangeID: 0,
		PoolAmount1:   big.NewInt(0),
		PoolAmount2:   big.NewInt(0),
	}
}
