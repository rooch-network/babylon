package app

import (
	appparams "github.com/babylonlabs-io/babylon/app/params"
	epochingkeeper "github.com/babylonlabs-io/babylon/x/epoching/keeper"
	epochingtypes "github.com/babylonlabs-io/babylon/x/epoching/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// WrapStakingMsgDecorator defines an AnteHandler decorator that rejects all messages that might change the validator set.
type WrapStakingMsgDecorator struct {
	epk  epochingkeeper.Keeper
	encC *appparams.EncodingConfig
}

// NewWrapStakingMsgDecorator creates a new DropValidatorMsgDecorator
func NewWrapStakingMsgDecorator(epk *epochingkeeper.Keeper, encC *appparams.EncodingConfig) *WrapStakingMsgDecorator {
	return &WrapStakingMsgDecorator{
		epk:  *epk,
		encC: encC,
	}
}

// AnteHandle performs an AnteHandler will wrap all the staking msgs that will be sent to epoch.
// It will replace the following types of messages:
// - MsgCreateValidator -> MsgWrappedDelegate
// - MsgDelegate ->
// - MsgUndelegate ->
// - MsgBeginRedelegate ->
// - MsgCancelUnbondingDelegation ->
func (wd WrapStakingMsgDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// skip if at genesis block, as genesis state contains txs that bootstrap the initial validator set
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	allMsgs := tx.GetMsgs()
	newWrappedMsgs := make([]sdk.Msg, len(allMsgs))
	// after genesis, if validator-related message, reject msg
	for i, msg := range allMsgs {
		switch msg := msg.(type) {
		case *stakingtypes.MsgDelegate:
			newWrappedMsgs[i] = epochingtypes.NewMsgWrappedDelegate(msg)
			continue
		default:
			newWrappedMsgs[i] = msg
		}
	}

	txBuilder := wd.encC.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(newWrappedMsgs...); err != nil {
		return ctx, err
	}

	tx = txBuilder.GetTx()
	return next(ctx, tx, simulate)
}

// EnqueueStakingMsgIfNeeded checks if the given message is of non-wrapped type, which should be rejected
// func (wd WrapStakingMsgDecorator) EnqueueStakingMsgIfNeeded(msg sdk.Msg) error {
// 	switch msg := msg.(type) {
// 	case *stakingtypes.MsgDelegate:
// 		// wd.epk.EnqueueMsg()
// 		return epochingtypes.NewMsgWrappedDelegate(msg), nil
// 		// Do for others...
// 	case *stakingtypes.MsgCreateValidator, *stakingtypes.MsgUndelegate, *stakingtypes.MsgBeginRedelegate, *stakingtypes.MsgCancelUnbondingDelegation:
// 		return nil
// 	default:
// 		return nil
// 	}
// }
