package cross

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"github.com/classzz/classzz/czzec"
)

// UnmarshalPubkey converts bytes to a secp256k1 public key.
func UnmarshalPubkey1(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(czzec.S256(), pub)
	if x == nil {
		return nil, ErrInvalidPubkey
	}
	return &ecdsa.PublicKey{Curve: czzec.S256(), X: x, Y: y}, nil
}
