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
)

var (
	MinStakingAmountForLightHouse = new(big.Int).Mul(big.NewInt(1000000),big.NewInt(1e9))
	MaxWhiteListCount  = 5
)
const (
	LhAssetBTC uint32 = 1 << iota
	LhAssetBCH
	LhAssetBSV
	LhAssetLTC
	LhAssetUSDT
	LhAssetDOGE
)

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