// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package netsync

import (
	"github.com/classzz/classzz/blockchain"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/mempool"
	"github.com/classzz/classzz/peer"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
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
