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
	// "github.com/classzz/czzutil"
)

var (
	ErrInvalidParam      = errors.New("Invalid Param")
	ErrLessThanMin		 = errors.New("less than min staking amount for lighthouse")
	ErrRepeatRegister    = errors.New("repeat register on this address")
	ErrNoRegister 	     = errors.New("not found the lighthouse")
	ErrNoUserReg 	     = errors.New("not entangle user in the lighthouse")
	ErrNoUserAsset 	     = errors.New("user no entangle asset in the lighthouse")
	ErrNotEnouthBurn 	 = errors.New("not enough burn amount in lighthouse")
)

var (
	MinStakingAmountForLightHouse = new(big.Int).Mul(big.NewInt(1000000),big.NewInt(1e9))
	MaxWhiteListCount  = 5
	MAXBASEFEE 		   = 10000
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
	Amount 	*big.Int
	Height 	uint64
}
type BurnInfos struct {
	Items 	[]*BurnItem
	BurnAmount *big.Int 		// add the user's burn amount
}
func newBurnInfos() *BurnInfos {
	return nil
}

func (b *BurnInfos) GetAllAmount() *big.Int {
	amount := big.NewInt(0)
	for _,v := range b.Items {
		amount = amount.Add(amount,v.Amount)
	}
	return amount
}

func (b *BurnInfos) GetValidAmount() *big.Int {
	return nil
}
// Update the valid amount for diffence height for entangle info
func (b *BurnInfos) Update() {

}

func ValidAssetType(utype uint32) bool {
	if utype & LhAssetBTC != 0 || utype & LhAssetBCH != 0 || utype & LhAssetBSV != 0 ||
	utype & LhAssetLTC != 0 || utype & LhAssetUSDT != 0 || utype & LhAssetDOGE != 0 {
		return true
	}
	return false
}
func ValidPK(pk []byte) bool {
	return true
}