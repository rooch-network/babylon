package chain

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	bbn "github.com/babylonlabs-io/babylon/types"
	bstypes "github.com/babylonlabs-io/babylon/x/btcstaking/types"
	"github.com/stretchr/testify/require"
)

func (n *NodeConfig) RegisterConsumer(consumerId, consumerName, consumerDesc string) {
	n.LogActionF("registering consumer on babylon")

	cmd := []string{
		"babylond", "tx", "btcstkconsumer", "register-consumer", consumerId, consumerName, consumerDesc, "--from=val",
	}
	_, _, err := n.containerManager.ExecTxCmd(n.t, n.chainId, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully registered consumer on babylon")
}

func (n *NodeConfig) CreateConsumerFinalityProvider(walletAddrOrName string, btcPK *bbn.BIP340PubKey, pop *bstypes.ProofOfPossessionBTC, consumerId, moniker, identity, website, securityContract, details string, commission *sdkmath.LegacyDec) {
	n.LogActionF("creating consumer finality provider")

	// get BTC PK hex
	btcPKHex := btcPK.MarshalHex()
	// get pop hex
	popHex, err := pop.ToHexStr()
	require.NoError(n.t, err)

	cmd := []string{
		"babylond", "tx", "btcstaking", "create-finality-provider", btcPKHex, popHex,
		fmt.Sprintf("--from=%s", walletAddrOrName),
		"--consumer-id", consumerId, "--moniker", moniker, "--identity", identity, "--website", website, "--security-contact", securityContract, "--details", details, "--commission-rate", commission.String(),
	}
	_, _, err = n.containerManager.ExecTxCmd(n.t, n.chainId, n.Name, cmd)
	require.NoError(n.t, err)
	n.LogActionF("successfully created consumer finality provider")
}