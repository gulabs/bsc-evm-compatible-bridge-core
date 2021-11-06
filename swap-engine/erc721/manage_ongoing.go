package engine

import (
	"github.com/pkg/errors"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/erc721"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

func (e *Engine) manageOngoingRequest() {
	fromChainID := e.chainID()
	ss, err := e.querySwap(fromChainID, []erc721.SwapState{
		erc721.SwapStateRequestOngoing,
	})
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageOngoingRequest]: failed to query onging Swaps"))
		return
	}

	// Fill required information without updating to DB
	if err := e.fillRequiredInfo(ss); err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageOngoingRequest]: failed to fill destination"))
		return
	}

	// Separate ready Swaps, pending Swaps, and rejected Swaps
	ss, pp, rr := e.separateSwapEvents(ss)
	for _, r := range rr {
		r.State = erc721.SwapStateRequestRejected
		if err := e.deps.DB.Save(&r).Error; err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageOngoingRequest]: failed to update Swap %s to state '%s'", r.ID, r.State),
			)
		}
	}
	for _, p := range pp {
		if err := e.deps.DB.Save(&p).Error; err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageOngoingRequest]: failed to update Swap %s", p.ID),
			)
		}
	}

	ss, err = e.filterConfirmedSwapEvents(ss)
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageOngoingRequest]: failed to filter confirmed Swaps"))
		return
	}

	for _, s := range ss {
		s.State = erc721.SwapStateRequestConfirmed
		if err := e.deps.DB.Save(&s).Error; err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageOngoingRequest]: failed to update Swap %s to state '%s'", s.ID, s.State),
			)
		}
	}
}
