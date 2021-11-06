package engine

import (
	"github.com/pkg/errors"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/erc721"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

func (e *Engine) manageTxSentSwap() {
	fromChainID := e.chainID()
	ss, err := e.querySwap(fromChainID, []erc721.SwapState{
		erc721.SwapStateFillTxSent,
	})
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageTxSentSwap]: failed to query tx_sent Swaps"))
		return
	}

	var ids []string
	for _, s := range ss {
		confirmed, err := e.hasBlockConfirmed(s.FillTxHash, s.DstChainID)
		if err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageTxSentSwap]: failed to check block confirmation for Swap %s", s.ID),
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
		"state": erc721.SwapStateFillTxConfirmed,
	}).Error; err != nil {
		util.Logger.Error(
			errors.Wrapf(err, "[Engine.manageTxSentSwap]: failed to update state '%s'", erc721.SwapStateFillTxConfirmed),
		)
	}

	for _, s := range ss {
		util.Logger.Infof("[Engine.manageTxSentSwap]: updated Swap %s state to '%s'", s.ID, s.State)
	}
}
