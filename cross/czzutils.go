package cross

import (
	// "bytes"
	// "encoding/binary"
	"errors"
	// "fmt"
	// "strings"
	"math/big"

	// "github.com/classzz/classzz/chaincfg"
	// "github.com/classzz/classzz/chaincfg/chainhash"
	// "github.com/classzz/classzz/czzec"
	// "github.com/classzz/classzz/txscript"
	// "github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

var (
	ErrInvalidParam   = errors.New("Invalid Param")
	ErrLessThanMin    = errors.New("less than min staking amount for lighthouse")
	ErrRepeatRegister = errors.New("repeat register on this address")
	ErrNoRegister     = errors.New("not found the lighthouse")
	ErrNoUserReg      = errors.New("not entangle user in the lighthouse")
	ErrNoUserAsset    = errors.New("user no entangle asset in the lighthouse")
	ErrNotEnouthBurn  = errors.New("not enough burn amount in lighthouse")
)

var (
	MinStakingAmountForBeaconAddress  = new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e9))
	MaxWhiteListCount                 = 5
	MAXBASEFEE                        = 10000
	LimitRedeemHeightForBeaconAddress = 5000
)

const (
	LhAssetBTC uint32 = 1 << iota
	LhAssetBCH
	LhAssetBSV
	LhAssetLTC
	LhAssetUSDT
	LhAssetDOGE
)

type BurnItem struct {
	Amount      *big.Int // czz asset amount
	Height      uint64
	RedeemState byte // 0--init, 1 -- redeem done by BeaconAddress payed,2--punishing,3-- punished
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
func (b *BurnInfos) getBurnTimeout(height uint64,update bool) []*BurnItem {
	res := make([]*BurnItem,0,0)
	for _, v := range b.Items {
		if v.RedeemState == 0 && int64(height-v.Height) >  int64(LimitRedeemHeightForBeaconAddress) {
			res = append(res,&BurnItem{
				Amount: 	new(big.Int).Set(v.Amount),
				Height:		v.Height,
				RedeemState: v.RedeemState,
			})
			if update {
				v.RedeemState = 2
			}
		}
	}
	return res
}
func (b *BurnInfos) addBurnItem(height uint64,amount *big.Int) {
	item := &BurnItem{
		Amount: new(big.Int).Set(amount),
		Height: height,
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
		b.Items = append(b.Items,item)
		b.BurnAmount = new(big.Int).Add(b.BurnAmount, amount)
	}
}

type TimeOutBurnInfo struct {
	Items     []*BurnItem
	AssetType uint32
}
type TypeTimeOutBurnInfo []*TimeOutBurnInfo
type UserTimeOutBurnInfo map[czzutil.Address]TypeTimeOutBurnInfo
type WhiteUnit struct {
	AssetType uint32
	Pk        []byte
}

type BaseAmountUint struct {
	AssetType uint32
	Amount    *big.Int
}

type EnAssetItem BaseAmountUint
type FreeQuotaItem BaseAmountUint

type BeaconAddressInfo struct {
	ExchangeID     uint64
	Address        czzutil.Address
	StakingAmount  *big.Int         // in
	EntangleAmount *big.Int         // out,express by czz,all amount of user's entangle
	EnAssets       []*EnAssetItem   // out,the extrinsic asset
	Frees          []*FreeQuotaItem // extrinsic asset
	AssetFlag      uint32
	Fee            uint64
	KeepTime       uint64 // the time as the block count for finally redeem time
	WhiteList      []*WhiteUnit
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

/////////////////////////////////////////////////////////////////
// Address > EntangleEntity
type EntangleEntity struct {
	ExchangeID      uint64
	Address         czzutil.Address
	AssetType       uint32
	Height          *big.Int // newest height for entangle
	OldHeight       *big.Int // oldest height for entangle
	EnOutsideAmount *big.Int // out asset
	OriginAmount    *big.Int // origin asset(czz) by entangle in
	MaxRedeem       *big.Int // out asset
	BurnAmount      *BurnInfos
}
type EntangleEntitys []*EntangleEntity
type UserEntangleInfos map[czzutil.Address]EntangleEntitys

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

/////////////////////////////////////////////////////////////////
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
func (ee *EntangleEntitys) getBurnTimeout(height uint64,update bool) TypeTimeOutBurnInfo {
	res := make([]*TimeOutBurnInfo,0,0)
	for _,entity := range *ee {
		items := entity.BurnAmount.getBurnTimeout(height,update)
		if len(items) > 0 {
			res = append(res,&TimeOutBurnInfo{
				Items:		items,
				AssetType:	entity.AssetType,
			})
		}
	}
	return TypeTimeOutBurnInfo(res)
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
	return true
}
func isValidAsset(atype, assetAll uint32) bool {
	return atype&assetAll != 0
}
