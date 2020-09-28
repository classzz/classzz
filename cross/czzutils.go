package cross

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/rlp"
	"github.com/classzz/czzutil"
)

var (
	ErrInvalidParam       = errors.New("Invalid Param")
	ErrLessThanMin        = errors.New("less than min staking amount for beaconAddress")
	ErrRepeatRegister     = errors.New("repeat register on this address")
	ErrNotRepeatRegister  = errors.New("repeat not register on this address")
	ErrRepeatToAddress    = errors.New("repeat to Address on this register")
	ErrNoRegister         = errors.New("not found the beaconAddress")
	ErrAddressInWhiteList = errors.New("the address in the whitelist")
	ErrNoUserReg          = errors.New("not entangle user in the beaconAddress")
	ErrNoUserAsset        = errors.New("user no entangle asset in the beaconAddress")
	ErrNotEnouthBurn      = errors.New("not enough burn amount in beaconAddress")
	ErrNotMatchUser       = errors.New("cann't find user address")
	ErrBurnProof          = errors.New("burn proof info not match")
	ErrWhiteListProof     = errors.New("white list proof not match")
	ErrStakingNotEnough   = errors.New("staking not enough")
	ErrRepeatProof        = errors.New("repeat proof")
	ErrNotEnouthEntangle  = errors.New("not enough entangle amount in beaconAddress")
	ErrNotKindOfAsset     = errors.New("no support the kind of asset")
)

var (
	MinStakingAmountForBeaconAddress  = new(big.Int).Mul(big.NewInt(100), big.NewInt(1e8))
	MaxWhiteListCount                 = 4
	MAXBASEFEE                        = 100000
	MAXFREEQUOTA                      = 100000 // about 30 days
	LimitRedeemHeightForBeaconAddress = 2000
	MaxCoinBase                       = 4
	MaxCoinType                       = 6
	ChechWhiteListProof               = true
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

func ValidAssetFlag(utype uint32) bool {
	if utype&LhAssetBTC != 0 || utype&LhAssetBCH != 0 || utype&LhAssetBSV != 0 ||
		utype&LhAssetLTC != 0 || utype&LhAssetUSDT != 0 || utype&LhAssetDOGE != 0 {
		return true
	}
	return false
}

func ValidAssetType(utype1 uint8) bool {
	utype := uint32(utype1)
	if utype&LhAssetBTC != 0 || utype&LhAssetBCH != 0 || utype&LhAssetBSV != 0 ||
		utype&LhAssetLTC != 0 || utype&LhAssetUSDT != 0 || utype&LhAssetDOGE != 0 {
		return true
	}
	return false
}
func ValidPK(pk []byte) bool {
	if len(pk) != 65 {
		return false
	}
	return true
}

func ExpandedTxTypeToAssetType(atype uint8) uint32 {
	switch atype {
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
	case ExpandedTxEntangle_Usdt:
		return LhAssetUSDT
	}
	return 0
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

//////////////////////////////////////////////////////////////////////////////

type WhiteUnit struct {
	AssetType uint8  `json:"asset_type"`
	Pk        []byte `json:"pk"`
}

func (w *WhiteUnit) toAddress() string {
	// pk to czz address
	return ""
}

type BaseAmountUint struct {
	AssetType uint8    `json:"asset_type"`
	Amount    *big.Int `json:"amount"`
}

type EnAssetItem BaseAmountUint
type FreeQuotaItem BaseAmountUint

type BeaconAddressInfo struct {
	BeaconID        uint64           `json:"beacon_id"`
	Address         string           `json:"address"`
	PubKey          []byte           `json:"pub_key"`
	ToAddress       []byte           `json:"toAddress"`
	StakingAmount   *big.Int         `json:"staking_amount"`  // in
	EntangleAmount  *big.Int         `json:"entangle_amount"` // out,express by czz,all amount of user's entangle
	EnAssets        []*EnAssetItem   `json:"en_assets"`       // out,the extrinsic asset
	Frees           []*FreeQuotaItem `json:"frees"`           // extrinsic asset
	AssetFlag       uint32           `json:"asset_flag"`
	Fee             uint64           `json:"fee"`
	KeepBlock       uint64           `json:"keep_block"` // the time as the block count for finally redeem time
	WhiteList       []*WhiteUnit     `json:"white_list"`
	CoinBaseAddress []string         `json:"coinbase_address"`
}

type BeaconAddressInfo2 struct {
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

type UpdateBeaconCoinbase struct {
	Address         string   `json:"address"`
	CoinBaseAddress []string `json:"coinbase_address"`
}

type UpdateBeaconFreeQuota struct {
	Address   string   `json:"address"`
	FreeQuota []uint64 `json:"free_quota"`
}

func (lh *BeaconAddressInfo) addEnAsset(atype uint8, amount *big.Int) {
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
func (lh *BeaconAddressInfo) reduceEntangleAmount(amount *big.Int) {
	lh.EntangleAmount = new(big.Int).Sub(lh.EntangleAmount, amount)
}
func (lh *BeaconAddressInfo) addFreeQuota(amount *big.Int, atype uint8) {
	for _, v := range lh.Frees {
		if atype == v.AssetType {
			v.Amount = new(big.Int).Add(v.Amount, amount)
		}
	}
}
func (lh *BeaconAddressInfo) useFreeQuota(amount *big.Int, atype uint8) {
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
func (lh *BeaconAddressInfo) canRedeem(amount *big.Int, atype uint8) bool {
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
func (lh *BeaconAddressInfo) getFreeQuotaInfo(atype uint8) *FreeQuotaItem {
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
func (lh *BeaconAddressInfo) getToAddress() []byte {
	return lh.ToAddress
}
func (lh *BeaconAddressInfo) getOutSideAsset(atype uint8) *big.Int {
	all := big.NewInt(0)
	for _, v := range lh.EnAssets {
		if v.AssetType == atype {
			all = new(big.Int).Add(all, v.Amount)
		}
	}
	return all
}
func (lh *BeaconAddressInfo) getWhiteList() []*WhiteUnit {
	return lh.WhiteList
}
func (lh *BeaconAddressInfo) EnoughToEntangle(enAmount *big.Int) error {
	tmp := new(big.Int).Sub(lh.StakingAmount, lh.EntangleAmount)
	if tmp.Sign() <= 0 {
		return ErrNotEnouthEntangle
	}
	if tmp.Cmp(new(big.Int).Add(enAmount, MinStakingAmountForBeaconAddress)) < 0 {
		return ErrNotEnouthEntangle
	}
	return nil
}

///////////////////////////////////////////////////////////////////
//// Address > EntangleEntity
type ExChangeEntity struct {
	AssetType       uint8
	EnOutsideAmount *big.Int `json:"en_outside_amount"` // out asset
}

func newExChangeEntitys() []*ExChangeEntity {

	exChangeEntitys := make([]*ExChangeEntity, 0, 0)
	exChangeEntitys = append(exChangeEntitys, &ExChangeEntity{
		AssetType:       ExpandedTxEntangle_Doge,
		EnOutsideAmount: big.NewInt(0),
	})

	exChangeEntitys = append(exChangeEntitys, &ExChangeEntity{
		AssetType:       ExpandedTxEntangle_Bsv,
		EnOutsideAmount: big.NewInt(0),
	})

	exChangeEntitys = append(exChangeEntitys, &ExChangeEntity{
		AssetType:       ExpandedTxEntangle_Bch,
		EnOutsideAmount: big.NewInt(0),
	})

	exChangeEntitys = append(exChangeEntitys, &ExChangeEntity{
		AssetType:       ExpandedTxEntangle_Btc,
		EnOutsideAmount: big.NewInt(0),
	})

	exChangeEntitys = append(exChangeEntitys, &ExChangeEntity{
		AssetType:       ExpandedTxEntangle_Ltc,
		EnOutsideAmount: big.NewInt(0),
	})
	exChangeEntitys = append(exChangeEntitys, &ExChangeEntity{
		AssetType:       ExpandedTxEntangle_Usdt,
		EnOutsideAmount: big.NewInt(0),
	})
	return exChangeEntitys
}

type UserExChangeInfo struct {
	BeaconID        uint64            `json:"beacon_id"`     // beaconID id
	Address         string            `json:"address"`       // 兑换的地址
	Height          *big.Int          `json:"height"`        // newest height for entangle
	OldHeight       *big.Int          `json:"old_height"`    // oldest height for entangle
	OriginAmount    *big.Int          `json:"origin_amount"` // origin asset(czz) by entangle in
	MaxRedeem       *big.Int          `json:"max_redeem"`    // redeem asset bt czz
	BurnAmounts     []*BurnInfo       `json:"burn_amount"`
	ExChangeEntitys []*ExChangeEntity `json:"ex_change_entitys"`
}

func NewUserExChangeInfo() *UserExChangeInfo {
	uci := &UserExChangeInfo{
		BeaconID:        0,
		Height:          big.NewInt(0),
		OldHeight:       big.NewInt(0),
		OriginAmount:    big.NewInt(0),
		MaxRedeem:       big.NewInt(0),
		BurnAmounts:     newBurnInfos(),
		ExChangeEntitys: newExChangeEntitys(),
	}
	return uci
}

type UserExChangeInfos map[string]*UserExChangeInfo

func NewUserExChangeInfos() UserExChangeInfos {
	return UserExChangeInfos(make(map[string]*UserExChangeInfo))
}

type StoreUserItme struct {
	Addr     string
	UserInfo *UserExChangeInfo
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

func (uinfos *UserExChangeInfos) toSlice() SortStoreUserItems {
	v1 := make([]*StoreUserItme, 0, 0)
	for k, v := range *uinfos {
		v1 = append(v1, &StoreUserItme{
			Addr:     k,
			UserInfo: v,
		})
	}
	sort.Sort(SortStoreUserItems(v1))
	return SortStoreUserItems(v1)
}

func (es *UserExChangeInfos) fromSlice(vv SortStoreUserItems) {
	userInfos := make(map[string]*UserExChangeInfo)
	for _, v := range vv {
		userInfos[v.Addr] = v.UserInfo
	}
	*es = userInfos
}

func (es *UserExChangeInfos) DecodeRLP(s *rlp.Stream) error {
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

func (es *UserExChangeInfos) EncodeRLP(w io.Writer) error {
	type Store1 struct {
		Value SortStoreUserItems
	}
	s1 := es.toSlice()
	return rlp.Encode(w, &Store1{
		Value: s1,
	})
}

func (es *UserExChangeInfos) GetRedeemableAmountAll() *big.Int {
	allAmount := big.NewInt(0)
	for _, v := range *es {
		allAmount = big.NewInt(0).Add(allAmount, v.MaxRedeem)
	}
	return allAmount
}

func (es *UserExChangeInfos) GetBurnAmount() *big.Int {
	allAmount := big.NewInt(0)
	for _, v := range *es {
		for _, burn := range v.BurnAmounts {
			for _, item := range burn.Items {
				if item.RedeemState == 0 {
					allAmount = big.NewInt(0).Add(allAmount, item.Amount)
				}
			}
		}
	}
	return allAmount
}

/////////////////////////////////////////////////////////////////
func (e *UserExChangeInfo) increaseOriginAmount(amount, height *big.Int) {
	e.OriginAmount = new(big.Int).Add(e.OriginAmount, amount)
	if e.MaxRedeem.Sign() == 0 {
		e.OldHeight = new(big.Int).Set(height)
	}
	e.MaxRedeem = new(big.Int).Add(e.MaxRedeem, amount)
}

// the returns maybe negative
func (e *UserExChangeInfo) getRedeemableAmount() *big.Int {
	allAmount := big.NewInt(0)
	for _, v := range e.BurnAmounts {
		allAmount = big.NewInt(0).Add(allAmount, v.GetAllAmountByOrigin())
	}
	return new(big.Int).Sub(e.MaxRedeem, allAmount)
}

func (e *UserExChangeInfo) getValidOriginAmount() *big.Int {
	allAmount := big.NewInt(0)
	for _, v := range e.BurnAmounts {
		allAmount = big.NewInt(0).Add(allAmount, v.GetAllAmountByOrigin())
	}
	return new(big.Int).Sub(e.OriginAmount, allAmount)
}

//func (e *ExChangeEntity) getValidOutsideAmount() *big.Int {
//	return new(big.Int).Sub(e.EnOutsideAmount, e.BurnAmount.GetAllBurnedAmountByOutside())
//}

// updateFreeQuotaOfHeight: update user's quota on the asset type by new entangle
func (e *UserExChangeInfo) updateFreeQuotaOfHeight(keepBlock, amount *big.Int) {
	t0, a0, f0 := e.OldHeight, e.getValidOriginAmount(), new(big.Int).Mul(keepBlock, amount)

	t1 := new(big.Int).Add(new(big.Int).Mul(t0, a0), f0)
	t2 := new(big.Int).Add(a0, amount)
	t := new(big.Int).Div(t1, t2)
	interval := big.NewInt(0)
	if t.Sign() > 0 {
		interval = t
	}
	e.OldHeight = new(big.Int).Add(e.OldHeight, interval)
}

// updateFreeQuota returns the czz asset by user who can redeemable
func (e *UserExChangeInfo) updateFreeQuota(curHeight, limitHeight *big.Int) *big.Int {

	if curHeight.Cmp(e.OldHeight) > 0 && e.MaxRedeem.Cmp(big.NewInt(0)) > 0 {
		// release user's quota
		left := e.getRedeemableAmount()
		e.MaxRedeem = big.NewInt(0)
		return left
	}
	return big.NewInt(0)
}

/////////////////////////////////////////////////////////////////
func (ee *UserExChangeInfo) getBurnByType(atype uint8) *BurnInfo {
	for _, v := range ee.BurnAmounts {
		if atype == v.AssetType {
			return v
		}
	}
	return nil
}

func (ee *UserExChangeInfo) getBurnTimeout(height uint64, update bool) TypeTimeOutBurnInfos {
	res := make([]*TimeOutBurnInfo, 0, 0)
	AmountSum := big.NewInt(0)

	for _, entity := range ee.BurnAmounts {
		items, Amount := entity.getBurnTimeout(height, update)
		if len(items) > 0 {
			res = append(res, &TimeOutBurnInfo{
				Items:     items,
				AssetType: entity.AssetType,
			})
		}
		AmountSum = big.NewInt(0).Add(AmountSum, Amount)
	}

	ttobi := TypeTimeOutBurnInfos{
		TypeTimeOutBurnInfo: res,
		AmountSum:           AmountSum,
	}

	return ttobi
}

func (ee *UserExChangeInfo) updateBurnState(state uint8, items TypeTimeOutBurnInfos) {
	for _, v := range items.TypeTimeOutBurnInfo {
		burn := ee.getBurnByType(v.AssetType)
		if burn != nil {
			burn.updateBurnState(state, v.Items)
		}
	}
}

func (ee *UserExChangeInfo) updateBurnState2(height uint64, amount *big.Int,
	atype uint8, proof *BurnProofItem) {
	for _, burn := range ee.BurnAmounts {
		if burn.AssetType == atype {
			burn.updateBurn(height, amount, proof)
		}
	}
}

func (ee *UserExChangeInfo) finishBurnState(height uint64, amount *big.Int,
	atype uint8, proof *BurnProofItem) {
	for _, burn := range ee.BurnAmounts {
		if burn.AssetType == atype {
			burn.finishBurn(height, amount, proof)
		}
	}
}
func (ee *UserExChangeInfo) getBurn() {

}

func (ee *UserExChangeInfo) verifyBurnProof(info *BurnProofInfo, outHeight, curHeight uint64) (*BurnItem, error) {
	for _, burn := range ee.BurnAmounts {
		if burn.AssetType == info.AssetType {
			return burn.verifyProof(info, outHeight, curHeight)
		}
	}
	return nil, ErrNoUserAsset
}

func (ee *UserExChangeInfo) closeProofForPunished(item *BurnItem, atype uint8) error {
	for _, burn := range ee.BurnAmounts {
		if burn.AssetType == atype {
			return burn.closeProofForPunished(item)
		}
	}
	return ErrNoUserAsset
}

/////////////////////////////////////////////////////////////////
func (u UserExChangeInfos) updateBurnState(state uint8, items UserTimeOutBurnInfo) {
	for addr, infos := range items {
		entitys, ok := u[addr]
		if ok {
			entitys.updateBurnState(state, infos)
		}
	}
}

/////////////////////////////////////////////////////////////////
type BurnItem struct {
	Amount      *big.Int       `json:"amount"`      // czz asset amount
	FeeAmount   *big.Int       `json:"fee_amount"`  // czz asset fee amount
	RAmount     *big.Int       `json:"ramount"`     // outside asset amount
	FeeRAmount  *big.Int       `json:"fee_ramount"` // outside asset fee amount
	Height      uint64         `json:"height"`
	ToAddress   string         `json:"to_address"`
	RedeemState byte           `json:"redeem_state"` // 0--init, 1 -- redeem done by BeaconAddress payed,2--punishing,3-- punished
	Proof       *BurnProofItem `json:"proof"`        // the tx of outside
}

func (b *BurnItem) equal(o *BurnItem) bool {
	return b.Height == o.Height && b.Amount.Cmp(o.Amount) == 0 &&
		b.RAmount.Cmp(o.Amount) == 0 && b.FeeRAmount.Cmp(o.FeeRAmount) == 0 &&
		b.FeeAmount.Cmp(o.FeeAmount) == 0
}

func (b *BurnItem) clone() *BurnItem {
	return &BurnItem{
		Amount:     new(big.Int).Set(b.Amount),
		RAmount:    new(big.Int).Set(b.RAmount),
		FeeAmount:  new(big.Int).Set(b.FeeAmount),
		FeeRAmount: new(big.Int).Set(b.FeeRAmount),
		Height:     b.Height,
		Proof: &BurnProofItem{
			Height: b.Proof.Height,
			TxHash: b.Proof.TxHash,
		},
		RedeemState: b.RedeemState,
	}
}

type BurnInfo struct {
	AssetType  uint8
	RAllAmount *big.Int // redeem asset for outside asset by burned czz
	BAllAmount *big.Int // all burned asset on czz by the account
	Items      []*BurnItem
}

type extBurnInfos struct {
	AssetType  uint8
	Items      []*BurnItem
	RAllAmount *big.Int // redeem asset for outside asset by burned czz
	BAllAmount *big.Int // all burned asset on czz by the account
}

// DecodeRLP decodes the
func (b *BurnInfo) DecodeRLP(s *rlp.Stream) error {

	var eb extBurnInfos
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.AssetType, b.Items, b.RAllAmount, b.BAllAmount = eb.AssetType, eb.Items, eb.RAllAmount, eb.BAllAmount
	return nil
}

// EncodeRLP serializes b into the  RLP block format.
func (b *BurnInfo) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extBurnInfos{
		AssetType:  b.AssetType,
		Items:      b.Items,
		RAllAmount: b.RAllAmount,
		BAllAmount: b.BAllAmount,
	})
}

func newBurnInfos() []*BurnInfo {

	burnInfos := make([]*BurnInfo, 0, 0)
	burnInfos = append(burnInfos, &BurnInfo{
		AssetType:  ExpandedTxEntangle_Doge,
		RAllAmount: big.NewInt(0),
		BAllAmount: big.NewInt(0),
		Items:      make([]*BurnItem, 0, 0),
	})

	burnInfos = append(burnInfos, &BurnInfo{
		AssetType:  ExpandedTxEntangle_Bsv,
		RAllAmount: big.NewInt(0),
		BAllAmount: big.NewInt(0),
		Items:      make([]*BurnItem, 0, 0),
	})

	burnInfos = append(burnInfos, &BurnInfo{
		AssetType:  ExpandedTxEntangle_Bch,
		RAllAmount: big.NewInt(0),
		BAllAmount: big.NewInt(0),
		Items:      make([]*BurnItem, 0, 0),
	})

	burnInfos = append(burnInfos, &BurnInfo{
		AssetType:  ExpandedTxEntangle_Btc,
		RAllAmount: big.NewInt(0),
		BAllAmount: big.NewInt(0),
		Items:      make([]*BurnItem, 0, 0),
	})

	burnInfos = append(burnInfos, &BurnInfo{
		AssetType:  ExpandedTxEntangle_Ltc,
		RAllAmount: big.NewInt(0),
		BAllAmount: big.NewInt(0),
		Items:      make([]*BurnItem, 0, 0),
	})
	burnInfos = append(burnInfos, &BurnInfo{
		AssetType:  ExpandedTxEntangle_Usdt,
		RAllAmount: big.NewInt(0),
		BAllAmount: big.NewInt(0),
		Items:      make([]*BurnItem, 0, 0),
	})
	return burnInfos
}

// GetAllAmountByOrigin returns all burned amount asset (czz)
func (b *BurnInfo) GetAllAmountByOrigin() *big.Int {
	return new(big.Int).Set(b.BAllAmount)
}

func (b *BurnInfo) GetAllBurnedAmountByOutside() *big.Int {
	return new(big.Int).Set(b.RAllAmount)
}

func (b *BurnInfo) getBurnTimeout(height uint64, update bool) ([]*BurnItem, *big.Int) {
	res := make([]*BurnItem, 0, 0)
	AmountSum := big.NewInt(0)
	for _, v := range b.Items {
		if v.RedeemState == 0 && int64(height-v.Height) > int64(LimitRedeemHeightForBeaconAddress) {
			res = append(res, &BurnItem{
				Amount:      new(big.Int).Set(v.Amount),
				RAmount:     new(big.Int).Set(v.RAmount),
				FeeAmount:   new(big.Int).Set(v.FeeAmount),
				FeeRAmount:  new(big.Int).Set(v.FeeRAmount),
				Height:      v.Height,
				RedeemState: v.RedeemState,
			})
			AmountSum = big.NewInt(0).Add(AmountSum, v.Amount)
			if update {
				v.RedeemState = 2
			}
		}
	}
	return res, AmountSum
}

func (b *BurnInfo) addBurnItem(toaddress string, height uint64, amount, fee, outFee, outAmount *big.Int) {
	item := &BurnItem{
		Amount:      new(big.Int).Set(amount),
		RAmount:     new(big.Int).Set(outAmount),
		FeeAmount:   new(big.Int).Set(fee),
		FeeRAmount:  new(big.Int).Set(outFee),
		ToAddress:   toaddress,
		Height:      height,
		Proof:       &BurnProofItem{},
		RedeemState: 0,
	}
	found := false
	for _, v := range b.Items {
		if v.RedeemState == 0 && v.equal(item) {
			found = true
			break
		}
	}
	if !found {
		b.Items = append(b.Items, item)
		b.RAllAmount = new(big.Int).Add(b.RAllAmount, outAmount)
		b.BAllAmount = new(big.Int).Add(b.BAllAmount, amount)
	}
}

func (b *BurnInfo) getItem(height uint64, amount *big.Int, state byte) *BurnItem {
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState == state && amount.Cmp(v.Amount) == 0 {
			return v
		}
	}
	return nil
}

func (b *BurnInfo) getBurnsItemByHeight(height uint64, state byte) []*BurnItem {
	items := []*BurnItem{}
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState == state && v.RAmount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) >= 0 {
			items = append(items, v)
		}
	}
	return items
}

func (b *BurnInfo) updateBurnState(state uint8, Items []*BurnItem) {
	for _, v := range Items {
		v.RedeemState = state
	}
}

func (b *BurnInfo) updateBurn(height uint64, amount *big.Int, proof *BurnProofItem) {
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState != 2 &&
			amount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) < 0 {
			v.RedeemState, v.Proof = 2, proof
		}
	}
}

func (b *BurnInfo) finishBurn(height uint64, amount *big.Int, proof *BurnProofItem) {
	for _, v := range b.Items {
		//&& v.RedeemState != 1 && sendAmount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) >= 0
		if v.Height == height && v.RedeemState != 1 && amount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) >= 0 {
			v.RedeemState, v.Proof = 1, proof
		}
	}
}

func (b *BurnInfo) recoverOutAmountForPunished(amount *big.Int) {
	b.RAllAmount = new(big.Int).Sub(b.RAllAmount, amount)
}

func (b *BurnInfo) EarliestHeightAndUsedTx(tx string) (uint64, bool) {
	height, used := uint64(0), false
	for _, v := range b.Items {
		if v.Proof.TxHash != "" {
			if height == 0 || height < v.Proof.Height {
				height = v.Proof.Height
			}
			if v.Proof.TxHash == tx {
				used = true
			}
		}
	}
	return height, used
}

func (b *BurnInfo) verifyProof(info *BurnProofInfo, outHeight, curHeight uint64) (*BurnItem, error) {
	eHeight, used := b.EarliestHeightAndUsedTx(info.TxHash)

	if outHeight >= eHeight && !used {
		if items := b.getBurnsItemByHeight(info.Height, byte(0)); len(items) > 0 {
			for _, v := range items {
				if info.Amount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) >= 0 && v.Proof.TxHash == "" {
					return v.clone(), nil
				}
			}
		}
	}

	return nil, ErrBurnProof
}

func (b *BurnInfo) closeProofForPunished(item *BurnItem) error {
	if v := b.getItem(item.Height, item.Amount, item.RedeemState); v != nil {
		v.RedeemState = 2
	}
	return nil
}

type TimeOutBurnInfo struct {
	Items     []*BurnItem
	AssetType uint8
}

func (t *TimeOutBurnInfo) getAll() *big.Int {
	res := big.NewInt(0)
	for _, v := range t.Items {
		res = res.Add(res, v.Amount)
	}
	return res
}

//type TypeTimeOutBurnInfo []*TimeOutBurnInfo

type TypeTimeOutBurnInfos struct {
	TypeTimeOutBurnInfo []*TimeOutBurnInfo
	AmountSum           *big.Int
}

type UserTimeOutBurnInfo map[string]TypeTimeOutBurnInfos

func (uu *TypeTimeOutBurnInfos) getAll() *big.Int {
	res := big.NewInt(0)
	for _, v := range uu.TypeTimeOutBurnInfo {
		res = res.Add(res, v.getAll())
	}
	return res
}

type BurnProofItem struct {
	Height uint64
	TxHash string
}

type extBurnProofItem struct {
	Height uint64
	TxHash string
}

func (b *BurnProofItem) DecodeRLP(s *rlp.Stream) error {
	var eb extBurnProofItem
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.Height, b.TxHash = eb.Height, eb.TxHash
	return nil
}

func (b *BurnProofItem) EncodeRLP(w io.Writer) error {
	var eb extBurnProofItem
	if b == nil {
		eb = extBurnProofItem{
			Height: b.Height,
			TxHash: b.TxHash,
		}
	}
	return rlp.Encode(w, eb)
}

type BurnProofInfo struct {
	BeaconID  uint64   // the BeaconID for beaconAddress of user burn's asset
	Height    uint64   // the height include the tx of user burn's asset
	Amount    *big.Int // the amount of burned asset (czz)
	Address   string
	AssetType uint8
	TxHash    string // the tx hash of outside
	OutIndex  uint64
}

type WhiteListProof struct {
	BeaconID  uint64 // the BeaconID for beaconAddress
	AssetType uint8
	Height    uint64 // the height of outside chain
	TxHash    string
	InIndex   uint64
	OutIndex  uint64
	Amount    *big.Int // the amount of outside chain
}

func (wl *WhiteListProof) Clone() *WhiteListProof {
	return &WhiteListProof{
		BeaconID:  wl.BeaconID,
		Height:    wl.Height,
		Amount:    new(big.Int).Set(wl.Amount),
		AssetType: wl.AssetType,
	}
}

type LHPunishedItem struct {
	All  *big.Int // czz amount(all user burned item in timeout)
	User string
}
type LHPunishedItems []*LHPunishedItem

//////////////////////////////////////////////////////////////////////////////
type ResItem struct {
	Index  int
	Amount *big.Int
}
type ResCoinBasePos []*ResItem

func NewResCoinBasePos() ResCoinBasePos {
	return []*ResItem{}
}
func (p *ResCoinBasePos) Put(i int, amount *big.Int) {
	*p = append(*p, &ResItem{
		Index:  i,
		Amount: new(big.Int).Set(amount),
	})
}
func (p ResCoinBasePos) IsIn(i int) bool {
	for _, v := range p {
		if v.Index == i {
			return true
		}
	}
	return false
}
func (p ResCoinBasePos) GetInCount() int {
	return len(p)
}
func (p ResCoinBasePos) GetOutCount() int {
	return len(p)
}

//////////////////////////////////////////////////////////////////////////////
