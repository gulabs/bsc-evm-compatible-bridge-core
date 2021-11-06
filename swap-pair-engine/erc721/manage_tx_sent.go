package engine

import (
	"github.com/pkg/errors"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/erc721"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

func (e *Engine) manageTxSentRegistration() {
	fromChainID := e.chainID()
	ss, err := e.querySwapPair(fromChainID, []erc721.SwapPairState{
		erc721.SwapPairStateCreationTxSent,
	})
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageTxSentRegistration]: failed to query tx_sent SwapPairs"))
		return
	}

	var ids []string
	for _, s := range ss {
		confirmed, err := e.hasBlockConfirmed(s.CreateTxHash, s.DstChainID)
		if err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageTxSentRegistration]: failed to check block confirmation for SwapPair %s", s.ID),
			)

			continue
		}

		if confirmed {
			ids = append(ids, s.ID)
		}
	}

	if len(ids) == 0 {
		return
	}

	if err := e.deps.DB.Model(&ss).Where("id in ?", ids).Updates(map[string]interface{}{
		"state":     erc721.SwapPairStateCreationTxConfirmed,
		"available": true,
	}).Error; err != nil {
		util.Logger.Error(
			errors.Wrapf(err, "[Engine.manageTxSentRegistration]: failed to update state '%s'", erc721.SwapPairStateCreationTxConfirmed),
		)
	}

	for _, s := range ss {
		util.Logger.Infof("[Engine.manageTxSentRegistration]: updated SwapPair %s state to '%s'", s.ID, s.State)
	}
}
