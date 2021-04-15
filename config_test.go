package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/czzutil"
	"github.com/classzz/czzutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
	"io/ioutil"
	"math/big"
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
	fmt.Println("pubk:", hex.EncodeToString((*czzec.PublicKey)(&key.PublicKey).SerializeUncompressed()))
	fmt.Println("pub:", hex.EncodeToString(pk))
	fmt.Println("pubhex:", hex.EncodeToString(key.PublicKey.Y.Bytes()))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.MainNetParams)

	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}
	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())

	add := crypto.Keccak256Hash(pk)
	fmt.Println("ETH address:", add.String())

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

	fmt.Println("pubk:", (*czzec.PublicKey)(&key.PublicKey).SerializeUncompressed())
	fmt.Println("pubk compressed :", pk)
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

	keyBy, _ := hex.DecodeString("3919cef699109d5fe06827f3121ae35116030913930c2d7280888c957e0195a6")
	key, _ := czzec.PrivKeyFromBytes(czzec.S256(), keyBy)
	wif, _ := czzutil.NewWIF(key, &chaincfg.MainNetParams, true)

	fmt.Println("wif:", wif.String())
	fmt.Println("priv:", hex.EncodeToString(key.Serialize()))
	pk := (*czzec.PublicKey)(&key.PublicKey).SerializeCompressed()

	fmt.Println("pubk:", hex.EncodeToString((*czzec.PublicKey)(&key.PublicKey).SerializeUncompressed()))
	fmt.Println("pubk compressed :", pk)
	fmt.Println("pub:", hex.EncodeToString(pk))
	fmt.Println("pubhex:", hex.EncodeToString(key.PublicKey.Y.Bytes()))
	address, err := czzutil.NewAddressPubKeyHash(czzutil.Hash160(pk), &chaincfg.MainNetParams)

	if err != nil {
		t.Errorf("failed to make address for: %v", err)
	}
	fmt.Println("addressScript:", hex.EncodeToString(address.ScriptAddress()))
	fmt.Println("address:", address.String())

	add := crypto.Keccak256Hash(pk)
	fmt.Println("ETH address:", add.String())
}

func TestWIFConvertAddr(t *testing.T) {

	wif, _ := czzutil.DecodeWIF("L3HTxLD2MyH7nw6jLYx1fVHRd3aLXp4LWcdYWrXG4kTYxyuadKPh")
	key := wif.PrivKey
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

func TestDecodeAddress(t *testing.T) {

	result, version, err := base58.CheckDecode("DKcSCmjSYUTXrFNSRiLHhDeLNrt4Dgu4G1")
	fmt.Println("address: ", err, hex.EncodeToString(result), version)

}

func TestToPubKey(t *testing.T) {

	hash_1, _ := hex.DecodeString("0a020fda220892dda773a530bd1e40e88fb6c6d72e5a67080112630a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412320a1541ef44430cbb180dab83319e74f06477b17b531140121541cd4055284b1143422467d2c82ebfb712a8e708a418a08d067088edb2c6d72e")
	hash := sha256.Sum256(hash_1)

	r, _ := hex.DecodeString("eaf47de25ad21054afe5af7c40b29df89d8cef082978ceb59f49a7cdd15bb262")
	s, _ := hex.DecodeString("26163f4b605945f667c56732125ae5488d29155b362cd8319d633e679822a864")

	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = 0

	pk, err := crypto.Ecrecover(hash[:], sig)
	pk_hex := hex.EncodeToString(pk)

	fmt.Println(pk_hex)
	fmt.Println(pk, err)

	add := crypto.Keccak256Hash(pk)
	fmt.Println("address:", add.String())

}

func TestName(t *testing.T) {

	pub_str := "0409c96aed053ece67c5e95456a23ff40d17ab446d6676f394b90b7d77fd739b823d23f972d86cc5fd6cbbddd4b59e37e9f567e3fcc7322ff1e9c65721e66ec972"

	//x, y := elliptic.Unmarshal(czzec.S256(), pub)
	publ1, _ := hex.DecodeString(pub_str)
	pubk, _ := czzec.ParsePubKey(publ1, czzec.S256())

	fmt.Println(crypto.PubkeyToAddress(*pubk.ToECDSA()).String())

}

func TestNumber(t *testing.T) {

	n1 := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(6), nil)
	fmt.Println(n1)
}
