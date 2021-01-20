package cross

import (
	"crypto/ecdsa"
	"errors"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/czzutil"
)

var (
	ErrInvalidPubkey = errors.New("invalid public key")
	ErrCryptoType    = errors.New("invalid crypto type")
)

// RecoverPublic returns the public key of the marshal bytes.
func RecoverPublicFromBytes(pub []byte, t uint8) (*ecdsa.PublicKey, error) {
	switch t {
	case ExpandedTxEntangle_Doge:
		return UnmarshalPubkey(pub)
	case ExpandedTxEntangle_Ltc:
		return UnmarshalPubkey(pub) // tmp exc
	default:
		return nil, ErrCryptoType
	}
}

func MakeAddress(puk ecdsa.PublicKey, params *chaincfg.Params) (czzutil.Address, error) {
	pub := (*czzec.PublicKey)(&puk).SerializeCompressed()
	if addrHash, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pub), params); err != nil {
		return nil, err
	} else {
		address, err1 := czzutil.DecodeAddress(addrHash.String(), params)
		if err1 != nil {
			return nil, err
		}
		return address, nil
	}

}
