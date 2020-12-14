package cross

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/database"
	_ "github.com/classzz/classzz/database/ffldb"
	"github.com/classzz/classzz/rpcclient"
	"github.com/classzz/classzz/wire"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"io/ioutil"
	"math/big"
	"net/http"
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

func NewExChangeVerify() *ExChangeVerify {

	dbPath := filepath.Join(os.TempDir(), "examplecreate")
	db, err := database.Create("ffldb", dbPath, wire.MainNet)
	if err != nil {
		fmt.Println("NewExChangeVerify", err)
		return nil
	}

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
		fmt.Println(err)
		return nil
	}
	dogeclients = append(dogeclients, client)

	entangleVerify := &ExChangeVerify{
		Cache:       cacheEntangleInfo,
		DogeCoinRPC: dogeclients,
		Params:      &chaincfg.TestNetParams,
	}

	return entangleVerify
}

func Close() {
	dbPath := filepath.Join(os.TempDir(), "examplecreate")
	os.RemoveAll(dbPath)
	//defer db.Close()
}

// registration
func TestExChangeVerify_VerifyBeaconRegistrationTx(t *testing.T) {

	state := NewEntangleState()
	ev := NewExChangeVerify()
	defer Close()

	if err := BeaconRegistration(state, ev, false); err != nil {
		t.Error(err)
	}

	if err := BeaconRegistration_StakingAmount(state, ev, false); err != ErrStakingAmount {
		t.Error(err)
	}

}

func BeaconRegistration(state *EntangleState, ev *ExChangeVerify, store bool) error {

	whiteList := make([]*WhiteUnit, 0)
	whiteUnit1 := &WhiteUnit{
		AssetType: 240,
		Pk:        []byte{4, 166, 247, 199, 108, 33, 195, 82, 32, 221, 1, 203, 206, 58, 106, 74, 172, 110, 216, 231, 207, 202, 230, 241, 203, 183, 15, 31, 240, 85, 196, 241, 127, 97, 228, 254, 196, 138, 222, 147, 162, 36, 215, 56, 166, 232, 123, 245, 173, 55, 160, 181, 72, 48, 173, 91, 216, 12, 162, 216, 229, 8, 30, 3, 153},
	}
	whiteList = append(whiteList, whiteUnit1)
	bai := &BeaconAddressInfo{
		Address:         "cz0t66qeut9mh2ha63s3x45ezymls55dpgjgz0na50",
		ToAddress:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20},
		PubKey:          []byte{3, 232, 11, 254, 202, 208, 40, 141, 66, 37, 57, 68, 219, 195, 131, 115, 104, 60, 5, 107, 150, 29, 115, 14, 116, 205, 200, 248, 126, 207, 90, 112, 201},
		StakingAmount:   new(big.Int).Mul(big.NewInt(500), big.NewInt(1e8)),
		EnAssets:        nil,
		Frees:           nil,
		AssetFlag:       63,
		Fee:             200,
		KeepBlock:       200,
		WhiteList:       whiteList,
		CoinBaseAddress: []string{"crmf3kkfmudmer2sm3zc3fnwlcydkx8eeqaedxw563"},
	}

	if err := ev.VerifyBeaconRegistrationTx(bai, state); err != nil {
		return err
	}

	if store {
		var err error
		if state != nil {
			err = state.RegisterBeaconAddress(bai.Address, bai.ToAddress, bai.PubKey, bai.StakingAmount, bai.Fee, bai.KeepBlock, bai.AssetFlag, bai.WhiteList, bai.CoinBaseAddress)
		}
		if err != nil {
			return err
		}
		beaconID := state.GetBeaconIdByTo(bai.ToAddress)
		if exInfos := state.GetBaExInfoByID(beaconID); exInfos != nil {
			ex := &ExBeaconInfo{
				EnItems: []*wire.OutPoint{&wire.OutPoint{
					Hash:  chainhash.Hash{1},
					Index: 1,
				}},
				Proofs: []*WhiteListProof{},
			}
			state.SetBaExInfo(beaconID, ex)
		} else {
			return errors.New(fmt.Sprintf("beacon merge failed,exInfo not nil,id:%v", beaconID))
		}
		return nil
	}

	return nil
}

func BeaconRegistration_StakingAmount(state *EntangleState, ev *ExChangeVerify, store bool) error {

	whiteList := make([]*WhiteUnit, 0)
	whiteUnit1 := &WhiteUnit{
		AssetType: 240,
		Pk:        []byte{4, 166, 247, 199, 108, 33, 195, 82, 32, 221, 1, 203, 206, 58, 106, 74, 172, 110, 216, 231, 207, 202, 230, 241, 203, 183, 15, 31, 240, 85, 196, 241, 127, 97, 228, 254, 196, 138, 222, 147, 162, 36, 215, 56, 166, 232, 123, 245, 173, 55, 160, 181, 72, 48, 173, 91, 216, 12, 162, 216, 229, 8, 30, 3, 153},
	}
	whiteList = append(whiteList, whiteUnit1)
	bai := &BeaconAddressInfo{
		Address:         "cz0t66qeut9mh2ha63s3x45ezymls55dpgjgz0na50",
		ToAddress:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20},
		PubKey:          []byte{3, 232, 11, 254, 202, 208, 40, 141, 66, 37, 57, 68, 219, 195, 131, 115, 104, 60, 5, 107, 150, 29, 115, 14, 116, 205, 200, 248, 126, 207, 90, 112, 201},
		StakingAmount:   new(big.Int).Mul(big.NewInt(10), big.NewInt(1e8)),
		EnAssets:        nil,
		Frees:           nil,
		AssetFlag:       63,
		Fee:             200,
		KeepBlock:       200,
		WhiteList:       whiteList,
		CoinBaseAddress: []string{"crmf3kkfmudmer2sm3zc3fnwlcydkx8eeqaedxw563"},
	}

	if err := ev.VerifyBeaconRegistrationTx(bai, state); err != nil {
		return err
	}

	if store {
		var err error
		if state != nil {
			err = state.RegisterBeaconAddress(bai.Address, bai.ToAddress, bai.PubKey, bai.StakingAmount, bai.Fee, bai.KeepBlock, bai.AssetFlag, bai.WhiteList, bai.CoinBaseAddress)
		}
		if err != nil {
			return err
		}
		beaconID := state.GetBeaconIdByTo(bai.ToAddress)
		if exInfos := state.GetBaExInfoByID(beaconID); exInfos != nil {
			ex := &ExBeaconInfo{
				EnItems: []*wire.OutPoint{&wire.OutPoint{
					Hash:  chainhash.Hash{1},
					Index: 1,
				}},
				Proofs: []*WhiteListProof{},
			}
			state.SetBaExInfo(beaconID, ex)
		} else {
			return errors.New(fmt.Sprintf("beacon merge failed,exInfo not nil,id:%v", beaconID))
		}
		return nil
	}

	return nil
}

func TestExChangeVerify_VerifyExChangeTx(t *testing.T) {

	state := NewEntangleState()
	ev := NewExChangeVerify()
	defer Close()

	if err := BeaconRegistration(state, ev, true); err != nil {
		t.Error(err)
	}

	if err := ExChangeTx(state, ev, true); err != nil {
		t.Error(err)
	}

}

func ExChangeTx(state *EntangleState, ev *ExChangeVerify, store bool) error {
	info := &ExChangeTxInfo{
		Address:   "cr3k7enc4ha867mx3k2y0t9x8u9z9wc5agjd64p3ea",
		AssetType: 240,
		Index:     0,
		Height:    3295847,
		Amount:    big.NewInt(10000000000),
		ExtTxHash: "bd0f745c9fedb60bd1e199af80849b72613c15000c514e2003285198a3997d36",
		BeaconID:  1,
	}

	if _, err := ev.verifyTx(info, state); err != nil {
		return err
	}

	//height := big.NewInt(int64(info.Height))
	//_, err := state.AddEntangleItem(info.Address, uint8(info.AssetType), info.BeaconID, height, info.Amount)
	//if err != nil {
	//	return err
	//}

	return nil
}

func TestExChangeVerify_VerifyBurn(t *testing.T) {

	state := NewEntangleState()
	ev := NewExChangeVerify()
	defer Close()

	if err := BeaconRegistration(state, ev, true); err != nil {
		t.Error(err)
	}

	if err := ExChangeTx(state, ev, true); err != nil {
		t.Error(err)
	}

	if err := BurnTx(state, ev, true); err != nil {
		t.Error(err)
	}
}

func BurnTx(state *EntangleState, ev *ExChangeVerify, store bool) error {
	//info := &BurnTxInfo{
	//	Address:   "cr3k7enc4ha867mx3k2y0t9x8u9z9wc5agjd64p3ea",
	//	AssetType: 240,
	//	Height:    30,
	//	Amount:    big.NewInt(10),
	//	BeaconID:  1,
	//}

	//if err := ev.VerifyBurn(info, state); err != nil {
	//	return err
	//}
	//
	//if store {
	//	if _, _, err := state.BurnAsset(info.Address, uint8(info.AssetType), info.BeaconID, 30, info.Amount); err != nil {
	//		return err
	//	}
	//}

	return nil
}

func TestExChangeVerify_VerifyBurnProofBeacon(t *testing.T) {
	state := NewEntangleState()
	ev := NewExChangeVerify()
	defer Close()

	if err := BeaconRegistration(state, ev, true); err != nil {
		t.Error(err)
	}

	if err := ExChangeTx(state, ev, true); err != nil {
		t.Error(err)
	}

	if err := BurnTx(state, ev, true); err != nil {
		t.Error(err)
	}

	if err := BurnProofTxBeacon(state, ev, true); err != nil {
		t.Error(err)
	}

}

func BurnProofTxBeacon(state *EntangleState, ev *ExChangeVerify, store bool) error {

	info := &BurnProofInfo{
		BeaconID:  1,
		Height:    30,
		Amount:    big.NewInt(250),
		Address:   "cr3k7enc4ha867mx3k2y0t9x8u9z9wc5agjd64p3ea",
		AssetType: 240,
		TxHash:    "28c1f349d687888f408392bb5a930852a5899502f98702d25d3786780134cd90",
		OutIndex:  0,
	}

	if _, _, err := ev.VerifyBurnProofBeacon(info, state, 30); err != nil {
		return err
	}

	if store {
		state.FinishHandleUserBurn(info, &BurnProofItem{
			Height: 30,
			TxHash: info.TxHash,
		})
	}

	return nil
}

func TestTrx(t *testing.T) {

	data := make(map[string]interface{})
	data["value"] = "d0807adb3c5412aa150787b944c96ee898c997debdc27e2f6a643c771edb5933"
	data["visible"] = true
	bytesData, _ := json.Marshal(data)

	// Get the current block count.
	// {"value":"d0807adb3c5412aa150787b944c96ee898c997debdc27e2f6a643c771edb5933","visible":true}
	// bytes.NewReader(bytesData)

	resp, err := http.Post("https://api.trongrid.io/wallet/gettransactionbyid", "application/json", bytes.NewReader(bytesData))
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)

	body, _ := ioutil.ReadAll(resp.Body)

	trxTx := &TrxTx{}
	fmt.Println(string(body))
	json.Unmarshal(body, trxTx)
	fmt.Println(trxTx.RawData)

}

func TestEth(t *testing.T) {

	client, err := rpc.Dial("http://cloud.tocloud.link:18545")
	if err != nil {
		t.Error(err)
	}

	var result hexutil.Uint64
	if err := client.Call(&result, "eth_blockNumber"); err != nil {
		t.Error(err)
	}

	fmt.Println("blockNumber", result)

}
