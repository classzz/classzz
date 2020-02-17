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
type WhiteUnit struct {
	AssetType 		uint32
	Pk				[]byte
}

type LightHouseInfo struct {
	ExchangeID		uint64
	Address 		czzutil.Address
	StakeAmount 	*big.Int 
	EntangleAmount  *big.Int
	AssetFlag 		uint32
	Fee 			uint64
	KeepTime		uint64 		// the time as the block count for finally redeem time
	WhiteList 		[]*WhiteUnit
}
// Address > EntangleEntity
type EntangleEntity struct {
	ExchangeID		uint64
	Address 		czzutil.Address
	AssetType		uint32
	Height			*big.Int
	EntangleAmount 	*big.Int
	BurnAmount 		*big.Int
}
type EntangleEntitys []*EntangleEntity
type UserEntangleInfos map[czzutil.Address]EntangleEntitys

type EntangleState struct {
	EnInfos 		map[czzutil.Address]*LightHouseInfo
	EnEntitys 		map[uint64]UserEntangleInfos
	CurExchangeID 	uint64
}

/////////////////////////////////////////////////////////////////

/////////////////////////////////////////////////////////////////
// keep staking enough amount asset
func (es *EntangleState) RegisterLightHouse(addr czzutil.Address,amount *big.Int,
	fee uint64,assetType uint32) error {
	if amount.Cmp(MinStakingAmountForLightHouse) < 0 {
		return ErrLessThanMin
	}
	if _,ok := es.EnInfos[addr]; ok {
		return ErrRepeatRegister
	}
	info := &LightHouseInfo{
		ExchangeID:		es.CurExchangeID+1,
		Address:		addr,
		StakeAmount:	new(big.Int).Set(amount),
		AssetFlag:		assetType,
		Fee:			fee,
		EntangleAmount:	big.NewInt(0),
		WhiteList:		make([]*WhiteUnit,0,0),
	}
	es.EnInfos[addr] = info
	return nil
}
func (es *EntangleState) AppendWhiteList(addr czzutil.Address,wlist []*WhiteUnit) error {
	if val,ok := es.EnInfos[addr]; ok {
		cnt := len(val.WhiteList)
		if cnt + len(wlist) >= MaxWhiteListCount {
			return errors.New("more than max white list")
		}
		for _,v := range wlist {
			if ValidAssetType(v.AssetType) && ValidPK(v.Pk) {
				val.WhiteList = append(val.WhiteList,v)
			}
		}
		return nil
	} else {
		return ErrNoRegister
	}	
}
// UnregisterLightHouse need to check all the proves and handle all the user's burn coins
func (es *EntangleState) UnregisterLightHouse(addr czzutil.Address) error {
	if val,ok := es.EnInfos[addr]; ok {
		last := new(big.Int).Sub(val.StakeAmount,val.EntangleAmount)
		redeemAmount(addr,last)
	} else {
		return ErrNoRegister
	}
	return nil
}
func (es *EntangleState) AddEntangleItem(addr czzutil.Address,aType uint,lightID uint64) error {
	return nil
}
func (es *EntangleState) BurnAsset(addr czzutil.Address,aType uint,lightID uint64) error {
	return nil
}
func (es *EntangleState) ConfiscateAsset() error {
	return nil
}

//////////////////////////////////////////////////////////////////////
func redeemAmount(addr czzutil.Address,amount *big.Int) error {
	if amount.Sign() > 0 {
		
	}
	return nil
}