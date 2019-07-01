// Copyright (c) 2018 The gcash developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package merkleblock_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/bourbaki-czz/classzz/chaincfg/chainhash"
	"github.com/bourbaki-czz/classzz/wire"
	"github.com/bourbaki-czz/czzutil/merkleblock"
)

// TestNewMerkleBlockFromMsg tests decoding of a partial merkle tree from
// wire.MsgMerkleBlock. Test values derived from random selection of block
// data extracted from bitcoin-cli ABC implementation's of RPC command
// "gettxoutproof"
func TestNewMerkleBlockFromMsg(t *testing.T) {

	tests := []struct {
		name       string
		block      string
		merkleRoot *chainhash.Hash
		txnIds     []*chainhash.Hash
		proof      string
	}{
		{
			name:       "Verify Testnet block 1267123, 2 hashes from 52 transactions",
			block:      "00000000eebdf440a2d44b20cea7d9bc50043bb6a7583d32ed88d0e200ce5b9c",
			merkleRoot: hashFromStr("8279d2bbf9bd03493a62b0af8bb9c4d2fcb289a9bc637702967c8a1fc8fc4681"),
			txnIds: []*chainhash.Hash{
				hashFromStr("4c988e91ecdd4ff35f4d033737e084bbee8e3fb1ab9953be244c7d458991222e"),
				hashFromStr("149ea58c61922e395354865885fd82a77db09713816164e24cb8d6490c1a0651"),
			},
			proof: "000000206f6383eb819fb001adbefba13e9adf9f5161267aec45960cfa010000000000008146fcc81f8a7c96027763bca989b2fcd2c4b98bafb0623a4903bdf9bbd27982787ee65bffff001dc6cf78c634000000098507a8580c7705cb3387146e84e89d5bcaa05a63f42de20135da73ad241a44aa2e229189457d4c24be5399abb13f8eeebb84e03737034d5ff34fddec918e984ccb09f770698f9eb555e1a8e49d2454a2bf0932dbe7a9dd00ff7b34d25ab53f7e66d1aeaee6d5c529f8efdb08c8967bb5e645a2445da97e6f4113b19e15e7944503304658ae08ac041aabcdab6f659e8014f6354ed46e2e96d6d52f0613666c8bb1aecf6d0ad5bcc1570d375c40b0136f1435e58927feb1b56700f7817636e729d54d27cd30958f152473db08484317999c51c049bd730663c77eeac101e9561651061a0c49d6b84ce26461811397b07da782fd8558865453392e92618ca59e14592829d43e020c272a11e3ea201d14a5d7d661940884c9444f0dd86da4897b2103fdf002",
		},
		{
			name:       "Verify Testnet block 1268246, 4 hashes from 4 transactions",
			block:      "000000006094499a2e47729fa1284be5f3d0a6d6b3144f9ab2b668c51ef89a81",
			merkleRoot: hashFromStr("8589c13da0282caeaa88b4fe94a7b9554062a6a3983cd0fb662184762225d6e7"),
			txnIds: []*chainhash.Hash{
				hashFromStr("f958b7d5654843dba1a749d1e8c452fb4668d793d2bedcd69c8dfeb2b6158c0c"),
				hashFromStr("5d16833cf59493d2f95377b4b5769f10ec61328e6982c9c4ca001f14418340c8"),
				hashFromStr("77ae677b32ec4ff0ad47f0b637958265e5bea50ac69e55171c1131ce7428a4f1"),
				hashFromStr("ffe0929c0077419c1633acc7aaa52081b5f20e51c49ad9814b93be349f953678"),
			},
			proof: "000000204d8a8dde721f22cc65d112aba0c86823efffd651bbc0fe951f7fdc7300000000e7d6252276842166fbd03c98a3a6624055b9a794feb488aaae2c28a03dc18985eefcf15bffff001d0410040004000000040c8c15b6b2fe8d9cd6dcbed293d76846fb52c4e8d149a7a1db434865d5b758f9c8408341141f00cac4c982698e3261ec109f76b5b47753f9d29394f53c83165df1a42874ce31111c17559ec60aa5bee565829537b6f047adf04fec327b67ae777836959f34be934b81d99ac4510ef2b58120a5aac7ac33169c4177009c92e0ff017f",
		},
		{
			name:       "Verify Testnet block 1268992, 12 hashes from 76 transactions",
			block:      "00000000f423978ee10fdf712a9e46e9a6e83614237a94fc8569078bceb4bbf0",
			merkleRoot: hashFromStr("a9391c2077fa2cf3a08eb76018f830e5b4f890d89b3b58200cecaaa0dd655e18"),
			txnIds: []*chainhash.Hash{
				hashFromStr("107e02ad080621339f7798257337c8d25a01621a0246114eddbb38f75ec66680"),
				hashFromStr("15196fb20c0404bb135eb0bb4eb946ec580094a7ec800193ede2ac0957cc6b90"),
				hashFromStr("316acf47d227b3eb7de769d4ecc9fc89375e4f76d09a68f3ad3e5cc6d914d920"),
				hashFromStr("3840230613331b8dda000b1b65fd1796625e3791a432171bb5dd697291752078"),
				hashFromStr("702d2809d50233c5c6331ee572714e5f8abce338ea3a338c1c16f3801bb021ea"),
				hashFromStr("728ca03f9c4500e52907f3a56446acd1ca6ab4cb9f7be14853c4b9c9957f6a7b"),
				hashFromStr("8ad93d8b1325a5c6974b7e6318b456f03d46fc3a3ab62df9b7798c27c2957839"),
				hashFromStr("a953ebc0e34b35cfdbb114ee0d26b97a22369e516dfcb53f11251ec45991724d"),
				hashFromStr("c869861f18451bbfb508f0e1a228dfd775e05de6e8de654d1f1d13ba0349cde1"),
				hashFromStr("d44fde9b6d885052948dde17cd848f78f7beb3eb1b87e5005c9a08063a60a407"),
				hashFromStr("e1ac2228e30dfd0a7162244f634bf6bc1acada35f49208be98c9e6b967ba492a"),
				hashFromStr("f99b9dac537a1264ffcc4c30f5227ce3583c50ae6d625dc39cd57f64020603ce"),
			},
			proof: "000000201b7ad15b2efd22dc4c284b8d25bbb956933ea45b9e7e1d5ff44494fc00000000185e65dda0aaec0c20583b9bd890f8b4e530f81860b78ea0f32cfa77201c39a90629f55bffff001d202c49014c0000002a1089ba96dfe7867231ab968c896ccb96439882d2649bb383f322d41cbf1469ca942a1d8a9439bf8b432c46c93bcfa6be20ca758627bf8a61df84eeb072689a0e8066c65ef738bbdd4e1146021a62015ad2c837732598779f33210608ad027e10f473e3fa4a2072ad18b9b31c8ff4bd936112f2da435f1c1836ff354b24114eb7acee01f03b799d20cee5614e584a2833dbff61b61780072a2e87e092dc748b12906bcc5709ace2ed930180eca7940058ec46b94ebbb05e13bb04040cb26f19158e7f7f95afcfb3d451e26dc9b6e12ccaf395ed034a0d828e508158326e240b28f324890ef7cf32156cefe3ffbc8fd2eeee5b0598c717daaa0e90d93e2fd4367446a1c927e5f69df92cd9ae2d7f15810f9ccfd43d3c7b6c85199ed5a4cf366d2620d914d9c65c3eadf3689ad0764f5e3789fcc9ecd469e77debb327d247cf6a31782075917269ddb51b1732a491375e629617fd651b0b00da8d1b33130623403880df43ba89a6a45d6abeecf49738c7ba130199313309e3d956c879e0f8ddb83b795e6bf8ee537ae79354bb9b138b38551964773d488e3de131d19f28a6b272de73a5a32d9ad8b2e841e3d2dc66821b1a4791f7c66ab084422a413c4a9576a9f4f3af05653ce6a1d535194614da5aeaeeb388355c368e863e6434ea1d9c2771863cbb84f4827c34e2d46576e3a672ddef0a9cdea8a9f9c77ae42220cb51fb3a6cea21b01b80f3161c8c333aea38e3bc8a5f4e7172e51e33c6c53302d509282d701bb1d7cf277c7d35150f799db8b7897cd18ca9f1cafcbd29991d2eee97da44717b6a7f95c9b9c45348e17b9fcbb46acad1ac4664a5f30729e500459c3fa08c7232a83b7059e8f74ca8aed4bdaad15fb39440e90d7a69d901b3505f44e8e227932369b63a83ec43e30a3d438af76aefdab144779f549c399f5391a6aa386d53ea48a46acf96e01865fffa3a30605c013511bd51048a735c5eee8b19cd6a5d039e5c7795cc966fb0a69d0b05933b8bc4239e6726f83c1fe8b69f0d75655da2ce87397895c2278c79b7f92db63a3afc463df056b418637e4b97c6a525138b3dd98af2a15d7d3bd66dd501987ce6f47511f0b4e666f482a3cf62a4645935b372db354dfb0da83b4581009cc1855a5a91648b3c3d6c09bedd88db702a2b0693253e4f337875a2bf2710c1edecdc1edd5f1960493cbe136ec0b5a4aa865fa5882939a74d729159c41e25113fb5fc6d519e36227ab9260dee14b1dbcf354be3c0eb53a9a42be8a7217a2757a5ecb1cc229cb5b6f2b1b154cd07ad6c590d1785c2fbbcbdca6179004701b050f88f6084480ccb85d744fd1851cf9b0f7b2ba4bc930b49bfe1cd4903ba131d1f4d65dee8e65de075d7df28a2e1f008b5bf1b45181f8669c8f505b58f73c9a3ec0e303e2bfb1682d1858188145e8dd31d9058f56d67af6ba2e0f99e06877046afe40b18477d95da59d265e7a95e912ed65e4d37f6aa8174d307a4603a06089a5c00e5871bebb3bef7788f84cd17de8d945250886d9bde4fd47c5b0253ce409d66f109e46456b3fa153bbc47822cc13109b0570d1eaadcfc57fe74ab379178bb729f076330f2888827be321668dfd2f5f722d5af6f9e1e25d72a49ba67b9e6c998be0892f435daca1abcf64b634f2462710afd0de32822ace113b05934da9ec5643898d783f44dcc658d06f28c2d21b988e4fd5b1acb9d6ef266d69d99a60539f9c0818006901d102449501308ab5ce6eb99d4a539e98308439f936c2adcaf9fbd15f5708076ca545a90b17a27e8c5134e7d2d823aa690e1ebce030602647fd59cc35d626dae503c58e37c22f5304cccff64127a53ac9d9bf9576d5494f837275fecf6c70bd056cbde91645271fdbd4e8158c264dbb356c4f90bdfbaeac7baae56b7ae751b",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// decode proof from hex to []bytes
			dec, err := hex.DecodeString(tt.proof)

			if err != nil {
				t.Errorf("hex.DecodeString error: %v\n", err)
				return
			}

			msg := wire.MsgMerkleBlock{}

			rbuf := bytes.NewReader(dec)

			err = msg.BchDecode(rbuf, wire.ProtocolVersion, wire.BaseEncoding)

			if err != nil {
				t.Errorf("BchDecode error: %v\n", err)
				return
			}

			mBlock := merkleblock.NewMerkleBlockFromMsg(msg)

			// extract transactions from our proof merkle block
			merkleRoot := mBlock.ExtractMatches()

			// check merkle root matches
			if !merkleRoot.IsEqual(tt.merkleRoot) {
				t.Errorf("Expected merkleRoot: %s\nGot: %s\n", tt.merkleRoot, merkleRoot.String())
			}

			// check extract transactions match
			if !reflect.DeepEqual(mBlock.GetMatches(), tt.txnIds) {
				t.Errorf("Expected transactions %v\nGot: %v", tt.txnIds, mBlock.GetMatches())
			}

			// check number of items matched is the same as the number of transactions
			if len(mBlock.GetItems()) != len(tt.txnIds) {
				t.Errorf("Invalid number of items found, got %d wanted %d",
					len(mBlock.GetItems()), len(tt.txnIds))
			}

			// check tree traversal was not bad
			if mBlock.BadTree() {
				t.Errorf("Tree traversal was bad")
			}
		})
	}
}

// hashFromStr provides function to wrap the primary function without
// the need to return an error since we can assert the static strings supplied
// in these tests will always decode
func hashFromStr(s string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(s)

	if err != nil {
		hash = &chainhash.Hash{}
	}

	return hash
}
