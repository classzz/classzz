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
)

var (
	MinStakingAmountForBeaconAddress  = new(big.Int).Mul(big.NewInt(100), big.NewInt(1e8))
	MaxWhiteListCount                 = 4
	MAXBASEFEE                        = 100000
	MAXFREEQUOTA                      = 100000 // about 30 days
	LimitRedeemHeightForBeaconAddress = 5000
	MaxCoinBase                       = 4
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
	if len(pk) != 64 {
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
	ExchangeID      uint64           `json:"exchange_id"`
	Address         string           `json:"address"`
	PubKey          []byte           `json:"pub_key"`
	ToAddress       []byte           `json:"toAddress"`
	StakingAmount   *big.Int         `json:"staking_amount"`  // in
	EntangleAmount  *big.Int         `json:"entangle_amount"` // out,express by czz,all amount of user's entangle
	EnAssets        []*EnAssetItem   `json:"en_assets"`       // out,the extrinsic asset
	Frees           []*FreeQuotaItem `json:"frees"`           // extrinsic asset
	AssetFlag       uint32           `json:"asset_flag"`
	Fee             uint64           `json:"fee"`
	KeepTime        uint64           `json:"keep_time"` // the time as the block count for finally redeem time
	WhiteList       []*WhiteUnit     `json:"white_list"`
	CoinBaseAddress []string         `json:"coinbase_address"`
}

type AddBeaconPledge struct {
	Address       string   `json:"address"`
	ToAddress     []byte   `json:"to_address"`
	StakingAmount *big.Int `json:"staking_amount"`
}

type AddBeaconCoinbase struct {
	Address         string   `json:"address"`
	ToAddress       []byte   `json:"to_address"`
	CoinBaseAddress []string `json:"coinbase_address"`
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

/////////////////////////////////////////////////////////////////
// Address > EntangleEntity
type EntangleEntity struct {
	ExchangeID      uint64     `json:"exchange_id"`
	Address         string     `json:"address"`
	AssetType       uint8      `json:"asset_type"`
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
func (ee *EntangleEntitys) getEntityByType(atype uint8) *EntangleEntity {
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

func (ee *EntangleEntitys) updateBurnState2(height uint64, amount *big.Int,
	atype uint8, proof *BurnProofItem) {
	for _, entity := range *ee {
		if entity.AssetType == atype {
			entity.BurnAmount.updateBurn(height, amount, proof)
		}
	}
}

func (ee *EntangleEntitys) finishBurnState(height uint64, amount *big.Int,
	atype uint8, proof *BurnProofItem) {
	for _, entity := range *ee {
		if entity.AssetType == atype {
			entity.BurnAmount.finishBurn(height, amount, proof)
		}
	}
}

func (ee *EntangleEntitys) verifyBurnProof(info *BurnProofInfo, outHeight, curHeight uint64) (*BurnItem, error) {
	for _, entity := range *ee {
		if entity.AssetType == info.Atype {
			return entity.BurnAmount.verifyProof(info, outHeight, curHeight)
		}
	}
	return nil, ErrNoUserAsset
}
func (ee *EntangleEntitys) closeProofForPunished(item *BurnItem, atype uint8) error {
	for _, entity := range *ee {
		if entity.AssetType == atype {
			return entity.BurnAmount.closeProofForPunished(item)
		}
	}
	return ErrNoUserAsset
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

/////////////////////////////////////////////////////////////////
type BurnItem struct {
	Amount      *big.Int       `json:"amount"`      // czz asset amount
	FeeAmount   *big.Int       `json:"fee_amount"`  // czz asset fee amount
	RAmount     *big.Int       `json:"ramount"`     // outside asset amount
	FeeRAmount  *big.Int       `json:"fee_ramount"` // outside asset fee amount
	Height      uint64         `json:"height"`
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

type BurnInfos struct {
	Items      []*BurnItem
	RAllAmount *big.Int // redeem asset for outside asset by burned czz
	BAllAmount *big.Int // all burned asset on czz by the account
}

type extBurnInfos struct {
	Items      []*BurnItem
	RAllAmount *big.Int // redeem asset for outside asset by burned czz
	BAllAmount *big.Int // all burned asset on czz by the account
}

// DecodeRLP decodes the truechain
func (b *BurnInfos) DecodeRLP(s *rlp.Stream) error {

	var eb extBurnInfos
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.Items, b.RAllAmount, b.BAllAmount = eb.Items, eb.RAllAmount, eb.BAllAmount
	return nil
}

// EncodeRLP serializes b into the truechain RLP block format.
func (b *BurnInfos) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extBurnInfos{
		Items:      b.Items,
		RAllAmount: b.RAllAmount,
		BAllAmount: b.BAllAmount,
	})
}

func newBurnInfos() *BurnInfos {
	return &BurnInfos{
		Items:      make([]*BurnItem, 0, 0),
		RAllAmount: big.NewInt(0),
		BAllAmount: big.NewInt(0),
	}
}

// GetAllAmountByOrigin returns all burned amount asset (czz)
func (b *BurnInfos) GetAllAmountByOrigin() *big.Int {
	return new(big.Int).Set(b.BAllAmount)
}
func (b *BurnInfos) GetAllBurnedAmountByOutside() *big.Int {
	return new(big.Int).Set(b.RAllAmount)
}
func (b *BurnInfos) getBurnTimeout(height uint64, update bool) []*BurnItem {
	res := make([]*BurnItem, 0, 0)
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
			if update {
				v.RedeemState = 2
			}
		}
	}
	return res
}
func (b *BurnInfos) addBurnItem(height uint64, amount, fee, outFee, outAmount *big.Int) {
	item := &BurnItem{
		Amount:      new(big.Int).Set(amount),
		RAmount:     new(big.Int).Set(outAmount),
		FeeAmount:   new(big.Int).Set(fee),
		FeeRAmount:  new(big.Int).Set(outFee),
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

func (b *BurnInfos) getItem(height uint64, amount *big.Int, state byte) *BurnItem {
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState == state && amount.Cmp(v.Amount) == 0 {
			return v
		}
	}
	return nil
}

func (b *BurnInfos) getBurnsItemByHeight(height uint64, state byte) []*BurnItem {
	items := []*BurnItem{}
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState == state {
			items = append(items, v)
		}
	}
	return items
}

func (b *BurnInfos) updateBurn(height uint64, amount *big.Int, proof *BurnProofItem) {
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState != 2 &&
			amount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) < 0 {
			v.RedeemState, v.Proof = 2, proof
		}
	}
}

func (b *BurnInfos) finishBurn(height uint64, amount *big.Int, proof *BurnProofItem) {
	for _, v := range b.Items {
		if v.Height == height && v.RedeemState != 1 &&
			amount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) >= 0 {
			v.RedeemState, v.Proof = 1, proof
		}
	}
}
func (b *BurnInfos) recoverOutAmountForPunished(amount *big.Int) {
	b.RAllAmount = new(big.Int).Sub(b.RAllAmount, amount)
}
func (b *BurnInfos) EarliestHeightAndUsedTx(tx string) (uint64, bool) {
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
func (b *BurnInfos) verifyProof(info *BurnProofInfo, outHeight, curHeight uint64) (*BurnItem, error) {
	eHeight, used := b.EarliestHeightAndUsedTx(info.TxHash)
	if info.IsBeacon {
		if outHeight >= eHeight && !used {
			if items := b.getBurnsItemByHeight(info.Height, byte(0)); len(items) > 0 {
				for _, v := range items {
					if info.Amount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) >= 0 && v.Proof.TxHash == "" {
						return v.clone(), nil
					}
				}
			}
		}
	} else {
		if items := b.getBurnsItemByHeight(info.Height, byte(0)); len(items) > 0 {
			for _, v := range items {
				if info.Amount.Cmp(new(big.Int).Sub(v.RAmount, v.FeeRAmount)) < 0 || int64(curHeight-v.Height) > int64(LimitRedeemHeightForBeaconAddress) {
					// deficiency or timeout
					return v.clone(), nil
				}
			}
		}
	}

	return nil, ErrBurnProof
}
func (b *BurnInfos) closeProofForPunished(item *BurnItem) error {
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

type TypeTimeOutBurnInfo []*TimeOutBurnInfo
type UserTimeOutBurnInfo map[string]TypeTimeOutBurnInfo

func (uu *TypeTimeOutBurnInfo) getAll() *big.Int {
	res := big.NewInt(0)
	for _, v := range *uu {
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
	LightID  uint64   // the lightid for beaconAddress of user burn's asset
	Height   uint64   // the height include the tx of user burn's asset
	Amount   *big.Int // the amount of burned asset (czz)
	Address  string
	Atype    uint8
	TxHash   string // the tx hash of outside
	OutIndex uint64
	IsBeacon bool
}

type WhiteListProof struct {
	LightID  uint64 // the lightid for beaconAddress
	Atype    uint8
	Height   uint64 // the height of outside chain
	TxHash   string
	InIndex  uint64
	OutIndex uint64
	Amount   *big.Int // the amount of outside chain
}

func (wl *WhiteListProof) Clone() *WhiteListProof {
	return &WhiteListProof{
		LightID: wl.LightID,
		Height:  wl.Height,
		Amount:  new(big.Int).Set(wl.Amount),
		Atype:   wl.Atype,
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
