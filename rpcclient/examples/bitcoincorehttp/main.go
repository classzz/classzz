// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/czzutil"
	"io/ioutil"
	"log"
	"path/filepath"
)

func main() {

	czzdHomeDir := czzutil.AppDataDir("classzz", false)
	certs, err := ioutil.ReadFile(filepath.Join(czzdHomeDir, "rpc.cert"))
	if err != nil {
		log.Fatal(err)
	}

	// Connect to local bitcoin core RPC server using HTTP POST mode.
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:8334",
		Endpoint:     "ws",
		User:         "root",
		Pass:         "admin",
		HTTPPostMode: true,  // Bitcoin core only supports HTTP POST mode
		DisableTLS:   false, // Bitcoin core does not provide TLS by default
		Certificates: certs,
	}

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	msginfo := make(map[int64]interface{})
	for i := 0; i < 20; i++ {
		bhash, err := client.GetBlockHash(int64(i))

		if err != nil {
			fmt.Println(err)
			return
		}
		block, err := client.GetBlock(bhash.String())
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, tx := range block.Transactions {
			txHash := tx.TxHash()
			txj, err := client.GetRawTransactionVerbose(&txHash)
			if err != nil {
				fmt.Println(err)
			}
			msginfo[int64(i)] = txj

		}

	}

	msg, _ := json.Marshal(msginfo)
	fmt.Println(string(msg))

}
