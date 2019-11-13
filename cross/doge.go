package cross

import (
	"crypto/ecdsa"
	"github.com/classzz/classzz/czzec"
)

// UnmarshalPubkey converts bytes to a secp256k1 public key.
func UnmarshalPubkey1(pub []byte) (*ecdsa.PublicKey, error) {
	//x, y := elliptic.Unmarshal(czzec.S256(), pub)
	pubk, err := czzec.ParsePubKey(pub, czzec.S256())

	if err != nil {
		return nil, ErrInvalidPubkey
	}
	return pubk.ToECDSA(), nil
}
