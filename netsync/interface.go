// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package netsync

import (
	"github.com/bourbaki-czz/classzz/blockchain"
	"github.com/bourbaki-czz/classzz/chaincfg"
	"github.com/bourbaki-czz/classzz/chaincfg/chainhash"
	"github.com/bourbaki-czz/classzz/mempool"
	"github.com/bourbaki-czz/classzz/peer"
	"github.com/bourbaki-czz/classzz/wire"
	"github.com/bourbaki-czz/czzutil"
)

// PeerNotifier exposes methods to notify peers of status changes to
// transactions, blocks, etc. Currently server (in the main package) implements
// this interface.
type PeerNotifier interface {
	AnnounceNewTransactions(newTxs []*mempool.TxDesc)

	UpdatePeerHeights(latestBlkHash *chainhash.Hash, latestHeight int32, updateSource *peer.Peer)

	RelayInventory(invVect *wire.InvVect, data interface{})

	TransactionConfirmed(tx *czzutil.Tx)
}

// Config is a configuration struct used to initialize a new SyncManager.
type Config struct {
	PeerNotifier PeerNotifier
	Chain        *blockchain.BlockChain
	TxMemPool    *mempool.TxPool
	ChainParams  *chaincfg.Params

	DisableCheckpoints bool
	MaxPeers           int

	FeeEstimator *mempool.FeeEstimator

	MinSyncPeerNetworkSpeed uint64

	FastSyncMode bool
}
