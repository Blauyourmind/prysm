package params

import (
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// MainnetConfig returns the configuration to be used in the main network.
func MainnetConfig() *BeaconChainConfig {
	return mainnetBeaconConfig
}

// UseMainnetConfig for beacon chain services.
func UseMainnetConfig() {
	beaconConfig = MainnetConfig()
}

var mainnetNetworkConfig = &NetworkConfig{
	GossipMaxSize:                   1 << 20, // 1 MiB
	MaxChunkSize:                    1 << 20, // 1 MiB
	AttestationSubnetCount:          64,
	AttestationPropagationSlotRange: 32,
	MaxRequestBlocks:                1 << 10, // 1024
	TtfbTimeout:                     5 * time.Second,
	RespTimeout:                     10 * time.Second,
	MaximumGossipClockDisparity:     500 * time.Millisecond,
	MessageDomainInvalidSnappy:      [4]byte{00, 00, 00, 00},
	MessageDomainValidSnappy:        [4]byte{01, 00, 00, 00},
	ETH2Key:                         "eth2",
	AttSubnetKey:                    "attnets",
	MinimumPeersInSubnet:            4,
	MinimumPeersInSubnetSearch:      20,
	ContractDeploymentBlock:         11184524, // Note: contract was deployed in block 11052984 but no transactions were sent until 11184524.
	BootstrapNodes: []string{
		// Proto's bootnode.
		"enr:-Ku4QFrzIH_DETC3gh3Dhl_rMzNOdlUH427ef3lbtArKVvbaX_YVs6_zEB0Z5Q72a6ZKlgvXJjS-yt20UXhftrFS2EoCh2F0dG5ldHOIAAAAAAAAAACEZXRoMpB5-A5dAAAHAf__________gmlkgnY0gmlwhANDKQyJc2VjcDI1NmsxoQIOfbH-pbhkBiSvD0wJxyjm8vA4DROacA7uT4Nfdc5QDYN1ZHCCIyg",
		"enr:-LK4QJO02L_YsaXjI0OUw_mIcfInBLiy9pqE2j6S8gyvVG31f2pemsBQWKSzh1o6FYKqkDYOOKCMRymcwvGqxHRXm0oBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpB5-A5dAAAHAf__________gmlkgnY0gmlwhA3UIxiJc2VjcDI1NmsxoQK-Ul9MNUZLNn5uITHeEwg8z4h26vshUJmux6yQm0d9ooN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QIVjWVem2LJoSZkqFLx0QM0xVJLD18jHZSNwUsMcqO0yaL_2bFznCLknH8hz9K2hPmPUUnfFsWPJfUwpgzOGNooBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpB5-A5dAAAHAf__________gmlkgnY0gmlwhANComqJc2VjcDI1NmsxoQOEccHNC733KCWsZFgRl7PctIYwbiHFrV-OIOtwvSFx14N0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QF0gc-Zj5y4k6typswN5ZYaYp1OIuNxgb0lhl7idG78OIsrmCOR5STJQT9RMp0tysjTElmp0hwaKmxizo_KyKOABh2F0dG5ldHOIAAAAAAAAAACEZXRoMpB5-A5dAAAHAf__________gmlkgnY0gmlwhAMHJfSJc2VjcDI1NmsxoQPwt51JI8k5b2cc1tcotu3BrjY2MOSOycIyBnpf5_LHX4N0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QGDdr6pFuRO95rX4HoMWbrhTqr_o26e2QgQVZBgoGQDAc207JcOqaf2lXxufVqCgSZ9Sevm0sywquBFTp94o3GkBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpB5-A5dAAAHAf__________gmlkgnY0gmlwhCNQDTuJc2VjcDI1NmsxoQI4ptVL1_Buv7If42yf1hfPJdvtsa36k4otY0OoYGBTYYN0Y3CCIyiDdWRwgiMo",
	},
}

var mainnetBeaconConfig = &BeaconChainConfig{
	// Constants (Non-configurable)
	FarFutureEpoch:           1<<64 - 1,
	FarFutureSlot:            1<<64 - 1,
	BaseRewardsPerEpoch:      4,
	DepositContractTreeDepth: 32,
	GenesisDelay:             604800, // 1 week.

	// Misc constant.
	TargetCommitteeSize:            128,
	MaxValidatorsPerCommittee:      2048,
	MaxCommitteesPerSlot:           64,
	MinPerEpochChurnLimit:          4,
	ChurnLimitQuotient:             1 << 16,
	ShuffleRoundCount:              90,
	MinGenesisActiveValidatorCount: 16384,
	MinGenesisTime:                 1606824000, // Dec 1, 2020, 12pm UTC.
	TargetAggregatorsPerCommittee:  16,
	HysteresisQuotient:             4,
	HysteresisDownwardMultiplier:   1,
	HysteresisUpwardMultiplier:     5,

	// Gwei value constants.
	MinDepositAmount:          1 * 1e9,
	MaxEffectiveBalance:       32 * 1e9,
	EjectionBalance:           16 * 1e9,
	EffectiveBalanceIncrement: 1 * 1e9,

	// Initial value constants.
	BLSWithdrawalPrefixByte: byte(0),
	ZeroHash:                [32]byte{},

	// Time parameter constants.
	MinAttestationInclusionDelay:     1,
	SecondsPerSlot:                   12,
	SlotsPerEpoch:                    32,
	MinSeedLookahead:                 1,
	MaxSeedLookahead:                 4,
	EpochsPerEth1VotingPeriod:        64,
	SlotsPerHistoricalRoot:           8192,
	MinValidatorWithdrawabilityDelay: 256,
	ShardCommitteePeriod:             256,
	MinEpochsToInactivityPenalty:     4,
	Eth1FollowDistance:               2048,
	SafeSlotsToUpdateJustified:       8,

	// Ethereum PoW parameters.
	DepositChainID:         1, // Chain ID of eth1 mainnet.
	DepositNetworkID:       1, // Network ID of eth1 mainnet.
	DepositContractAddress: "0x00000000219ab540356cBB839Cbe05303d7705Fa",

	// Validator params.
	RandomSubnetsPerValidator:         1 << 0,
	EpochsPerRandomSubnetSubscription: 1 << 8,

	// While eth1 mainnet block times are closer to 13s, we must conform with other clients in
	// order to vote on the correct eth1 blocks.
	//
	// Additional context: https://github.com/ethereum/eth2.0-specs/issues/2132
	// Bug prompting this change: https://github.com/prysmaticlabs/prysm/issues/7856
	// Future optimization: https://github.com/prysmaticlabs/prysm/issues/7739
	SecondsPerETH1Block: 14,

	// State list length constants.
	EpochsPerHistoricalVector: 65536,
	EpochsPerSlashingsVector:  8192,
	HistoricalRootsLimit:      16777216,
	ValidatorRegistryLimit:    1099511627776,

	// Reward and penalty quotients constants.
	BaseRewardFactor:               64,
	WhistleBlowerRewardQuotient:    512,
	ProposerRewardQuotient:         8,
	InactivityPenaltyQuotient:      67108864,
	MinSlashingPenaltyQuotient:     128,
	ProportionalSlashingMultiplier: 1,

	// Max operations per block constants.
	MaxProposerSlashings:         16,
	MaxAttesterSlashings:         2,
	MaxAttestations:              128,
	MaxDeposits:                  16,
	MaxVoluntaryExits:            16,
	MaxExecutionTransactions:     16384,
	MaxBytesPerOpaqueTransaction: 1048576,

	// Sharding related constants.
	MaxShards:                     1024,
	InitialActiveShards:           64,
	GaspriceAdjustmentCoefficient: 8,
	MaxShardHeadersPerShard:       4,
	MaxShardProposerSlashings:     16,
	MaxSamplesPerBlock:            2048,
	TargetSamplesPerBlock:         1024,
	MaxGasPrice:                   1 << 33,
	MinGasPrice:                   8,

	// BLS domain values.
	DomainBeaconProposer:    bytesutil.ToBytes4(bytesutil.Bytes4(0)),
	DomainBeaconAttester:    bytesutil.ToBytes4(bytesutil.Bytes4(1)),
	DomainRandao:            bytesutil.ToBytes4(bytesutil.Bytes4(2)),
	DomainDeposit:           bytesutil.ToBytes4(bytesutil.Bytes4(3)),
	DomainVoluntaryExit:     bytesutil.ToBytes4(bytesutil.Bytes4(4)),
	DomainSelectionProof:    bytesutil.ToBytes4(bytesutil.Bytes4(5)),
	DomainAggregateAndProof: bytesutil.ToBytes4(bytesutil.Bytes4(6)),
	DomainShardProposer:     bytesutil.ToBytes4(bytesutil.Bytes4(128)),
	DomainShardCommittee:    bytesutil.ToBytes4(bytesutil.Bytes4(129)),

	// Prysm constants.
	GweiPerEth:                1000000000,
	BLSSecretKeyLength:        32,
	BLSPubkeyLength:           48,
	BLSSignatureLength:        96,
	DefaultBufferSize:         10000,
	WithdrawalPrivkeyFileName: "/shardwithdrawalkey",
	ValidatorPrivkeyFileName:  "/validatorprivatekey",
	RPCSyncCheck:              1,
	EmptySignature:            [96]byte{},
	DefaultPageSize:           250,
	MaxPeersToSync:            15,
	SlotsPerArchivedPoint:     2048,
	GenesisCountdownInterval:  time.Minute,
	ConfigName:                ConfigNames[Mainnet],
	BeaconStateFieldCount:     22,

	// Slasher related values.
	WeakSubjectivityPeriod:          54000,
	PruneSlasherStoragePeriod:       10,
	SlashingProtectionPruningEpochs: 512,

	// Weak subjectivity values.
	SafetyDecay: 10,

	// Fork related values.
	GenesisForkVersion:  []byte{0, 0, 0, 0},
	NextForkVersion:     []byte{0, 0, 0, 0}, // Set to GenesisForkVersion unless there is a scheduled fork
	NextForkEpoch:       1<<64 - 1,          // Set to FarFutureEpoch unless there is a scheduled fork.
	ForkVersionSchedule: map[types.Epoch][]byte{
		// Any further forks must be specified here by their epoch number.
	},
}
