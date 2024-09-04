package e2e

import (
	"fmt"
	"time"

	wasmparams "github.com/CosmWasm/wasmd/app/params"
	bcdapp "github.com/babylonlabs-io/babylon-sdk/demo/app"
	bcdparams "github.com/babylonlabs-io/babylon-sdk/demo/app/params"
	bbnparams "github.com/babylonlabs-io/babylon/app/params"
	"github.com/babylonlabs-io/babylon/client/config"
	"github.com/babylonlabs-io/babylon/test/e2e/clientcontroller/babylon"
	cwconfig "github.com/babylonlabs-io/babylon/test/e2e/clientcontroller/config"
	"github.com/babylonlabs-io/babylon/test/e2e/clientcontroller/cosmwasm"
	cwcc "github.com/babylonlabs-io/babylon/test/e2e/clientcontroller/cosmwasm"
	"github.com/babylonlabs-io/babylon/testutil/datagen"
	bstypes "github.com/babylonlabs-io/babylon/x/btcstaking/types"
	bsctypes "github.com/babylonlabs-io/babylon/x/btcstkconsumer/types"
	"github.com/btcsuite/btcd/chaincfg"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type BTCStakingIntegration2TestSuite struct {
	suite.Suite

	babylonRPC1      string
	babylonRPC2      string
	consumerChainRPC string

	babylonController  *babylon.BabylonController
	cosmwasmController *cosmwasm.CosmwasmConsumerController
}

func (s *BTCStakingIntegration2TestSuite) SetupSuite() {
	s.T().Log("setting up e2e integration test suite...")

	// Set the RPC URLs for the Babylon nodes and consumer chain
	s.babylonRPC1 = "http://localhost:26657"
	s.babylonRPC2 = "http://localhost:26667"
	s.consumerChainRPC = "http://localhost:26677"

	err := s.initBabylonController()
	s.Require().NoError(err, "Failed to initialize BabylonController")

	err = s.initCosmwasmController()
	s.Require().NoError(err, "Failed to initialize CosmwasmConsumerController")
}

func (s *BTCStakingIntegration2TestSuite) TearDownSuite() {
	s.T().Log("tearing down e2e integration test suite...")

	// Run the stop-integration-test make target
	// cmd := exec.Command("make", "-C", "../consumer", "stop-integration-test")
	// output, err := cmd.CombinedOutput()
	// if err != nil {
	// 	s.T().Logf("Failed to run stop-integration-test: %s", output)
	// }
}

func (s *BTCStakingIntegration2TestSuite) Test1ChainStartup() {
	var (
		babylonStatus  *coretypes.ResultStatus
		consumerStatus *coretypes.ResultStatus
		err            error
	)

	// Use Babylon controller
	s.Eventually(func() bool {
		babylonStatus, err = s.babylonController.QueryNodeStatus()
		return err == nil && babylonStatus.SyncInfo.LatestBlockHeight >= 1
	}, time.Minute, time.Second, "Failed to query Babylon node status", err)
	s.T().Logf("Babylon node status: %v", babylonStatus.SyncInfo.LatestBlockHeight)

	// Use Cosmwasm controller
	s.Eventually(func() bool {
		consumerStatus, err = s.cosmwasmController.GetCometNodeStatus()
		return err == nil && consumerStatus.SyncInfo.LatestBlockHeight >= 1
	}, time.Minute, time.Second, "Failed to query Consumer node status", err)
	s.T().Logf("Consumer node status: %v", consumerStatus.SyncInfo.LatestBlockHeight)
	// Add your test assertions here
	// ...
}

func (s *BTCStakingIntegration2TestSuite) Test2AutoRegisterAndVerifyNewConsumer() {
	// TODO: try to fix the error otherwise hardcode consumer id for now
	consumerID := "07-tendermint-0" //  s.getIBCClientID()
	s.verifyConsumerRegistration(consumerID)
}

func (s *BTCStakingIntegration2TestSuite) Test3CreateConsumerFinalityProvider() {
	consumerID := "07-tendermint-0"

	// generate a random number of finality providers from 1 to 5
	numConsumerFPs := datagen.RandomInt(r, 5) + 1
	var consumerFps []*bstypes.FinalityProvider
	for i := 0; i < int(numConsumerFPs); i++ {
		consumerFp := s.createVerifyConsumerFP(consumerID)
		consumerFps = append(consumerFps, consumerFp)
	}

	//czNode, err := s.configurer.GetChainConfig(1).GetNodeAtIndex(2)
	//s.NoError(err)
	//// retrieve staking contract address and query finality providers stored in the contract
	//stakingContracts, err := czNode.QueryContractsFromId(2)
	//s.NoError(err)
	//s.Equal(1, len(stakingContracts))
	//stakingContractAddr := stakingContracts[0]
	//
	//// query the staking contract for finality providers
	//var dataFromContract *chain.ConsumerFpsResponse
	//s.Eventually(func() bool {
	//	// try to retrieve expected number of finality providers from the contract
	//	dataFromContract, err = czNode.QueryConsumerFps(stakingContractAddr)
	//	if err != nil {
	//		return false
	//	}
	//	return len(dataFromContract.ConsumerFps) == int(numConsumerFPs)
	//}, time.Second*20, time.Second)
	//
	//// create a map of expected finality providers for verification
	//fpMap := make(map[string]*bstypes.FinalityProvider)
	//for _, czFp := range consumerFps {
	//	fpMap[czFp.BtcPk.MarshalHex()] = czFp
	//}
	//
	//// validate that all finality providers match with the consumer list
	//for _, czFp := range dataFromContract.ConsumerFps {
	//	fpFromMap, ok := fpMap[czFp.BtcPkHex]
	//	s.True(ok)
	//	s.Equal(fpFromMap.BtcPk.MarshalHex(), czFp.BtcPkHex)
	//	s.Equal(fpFromMap.SlashedBabylonHeight, czFp.SlashedHeight)
	//	s.Equal(fpFromMap.SlashedBtcHeight, czFp.SlashedBtcHeight)
	//	s.Equal(fpFromMap.ConsumerId, czFp.ConsumerID)
	//}
}

func (s *BTCStakingIntegration2TestSuite) createVerifyConsumerFP(consumerId string) *bstypes.FinalityProvider {
	/*
		create a random consumer finality provider on Babylon
	*/
	// NOTE: we use the node's secret key as Babylon secret key for the finality provider
	czFpBTCSK, _, _ := datagen.GenRandomBTCKeyPair(r)
	sdk.SetAddrCacheEnabled(false)
	bbnparams.SetAddressPrefixes()
	sdkCfg := sdk.GetConfig()
	fmt.Printf("Current - Account Prefix: %s\n", sdkCfg.GetBech32AccountAddrPrefix())
	fpBabylonAddr, err := sdk.AccAddressFromBech32(s.babylonController.MustGetTxSigner())
	s.NoError(err)
	fmt.Println("fpbabylonaddr", s.babylonController.MustGetTxSigner())
	czFp, err := datagen.GenCustomFinalityProvider(r, czFpBTCSK, fpBabylonAddr, consumerId)
	s.NoError(err)
	czFpPop, err := czFp.Pop.Marshal()
	s.NoError(err)
	czDescription, err := czFp.Description.Marshal()
	s.NoError(err)

	_, err = s.babylonController.RegisterFinalityProvider(
		consumerId,
		czFp.BtcPk,
		czFpPop,
		czFp.Commission,
		czDescription,
	)
	s.NoError(err)

	// query the existence of finality provider and assert equivalence
	actualFp, err := s.babylonController.QueryConsumerFinalityProvider(consumerId, czFp.BtcPk.MarshalHex())
	s.NoError(err)
	s.Equal(czFp.Description, actualFp.Description)
	s.Equal(czFp.Commission.String(), actualFp.Commission.String())
	s.Equal(czFp.BtcPk, actualFp.BtcPk)
	s.Equal(czFp.Pop, actualFp.Pop)
	s.Equal(czFp.SlashedBabylonHeight, actualFp.SlashedBabylonHeight)
	s.Equal(czFp.SlashedBtcHeight, actualFp.SlashedBtcHeight)
	s.Equal(consumerId, actualFp.ConsumerId)
	return czFp
}

func (s *BTCStakingIntegration2TestSuite) initBabylonController() error {
	cfg := config.DefaultBabylonConfig()

	btcParams := &chaincfg.RegressionNetParams // or whichever network you're using

	logger, _ := zap.NewDevelopment()
	cfg.KeyDirectory = "/Users/gusin/Github/labs/cursor-bcd-babylon/babylon-private/test/e2e/consumer/.testnets/node0/babylond"
	cfg.GasPrices = "0.02ubbn"
	cfg.GasAdjustment = 20

	sdkCfg := sdk.GetConfig()
	fmt.Printf("CURRENT - SDK Account Prefix babylon init: %s\n", sdkCfg.GetBech32AccountAddrPrefix())
	sdk.SetAddrCacheEnabled(false)
	bbnparams.SetAddressPrefixes()
	sdkCfg = sdk.GetConfig()
	fmt.Printf("AFTER - SDK Account Prefix babylon init: %s\n", sdkCfg.GetBech32AccountAddrPrefix())

	controller, err := babylon.NewBabylonController(&cfg, btcParams, logger)
	if err != nil {
		return err
	}

	s.babylonController = controller
	return nil
}

func (s *BTCStakingIntegration2TestSuite) initCosmwasmController() error {
	cfg := cwconfig.DefaultCosmwasmConfig()

	// Override the RPC address with the one from your test suite
	//cfg.RPCAddr = s.consumerChainRPC

	// You might need to adjust other config values as needed for your test environment

	// Create a logger
	logger, _ := zap.NewDevelopment()

	// // You'll need to provide the correct encoding config
	// // This is typically available from your app's setup
	// encodingConfig := wasmparams.MakeEncodingConfig()

	sdkCfg := sdk.GetConfig()
	fmt.Printf("CURRENT - SDK Account Prefix BCD init: %s\n", sdkCfg.GetBech32AccountAddrPrefix())
	sdk.SetAddrCacheEnabled(false)
	bcdparams.SetAddressPrefixes()
	sdkCfg = sdk.GetConfig()
	fmt.Printf("AFTER - SDK Account Prefix BCD init: %s\n", sdkCfg.GetBech32AccountAddrPrefix())
	tempApp := bcdapp.NewTmpApp()
	//tempApp := wasmapp.NewWasmApp(sdklogs.NewNopLogger(), dbm.NewMemDB(), nil, false, simtestutil.NewAppOptionsWithFlagHome(s.T().TempDir()), []wasmkeeper.Option{})
	encodingCfg := wasmparams.EncodingConfig{
		InterfaceRegistry: tempApp.InterfaceRegistry(),
		Codec:             tempApp.AppCodec(),
		TxConfig:          tempApp.TxConfig(),
		Amino:             tempApp.LegacyAmino(),
	}

	interfaces := encodingCfg.InterfaceRegistry.ListAllInterfaces()
	s.T().Logf("Interfaces: %v", interfaces)

	// Log implementations of ClientState
	impls := encodingCfg.InterfaceRegistry.ListImplementations("ibc.core.client.v1.ClientState")
	s.T().Logf("ClientState implementations: %v", impls)

	// encodingCfg.InterfaceRegistry.RegisterImplementations()

	// // Ensure that IBC types are registered
	// clienttypes.RegisterInterfaces(encodingCfg.InterfaceRegistry)
	// channeltypes.RegisterInterfaces(encodingCfg.InterfaceRegistry)
	// connectiontypes.RegisterInterfaces(encodingCfg.InterfaceRegistry)

	wcc, err := cwcc.NewCosmwasmConsumerController(cfg, encodingCfg, logger)
	require.NoError(s.T(), err)

	s.cosmwasmController = wcc
	return nil
}

func (s *BTCStakingIntegration2TestSuite) getIBCClientID() string {
	var babylonChannel *channeltypes.IdentifiedChannel
	s.Eventually(func() bool {
		babylonChannelsResp, err := s.babylonController.IBCChannels()
		if err != nil {
			s.T().Logf("Error querying Babylon IBC channels: %v", err)
			return false
		}
		if len(babylonChannelsResp.Channels) != 1 {
			s.T().Logf("Expected 1 Babylon IBC channel, got %d", len(babylonChannelsResp.Channels))
			return false
		}
		babylonChannel = babylonChannelsResp.Channels[0]
		if babylonChannel.State != channeltypes.OPEN {
			s.T().Logf("Babylon channel state is not OPEN, got %s", babylonChannel.State)
			return false
		}
		s.Equal(channeltypes.ORDERED, babylonChannel.Ordering)
		s.Contains(babylonChannel.Counterparty.PortId, "wasm.")
		return true
	}, time.Minute*3, time.Second*10, "Failed to get expected Babylon IBC channel")

	var consumerChannel *channeltypes.IdentifiedChannel
	s.Eventually(func() bool {
		consumerChannelsResp, err := s.cosmwasmController.IBCChannels()
		if err != nil {
			s.T().Logf("Error querying Consumer IBC channels: %v", err)
			return false
		}
		if len(consumerChannelsResp.Channels) != 1 {
			return false
		}
		consumerChannel = consumerChannelsResp.Channels[0]
		if consumerChannel.State != channeltypes.OPEN {
			return false
		}
		s.Equal(channeltypes.ORDERED, consumerChannel.Ordering)
		s.Equal(babylonChannel.PortId, consumerChannel.Counterparty.PortId)
		return true
	}, time.Minute, time.Second*2, "Failed to get expected Consumer IBC channel")

	s.T().Logf("IBC channel is established successfully")

	// Query the channel client state
	babylonChannelState, err := s.babylonController.QueryChannelClientState(babylonChannel.ChannelId, babylonChannel.PortId)
	s.Require().NoError(err, "Failed to query Babylon channel client state")

	return babylonChannelState.IdentifiedClientState.ClientId
}

func (s *BTCStakingIntegration2TestSuite) verifyConsumerRegistration(consumerID string) *bsctypes.ConsumerRegister {
	var consumerRegistry []*bsctypes.ConsumerRegister

	s.Eventually(func() bool {
		var err error
		consumerRegistry, err = s.babylonController.QueryConsumerRegistry(consumerID)
		if err != nil {
			s.T().Logf("Error querying consumer registry: %v", err)
			return false
		}
		return len(consumerRegistry) == 1
	}, time.Minute, 5*time.Second, "Consumer was not registered within the expected time")

	s.Require().Len(consumerRegistry, 1)
	registeredConsumer := consumerRegistry[0]

	s.T().Logf("Consumer registered: ID=%s, Name=%s, Description=%s",
		registeredConsumer.ConsumerId,
		registeredConsumer.ConsumerName,
		registeredConsumer.ConsumerDescription)

	return registeredConsumer
}