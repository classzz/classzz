package cross

import (
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

type trxTx struct {
	Ret        []trxState `json:"ret"`
	Signature  []string   `json:"signature"`
	TxID       string     `json:"txID"`
	RawDataHex string     `json:"raw_data_hex"`
	RawData    string     `json:"raw_data"`
}

type trxState struct {
	ContractRet string `json:"contractRet"`
}

type Parameter_value struct {
	amount       int64
	assetName    string
	ownerAddress string
	toAddress    string
}

type Parameter struct {
	parameterValue Parameter_value
	typeUrl        string
}

type Contract struct {
	parameter Parameter
	type_     string
}

type trxRawData struct {
	contract []Contract
}
