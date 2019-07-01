// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
This test file is part of the bchutil package rather than than the
bchutil_test package so it can bridge access to the internals to properly test
cases which are either not possible or can't reliably be tested via the public
interface. The functions are only exported while the tests are being run.
*/

package czzutil

import (
	"github.com/bourbaki-czz/classzz/bchec"
	"github.com/bourbaki-czz/classzz/chaincfg"
	"github.com/bourbaki-czz/czzutil/base58"
	"golang.org/x/crypto/ripemd160"
	"strings"
)

// SetBlockBytes sets the internal serialized block byte buffer to the passed
// buffer.  It is used to inject errors and is only available to the test
// package.
func (b *Block) SetBlockBytes(buf []byte) {
	b.serializedBlock = buf
}

// TstAppDataDir makes the internal appDataDir function available to the test
// package.
func TstAppDataDir(goos, appName string, roaming bool) string {
	return appDataDir(goos, appName, roaming)
}

// TstAddressPubKeyHash makes a AddressPubKeyHash, setting the
// unexported fields with the parameters hash and netID.
func TstAddressPubKeyHash(hash [ripemd160.Size]byte,
	params *chaincfg.Params) *AddressPubKeyHash {
	return &AddressPubKeyHash{
		hash:   hash,
		prefix: params.CashAddressPrefix,
	}
}

// TstAddressScriptHash makes a AddressScriptHash, setting the
// unexported fields with the parameters hash and netID.
func TstAddressScriptHash(hash [ripemd160.Size]byte,
	params *chaincfg.Params) *AddressScriptHash {
	return &AddressScriptHash{
		hash:   hash,
		prefix: params.CashAddressPrefix,
	}
}

// TstLegacyAddressPubKeyHash makes a LegacyAddressPubKeyHash, setting the
// unexported fields with the parameters hash and netID.
func TstLegacyAddressPubKeyHash(hash [ripemd160.Size]byte,
	netID byte) *LegacyAddressPubKeyHash {

	return &LegacyAddressPubKeyHash{
		hash:  hash,
		netID: netID,
	}
}

// TstLegacyAddressScriptHash makes a LegacyAddressScriptHash, setting the
// unexported fields with the parameters hash and netID.
func TstLegacyAddressScriptHash(hash [ripemd160.Size]byte,
	netID byte) *LegacyAddressScriptHash {

	return &LegacyAddressScriptHash{
		hash:  hash,
		netID: netID,
	}
}

// TstAddressPubKey makes an AddressPubKey, setting the unexported fields with
// the parameters.
func TstAddressPubKey(serializedPubKey []byte, pubKeyFormat PubKeyFormat,
	netID byte) *AddressPubKey {

	pubKey, _ := bchec.ParsePubKey(serializedPubKey, bchec.S256())
	return &AddressPubKey{
		pubKeyFormat: pubKeyFormat,
		pubKey:       (*bchec.PublicKey)(pubKey),
		pubKeyHashID: netID,
	}
}

// TstLegacyAddressSAddr returns the expected script address bytes for
// P2PKH and P2SH legacy addresses.
func TstLegacyAddressSAddr(addr string) []byte {
	decoded := base58.Decode(addr)
	return decoded[1 : 1+ripemd160.Size]
}

// TstAddressSAddr returns the expected script address bytes for
// P2PKH and P2SH cashaddr addresses.
func TstAddressSAddr(addr string, params *chaincfg.Params) []byte {
	prefix := params.CashAddressPrefix
	if !strings.HasPrefix(addr, prefix) {
		addr = prefix + ":" + addr
	}
	decoded, _, _, _ := checkDecodeCashAddress(addr)
	return decoded[:ripemd160.Size]
}
