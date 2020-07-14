package main

import (
	"encoding/hex"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/czzutil"
	"github.com/jessevdk/go-flags"
	"os"
)

type config struct {
	NetType string `short:"t" long:"type" description:"mainnet, testnet, regtest, simnet"`
}

func main() {

	cfg := config{
		NetType: "mainnet",
	}
	parser := flags.NewParser(&cfg, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stderr)
		}
		return
	}

	params := &chaincfg.MainNetParams

	switch cfg.NetType {
	case "mainnet":
		params = &chaincfg.MainNetParams
	case "testnet":
		params = &chaincfg.TestNetParams
	case "regtest":
		params = &chaincfg.RegressionNetParams
	case "simnet":
		params = &chaincfg.SimNetParams
	}

	key, _ := czzec.NewPrivateKey(czzec.S256())
	wif, _ := czzutil.NewWIF(key, params, true)

	fmt.Println("wif:", wif.String())
	fmt.Println("priv:", hex.EncodeToString(key.Serialize()))
	pk := (*czzec.PublicKey)(&key.PublicKey).SerializeCompressed()
	fmt.Println("pub:", hex.EncodeToString(pk))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.RegressionNetParams)

	if err != nil {
		fmt.Println("failed to make address for: ", err)
		return
	}

	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())

}
