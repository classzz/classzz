// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"time"

	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/wire"
)

// genesisCoinbaseTx is the coinbase transaction for the genesis blocks for
// the main network, regression test network, and test network (version 3).
var signatureScript = []byte("The CZZ network has a minimal mining difficulty of 1mh/s, regardless of total hashpower available to the network. The downside is that during the early days of mining, block production rate could be quite low. The advantage is that it will completely eliminate all near-zero-cost tokens in the system, thereby making the token value less volatile.")
var genesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: signatureScript,
			Sequence:        0xffffffff,
		},
	},
	TxOut:    []*wire.TxOut{},
	LockTime: 0,
}

// genesisHash is the hash of the first block in the block chain for the main
// network (genesis block).
var genesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x17, 0x05, 0xd3, 0x0e,
	0xd3, 0xf1, 0xd1, 0x14,
	0xfb, 0xbd, 0x0c, 0x1e,
	0xa6, 0xc8, 0x5f, 0x82,
	0xcf, 0x6a, 0x6f, 0x67,
	0xef, 0x9b, 0x6e, 0x3d,
	0xea, 0xc7, 0xa2, 0xc4,
	0xa6, 0xdc, 0x9f, 0x81,
})

// genesisMerkleRoot is the hash of the first transaction in the genesis block
// for the main network.
var genesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0x7f, 0x3d, 0xc9, 0x6d,
	0x40, 0x48, 0x0b, 0x5b,
	0xa6, 0xb4, 0x69, 0xe1,
	0x9e, 0x92, 0x88, 0x6f,
	0xf9, 0xf4, 0x75, 0x9f,
	0x37, 0xdc, 0x75, 0xf0,
	0x68, 0xf0, 0xf8, 0xe1,
	0xd3, 0xbd, 0x38, 0x9b,
})

// genesisBlock defines the genesis block of the block chain which serves as the
// public transaction ledger for the main network.
var genesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{}, // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: genesisMerkleRoot,
		CIDRoot:    chainhash.Hash{},
		Timestamp:  time.Unix(1561895999, 0), // 2019-06-30 59:59:59 +1200 UTC
		Bits:       0x1e10624d,               // 504390221
		Nonce:      0x1d5f3f,                 // 1924927
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// regTestGenesisHash is the hash of the first block in the block chain for the
// regression test network (genesis block).
var regTestGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xcb, 0x8a, 0xbe, 0x67,
	0x3d, 0x4d, 0x9e, 0xb5,
	0xfc, 0x81, 0x52, 0x11,
	0xb9, 0xdd, 0xfe, 0x21,
	0x0b, 0x7b, 0xbf, 0x4f,
	0x54, 0x67, 0x7e, 0x77,
	0xe3, 0xd8, 0x00, 0x93,
	0xb0, 0x84, 0xf8, 0x31,
})

// regTestGenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the regression test network.  It is the same as the merkle root for
// the main network.
var regTestGenesisMerkleRoot = genesisMerkleRoot

// regTestGenesisBlock defines the genesis block of the block chain which serves
// as the public transaction ledger for the regression test network.
var regTestGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: regTestGenesisMerkleRoot, // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		CIDRoot:    chainhash.Hash{},
		Timestamp:  time.Unix(1561895999, 0), // 2019-06-30 59:59:59 +1200 UTC
		Bits:       0x207fffff,               // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
		Nonce:      0,
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// testNet3GenesisHash is the hash of the first block in the block chain for the
// test network (version 3).
var testNetGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xca, 0x56, 0x8c, 0x3e,
	0x4d, 0xac, 0x89, 0x95,
	0xcd, 0x74, 0xbe, 0xef,
	0x36, 0x06, 0xcb, 0x2b,
	0xb9, 0x1b, 0x26, 0x43,
	0x96, 0x65, 0x4a, 0x75,
	0x12, 0xdd, 0x6f, 0xb9,
	0x55, 0x26, 0xc6, 0xcf,
})

// testNet3GenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the test network (version 3).  It is the same as the merkle root
// for the main network.
var testNetGenesisMerkleRoot = genesisMerkleRoot

// testNet3GenesisBlock defines the genesis block of the block chain which
// serves as the public transaction ledger for the test network (version 3).
var testNetGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: testNetGenesisMerkleRoot, // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		CIDRoot:    chainhash.Hash{},
		Timestamp:  time.Unix(1574672932, 0), // 2019-11-25 17:08:52 +1200 UTC
		Bits:       0x1f0ccccc,               // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
		Nonce:      1324,
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

// simNetGenesisHash is the hash of the first block in the block chain for the
// simulation test network.
var simNetGenesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xca, 0x56, 0x8c, 0x3e,
	0x4d, 0xac, 0x89, 0x95,
	0xcd, 0x74, 0xbe, 0xef,
	0x36, 0x06, 0xcb, 0x2b,
	0xb9, 0x1b, 0x26, 0x43,
	0x96, 0x65, 0x4a, 0x75,
	0x12, 0xdd, 0x6f, 0xb9,
	0x55, 0x26, 0xc6, 0xcf,
})

// simNetGenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the simulation test network.  It is the same as the merkle root for
// the main network.
var simNetGenesisMerkleRoot = genesisMerkleRoot

// simNetGenesisBlock defines the genesis block of the block chain which serves
// as the public transaction ledger for the simulation test network.
var simNetGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},        // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: simNetGenesisMerkleRoot, // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		CIDRoot:    chainhash.Hash{},
		Timestamp:  time.Unix(1574672932, 0), // 2019-11-25 17:08:52 +1200 UTC
		Bits:       0x1f0ccccc,               // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
		Nonce:      1324,
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}
