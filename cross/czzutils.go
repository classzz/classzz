package cross

import (
	"bytes"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/czzutil"

	// "encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"

	// "github.com/classzz/classzz/chaincfg"
	// "github.com/classzz/classzz/chaincfg/chainhash"
	// "github.com/classzz/classzz/czzec"
	// "github.com/classzz/classzz/txscript"
	// "github.com/classzz/classzz/wire"
	"github.com/classzz/classzz/rlp"
)

var (
	ErrInvalidParam       = errors.New("Invalid Param")
	ErrLessThanMin        = errors.New("less than min staking amount for beaconAddress")
	ErrRepeatRegister     = errors.New("repeat register on this address")
	ErrRepeatToAddress    = errors.New("repeat to Address on this register")
	ErrNoRegister         = errors.New("not found the beaconAddress")
	ErrAddressInWhiteList = errors.New("the address in the whitelist")
	ErrNoUserReg          = errors.New("not entangle user in the beaconAddress")
	ErrNoUserAsset        = errors.New("user no entangle asset in the beaconAddress")
	ErrNotEnouthBurn      = errors.New("not enough burn amount in beaconAddress")
	ErrStakingNotEnough   = errors.New("staking not enough")
)

var (
	MinStakingAmountForBeaconAddress  = new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e8))
	MaxWhiteListCount                 = 4
	MAXBASEFEE                        = 100000
	MAXFREEQUOTA                      = 100000 // about 30 days
	LimitRedeemHeightForBeaconAddress = 5000
	MaxCoinBase                       = 5
)

const (
	LhAssetBTC uint32 = 1 << iota
	LhAssetBCH
	LhAssetBSV
	LhAssetLTC
	LhAssetUSDT
	LhAssetDOGE
)

func equalAddress(addr1, addr2 string) bool {
	return bytes.Equal([]byte(addr1), []byte(addr2))
}
func validFee(fee *big.Int) bool {
	if fee.Sign() < 0 || fee.Int64() > int64(MAXBASEFEE) {
		return false
	}
	return true
}
func validKeepTime(kt *big.Int) bool {
	if kt.Sign() < 0 || kt.Int64() > int64(MAXFREEQUOTA) {
		return false
	}
	return true
}

type BurnItem struct {
	Amount      *big.Int `json:"amount"` // czz asset amount
	Height      uint64   `json:"height"`
	RedeemState byte     `json:"redeem_state"` // 0--init, 1 -- redeem done by BeaconAddress payed,2--punishing,3-- punished
}
type BurnInfos struct {
	Items      []*BurnItem
	BurnAmount *big.Int // burned amount for outside asset
}

func newBurnInfos() *BurnInfos {
	return &BurnInfos{
		Items:      make([]*BurnItem, 0, 0),
		BurnAmount: big.NewInt(0),
	}
}

// GetAllAmountByOrigin returns all burned amount asset (czz)
func (b *BurnInfos) GetAllAmountByOrigin() *big.Int {
	amount := big.NewInt(0)
	for _, v := range b.Items {
		amount = amount.Add(amount, v.Amount)
	}
	return amount
}
func (b *BurnInfos) GetAllBurnedAmountByOutside() *big.Int {
	return b.BurnAmount
}
func (b *BurnInfos) getBurnTimeout(height uint64, update bool) []*BurnItem {
	res := make([]*BurnItem, 0, 0)
	for _, v := range b.Items {
		if v.RedeemState == 0 && int64(height-v.Height) > int64(LimitRedeemHeightForBeaconAddress) {
			res = append(res, &BurnItem{
				Amount:      new(big.Int).Set(v.Amount),
				Height:      v.Height,
				RedeemState: v.RedeemState,
			})
			if update {
				v.RedeemState = 2
			}
		}
	}
	return res
}
func (b *BurnInfos) addBurnItem(height uint64, amount *big.Int) {
	item := &BurnItem{
		Amount:      new(big.Int).Set(amount),
		Height:      height,
		RedeemState: 0,
	}
	found := false
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState == 0 && amount.Cmp(v.Amount) == 0 {
			found = true
			break
		}
	}
	if !found {
		b.Items = append(b.Items, item)
		b.BurnAmount = new(big.Int).Add(b.BurnAmount, amount)
	}
}
func (b *BurnInfos) getItem(height uint64, amount *big.Int, state byte) *BurnItem {
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState == state && amount.Cmp(v.Amount) == 0 {
			return v
		}
	}
	return nil
}
func (b *BurnInfos) finishBurn(height uint64, amount *big.Int) {
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState == 3 && amount.Cmp(v.Amount) == 0 {
			v.RedeemState = 1
		}
	}
}

type TimeOutBurnInfo struct {
	Items     []*BurnItem
	AssetType uint32
}

func (t *TimeOutBurnInfo) getAll() *big.Int {
	res := big.NewInt(0)
	for _, v := range t.Items {
		res = res.Add(res, v.Amount)
	}
	return res
}

type TypeTimeOutBurnInfo []*TimeOutBurnInfo
type UserTimeOutBurnInfo map[string]TypeTimeOutBurnInfo

type LHPunishedItem struct {
	All  *big.Int // czz amount(all user burned item in timeout)
	User string
}
type LHPunishedItems []*LHPunishedItem

//////////////////////////////////////////////////////////////////////////////

type WhiteUnit struct {
	AssetType uint32 `json:"asset_type"`
	Pk        []byte `json:"pk"`
}

func (w *WhiteUnit) toAddress() string {
	// pk to czz address
	return ""
}

type BaseAmountUint struct {
	AssetType uint32   `json:"asset_type"`
	Amount    *big.Int `json:"amount"`
}

type EnAssetItem BaseAmountUint
type FreeQuotaItem BaseAmountUint

type BeaconAddressInfo struct {
	ExchangeID      uint64           `json:"exchange_id"`
	Address         string           `json:"address"`
	ToAddress       []byte           `json:"toAddress"`
	StakingAmount   *big.Int         `json:"staking_amount"`  // in
	EntangleAmount  *big.Int         `json:"entangle_amount"` // out,express by czz,all amount of user's entangle
	EnAssets        []*EnAssetItem   `json:"en_assets"`       // out,the extrinsic asset
	Frees           []*FreeQuotaItem `json:"frees"`           // extrinsic asset
	AssetFlag       uint32           `json:"asset_flag"`
	Fee             uint64           `json:"fee"`
	KeepTime        uint64           `json:"keep_time"` // the time as the block count for finally redeem time
	WhiteList       []*WhiteUnit     `json:"white_list"`
	CoinBaseAddress []string         `json:"CoinBaseAddress"`
}

type AddBeaconPledge struct {
	Address       string   `json:"address"`
	ToAddress     []byte   `json:"to_address"`
	StakingAmount *big.Int `json:"staking_amount"`
}

func (lh *BeaconAddressInfo) addEnAsset(atype uint32, amount *big.Int) {
	found := false
	for _, val := range lh.EnAssets {
		if val.AssetType == atype {
			found = true
			val.Amount = new(big.Int).Add(val.Amount, amount)
		}
	}
	if !found {
		lh.EnAssets = append(lh.EnAssets, &EnAssetItem{
			AssetType: atype,
			Amount:    amount,
		})
	}
}
func (lh *BeaconAddressInfo) recordEntangleAmount(amount *big.Int) {
	lh.EntangleAmount = new(big.Int).Add(lh.EntangleAmount, amount)
}
func (lh *BeaconAddressInfo) addFreeQuota(amount *big.Int, atype uint32) {
	for _, v := range lh.Frees {
		if atype == v.AssetType {
			v.Amount = new(big.Int).Add(v.Amount, amount)
		}
	}
}
func (lh *BeaconAddressInfo) useFreeQuota(amount *big.Int, atype uint32) {
	for _, v := range lh.Frees {
		if atype == v.AssetType {
			if v.Amount.Cmp(amount) >= 0 {
				v.Amount = new(big.Int).Sub(v.Amount, amount)
			} else {
				// panic
				v.Amount = big.NewInt(0)
			}
		}
	}
}
func (lh *BeaconAddressInfo) canRedeem(amount *big.Int, atype uint32) bool {
	for _, v := range lh.Frees {
		if atype == v.AssetType {
			if v.Amount.Cmp(amount) >= 0 {
				return true
			} else {
				return false
			}
		}
	}
	return false
}
func (lh *BeaconAddressInfo) updateFreeQuota(res []*BaseAmountUint) error {
	// add free quota for lighthouse
	for _, val := range res {
		if val.Amount != nil && val.Amount.Sign() > 0 {
			item := lh.getFreeQuotaInfo(val.AssetType)
			if item != nil {
				item.Amount = new(big.Int).Add(item.Amount, val.Amount)
			}
		}
	}
	return nil
}
func (lh *BeaconAddressInfo) getFreeQuotaInfo(atype uint32) *FreeQuotaItem {
	for _, v := range lh.Frees {
		if atype == v.AssetType {
			return v
		}
	}
	return nil
}
func (lh *BeaconAddressInfo) addressInWhiteList(addr string) bool {
	for _, val := range lh.WhiteList {
		if equalAddress(addr, val.toAddress()) {
			return true
		}
	}
	return false
}
func (lh *BeaconAddressInfo) updatePunished(amount *big.Int) error {
	var err error
	if amount.Cmp(lh.StakingAmount) > 0 {
		err = ErrStakingNotEnough
		fmt.Println("beacon punished has not enough staking,[current:",
			lh.StakingAmount.String(), "want:", amount.String())
	}
	lh.StakingAmount = new(big.Int).Sub(lh.StakingAmount, amount)
	return err
}

/////////////////////////////////////////////////////////////////
// Address > EntangleEntity
type EntangleEntity struct {
	ExchangeID      uint64     `json:"exchange_id"`
	Address         string     `json:"address"`
	AssetType       uint32     `json:"asset_type"`
	Height          *big.Int   `json:"height"`            // newest height for entangle
	OldHeight       *big.Int   `json:"old_height"`        // oldest height for entangle
	EnOutsideAmount *big.Int   `json:"en_outside_amount"` // out asset
	OriginAmount    *big.Int   `json:"origin_amount"`     // origin asset(czz) by entangle in
	MaxRedeem       *big.Int   `json:"max_redeem"`        // out asset
	BurnAmount      *BurnInfos `json:"burn_amount"`
}
type EntangleEntitys []*EntangleEntity
type UserEntangleInfos map[string]EntangleEntitys
type StoreUserItme struct {
	Addr      string
	UserInfos EntangleEntitys
}
type SortStoreUserItems []*StoreUserItme

func (vs SortStoreUserItems) Len() int {
	return len(vs)
}
func (vs SortStoreUserItems) Less(i, j int) bool {
	return bytes.Compare([]byte(vs[i].Addr), []byte(vs[j].Addr)) == -1
}
func (vs SortStoreUserItems) Swap(i, j int) {
	it := vs[i]
	vs[i] = vs[j]
	vs[j] = it
}
func (uinfos *UserEntangleInfos) toSlice() SortStoreUserItems {
	v1 := make([]*StoreUserItme, 0, 0)
	for k, v := range *uinfos {
		v1 = append(v1, &StoreUserItme{
			Addr:      k,
			UserInfos: v,
		})
	}
	sort.Sort(SortStoreUserItems(v1))
	return SortStoreUserItems(v1)
}
func (es *UserEntangleInfos) fromSlice(vv SortStoreUserItems) {
	userInfos := make(map[string]EntangleEntitys)
	for _, v := range vv {
		userInfos[v.Addr] = v.UserInfos
	}
	*es = UserEntangleInfos(userInfos)
}
func (es *UserEntangleInfos) DecodeRLP(s *rlp.Stream) error {
	type Store1 struct {
		Value SortStoreUserItems
	}
	var eb Store1
	if err := s.Decode(&eb); err != nil {
		return err
	}
	es.fromSlice(eb.Value)
	return nil
}
func (es *UserEntangleInfos) EncodeRLP(w io.Writer) error {
	type Store1 struct {
		Value SortStoreUserItems
	}
	s1 := es.toSlice()
	return rlp.Encode(w, &Store1{
		Value: s1,
	})
}

/////////////////////////////////////////////////////////////////
func (e *EntangleEntity) increaseOriginAmount(amount *big.Int) {
	e.OriginAmount = new(big.Int).Add(e.OriginAmount, amount)
	e.MaxRedeem = new(big.Int).Add(e.MaxRedeem, amount)
}

// the returns maybe negative
func (e *EntangleEntity) GetValidRedeemAmount() *big.Int {
	return new(big.Int).Sub(e.MaxRedeem, e.BurnAmount.GetAllBurnedAmountByOutside())
}
func (e *EntangleEntity) getValidOriginAmount() *big.Int {
	return new(big.Int).Sub(e.OriginAmount, e.BurnAmount.GetAllAmountByOrigin())
}
func (e *EntangleEntity) getValidOutsideAmount() *big.Int {
	return new(big.Int).Sub(e.EnOutsideAmount, e.BurnAmount.GetAllBurnedAmountByOutside())
}

// updateFreeQuotaOfHeight: update user's quota on the asset type by new entangle
func (e *EntangleEntity) updateFreeQuotaOfHeight(height, amount *big.Int) {
	t0, a0, f0 := e.OldHeight, e.getValidOriginAmount(), new(big.Int).Mul(big.NewInt(90), amount)

	t1 := new(big.Int).Add(new(big.Int).Mul(t0, a0), f0)
	t2 := new(big.Int).Add(a0, amount)
	t := new(big.Int).Div(t1, t2)
	interval := big.NewInt(0)
	if t.Sign() > 0 {
		interval = t
	}
	e.OldHeight = new(big.Int).Add(e.OldHeight, interval)
}

// updateFreeQuota returns the outside asset by user who can redeemable
func (e *EntangleEntity) updateFreeQuota(curHeight, limitHeight *big.Int) *big.Int {
	limit := new(big.Int).Sub(curHeight, e.OldHeight)
	if limit.Cmp(limitHeight) < 0 {
		// release user's quota
		e.MaxRedeem = big.NewInt(0)
	}
	return e.getValidOutsideAmount()
}
func (e *EntangleEntity) updateBurnState(state byte, items []*BurnItem) {
	for _, v := range items {
		ii := e.BurnAmount.getItem(v.Height, v.Amount, v.RedeemState)
		if ii != nil {
			ii.RedeemState = state
		}
	}
}

/////////////////////////////////////////////////////////////////
func (ee *EntangleEntitys) getEntityByType(atype uint32) *EntangleEntity {
	for _, v := range *ee {
		if atype == v.AssetType {
			return v
		}
	}
	return nil
}
func (ee *EntangleEntitys) updateFreeQuotaForAllType(curHeight, limit *big.Int) []*BaseAmountUint {
	res := make([]*BaseAmountUint, 0, 0)
	for _, v := range *ee {
		item := &BaseAmountUint{
			AssetType: v.AssetType,
		}
		item.Amount = v.updateFreeQuota(curHeight, limit)
		res = append(res, item)
	}
	return res
}
func (ee *EntangleEntitys) getAllRedeemableAmount() *big.Int {
	res := big.NewInt(0)
	for _, v := range *ee {
		a := v.GetValidRedeemAmount()
		if a != nil {
			res = res.Add(res, a)
		}
	}
	return res
}
func (ee *EntangleEntitys) getBurnTimeout(height uint64, update bool) TypeTimeOutBurnInfo {
	res := make([]*TimeOutBurnInfo, 0, 0)
	for _, entity := range *ee {
		items := entity.BurnAmount.getBurnTimeout(height, update)
		if len(items) > 0 {
			res = append(res, &TimeOutBurnInfo{
				Items:     items,
				AssetType: entity.AssetType,
			})
		}
	}
	return TypeTimeOutBurnInfo(res)
}
func (ee *EntangleEntitys) updateBurnState(state byte, items TypeTimeOutBurnInfo) {
	for _, v := range items {
		entity := ee.getEntityByType(v.AssetType)
		if entity != nil {
			entity.updateBurnState(state, v.Items)
		}
	}
}
func (ee *EntangleEntitys) finishBurnState(height uint64, amount *big.Int, atype uint32) {
	for _, entity := range *ee {
		if entity.AssetType == atype {
			entity.BurnAmount.finishBurn(height, amount)
		}
	}
}

/////////////////////////////////////////////////////////////////
func (u UserEntangleInfos) updateBurnState(state byte, items UserTimeOutBurnInfo) {
	for addr, infos := range items {
		entitys, ok := u[addr]
		if ok {
			entitys.updateBurnState(state, infos)
		}
	}
}
func (uu *TypeTimeOutBurnInfo) getAll() *big.Int {
	res := big.NewInt(0)
	for _, v := range *uu {
		res = res.Add(res, v.getAll())
	}
	return res
}

/////////////////////////////////////////////////////////////////
func ValidAssetType(utype uint32) bool {
	if utype&LhAssetBTC != 0 || utype&LhAssetBCH != 0 || utype&LhAssetBSV != 0 ||
		utype&LhAssetLTC != 0 || utype&LhAssetUSDT != 0 || utype&LhAssetDOGE != 0 {
		return true
	}
	return false
}
func ValidPK(pk []byte) bool {
	if len(pk) != 64 {
		return false
	}
	return true
}
func isValidAsset(atype, assetAll uint32) bool {
	return atype&assetAll != 0
}

func ComputeDiff(params *chaincfg.Params, target *big.Int, address czzutil.Address, eState *EntangleState) *big.Int {
	found_t := 0
	StakingAmount := big.NewInt(0)
	for _, eninfo := range eState.EnInfos {
		for _, eAddr := range eninfo.CoinBaseAddress {
			if address.String() == eAddr {
				StakingAmount = big.NewInt(0).Add(StakingAmount, eninfo.StakingAmount)
				found_t = 1
				break
			}
		}
	}
	if found_t == 1 {
		result := big.NewInt(0).Div(StakingAmount, MinStakingAmountForBeaconAddress)
		result1 := big.NewInt(0).Mul(result, big.NewInt(10))
		target = big.NewInt(0).Mul(target, result1)
	}
	if target.Cmp(params.PowLimit) > 0 {
		target.Set(params.PowLimit)
	}
	return target
}
