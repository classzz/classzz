package rpcclient

import (
	"encoding/json"
	"fmt"
	"github.com/classzz/classzz/chaincfg/chainhash"
	"testing"
)

func Test_usdt(t *testing.T) {

	// Connect to local bitcoin core RPC server using HTTP POST mode.
	connCfg := &ConnConfig{
		Host:         "localhost:8335",
		Endpoint:     "ws",
		User:         "root",
		Pass:         "admin",
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := New(connCfg, nil)
	if err != nil {
		t.Error(err)
	}

	txhash, _ := chainhash.NewHashFromStr("5ed3694e8a4fa8d3ec5c75eb6789492c69e65511522b220e94ab51da2b6dd53f")
	result, err := client.OmniGetTransactionResult(txhash)
	if err != nil {
		t.Error(err)
	}

	re_byte, _ := json.Marshal(result)
	fmt.Println(string(re_byte))

}
