package cross

import (
	"fmt"
	"github.com/classzz/classzz/btcjson"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/database"
	_ "github.com/classzz/classzz/database/ffldb"
	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/wire"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

var (
	dogecoinrpc     = "127.0.0.1:9999"
	dogecoinrpcuser = "root"
	dogecoinrpcpass = "admin"

	ltccoinrpc     = "127.0.0.1:9998"
	ltccoinrpcuser = "root"
	ltccoinrpcpass = "admin"
)

func TestVerifyTx(t *testing.T) {

	dbPath := filepath.Join(os.TempDir(), "examplecreate")
	db, err := database.Create("ffldb", dbPath, wire.MainNet)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dbPath)
	defer db.Close()

	cacheEntangleInfo := &CacheEntangleInfo{
		DB: db,
	}

	var dogeclients []*rpcclient.Client
	connCfg := &rpcclient.ConnConfig{
		Host:         dogecoinrpc,
		Endpoint:     "ws",
		User:         dogecoinrpcuser,
		Pass:         dogecoinrpcpass,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		t.Error("err", err)
	}
	dogeclients = append(dogeclients, client)

	// entangleVerify := &EntangleVerify{
	// 	Cache:       cacheEntangleInfo,
	// 	DogeCoinRPC: dogeclients,
	// }

	//create tx
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		Sequence: wire.MaxTxInSequenceNum,
	})

	// {\"extxtype\":240,\"index\":1,\"height\":2975150,\"amount\":1000000000,\"exttxhash\":\"6800e9579dad0e6667e6e897e2d65f08afdba18ed1eb9956fe5f0b936162404a\"\}\]
	EntangleOut := &btcjson.EntangleOut{
		ExTxType:  240,
		Index:     0,
		Height:    2972841,
		Amount:    big.NewInt(225226803000),
		ExtTxHash: "6800e9579dad0e6667e6e897e2d65f08afdba18ed1eb9956fe5f0b936162404a",
	}

	scriptInfo, err := txscript.EntangleScript(EntangleOut.Serialize())
	if err != nil {
		t.Error("err", err)
	}
	txout := &wire.TxOut{
		Value:    0,
		PkScript: scriptInfo,
	}
	tx.AddTxOut(txout)
	// puk, err := entangleVerify.VerifyEntangleTx(tx)
	// if err != nil {
	// 	t.Error("err", err)
	// }

	// t.Log(puk[0].Pub)

}
