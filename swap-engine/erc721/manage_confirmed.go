package engine

import (
	"math"

	"github.com/ethereum/go-ethereum/core"
	"github.com/pkg/errors"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/erc721"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

func (e *Engine) manageConfirmedSwap() {
	fromChainID := e.chainID()
	ss, err := e.querySwap(fromChainID, []erc721.SwapState{
		erc721.SwapStateRequestConfirmed,
	})
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageConfirmedSwap]: failed to query confirmed Swaps"))
		return
	}

	for _, s := range ss {
		txHash, err := e.generateTxHash(s)
		if err != nil {
			// this error might comes from gas estimation, so it means we cannot send the real tx to the chain
			util.Logger.Warningf("[Engine.manageConfirmedSwap]: failed to dry run tx of Swap %s", s.ID)

			s.State = erc721.SwapStateFillTxDryRunFailed
			s.MessageLog = err.Error()
			if err := e.deps.DB.Save(s).Error; err != nil {
				util.Logger.Error(
					errors.Wrapf(err, "[Engine.manageConfirmedSwap]: failed to update Swap %s to '%s' state", s.ID, s.State),
				)
			}

			continue
		}

		// We save the tx as our checkpoint to probe the stats later
		// It tells that this tx might be sent or might not, but it is okay
		// We will set the state to failed later
		s.State = erc721.SwapStateFillTxCreated
		s.FillTxHash = txHash
		s.FillHeight = math.MaxInt64
		if err := e.deps.DB.Save(s).Error; err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageConfirmedSwap]: failed to update Swap %s to '%s' state", s.ID, s.State),
			)

			continue
		}

		util.Logger.Infof(
			"[Engine.manageConfirmedSwap]: sent dry run tx to chain id %s, %s",
			e.chainID(),
			txHash,
		)

		request, err := e.sendFillSwapRequest(s, false)
		if err != nil {
			// retry when a transaction is attempted to be replaced
			// with a different one without the required price bump.
			if errors.Cause(err).Error() == core.ErrReplaceUnderpriced.Error() {
				s.State = erc721.SwapStateRequestConfirmed
				s.MessageLog = err.Error()
				if dbErr := e.deps.DB.Save(s).Error; dbErr != nil {
					util.Logger.Error(
						errors.Wrapf(dbErr, "[Engine.manageConfirmedSwap]: failed to update Swap %s to '%s' state", s.ID, s.State),
					)

					continue
				}
			}

			s.State = erc721.SwapStateFillTxFailed
			s.MessageLog = err.Error()
			if dbErr := e.deps.DB.Save(s).Error; dbErr != nil {
				util.Logger.Error(
					errors.Wrapf(dbErr, "[Engine.manageConfirmedSwap]: failed to update Swap %s to '%s' state", s.ID, s.State),
				)

				continue
			}

			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageConfirmedSwap]: failed to send a real tx %s of Swap %s", s.FillTxHash, s.ID),
			)

			continue
		}

		util.Logger.Infof(
			"[Engine.manageConfirmedSwap]: sent tx to chain id %s, %s/%s",
			e.chainID(),
			e.conf.ExplorerURL,
			request.Hash().String(),
		)

		// update tx hash again in case there are some parameters might change tx hash
		// for example, gas limit which comes from estimation
		s.FillTxHash = request.Hash().String()
		if dbErr := e.deps.DB.Save(s).Error; dbErr != nil {
			util.Logger.Error(
				errors.Wrapf(dbErr, "[Engine.manageConfirmedSwap]: failed to update Swap %s fill tx hash %s right after sending out", s.ID, s.FillTxHash),
			)

			continue
		}
	}
}
