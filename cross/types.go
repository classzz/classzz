package cross

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

type TrxBlock struct {
	block []*TrxBlock_
}

type TrxBlock_ struct {
	BlockID     string          `json:"blockID"`
	BlockHeader *TrxBlockHeader `json:"block_header"`
}

type TrxBlockHeader struct {
	RawData          *TrxBlockRawData `json:"raw_data"`
	WitnessSignature string           `json:"witness_signature"`
}

type TrxBlockRawData struct {
	Number         uint64 `json:"number"`
	TxTrieRoot     string `json:"txTrieRoot"`
	WitnessAddress string `json:"witness_address"`
	ParentHash     string `json:"parentHash"`
	Version        uint64 `json:"version"`
	Timestamp      uint64 `json:"timestamp"`
}

type TrxTx struct {
	Ret        []*trxState `json:"ret"`
	Signature  []string    `json:"signature"`
	TxID       string      `json:"txID"`
	RawDataHex string      `json:"raw_data_hex"`
	RawData    *TrxRawData `json:"raw_data"`
}

type trxState struct {
	ContractRet string `json:"contractRet"`
}

type Parameter_value struct {
	Amount       int64  `json:"amount"`
	AssetName    string `json:"asset_name"`
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
}

type Parameter struct {
	ParameterValue *Parameter_value `json:"value"`
	TypeUrl        string           `json:"type_url"`
}

type Contract struct {
	Parameter *Parameter `json:"parameter"`
	Type_     string     `json:"type"`
}

type TrxRawData struct {
	Contract      []*Contract `json:"contract"`
	RefBlockBytes string      `json:"ref_block_bytes"`
	RefBlockHash  string      `json:"ref_block_hash"`
	Expiration    uint64      `json:"expiration"`
	Timestamp     uint64      `json:"timestamp"`
}
