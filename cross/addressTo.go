package cross

import (
	"crypto/ecdsa"
	"errors"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/czzutil"
)

var (
	ErrInvalidMsgLen       = errors.New("invalid message length, need 32 bytes")
	ErrInvalidSignatureLen = errors.New("invalid signature length")
	ErrInvalidRecoveryID   = errors.New("invalid signature recovery id")
	ErrInvalidKey          = errors.New("invalid private key")
	ErrInvalidPubkey       = errors.New("invalid public key")
	ErrSignFailed          = errors.New("signing failed")
	ErrRecoverFailed       = errors.New("recovery failed")
	ErrCryptoType          = errors.New("invalid crypto type")
)

// RecoverPublic returns the public key of the marshal bytes.
func RecoverPublicFromBytes(pub []byte, t ExpandedTxType) (*ecdsa.PublicKey, error) {
	switch t {
	case ExpandedTxEntangle_Doge:
		return UnmarshalPubkey1(pub)
	default:
		return nil, ErrCryptoType
	}
}

func MakeAddress(puk ecdsa.PublicKey) (error, czzutil.Address) {
	pub := (*czzec.PublicKey)(&puk).SerializeCompressed()
	if addrHash, err := czzutil.NewAddressPubKeyHash(
		czzutil.Hash160(pub), &chaincfg.MainNetParams); err != nil {
		return err, nil
	} else {
		address, err1 := czzutil.DecodeAddress(addrHash.String(), &chaincfg.MainNetParams)
		if err1 != nil {
			return err, nil
		}
		return nil, address
	}
}
