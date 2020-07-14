package main

import (
	"encoding/hex"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/czzutil"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var (
	rpcuserRegexp = regexp.MustCompile("(?m)^rpcuser=.+$")
	rpcpassRegexp = regexp.MustCompile("(?m)^rpcpass=.+$")
)

func TestExcessiveBlockSizeUserAgentComment(t *testing.T) {
	// Wipe test args.
	os.Args = []string{"classzz"}

	cfg, _, err := loadConfig()
	if err != nil {
		t.Fatal("Failed to load configuration")
	}

	if len(cfg.UserAgentComments) != 1 {
		t.Fatal("Expected EB UserAgentComment")
	}

	uac := cfg.UserAgentComments[0]
	uacExpected := "EB32.0"
	if uac != uacExpected {
		t.Fatalf("Expected UserAgentComments to contain %s but got %s", uacExpected, uac)
	}

	// Custom excessive block size.
	os.Args = []string{"classzz", "--excessiveblocksize=64000000"}

	cfg, _, err = loadConfig()
	if err != nil {
		t.Fatal("Failed to load configuration")
	}

	if len(cfg.UserAgentComments) != 1 {
		t.Fatal("Expected EB UserAgentComment")
	}

	uac = cfg.UserAgentComments[0]
	uacExpected = "EB64.0"
	if uac != uacExpected {
		t.Fatalf("Expected UserAgentComments to contain %s but got %s", uacExpected, uac)
	}
}

func TestCreateDefaultConfigFile(t *testing.T) {
	// Setup a temporary directory
	tmpDir, err := ioutil.TempDir("", "classzz")
	if err != nil {
		t.Fatalf("Failed creating a temporary directory: %v", err)
	}
	testpath := filepath.Join(tmpDir, "test.conf")

	// Clean-up
	defer func() {
		os.Remove(testpath)
		os.Remove(tmpDir)
	}()

	err = createDefaultConfigFile(testpath)

	if err != nil {
		t.Fatalf("Failed to create a default config file: %v", err)
	}

	content, err := ioutil.ReadFile(testpath)
	if err != nil {
		t.Fatalf("Failed to read generated default config file: %v", err)
	}

	if !rpcuserRegexp.Match(content) {
		t.Error("Could not find rpcuser in generated default config file.")
	}

	if !rpcpassRegexp.Match(content) {
		t.Error("Could not find rpcpass in generated default config file.")
	}
}

func TestGenesisAdderss(t *testing.T) {

	key, _ := czzec.NewPrivateKey(czzec.S256())
	wif, _ := czzutil.NewWIF(key, &chaincfg.MainNetParams, true)

	fmt.Println("wif:", wif.String())
	fmt.Println("priv:", hex.EncodeToString(key.Serialize()))
	pk := (*czzec.PublicKey)(&key.PublicKey).SerializeCompressed()
	fmt.Println("pub:", hex.EncodeToString(pk))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.MainNetParams)

	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}
	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())
}

func TestGenesisRegTestAdderss(t *testing.T) {

	key, _ := czzec.NewPrivateKey(czzec.S256())
	wif, _ := czzutil.NewWIF(key, &chaincfg.RegressionNetParams, true)

	fmt.Println("wif:", wif.String())
	fmt.Println("priv:", hex.EncodeToString(key.Serialize()))
	pk := (*czzec.PublicKey)(&key.PublicKey).SerializeCompressed()
	fmt.Println("pub:", hex.EncodeToString(pk))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.RegressionNetParams)

	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}
	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())
}

func TestGenesisTestAdderss(t *testing.T) {

	key, _ := czzec.NewPrivateKey(czzec.S256())
	wif, _ := czzutil.NewWIF(key, &chaincfg.TestNetParams, true)

	fmt.Println("wif:", wif.String())
	fmt.Println("priv:", hex.EncodeToString(key.Serialize()))
	pk := (*czzec.PublicKey)(&key.PublicKey).SerializeCompressed()
	fmt.Println("pub:", hex.EncodeToString(pk))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.TestNetParams)

	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}
	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())
}

func TestGenesisSimNetAdderss(t *testing.T) {

	key, _ := czzec.NewPrivateKey(czzec.S256())
	wif, _ := czzutil.NewWIF(key, &chaincfg.SimNetParams, true)

	fmt.Println("wif:", wif.String())
	fmt.Println("priv:", hex.EncodeToString(key.Serialize()))
	pk := (*czzec.PublicKey)(&key.PublicKey).SerializeCompressed()
	fmt.Println("pub:", hex.EncodeToString(pk))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.SimNetParams)

	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}
	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())
}

func TestConvertAddr(t *testing.T) {

	keyBy, _ := hex.DecodeString("496a5621a8210ec7f28521e104ea8c910d2eaddb4e57b282ddb67ae7a7fcf70b")
	key, _ := czzec.PrivKeyFromBytes(czzec.S256(), keyBy)
	wif, _ := czzutil.NewWIF(key, &chaincfg.MainNetParams, true)

	fmt.Println("wif:", wif.String())
	fmt.Println("priv:", hex.EncodeToString(key.Serialize()))
	pk := (*czzec.PublicKey)(&key.PublicKey).SerializeCompressed()
	fmt.Println("pub:", hex.EncodeToString(pk))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.MainNetParams)

	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}
	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())

}

func TestWIFConvertAddr(t *testing.T) {

	wif, _ := czzutil.DecodeWIF("QR5LWbjSyimFo7CkjeLdFEuiXAXP9nfqwQV9DjWmPg2ytET4647D")
	key := wif.PrivKey
	fmt.Println("wif:", wif.String())
	fmt.Println("priv:", hex.EncodeToString(key.Serialize()))
	pk := (*czzec.PublicKey)(&key.PublicKey).SerializeCompressed()
	fmt.Println("pub:", hex.EncodeToString(pk))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.MainNetParams)

	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}
	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())

}

func TestNewAddressFromPub(t *testing.T) {

	// btc 0x00
	// bsv 0x00
	// bch 0x00

	pub, _ := hex.DecodeString("1fad1e999a021ffbe97556f2b7b6ab4ac25c95fd")
	params := &chaincfg.Params{
		LegacyScriptHashAddrID: 0x00,
	}

	addr, err := czzutil.NewLegacyAddressScriptHashFromHash(pub, params)
	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}

	fmt.Println("address: ", addr.String())

}
