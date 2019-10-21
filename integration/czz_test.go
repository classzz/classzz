package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/classzz/integration/rpctest"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/czzutil"
)

var (
	minAddress, _ = czzutil.DecodeAddress("czp5g27p3lz02astuyrnzd0sm90gh4280g3hgr2l0t", &chaincfg.MainNetParams)
	pubKey        = "02cd77593671ecaac86f942ac99cccaa53810bb23d7b8dd38610b068d388cbd899"
	privKey       = "bcd7220fae4f1fcff9bb6d9fd7861c880e0c522abfaa3a37ab17dad512a54885"
)

func CreateBlocks(count int, address czzutil.Address) ([]*czzutil.Block, error) {
	if count <= 0 {
		return nil, nil
	}
	chainParams := chaincfg.MainNetParams
	genesisBlock := czzutil.NewBlock(chainParams.GenesisBlock)
	var nullTime time.Time

	blocks := make([]*czzutil.Block, 0, count)
	blockVersion := int32(2)
	prevBlock := genesisBlock
	for i := 0; i < count; i++ {
		block, err := rpctest.CreateBlock(prevBlock, nil, blockVersion,
			nullTime, address, []wire.TxOut{}, &chainParams)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
		prevBlock = block
	}
	return blocks, nil
}

func TestCZZ1(t *testing.T) {
	address := minAddress
	_, err := CreateBlocks(5, address)
	if err != nil {
		fmt.Println(err)
	}

}

