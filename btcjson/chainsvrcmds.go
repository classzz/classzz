// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// NOTE: This file is intended to house the RPC commands that are supported by
// a chain server.

package btcjson

import (
	"encoding/json"
	"fmt"
	"github.com/classzz/classzz/wire"
	"math/big"
)

// AddNodeSubCmd defines the type used in the addnode JSON-RPC command for the
// sub command field.
type AddNodeSubCmd string

const (
	// ANAdd indicates the specified host should be added as a persistent
	// peer.
	ANAdd AddNodeSubCmd = "add"

	// ANRemove indicates the specified peer should be removed.
	ANRemove AddNodeSubCmd = "remove"

	// ANOneTry indicates the specified host should try to connect once,
	// but it should not be made persistent.
	ANOneTry AddNodeSubCmd = "onetry"
)

// AddNodeCmd defines the addnode JSON-RPC command.
type AddNodeCmd struct {
	Addr   string
	SubCmd AddNodeSubCmd `jsonrpcusage:"\"add|remove|onetry\""`
}

// NewAddNodeCmd returns a new instance which can be used to issue an addnode
// JSON-RPC command.
func NewAddNodeCmd(addr string, subCmd AddNodeSubCmd) *AddNodeCmd {
	return &AddNodeCmd{
		Addr:   addr,
		SubCmd: subCmd,
	}
}

// TransactionInput represents the inputs to a transaction.  Specifically a
// transaction hash and output number pair.
type TransactionInput struct {
	Txid string `json:"txid"`
	Vout uint32 `json:"vout"`
}

// CreateRawTransactionCmd defines the createrawtransaction JSON-RPC command.
type CreateRawTransactionCmd struct {
	Inputs   []TransactionInput
	Amounts  map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime *int64
}

type ExChangeOut struct {
	Address   string         `json:"address"`
	ExTxType  ExpandedTxType `json:"extxtype"`
	Index     uint32         `json:"index"`
	Height    uint64         `json:"height"`
	Amount    *big.Int       `json:"amount"`
	ExtTxHash string         `json:"exttxhash"`
	BID       uint64         `json:"bid"`
}

type WhiteUnit struct {
	AssetType uint32 `json:"asset_type"`
	Pk        []byte `json:"pk"`
}

type BaseAmountUint struct {
	AssetType uint32   `json:"asset_type"`
	Amount    *big.Int `json:"amount"`
}

type EnAssetItem BaseAmountUint
type FreeQuotaItem BaseAmountUint

type BeaconRegistrationOut struct {
	ToAddress       []byte
	StakingAmount   float64
	AssetFlag       uint32
	Fee             uint64
	KeepTime        uint64 // the time as the block count for finally redeem time
	WhiteList       []*WhiteUnit
	CoinBaseAddress []string
}

type AddBeaconPledgeOut struct {
	Address       string
	ToAddress     []byte
	StakingAmount float64
}

type AddBeaconCoinbaseOut struct {
	Address         string
	ToAddress       []byte
	CoinBaseAddress []string
}

type BurnTransactionOut struct {
	ExTxType uint8
	Address  string
	LightID  uint64
	Amount   float64
}

type BurnProofOut struct {
	LightID  uint64   // the lightid for beaconAddress of user burn's asset
	Height   uint64   // the height include the tx of user burn's asset
	Amount   *big.Int // the amount of burned asset (czz)
	Address  string
	Atype    uint32
	TxHash   string // the tx hash of outside
	OutIndex uint64
	IsBeacon bool
}

type BurnReportWhiteListOut struct {
	LightID  uint64 // the lightid for beaconAddress
	Atype    uint32
	Height   uint64 // the height of outside chain
	TxHash   string
	InIndex  int64
	OutIndex int64
	Amount   *big.Int // the amount of outside chain
}

// ExChangeTransaction defines the CreateRawExChangeTransactionCmd JSON-RPC command.
type ExChangeTransactionCmd struct {
	Inputs       []TransactionInput
	ExChangeOuts []ExChangeOut
	Amounts      *map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime     *int64
}

// CreatePledgeRegistrationCmd defines JSON-RPC command.
type BeaconRegistrationCmd struct {
	Inputs             []TransactionInput
	BeaconRegistration BeaconRegistrationOut
	Amounts            *map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime           *int64
}

// AddBeaconPledge defines JSON-RPC command.
type AddBeaconPledgeCmd struct {
	Inputs          []TransactionInput
	AddBeaconPledge AddBeaconPledgeOut
	Amounts         *map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime        *int64
}

// AddBeaconCoinbase defines JSON-RPC command.
type AddBeaconCoinbaseCmd struct {
	Inputs            []TransactionInput
	AddBeaconCoinbase AddBeaconCoinbaseOut
	Amounts           *map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime          *int64
}

// BurnTransaction defines JSON-RPC command.
type BurnTransactionCmd struct {
	Inputs          []TransactionInput
	BurnTransaction BurnTransactionOut
	Amounts         *map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime        *int64
}

type BurnProofCmd struct {
	Inputs    []TransactionInput
	BurnProof BurnProofOut
	Amounts   *map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime  *int64
}

type BurnReportCmd struct {
	Inputs    []TransactionInput
	BurnProof BurnProofOut
	Amounts   *map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime  *int64
}

type BurnReportWhiteListCmd struct {
	Inputs    []TransactionInput
	BurnProof BurnReportWhiteListOut
	Amounts   *map[string]float64 `jsonrpcusage:"{\"address\":amount,...}"`
	LockTime  *int64
}

func (w *WhiteUnit) toAddress() string {
	// pk to czz address
	return ""
}

type BeaconAddressInfo struct {
	ExchangeID      uint64           `json:"exchange_id"`
	Address         string           `json:"address"`
	PubKey          []byte           `json:"pub_key"`
	ToAddress       []byte           `json:"toAddress"`
	StakingAmount   *big.Int         `json:"staking_amount"`  // in
	EntangleAmount  *big.Int         `json:"entangle_amount"` // out,express by czz,all amount of user's entangle
	EnAssets        []*EnAssetItem   `json:"en_assets"`       // out,the extrinsic asset
	Frees           []*FreeQuotaItem `json:"frees"`           // extrinsic asset
	AssetFlag       uint32           `json:"asset_flag"`
	Fee             uint64           `json:"fee"`
	KeepTime        uint64           `json:"keep_time"` // the time as the block count for finally redeem time
	WhiteList       []*WhiteUnit     `json:"white_list"`
	CoinBaseAddress []string         `json:"coinbase_address"`
}

type AddBeaconPledge struct {
	Address       string   `json:"address"`
	ToAddress     []byte   `json:"to_address"`
	StakingAmount *big.Int `json:"staking_amount"`
}

type AddBeaconCoinbase struct {
	Address         string   `json:"address"`
	ToAddress       []byte   `json:"to_address"`
	CoinBaseAddress []string `json:"coinbase_address"`
}

type ExpandedTxType uint8

type BurnInfo struct {
	ExTxType ExpandedTxType
	Address  string
	LightID  uint64
	Amount   *big.Int
}

type BurnProofInfo struct {
	LightID  uint64   // the lightid for beaconAddress of user burn's asset
	Height   uint64   // the height include the tx of user burn's asset
	Amount   *big.Int // the amount of burned asset (czz)
	Address  string
	Atype    uint32
	TxHash   string // the tx hash of outside
	OutIndex uint64
	IsBeacon bool
}

// NewCreateRawTransactionCmd returns a new instance which can be used to issue
// a createrawtransaction JSON-RPC command.
//
// Amounts are in BTC.
func NewCreateRawTransactionCmd(inputs []TransactionInput, amounts map[string]float64,
	lockTime *int64) *CreateRawTransactionCmd {

	return &CreateRawTransactionCmd{
		Inputs:   inputs,
		Amounts:  amounts,
		LockTime: lockTime,
	}
}

// NewCreateRawTransactionCmd returns a new instance which can be used to issue
// a createrawtransaction JSON-RPC command.
//
// Amounts are in BTC.
func NewExChangeTransactionCmd(inputs []TransactionInput, entangleOuts []ExChangeOut, amounts *map[string]float64,
	lockTime *int64) *ExChangeTransactionCmd {
	return &ExChangeTransactionCmd{
		Inputs:       inputs,
		ExChangeOuts: entangleOuts,
		Amounts:      amounts,
		LockTime:     lockTime,
	}
}

// NewCreateRawTransactionCmd returns a new instance which can be used to issue
// a createrawtransaction JSON-RPC command.
//
// Amounts are in BTC.
func NewBeaconRegistrationCmd(inputs []TransactionInput, beaconRegistrationOut BeaconRegistrationOut, amounts *map[string]float64,
	lockTime *int64) *BeaconRegistrationCmd {
	return &BeaconRegistrationCmd{
		Inputs:             inputs,
		BeaconRegistration: beaconRegistrationOut,
		Amounts:            amounts,
		LockTime:           lockTime,
	}
}

// NewCreateRawTransactionCmd returns a new instance which can be used to issue
// a createrawtransaction JSON-RPC command.
//
// Amounts are in BTC.
func NewAddBeaconPledgeCmd(inputs []TransactionInput, addBeaconPledgeOut AddBeaconPledgeOut, amounts *map[string]float64,
	lockTime *int64) *AddBeaconPledgeCmd {
	return &AddBeaconPledgeCmd{
		Inputs:          inputs,
		AddBeaconPledge: addBeaconPledgeOut,
		Amounts:         amounts,
		LockTime:        lockTime,
	}
}

// NewCreateRawTransactionCmd returns a new instance which can be used to issue
// a createrawtransaction JSON-RPC command.
//
// Amounts are in BTC.
func NewAddBeaconCoinbaseCmd(inputs []TransactionInput, outs AddBeaconCoinbaseOut, amounts *map[string]float64,
	lockTime *int64) *AddBeaconCoinbaseCmd {
	return &AddBeaconCoinbaseCmd{
		Inputs:            inputs,
		AddBeaconCoinbase: outs,
		Amounts:           amounts,
		LockTime:          lockTime,
	}
}

func NewBurnTransactionCmd(inputs []TransactionInput, out BurnTransactionOut, amounts *map[string]float64,
	lockTime *int64) *BurnTransactionCmd {
	return &BurnTransactionCmd{
		Inputs:          inputs,
		BurnTransaction: out,
		Amounts:         amounts,
		LockTime:        lockTime,
	}
}

// DecodeRawTransactionCmd defines the decoderawtransaction JSON-RPC command.
type DecodeRawTransactionCmd struct {
	HexTx string
}

// NewDecodeRawTransactionCmd returns a new instance which can be used to issue
// a decoderawtransaction JSON-RPC command.
func NewDecodeRawTransactionCmd(hexTx string) *DecodeRawTransactionCmd {
	return &DecodeRawTransactionCmd{
		HexTx: hexTx,
	}
}

// DecodeScriptCmd defines the decodescript JSON-RPC command.
type DecodeScriptCmd struct {
	HexScript string
}

// NewDecodeScriptCmd returns a new instance which can be used to issue a
// decodescript JSON-RPC command.
func NewDecodeScriptCmd(hexScript string) *DecodeScriptCmd {
	return &DecodeScriptCmd{
		HexScript: hexScript,
	}
}

// GetAddedNodeInfoCmd defines the getaddednodeinfo JSON-RPC command.
type GetAddedNodeInfoCmd struct {
	DNS  bool
	Node *string
}

// NewGetAddedNodeInfoCmd returns a new instance which can be used to issue a
// getaddednodeinfo JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetAddedNodeInfoCmd(dns bool, node *string) *GetAddedNodeInfoCmd {
	return &GetAddedNodeInfoCmd{
		DNS:  dns,
		Node: node,
	}
}

// GetBestBlockHashCmd defines the getbestblockhash JSON-RPC command.
type GetBestBlockHashCmd struct{}

// NewGetBestBlockHashCmd returns a new instance which can be used to issue a
// getbestblockhash JSON-RPC command.
func NewGetBestBlockHashCmd() *GetBestBlockHashCmd {
	return &GetBestBlockHashCmd{}
}

// GetBlockCmd defines the getblock JSON-RPC command.
type GetBlockCmd struct {
	Hash      string
	Verbosity *uint32 `jsonrpcdefault:"1"`
}

// NewGetBlockCmd returns a new instance which can be used to issue a getblock
// JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetBlockCmd(hash string, verbosity *uint32) *GetBlockCmd {
	return &GetBlockCmd{
		Hash:      hash,
		Verbosity: verbosity,
	}
}

// GetBlockCmd defines the getblock JSON-RPC command.
type GetDogecoinBlockCmd struct {
	Hash string
}

func NewGetDogecoinBlockCmd(hash string) *GetDogecoinBlockCmd {
	return &GetDogecoinBlockCmd{
		Hash: hash,
	}
}

// GetBlockChainInfoCmd defines the getblockchaininfo JSON-RPC command.
type GetBlockChainInfoCmd struct{}

// NewGetBlockChainInfoCmd returns a new instance which can be used to issue a
// getblockchaininfo JSON-RPC command.
func NewGetBlockChainInfoCmd() *GetBlockChainInfoCmd {
	return &GetBlockChainInfoCmd{}
}

// GetBlockCountCmd defines the getblockcount JSON-RPC command.
type GetBlockCountCmd struct{}

// NewGetBlockCountCmd returns a new instance which can be used to issue a
// getblockcount JSON-RPC command.
func NewGetBlockCountCmd() *GetBlockCountCmd {
	return &GetBlockCountCmd{}
}

// GetBlockHashCmd defines the getblockhash JSON-RPC command.
type GetBlockHashCmd struct {
	Index int64
}

// NewGetBlockHashCmd returns a new instance which can be used to issue a
// getblockhash JSON-RPC command.
func NewGetBlockHashCmd(index int64) *GetBlockHashCmd {
	return &GetBlockHashCmd{
		Index: index,
	}
}

// GetBlockHeaderCmd defines the getblockheader JSON-RPC command.
type GetBlockHeaderCmd struct {
	Hash    string
	Verbose *bool `jsonrpcdefault:"true"`
}

// NewGetBlockHeaderCmd returns a new instance which can be used to issue a
// getblockheader JSON-RPC command.
func NewGetBlockHeaderCmd(hash string, verbose *bool) *GetBlockHeaderCmd {
	return &GetBlockHeaderCmd{
		Hash:    hash,
		Verbose: verbose,
	}
}

// TemplateRequest is a request object as defined in BIP22
// (https://en.bitcoin.it/wiki/BIP_0022), it is optionally provided as an
// pointer argument to GetBlockTemplateCmd.
type TemplateRequest struct {
	Mode         string   `json:"mode,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`

	// Optional long polling.
	LongPollID string `json:"longpollid,omitempty"`

	// Optional template tweaking.  SigOpLimit and SizeLimit can be int64
	// or bool.
	SigOpLimit interface{} `json:"sigoplimit,omitempty"`
	SizeLimit  interface{} `json:"sizelimit,omitempty"`
	MaxVersion uint32      `json:"maxversion,omitempty"`

	// Basic pool extension from BIP 0023.
	Target string `json:"target,omitempty"`

	// Block proposal from BIP 0023.  Data is only provided when Mode is
	// "proposal".
	Data   string `json:"data,omitempty"`
	WorkID string `json:"workid,omitempty"`
}

// convertTemplateRequestField potentially converts the provided value as
// needed.
func convertTemplateRequestField(fieldName string, iface interface{}) (interface{}, error) {
	switch val := iface.(type) {
	case nil:
		return nil, nil
	case bool:
		return val, nil
	case float64:
		if val == float64(int64(val)) {
			return int64(val), nil
		}
	}

	str := fmt.Sprintf("the %s field must be unspecified, a boolean, or "+
		"a 64-bit integer", fieldName)
	return nil, makeError(ErrInvalidType, str)
}

// UnmarshalJSON provides a custom Unmarshal method for TemplateRequest.  This
// is necessary because the SigOpLimit and SizeLimit fields can only be specific
// types.
func (t *TemplateRequest) UnmarshalJSON(data []byte) error {
	type templateRequest TemplateRequest

	request := (*templateRequest)(t)
	if err := json.Unmarshal(data, &request); err != nil {
		return err
	}

	// The SigOpLimit field can only be nil, bool, or int64.
	val, err := convertTemplateRequestField("sigoplimit", request.SigOpLimit)
	if err != nil {
		return err
	}
	request.SigOpLimit = val

	// The SizeLimit field can only be nil, bool, or int64.
	val, err = convertTemplateRequestField("sizelimit", request.SizeLimit)
	if err != nil {
		return err
	}
	request.SizeLimit = val

	return nil
}

// GetBlockTemplateCmd defines the getblocktemplate JSON-RPC command.
type GetBlockTemplateCmd struct {
	Request *TemplateRequest
}

// NewGetBlockTemplateCmd returns a new instance which can be used to issue a
// getblocktemplate JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetBlockTemplateCmd(request *TemplateRequest) *GetBlockTemplateCmd {
	return &GetBlockTemplateCmd{
		Request: request,
	}
}

// GetCFilterCmd defines the getcfilter JSON-RPC command.
type GetCFilterCmd struct {
	Hash       string
	FilterType wire.FilterType
}

// NewGetCFilterCmd returns a new instance which can be used to issue a
// getcfilter JSON-RPC command.
func NewGetCFilterCmd(hash string, filterType wire.FilterType) *GetCFilterCmd {
	return &GetCFilterCmd{
		Hash:       hash,
		FilterType: filterType,
	}
}

// GetCFilterHeaderCmd defines the getcfilterheader JSON-RPC command.
type GetCFilterHeaderCmd struct {
	Hash       string
	FilterType wire.FilterType
}

// NewGetCFilterHeaderCmd returns a new instance which can be used to issue a
// getcfilterheader JSON-RPC command.
func NewGetCFilterHeaderCmd(hash string,
	filterType wire.FilterType) *GetCFilterHeaderCmd {
	return &GetCFilterHeaderCmd{
		Hash:       hash,
		FilterType: filterType,
	}
}

// GetChainTipsCmd defines the getchaintips JSON-RPC command.
type GetChainTipsCmd struct{}

// NewGetChainTipsCmd returns a new instance which can be used to issue a
// getchaintips JSON-RPC command.
func NewGetChainTipsCmd() *GetChainTipsCmd {
	return &GetChainTipsCmd{}
}

// GetConnectionCountCmd defines the getconnectioncount JSON-RPC command.
type GetConnectionCountCmd struct{}

// NewGetConnectionCountCmd returns a new instance which can be used to issue a
// getconnectioncount JSON-RPC command.
func NewGetConnectionCountCmd() *GetConnectionCountCmd {
	return &GetConnectionCountCmd{}
}

// GetDifficultyCmd defines the getdifficulty JSON-RPC command.
type GetDifficultyCmd struct{}

// NewGetDifficultyCmd returns a new instance which can be used to issue a
// getdifficulty JSON-RPC command.
func NewGetDifficultyCmd() *GetDifficultyCmd {
	return &GetDifficultyCmd{}
}

// GetGenerateCmd defines the getgenerate JSON-RPC command.
type GetGenerateCmd struct{}

// NewGetGenerateCmd returns a new instance which can be used to issue a
// getgenerate JSON-RPC command.
func NewGetGenerateCmd() *GetGenerateCmd {
	return &GetGenerateCmd{}
}

// GetHashesPerSecCmd defines the gethashespersec JSON-RPC command.
type GetHashesPerSecCmd struct{}

// NewGetHashesPerSecCmd returns a new instance which can be used to issue a
// gethashespersec JSON-RPC command.
func NewGetHashesPerSecCmd() *GetHashesPerSecCmd {
	return &GetHashesPerSecCmd{}
}

// GetInfoCmd defines the getinfo JSON-RPC command.
type GetInfoCmd struct{}

// NewGetInfoCmd returns a new instance which can be used to issue a
// getinfo JSON-RPC command.
func NewGetInfoCmd() *GetInfoCmd {
	return &GetInfoCmd{}
}

// GetInfoCmd defines the getinfo JSON-RPC command.
type GetWorkTemplateCmd struct{}

// NewGetInfoCmd returns a new instance which can be used to issue a
// getinfo JSON-RPC command.
func NewGetWorkTemplateCmd() *GetWorkTemplateCmd {
	return &GetWorkTemplateCmd{}
}

// GetMempoolEntryCmd defines the getmempoolentry JSON-RPC command.
type GetMempoolEntryCmd struct {
	TxID string
}

// NewGetMempoolEntryCmd returns a new instance which can be used to issue a
// getmempoolentry JSON-RPC command.
func NewGetMempoolEntryCmd(txHash string) *GetMempoolEntryCmd {
	return &GetMempoolEntryCmd{
		TxID: txHash,
	}
}

// GetMempoolInfoCmd defines the getmempoolinfo JSON-RPC command.
type GetMempoolInfoCmd struct{}

// NewGetMempoolInfoCmd returns a new instance which can be used to issue a
// getmempool JSON-RPC command.
func NewGetMempoolInfoCmd() *GetMempoolInfoCmd {
	return &GetMempoolInfoCmd{}
}

// GetMiningInfoCmd defines the getmininginfo JSON-RPC command.
type GetMiningInfoCmd struct{}

// NewGetMiningInfoCmd returns a new instance which can be used to issue a
// getmininginfo JSON-RPC command.
func NewGetMiningInfoCmd() *GetMiningInfoCmd {
	return &GetMiningInfoCmd{}
}

// GetNetworkInfoCmd defines the getnetworkinfo JSON-RPC command.
type GetNetworkInfoCmd struct{}

// NewGetNetworkInfoCmd returns a new instance which can be used to issue a
// getnetworkinfo JSON-RPC command.
func NewGetNetworkInfoCmd() *GetNetworkInfoCmd {
	return &GetNetworkInfoCmd{}
}

// GetNetTotalsCmd defines the getnettotals JSON-RPC command.
type GetNetTotalsCmd struct{}

// NewGetNetTotalsCmd returns a new instance which can be used to issue a
// getnettotals JSON-RPC command.
func NewGetNetTotalsCmd() *GetNetTotalsCmd {
	return &GetNetTotalsCmd{}
}

// GetNetworkHashPSCmd defines the getnetworkhashps JSON-RPC command.
type GetNetworkHashPSCmd struct {
	Blocks *int `jsonrpcdefault:"120"`
	Height *int `jsonrpcdefault:"-1"`
}

// NewGetNetworkHashPSCmd returns a new instance which can be used to issue a
// getnetworkhashps JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetNetworkHashPSCmd(numBlocks, height *int) *GetNetworkHashPSCmd {
	return &GetNetworkHashPSCmd{
		Blocks: numBlocks,
		Height: height,
	}
}

// GetPeerInfoCmd defines the getpeerinfo JSON-RPC command.
type GetPeerInfoCmd struct{}

// NewGetPeerInfoCmd returns a new instance which can be used to issue a getpeer
// JSON-RPC command.
func NewGetPeerInfoCmd() *GetPeerInfoCmd {
	return &GetPeerInfoCmd{}
}

// GetPeerInfoCmd defines the getpeerinfo JSON-RPC command.
type GetEntangleInfoCmd struct{}

// NewGetPeerInfoCmd returns a new instance which can be used to issue a getpeer
// JSON-RPC command.
func NewGetEntangleInfoCmd() *GetEntangleInfoCmd {
	return &GetEntangleInfoCmd{}
}

// GetPeerInfoCmd defines the getpeerinfo JSON-RPC command.
type GetStateInfoCmd struct{}

// NewGetPeerInfoCmd returns a new instance which can be used to issue a getpeer
// JSON-RPC command.
func NewGetStateInfoCmd() *GetStateInfoCmd {
	return &GetStateInfoCmd{}
}

// GetRawMempoolCmd defines the getmempool JSON-RPC command.
type GetRawMempoolCmd struct {
	Verbose *bool `jsonrpcdefault:"false"`
}

// NewGetRawMempoolCmd returns a new instance which can be used to issue a
// getrawmempool JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetRawMempoolCmd(verbose *bool) *GetRawMempoolCmd {
	return &GetRawMempoolCmd{
		Verbose: verbose,
	}
}

// GetRawTransactionCmd defines the getrawtransaction JSON-RPC command.
//
// NOTE: This field is an int versus a bool to remain compatible with Bitcoin
// Core even though it really should be a bool.
type GetRawTransactionCmd struct {
	Txid    string
	Verbose *int `jsonrpcdefault:"0"`
}

// NewGetRawTransactionCmd returns a new instance which can be used to issue a
// getrawtransaction JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetRawTransactionCmd(txHash string, verbose *int) *GetRawTransactionCmd {
	return &GetRawTransactionCmd{
		Txid:    txHash,
		Verbose: verbose,
	}
}

// GetTxOutCmd defines the gettxout JSON-RPC command.
type GetTxOutCmd struct {
	Txid           string
	Vout           uint32
	IncludeMempool *bool `jsonrpcdefault:"true"`
}

// NewGetTxOutCmd returns a new instance which can be used to issue a gettxout
// JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetTxOutCmd(txHash string, vout uint32, includeMempool *bool) *GetTxOutCmd {
	return &GetTxOutCmd{
		Txid:           txHash,
		Vout:           vout,
		IncludeMempool: includeMempool,
	}
}

// GetTxOutProofCmd defines the gettxoutproof JSON-RPC command.
type GetTxOutProofCmd struct {
	TxIDs     []string
	BlockHash *string
}

// NewGetTxOutProofCmd returns a new instance which can be used to issue a
// gettxoutproof JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetTxOutProofCmd(txIDs []string, blockHash *string) *GetTxOutProofCmd {
	return &GetTxOutProofCmd{
		TxIDs:     txIDs,
		BlockHash: blockHash,
	}
}

// GetTxOutSetInfoCmd defines the gettxoutsetinfo JSON-RPC command.
type GetTxOutSetInfoCmd struct{}

// NewGetTxOutSetInfoCmd returns a new instance which can be used to issue a
// gettxoutsetinfo JSON-RPC command.
func NewGetTxOutSetInfoCmd() *GetTxOutSetInfoCmd {
	return &GetTxOutSetInfoCmd{}
}

// GetWorkCmd defines the getwork JSON-RPC command.
type GetWorkCmd struct {
	Data *string
}

// NewGetWorkCmd returns a new instance which can be used to issue a getwork
// JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewGetWorkCmd(data *string) *GetWorkCmd {
	return &GetWorkCmd{
		Data: data,
	}
}

// HelpCmd defines the help JSON-RPC command.
type HelpCmd struct {
	Command *string
}

// NewHelpCmd returns a new instance which can be used to issue a help JSON-RPC
// command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewHelpCmd(command *string) *HelpCmd {
	return &HelpCmd{
		Command: command,
	}
}

// InvalidateBlockCmd defines the invalidateblock JSON-RPC command.
type InvalidateBlockCmd struct {
	BlockHash string
}

// NewInvalidateBlockCmd returns a new instance which can be used to issue a
// invalidateblock JSON-RPC command.
func NewInvalidateBlockCmd(blockHash string) *InvalidateBlockCmd {
	return &InvalidateBlockCmd{
		BlockHash: blockHash,
	}
}

// PingCmd defines the ping JSON-RPC command.
type PingCmd struct{}

// NewPingCmd returns a new instance which can be used to issue a ping JSON-RPC
// command.
func NewPingCmd() *PingCmd {
	return &PingCmd{}
}

// PreciousBlockCmd defines the preciousblock JSON-RPC command.
type PreciousBlockCmd struct {
	BlockHash string
}

// NewPreciousBlockCmd returns a new instance which can be used to issue a
// preciousblock JSON-RPC command.
func NewPreciousBlockCmd(blockHash string) *PreciousBlockCmd {
	return &PreciousBlockCmd{
		BlockHash: blockHash,
	}
}

// ReconsiderBlockCmd defines the reconsiderblock JSON-RPC command.
type ReconsiderBlockCmd struct {
	BlockHash string
}

// NewReconsiderBlockCmd returns a new instance which can be used to issue a
// reconsiderblock JSON-RPC command.
func NewReconsiderBlockCmd(blockHash string) *ReconsiderBlockCmd {
	return &ReconsiderBlockCmd{
		BlockHash: blockHash,
	}
}

// SearchRawTransactionsCmd defines the searchrawtransactions JSON-RPC command.
type SearchRawTransactionsCmd struct {
	Address     string
	Verbose     *int  `jsonrpcdefault:"1"`
	Skip        *int  `jsonrpcdefault:"0"`
	Count       *int  `jsonrpcdefault:"100"`
	VinExtra    *int  `jsonrpcdefault:"0"`
	Reverse     *bool `jsonrpcdefault:"false"`
	FilterAddrs *[]string
}

// NewSearchRawTransactionsCmd returns a new instance which can be used to issue a
// sendrawtransaction JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSearchRawTransactionsCmd(address string, verbose, skip, count *int, vinExtra *int, reverse *bool, filterAddrs *[]string) *SearchRawTransactionsCmd {
	return &SearchRawTransactionsCmd{
		Address:     address,
		Verbose:     verbose,
		Skip:        skip,
		Count:       count,
		VinExtra:    vinExtra,
		Reverse:     reverse,
		FilterAddrs: filterAddrs,
	}
}

// SendRawTransactionCmd defines the sendrawtransaction JSON-RPC command.
type SendRawTransactionCmd struct {
	HexTx         string
	AllowHighFees *bool `jsonrpcdefault:"false"`
}

// NewSendRawTransactionCmd returns a new instance which can be used to issue a
// sendrawtransaction JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSendRawTransactionCmd(hexTx string, allowHighFees *bool) *SendRawTransactionCmd {
	return &SendRawTransactionCmd{
		HexTx:         hexTx,
		AllowHighFees: allowHighFees,
	}
}

// SetGenerateCmd defines the setgenerate JSON-RPC command.
type SetGenerateCmd struct {
	Generate     bool
	GenProcLimit *int `jsonrpcdefault:"-1"`
}

// NewSetGenerateCmd returns a new instance which can be used to issue a
// setgenerate JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSetGenerateCmd(generate bool, genProcLimit *int) *SetGenerateCmd {
	return &SetGenerateCmd{
		Generate:     generate,
		GenProcLimit: genProcLimit,
	}
}

// StopCmd defines the stop JSON-RPC command.
type StopCmd struct{}

// NewStopCmd returns a new instance which can be used to issue a stop JSON-RPC
// command.
func NewStopCmd() *StopCmd {
	return &StopCmd{}
}

// SubmitBlockOptions represents the optional options struct provided with a
// SubmitBlockCmd command.
type SubmitBlockOptions struct {
	// must be provided if server provided a workid with template.
	WorkID string `json:"workid,omitempty"`
}

// SubmitBlockCmd defines the submitblock JSON-RPC command.
type SubmitBlockCmd struct {
	HexBlock string
	Options  *SubmitBlockOptions
}

// SubmitWorkCmd defines the submitblock JSON-RPC command.
type SubmitWorkCmd struct {
	Hash  string
	Nonce uint64
}

// NewSubmitBlockCmd returns a new instance which can be used to issue a
// submitblock JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSubmitBlockCmd(hexBlock string, options *SubmitBlockOptions) *SubmitBlockCmd {
	return &SubmitBlockCmd{
		HexBlock: hexBlock,
		Options:  options,
	}
}

// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewSubmitWorkCmd(Hash string, Nonce uint64) *SubmitWorkCmd {
	return &SubmitWorkCmd{
		Hash:  Hash,
		Nonce: Nonce,
	}
}

// UptimeCmd defines the uptime JSON-RPC command.
type UptimeCmd struct{}

// NewUptimeCmd returns a new instance which can be used to issue an uptime JSON-RPC command.
func NewUptimeCmd() *UptimeCmd {
	return &UptimeCmd{}
}

// ValidateAddressCmd defines the validateaddress JSON-RPC command.
type ValidateAddressCmd struct {
	Address string
}

// NewValidateAddressCmd returns a new instance which can be used to issue a
// validateaddress JSON-RPC command.
func NewValidateAddressCmd(address string) *ValidateAddressCmd {
	return &ValidateAddressCmd{
		Address: address,
	}
}

// VerifyChainCmd defines the verifychain JSON-RPC command.
type VerifyChainCmd struct {
	CheckLevel *int32 `jsonrpcdefault:"3"`
	CheckDepth *int32 `jsonrpcdefault:"288"` // 0 = all
}

// NewVerifyChainCmd returns a new instance which can be used to issue a
// verifychain JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewVerifyChainCmd(checkLevel, checkDepth *int32) *VerifyChainCmd {
	return &VerifyChainCmd{
		CheckLevel: checkLevel,
		CheckDepth: checkDepth,
	}
}

// VerifyMessageCmd defines the verifymessage JSON-RPC command.
type VerifyMessageCmd struct {
	Address   string
	Signature string
	Message   string
}

// NewVerifyMessageCmd returns a new instance which can be used to issue a
// verifymessage JSON-RPC command.
func NewVerifyMessageCmd(address, signature, message string) *VerifyMessageCmd {
	return &VerifyMessageCmd{
		Address:   address,
		Signature: signature,
		Message:   message,
	}
}

// VerifyTxOutProofCmd defines the verifytxoutproof JSON-RPC command.
type VerifyTxOutProofCmd struct {
	Proof string
}

// NewVerifyTxOutProofCmd returns a new instance which can be used to issue a
// verifytxoutproof JSON-RPC command.
func NewVerifyTxOutProofCmd(proof string) *VerifyTxOutProofCmd {
	return &VerifyTxOutProofCmd{
		Proof: proof,
	}
}

func init() {
	// No special flags for commands in this file.
	flags := UsageFlag(0)

	MustRegisterCmd("addnode", (*AddNodeCmd)(nil), flags)
	MustRegisterCmd("createrawtransaction", (*CreateRawTransactionCmd)(nil), flags)
	MustRegisterCmd("exchangetransaction", (*ExChangeTransactionCmd)(nil), flags)
	MustRegisterCmd("beaconregistration", (*BeaconRegistrationCmd)(nil), flags)
	MustRegisterCmd("addbeaconpledge", (*AddBeaconPledgeCmd)(nil), flags)
	MustRegisterCmd("addbeaconcoinbase", (*AddBeaconCoinbaseCmd)(nil), flags)
	MustRegisterCmd("burntransaction", (*BurnTransactionCmd)(nil), flags)
	MustRegisterCmd("burnprooft", (*BurnProofCmd)(nil), flags)
	MustRegisterCmd("burnreport", (*BurnReportCmd)(nil), flags)
	MustRegisterCmd("burnreportwhitelist", (*BurnReportWhiteListCmd)(nil), flags)
	MustRegisterCmd("decoderawtransaction", (*DecodeRawTransactionCmd)(nil), flags)
	MustRegisterCmd("decodescript", (*DecodeScriptCmd)(nil), flags)
	MustRegisterCmd("getaddednodeinfo", (*GetAddedNodeInfoCmd)(nil), flags)
	MustRegisterCmd("getbestblockhash", (*GetBestBlockHashCmd)(nil), flags)
	MustRegisterCmd("getblock", (*GetBlockCmd)(nil), flags)
	MustRegisterCmd("getdogeblock", (*GetDogecoinBlockCmd)(nil), flags)
	MustRegisterCmd("getblockchaininfo", (*GetBlockChainInfoCmd)(nil), flags)
	MustRegisterCmd("getblockcount", (*GetBlockCountCmd)(nil), flags)
	MustRegisterCmd("getblockhash", (*GetBlockHashCmd)(nil), flags)
	MustRegisterCmd("getblockheader", (*GetBlockHeaderCmd)(nil), flags)
	MustRegisterCmd("getblocktemplate", (*GetBlockTemplateCmd)(nil), flags)
	MustRegisterCmd("getcfilter", (*GetCFilterCmd)(nil), flags)
	MustRegisterCmd("getcfilterheader", (*GetCFilterHeaderCmd)(nil), flags)
	MustRegisterCmd("getchaintips", (*GetChainTipsCmd)(nil), flags)
	MustRegisterCmd("getconnectioncount", (*GetConnectionCountCmd)(nil), flags)
	MustRegisterCmd("getdifficulty", (*GetDifficultyCmd)(nil), flags)
	MustRegisterCmd("getgenerate", (*GetGenerateCmd)(nil), flags)
	MustRegisterCmd("gethashespersec", (*GetHashesPerSecCmd)(nil), flags)
	MustRegisterCmd("getinfo", (*GetInfoCmd)(nil), flags)
	MustRegisterCmd("getstateinfo", (*GetStateInfoCmd)(nil), flags)
	MustRegisterCmd("getentangleinfo", (*GetEntangleInfoCmd)(nil), flags)
	MustRegisterCmd("getmempoolentry", (*GetMempoolEntryCmd)(nil), flags)
	MustRegisterCmd("getmempoolinfo", (*GetMempoolInfoCmd)(nil), flags)
	MustRegisterCmd("getmininginfo", (*GetMiningInfoCmd)(nil), flags)
	MustRegisterCmd("getnetworkinfo", (*GetNetworkInfoCmd)(nil), flags)
	MustRegisterCmd("getnettotals", (*GetNetTotalsCmd)(nil), flags)
	MustRegisterCmd("getnetworkhashps", (*GetNetworkHashPSCmd)(nil), flags)
	MustRegisterCmd("getpeerinfo", (*GetPeerInfoCmd)(nil), flags)
	MustRegisterCmd("getrawmempool", (*GetRawMempoolCmd)(nil), flags)
	MustRegisterCmd("getrawtransaction", (*GetRawTransactionCmd)(nil), flags)
	MustRegisterCmd("gettxout", (*GetTxOutCmd)(nil), flags)
	MustRegisterCmd("gettxoutproof", (*GetTxOutProofCmd)(nil), flags)
	MustRegisterCmd("gettxoutsetinfo", (*GetTxOutSetInfoCmd)(nil), flags)
	MustRegisterCmd("getwork", (*GetWorkCmd)(nil), flags)
	MustRegisterCmd("getworktemplate", (*GetWorkTemplateCmd)(nil), flags)
	MustRegisterCmd("help", (*HelpCmd)(nil), flags)
	MustRegisterCmd("invalidateblock", (*InvalidateBlockCmd)(nil), flags)
	MustRegisterCmd("ping", (*PingCmd)(nil), flags)
	MustRegisterCmd("preciousblock", (*PreciousBlockCmd)(nil), flags)
	MustRegisterCmd("reconsiderblock", (*ReconsiderBlockCmd)(nil), flags)
	MustRegisterCmd("searchrawtransactions", (*SearchRawTransactionsCmd)(nil), flags)
	MustRegisterCmd("sendrawtransaction", (*SendRawTransactionCmd)(nil), flags)
	MustRegisterCmd("setgenerate", (*SetGenerateCmd)(nil), flags)
	MustRegisterCmd("stop", (*StopCmd)(nil), flags)
	MustRegisterCmd("submitblock", (*SubmitBlockCmd)(nil), flags)
	MustRegisterCmd("submitwork", (*SubmitWorkCmd)(nil), flags)
	MustRegisterCmd("uptime", (*UptimeCmd)(nil), flags)
	MustRegisterCmd("validateaddress", (*ValidateAddressCmd)(nil), flags)
	MustRegisterCmd("verifychain", (*VerifyChainCmd)(nil), flags)
	MustRegisterCmd("verifymessage", (*VerifyMessageCmd)(nil), flags)
	MustRegisterCmd("verifytxoutproof", (*VerifyTxOutProofCmd)(nil), flags)
}
