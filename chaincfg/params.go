// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/classzz/classzz/chaincfg/chainhash"
	"github.com/classzz/classzz/wire"
)

// These variables are the chain proof-of-work limit parameters for each default
// network.
var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// mainPowLimit is the highest proof of work value a Bitcoin block can
	// have for the main network.  It is the value 2^224 - 1.
	mainPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 236), bigOne)

	// regressionPowLimit is the highest proof of work value a Bitcoin block
	// can have for the regression test network.  It is the value 2^255 - 1.
	regressionPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// testNet3PowLimit is the highest proof of work value a Bitcoin block
	// can have for the test network (version 3).  It is the value
	// 2^224 - 1.
	testNetPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)

	// simNetPowLimit is the highest proof of work value a Bitcoin block
	// can have for the simulation test network.  It is the value 2^255 - 1.
	simNetPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
)

// Checkpoint identifies a known good point in the block chain.  Using
// checkpoints allows a few optimizations for old blocks during initial download
// and also prevents forks from old blocks.
//
// Each checkpoint is selected based upon several factors.  See the
// documentation for blockchain.IsCheckpointCandidate for details on the
// selection criteria.
type Checkpoint struct {
	Height         int32
	Hash           *chainhash.Hash
	UtxoSetHash    *chainhash.Hash
	UtxoSetSources []string
	UtxoSetSize    uint32
}

// DNSSeed identifies a DNS seed.
type DNSSeed struct {
	// Host defines the hostname of the seed.
	Host string

	// HasFiltering defines whether the seed supports filtering
	// by service flags (wire.ServiceFlag).
	HasFiltering bool
}

// ConsensusDeployment defines details related to a specific consensus rule
// change that is voted in.  This is part of BIP0009.
type ConsensusDeployment struct {
	// BitNumber defines the specific bit number within the block version
	// this particular soft-fork deployment refers to.
	BitNumber uint8

	// StartTime is the median block time after which voting on the
	// deployment starts.
	StartTime uint64

	// ExpireTime is the median block time after which the attempted
	// deployment expires.
	ExpireTime uint64
}

// Constants that define the deployment offset in the deployments field of the
// parameters for each deployment.  This is useful to be able to get the details
// of a specific deployment by name.
const (
	// DeploymentTestDummy defines the rule change deployment ID for testing
	// purposes.
	DeploymentTestDummy = iota

	// DeploymentCSV defines the rule change deployment ID for the CSV
	// soft-fork package. The CSV package includes the deployment of BIPS
	// 68, 112, and 113.
	DeploymentCSV

	//Ensure that the time of the parent block is less than the current time
	DeploymentSEQ

	// NOTE: DefinedDeployments must always come last since it is used to
	// determine how many defined deployments there currently are.
	// DefinedDeployments is the number of currently defined deployments.
	DefinedDeployments
)

// Params defines a Bitcoin network by its parameters.  These parameters may be
// used by Bitcoin applications to differentiate networks as well as addresses
// and keys for one network from those intended for use on another network.
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name string

	// Net defines the magic bytes used to identify the network.
	Net wire.BitcoinNet

	// DefaultPort defines the default peer-to-peer port for the network.
	DefaultPort string

	// DNSSeeds defines a list of DNS seeds for the network that are used
	// as one method to discover peers.
	DNSSeeds []DNSSeed

	// GenesisBlock defines the first block of the chain.
	GenesisBlock *wire.MsgBlock

	// GenesisHash is the starting block hash.
	GenesisHash *chainhash.Hash

	// PowLimit defines the highest allowed proof of work value for a block
	// as a uint256.
	PowLimit *big.Int

	// PowLimitBits defines the highest allowed proof of work value for a
	// block in compact form.
	PowLimitBits uint32

	// Planned hardforks
	GravitonActivationTime uint64 // Nov 15, 2019 hard fork

	// CoinbaseMaturity is the number of blocks required before newly mined
	// coins (coinbase transactions) can be spent.
	CoinbaseMaturity uint16

	// SubsidyReductionInterval is the interval of blocks before the subsidy
	// is reduced.
	SubsidyReductionInterval int32

	// TargetTimespan is the desired amount of time that should elapse
	// before the block difficulty requirement is examined to determine how
	// it should be changed in order to maintain the desired block
	// generation rate.
	TargetTimespan time.Duration

	// TargetTimePerBlock is the desired amount of time to generate each
	// block.
	TargetTimePerBlock time.Duration

	// RetargetAdjustmentFactor is the adjustment factor used to limit
	// the minimum and maximum amount of adjustment that can occur between
	// difficulty retargets.
	RetargetAdjustmentFactor int64

	// ReduceMinDifficulty defines whether the network should reduce the
	// minimum required difficulty after a long enough period of time has
	// passed without finding a block.  This is really only useful for test
	// networks and should not be set on a main network.
	ReduceMinDifficulty bool

	// NoDifficultyAdjustment defines whether the network should skip the
	// normal difficulty adjustment and keep the current difficulty.
	NoDifficultyAdjustment bool

	// MinDiffReductionTime is the amount of time after which the minimum
	// required difficulty should be reduced when a block hasn't been found.
	//
	// NOTE: This only applies if ReduceMinDifficulty is true.
	MinDiffReductionTime time.Duration

	// GenerateSupported specifies whether or not CPU mining is allowed.
	GenerateSupported bool

	EntangleHeight int32

	BeaconHeight int32

	//ExChangeHeight int32

	ConverHeight int32

	// Checkpoints ordered from oldest to newest.
	Checkpoints []Checkpoint

	// These fields are related to voting on consensus rule changes as
	// defined by BIP0009.
	//
	// RuleChangeActivationThreshold is the number of blocks in a threshold
	// state retarget window for which a positive vote for a rule change
	// must be cast in order to lock in a rule change. It should typically
	// be 95% for the main network and 75% for test networks.
	//
	// MinerConfirmationWindow is the number of blocks in each threshold
	// state retarget window.
	//
	// Deployments define the specific consensus rule changes to be voted
	// on.
	RuleChangeActivationThreshold uint32
	MinerConfirmationWindow       uint32
	Deployments                   [DefinedDeployments]ConsensusDeployment

	// Mempool parameters
	RelayNonStdTxs bool

	// The prefix used for the cashaddress. This is different for each network.
	CashAddressPrefix string

	// Address encoding magics
	LegacyPubKeyHashAddrID byte // First byte of a P2PKH address
	LegacyScriptHashAddrID byte // First byte of a P2SH address
	PrivateKeyID           byte // First byte of a WIF private key

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID [4]byte
	HDPublicKeyID  [4]byte

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType uint32
}

// MainNetParams defines the network parameters for the main Bitcoin network.
var MainNetParams = Params{
	Name:        "mainnet",
	Net:         wire.MainNet,
	DefaultPort: "18883",
	DNSSeeds:    []DNSSeed{},

	// Chain parameters
	GenesisBlock: &genesisBlock,
	GenesisHash:  &genesisHash,

	PowLimit:     mainPowLimit,
	PowLimitBits: 0x1e10624d,

	CoinbaseMaturity:         14,
	SubsidyReductionInterval: 1000000,
	TargetTimePerBlock:       30, // 30 seconds
	GenerateSupported:        true,

	EntangleHeight: 120000,
	BeaconHeight:   420000,

	ConverHeight: 100000000,
	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{
		{Height: 11111, Hash: newHashFromStr("1faf0d2246f07608c6a97a6ca698055a89d07f84c52db4455addad0cc86175aa")},
		{Height: 33333, Hash: newHashFromStr("cf3de795f31dbc20fbefc0e1b8aeeb07c41fc7e8ef748c9e7d74af767beaf1d2")},
		{Height: 55000, Hash: newHashFromStr("1f6daf0bc028bea6183120d740df256162c645de98e3c221625b8405d54bad66")},
		{Height: 74000, Hash: newHashFromStr("0e14e6a7afb47846296111d2ade1b75527a96e898c11a8422325aad480adcc1d")},
		{Height: 85000, Hash: newHashFromStr("bdf3bc34deb6a19df11f626cc18c5230777124cb2a83c0c3bca90dd2b523a417")},
		{Height: 91000, Hash: newHashFromStr("676e45ca46d01099763a4b693d7aa63068e3280a9c6f576dd0fade5d01cc1439")},
		{Height: 101000, Hash: newHashFromStr("aed106b6ea933822fe66e1be6e369c976163c9b4173c63f56c717bc08fe9690d")},
		{Height: 120100, Hash: newHashFromStr("ad185c5b8cb742bc113791b171c63fbd9ded1a1e33ad1aeed137a7f31b44fc70")},
		{Height: 150000, Hash: newHashFromStr("621732d353237090755fd0ae2bcd1dbc62ff3e16730799f8da7e57ee7f1e7f6b")},
		{Height: 170000, Hash: newHashFromStr("b322b92797469e366849947a6daabce890f74c8be58af5096f79efe629bd2b4c")},
		{Height: 190000, Hash: newHashFromStr("bbccbab407a2755d3bac95c1dd02d4a25641e7fe40cc3e866f485125e5733b15")},
		{Height: 210000, Hash: newHashFromStr("07f5e9acc27edfd1ff6252f1005e3984a374d05a92aead6914c9f5b2b1ef14c8")},
		{Height: 225000, Hash: newHashFromStr("fc0e1538e7369862e41c8cedbf2b0b73eaf1a3f76cdad7496ec7884e3aad3e9d")},
		{Height: 260000, Hash: newHashFromStr("c37dfc73a91484ef4279f67d14aeb7537c32203cc7463ea871877ead64af57eb")},
		{Height: 290000, Hash: newHashFromStr("e105ec96c674d53b17eda46dd464ba0b3bcb9d4706d9a5c497f995f7e7d472b9")},
		{Height: 320757, Hash: newHashFromStr("b5e2b5431de01a93efbcdef77b5a72ba216dcdcf14728e66e22d96387da3089b")},
		{Height: 350757, Hash: newHashFromStr("19d78e5c64bd4f414ca13765dd7ff7639e3b58749969fd650170ff10db6b8b92")},
		{Height: 370000, Hash: newHashFromStr("80258d53649075b0118ac709faa639061b1fc5780091487b71ec1dd9b1db2e6b")},
		{Height: 385000, Hash: newHashFromStr("62ab9700f2ed79a6e059890d0385254bdd0e572ed4634c0f7fec012dd18f660e")},
		{Height: 400100, Hash: newHashFromStr("7c7fd0304aa4409c091a7681e4b094dcf82fe497b6831bf96c7ea31352c757f2")},
		{Height: 422555, Hash: newHashFromStr("095dd0703fb7ece922bb157cfaa5ccd25469e6a95bf1fc226d734c09722d6e64")},
		{Height: 436000, Hash: newHashFromStr("57f6f4ead9aee8becc9acab3fad18c6194684069e3ca781c64ce79c22bca83f4")},
		{Height: 466000, Hash: newHashFromStr("0f97de060983f83533820977754b279d66a9085ed6cb2a628448bec7c9cbd0ad")},
		{Height: 486000, Hash: newHashFromStr("2986367d7871a46c9edf6e9e6aa6f334c43d9874cd0fecbb5e1f9d6c6e9de200")},
		{Height: 506000, Hash: newHashFromStr("5b3c6a48e7f3f3b641cdc22d5b949c0404da17ffef2054255d5352c59c061ce0")},
		{Height: 526000, Hash: newHashFromStr("3a0b0842c25416d7cf982df463bbb3e236d0fd2404c95722d56965995bf15989")},
		{Height: 546000, Hash: newHashFromStr("6d49304d7328d291f2fa531aff8e609df28706e56f72dca528ac2f821de9b032")},
		{Height: 566000, Hash: newHashFromStr("228b8a0260cab3c15b2e63e46a639eea67bb5b693561af1dd881b5277c38d0ef")},
		{Height: 586000, Hash: newHashFromStr("b7557ea0a6c62de01c7a613f4ec209c3857f8597f57781c7ea521e160698b567")},
		{Height: 606000, Hash: newHashFromStr("a3b1bfe7cdfd0fa3fbe25cdee3b9a7956bb040749d5f79f122aad321e8364a76")},
		{Height: 626000, Hash: newHashFromStr("17f1caac421c561d3738c2826915653c636edcff0a2ed04d3493eb36f9dfd594")},
		{Height: 646000, Hash: newHashFromStr("b0ccde768d9de6a5f2e589b4e737a2961314eab0756330a3cba226cf961fc46b")},
		{Height: 666000, Hash: newHashFromStr("2ec96ab946b82f9eaebaf12db102b5a49ede18a09c0e6f3d7e8341efa4c26897")},
		{Height: 686000, Hash: newHashFromStr("fe9f270d3e1bd758f67b29c2f78caf60a53af0c45e8719bd433b1917c50ace5f")},
		{Height: 706000, Hash: newHashFromStr("66861fb39dbe96027eadcfc3b03a609ea68df18cb975225b70c1a5260d413332")},
		{Height: 726000, Hash: newHashFromStr("142225d4eceafc088cfbcfc1930a9418393393a3eaf99b6e2827a315958377c8")},
		{Height: 746000, Hash: newHashFromStr("40698d35987861d648604b6955553b0760e05bee1ca82eca1de2351933e704e6")},
		{Height: 766000, Hash: newHashFromStr("58427d3bdd1ace8a0c86966a22702ae6f0c24ed5de89967758ab9a2aab2d7e93")},
		{Height: 786000, Hash: newHashFromStr("4800f3c3d0315389099f09a1199f3e8122e66daf04578ff55e51f61f277c9e8a")},
		{Height: 806000, Hash: newHashFromStr("a32b451de977e88a4729ee86a5875c68530564aee11937ca33bf9065f27ff3e4")},
		{Height: 826000, Hash: newHashFromStr("bbfa4fec2842900007290081b6ec433000f905f9955080485a648186fdb6964c")},
		{Height: 846000, Hash: newHashFromStr("80cf81e76251fdca931fa76c1051b663f51ed29164952051763e9dd38832daac")},
		{Height: 866000, Hash: newHashFromStr("cc16622ecce3b0f057fd9cdebc4154c1b64f1b2590f55593b5130457f5f5694b")},
		{Height: 886000, Hash: newHashFromStr("3e2315b566baf2a6f99d5e9969b133bb8be59ba054653956ea8cffcc0b7d1c5b")},
		{Height: 906000, Hash: newHashFromStr("f3ea95e3a27af331fbd177ae949827ea0e6e86c7ccdf06efbfc2e8fa4a9acf06")},
		{Height: 926000, Hash: newHashFromStr("a649fb9e2286d9be11636d482ac45c90f2dd852507cb021f4a5f9f94dacbde37")},
		{Height: 946000, Hash: newHashFromStr("c211bc8d3abbf50afaeaf69079dd05bba65f32d259cf7074a0c0469724ea400a")},
		{Height: 966000, Hash: newHashFromStr("f01e1d6959069c5b2957a835244e6e41190a65e9178bc7cea4ca464e0ba81594")},
		{Height: 986000, Hash: newHashFromStr("4f8fc5383d2377737ecb35ddb0edc6b0086e4a9d5fab874fc16e069aa3588eb9")},
		{Height: 1006000, Hash: newHashFromStr("a9068a3b7ef30e474449f304723d2b4d0737b778c8c7ef028ba898450ce2b58c")},
		{Height: 1026000, Hash: newHashFromStr("9ba6788069a69c642058aa70a59b216b40b7dc0713a24ff34a68912403849110")},
		{Height: 1046000, Hash: newHashFromStr("8cac05b7495c9efef5c49562260bdc36a25b48702a21fba18f68e7404be33b53")},
		{Height: 1066000, Hash: newHashFromStr("914e9f34f2e187b54940419e74c60142ec69c703a8b3052e71bc3c1d6c63e9fd")},
	},

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1916, // 95% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016, //
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  1199145601, // January 1, 2008 UTC
			ExpireTime: 1230767999, // December 31, 2008 UTC
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  1462060800, // May 1st, 2016
			ExpireTime: 1493596800, // May 1st, 2017
		},
		DeploymentSEQ: {
			BitNumber:  0,
			StartTime:  1572868800,    //
			ExpireTime: math.MaxInt64, // Never expires
		},
	},

	// Mempool parameters
	RelayNonStdTxs: false,

	// The prefix for the cashaddress
	CashAddressPrefix: "classzz", // always class-zz for mainnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x00, // starts with 1
	LegacyScriptHashAddrID: 0x05, // starts with 3
	PrivateKeyID:           0x80, // starts with 5 (uncompressed) or K (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e}, // starts with xpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 706, // 706
}

// RegressionNetParams defines the network parameters for the regression test
// Bitcoin network.  Not to be confused with the test Bitcoin network (version
// 3), this network is sometimes simply called "testnet".
var RegressionNetParams = Params{
	Name:        "regtest",
	Net:         wire.RegTestNet,
	DefaultPort: "18884",
	DNSSeeds:    []DNSSeed{},

	// Chain parameters
	GenesisBlock:     &regTestGenesisBlock,
	GenesisHash:      &regTestGenesisHash,
	PowLimit:         regressionPowLimit,
	PowLimitBits:     0x207fffff,
	CoinbaseMaturity: 14,

	SubsidyReductionInterval: 1000000,
	TargetTimePerBlock:       30, // 10 minutes
	RetargetAdjustmentFactor: 4,  // 25% less, 400% more
	ReduceMinDifficulty:      true,
	NoDifficultyAdjustment:   false,
	MinDiffReductionTime:     time.Minute * 20, // TargetTimePerBlock * 2
	GenerateSupported:        true,

	EntangleHeight: 120000,
	BeaconHeight:   2,
	//ExChangeHeight: 500000,
	// Checkpoints ordered from oldest to newest.
	Checkpoints: nil,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1916, // 95% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016, //
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentSEQ: {
			BitNumber:  0,
			StartTime:  0,             //
			ExpireTime: math.MaxInt64, // Never expires
		},
	},

	// Mempool parameters
	RelayNonStdTxs: false,

	// The prefix for the cashaddress
	CashAddressPrefix: "czzreg", // always czzreg for reg testnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x6f, // starts with m or n
	LegacyScriptHashAddrID: 0xc4, // starts with 2
	PrivateKeyID:           0xef, // starts with 9 (uncompressed) or c (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1, // all coins use 1
}

// TestNet3Params defines the network parameters for the test Bitcoin network
// (version 3).  Not to be confused with the regression test network, this
// network is sometimes simply called "testnet".
var TestNetParams = Params{
	Name:        "testnet",
	Net:         wire.TestNet,
	DefaultPort: "18885",
	DNSSeeds:    []DNSSeed{},

	// Chain parameters
	GenesisBlock: &testNetGenesisBlock,
	GenesisHash:  &testNetGenesisHash,
	PowLimit:     testNetPowLimit,
	PowLimitBits: 0x207fffff,

	CoinbaseMaturity:         14,
	TargetTimespan:           time.Hour * 24 * 14, // 14 days
	SubsidyReductionInterval: 100,
	TargetTimePerBlock:       30, // 10 minutes
	GenerateSupported:        true,

	EntangleHeight: 5,
	BeaconHeight:   10,
	ConverHeight:   50,
	// Checkpoints ordered from oldest to newest.
	Checkpoints: []Checkpoint{},

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1512, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentSEQ: {
			BitNumber:  0,
			StartTime:  0,             //
			ExpireTime: math.MaxInt64, // Never expires
		},
	},

	// Mempool parameters
	RelayNonStdTxs: true,

	// The prefix for the cashaddress
	CashAddressPrefix: "czztest", // always czztest for testnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x00, // starts with 1
	LegacyScriptHashAddrID: 0x05, // starts with 3
	PrivateKeyID:           0x80, // starts with 5 (uncompressed) or K (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e}, // starts with xpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 706, // 706
}

// SimNetParams defines the network parameters for the simulation test Bitcoin
// network.  This network is similar to the normal test network except it is
// intended for private use within a group of individuals doing simulation
// testing.  The functionality is intended to differ in that the only nodes
// which are specifically specified are used to create the network rather than
// following normal discovery rules.  This is important as otherwise it would
// just turn into another public testnet.
var SimNetParams = Params{
	Name:        "simnet",
	Net:         wire.SimNet,
	DefaultPort: "18885",
	DNSSeeds:    []DNSSeed{}, // NOTE: There must NOT be any seeds.

	// Chain parameters
	GenesisBlock:             &simNetGenesisBlock,
	GenesisHash:              &simNetGenesisHash,
	PowLimit:                 simNetPowLimit,
	PowLimitBits:             0x207fffff,
	CoinbaseMaturity:         14,
	SubsidyReductionInterval: 1000000,
	TargetTimespan:           time.Hour * 24 * 14, // 14 days
	TargetTimePerBlock:       30,                  // 10 minutes
	RetargetAdjustmentFactor: 4,                   // 25% less, 400% more
	ReduceMinDifficulty:      true,
	NoDifficultyAdjustment:   false,
	MinDiffReductionTime:     time.Minute * 20, // TargetTimePerBlock * 2
	GenerateSupported:        true,

	EntangleHeight: 10,
	BeaconHeight:   12,
	//ExChangeHeight: 20,
	ConverHeight: 25,
	// Checkpoints ordered from oldest to newest.
	Checkpoints: nil,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 75, // 75% of MinerConfirmationWindow
	MinerConfirmationWindow:       14,
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentSEQ: {
			BitNumber:  0,
			StartTime:  0,             //
			ExpireTime: math.MaxInt64, // Never expires
		},
	},

	// Mempool parameters
	RelayNonStdTxs: true,

	// The prefix for the cashaddress
	CashAddressPrefix: "czzsim", // always czzsim for simnet

	// Address encoding magics
	LegacyPubKeyHashAddrID: 0x3f, // starts with S
	LegacyScriptHashAddrID: 0x7b, // starts with s
	PrivateKeyID:           0x64, // starts with 4 (uncompressed) or F (compressed)

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x20, 0xb9, 0x00}, // starts with sprv
	HDPublicKeyID:  [4]byte{0x04, 0x20, 0xbd, 0x3a}, // starts with spub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 115, //
}

var (
	// ErrDuplicateNet describes an error where the parameters for a Bitcoin
	// network could not be set due to the network already being a standard
	// network or previously-registered into this package.
	ErrDuplicateNet = errors.New("duplicate Bitcoin network")

	// ErrUnknownHDKeyID describes an error where the provided id which
	// is intended to identify the network for a hierarchical deterministic
	// private extended key is not registered.
	ErrUnknownHDKeyID = errors.New("unknown hd private extended key bytes")
)

var (
	registeredNets      = make(map[wire.BitcoinNet]struct{})
	pubKeyHashAddrIDs   = make(map[byte]struct{})
	scriptHashAddrIDs   = make(map[byte]struct{})
	cashAddressPrefixes = make(map[string]struct{})
	hdPrivToPubKeyIDs   = make(map[[4]byte][]byte)
)

// String returns the hostname of the DNS seed in human-readable form.
func (d DNSSeed) String() string {
	return d.Host
}

// Register registers the network parameters for a Bitcoin network.  This may
// error with ErrDuplicateNet if the network is already registered (either
// due to a previous Register call, or the network being one of the default
// networks).
//
// Network parameters should be registered into this package by a main package
// as early as possible.  Then, library packages may lookup networks or network
// parameters based on inputs and work regardless of the network being standard
// or not.
func Register(params *Params) error {
	if _, ok := registeredNets[params.Net]; ok {
		return ErrDuplicateNet
	}
	registeredNets[params.Net] = struct{}{}
	pubKeyHashAddrIDs[params.LegacyPubKeyHashAddrID] = struct{}{}
	scriptHashAddrIDs[params.LegacyScriptHashAddrID] = struct{}{}
	hdPrivToPubKeyIDs[params.HDPrivateKeyID] = params.HDPublicKeyID[:]

	// A valid cashaddress prefix for the given net followed by ':'.
	cashAddressPrefixes[params.CashAddressPrefix+":"] = struct{}{}
	return nil
}

// mustRegister performs the same function as Register except it panics if there
// is an error.  This should only be called from package init functions.
func mustRegister(params *Params) {
	if err := Register(params); err != nil {
		panic("failed to register network: " + err.Error())
	}
}

// IsPubKeyHashAddrID returns whether the id is an identifier known to prefix a
// pay-to-pubkey-hash address on any default or registered network.  This is
// used when decoding an address string into a specific address type.  It is up
// to the caller to check both this and IsScriptHashAddrID and decide whether an
// address is a pubkey hash address, script hash address, neither, or
// undeterminable (if both return true).
func IsPubKeyHashAddrID(id byte) bool {
	_, ok := pubKeyHashAddrIDs[id]
	return ok
}

// IsScriptHashAddrID returns whether the id is an identifier known to prefix a
// pay-to-script-hash address on any default or registered network.  This is
// used when decoding an address string into a specific address type.  It is up
// to the caller to check both this and IsPubKeyHashAddrID and decide whether an
// address is a pubkey hash address, script hash address, neither, or
// undeterminable (if both return true).
func IsScriptHashAddrID(id byte) bool {
	_, ok := scriptHashAddrIDs[id]
	return ok
}

// IsCashAddressPrefix returns whether the prefix is a known prefix for the
// cashaddress on any default or registered network.  This is used when decoding
// an address string into a specific address type.
func IsCashAddressPrefix(prefix string) bool {
	prefix = strings.ToLower(prefix)
	_, ok := cashAddressPrefixes[prefix]
	return ok
}

// HDPrivateKeyToPublicKeyID accepts a private hierarchical deterministic
// extended key id and returns the associated public key id.  When the provided
// id is not registered, the ErrUnknownHDKeyID error will be returned.
func HDPrivateKeyToPublicKeyID(id []byte) ([]byte, error) {
	if len(id) != 4 {
		return nil, ErrUnknownHDKeyID
	}

	var key [4]byte
	copy(key[:], id)
	pubBytes, ok := hdPrivToPubKeyIDs[key]
	if !ok {
		return nil, ErrUnknownHDKeyID
	}

	return pubBytes, nil
}

// newHashFromStr converts the passed big-endian hex string into a
// chainhash.Hash.  It only differs from the one available in chainhash in that
// it panics on an error since it will only (and must only) be called with
// hard-coded, and therefore known good, hashes.
func newHashFromStr(hexStr string) *chainhash.Hash {
	hash, err := chainhash.NewHashFromStr(hexStr)
	if err != nil {
		// Ordinarily I don't like panics in library code since it
		// can take applications down without them having a chance to
		// recover which is extremely annoying, however an exception is
		// being made in this case because the only way this can panic
		// is if there is an error in the hard-coded hashes.  Thus it
		// will only ever potentially panic on init and therefore is
		// 100% predictable.
		panic(err)
	}
	return hash
}

func init() {
	// Register all default networks when the package is initialized.
	mustRegister(&MainNetParams)
	mustRegister(&TestNetParams)
	mustRegister(&RegressionNetParams)
	mustRegister(&SimNetParams)
}
