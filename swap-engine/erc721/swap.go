package engine

import (
	"context"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/erc721"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
	"gorm.io/gorm"
)

var (
	watchSwapEventDelay = time.Duration(3) * time.Second
	querySwapLimit      = 50
)

// TODO: Implement wait steps for waiting register pair ongoing
// might have to implement timeout for that

func (e *Engine) manageOngoingRequest() {
	fromChainID := e.chainID()
	ss, err := e.querySwap(fromChainID, []erc721.SwapState{
		erc721.SwapStateRequestOngoing,
	})
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageOngoingRequest]: failed to query onging Swaps"))
		return
	}

	if err := e.fillTokenURIs(ss); err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageOngoingRequest]: failed to fill tokenURI"))
		return
	}

	if err := e.fillDestination(ss); err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageOngoingRequest]: failed to fill destination"))
		return
	}

	ss, rr := e.filterRejectedSwapEvents(ss)
	for _, r := range rr {
		r.State = erc721.SwapStateRequestRejected
		if err := e.deps.DB.Save(&r).Error; err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageOngoingRequest]: failed to update updated Swap %s to state '%s'", r.ID, r.State),
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
				errors.Wrapf(err, "[Engine.manageOngoingRequest]: failed to update updated Swap %s to state '%s'", s.ID, s.State),
			)
		}
	}
}

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

func (e *Engine) generateTxHash(s *erc721.Swap) (string, error) {
	request, err := e.sendFillSwapRequest(s, true)
	if err != nil {
		return "", errors.Wrap(err, "[Engine.generateTxHash]: failed to dry run sending a request")
	}

	return request.Hash().String(), nil
}

// sendFillSwapRequest sends transaction to fill a swap on destination chain
func (e *Engine) sendFillSwapRequest(s *erc721.Swap, dryRun bool) (*types.Transaction, error) {
	dstChainID := s.DstChainID
	dstChainIDInt := util.StrToBigInt(dstChainID)
	if _, ok := e.deps.Client[dstChainID]; !ok {
		return nil, errors.Errorf("[Engine.sendFillSwapRequest]: client for chain id %s is not supported", dstChainID)
	}
	if _, ok := e.deps.SwapAgent[dstChainID]; !ok {
		return nil, errors.Errorf("[Engine.sendFillSwapRequest]: swap agent for chain id %s is not supported", dstChainID)
	}

	ctx := context.Background()
	txOpts, err := util.TxOpts(ctx, e.deps.Client[dstChainID], e.conf.PrivateKey, dstChainIDInt)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.sendFillSwapRequest]: failed to create tx opts")
	}

	txOpts.NoSend = dryRun

	tx, err := e.deps.SwapAgent[dstChainID].Fill(
		txOpts,
		common.HexToHash(s.RequestTxHash),
		common.HexToAddress(s.SrcTokenAddr),
		common.HexToAddress(s.Recipient),
		util.StrToBigInt(s.SrcChainID),
		util.StrToBigInt(s.TokenID),
		s.TokenURI,
	)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.sendFillSwapRequest]: failed to send swap pair creation tx")
	}

	return tx, nil
}

// querySwap queries Swap this engine is responsible
func (e *Engine) querySwap(fromChainID string, states []erc721.SwapState) ([]*erc721.Swap, error) {
	// TODO: check the index
	var ss []*erc721.Swap
	err := e.deps.DB.Where(
		"state in ? and src_chain_id = ?",
		states,
		fromChainID,
	).Order(
		"request_height asc",
	).Limit(
		querySwapLimit,
	).Find(&ss).Error
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.querySwap]: failed to query Swap")
	}

	return ss, nil
}

func (e *Engine) fillTokenURIs(ss []*erc721.Swap) error {
	for _, s := range ss {
		uri, err := e.retrieveTokenURI(s.SrcTokenAddr, s.TokenID, s.SrcChainID)
		if err != nil {
			return errors.Wrapf(err, "[Engine.fillTokenURIs]: failed to retrieve token uri of token %s, chain id %", s.SrcTokenAddr, s.SrcChainID)
		}
		if uri == "" {
			util.Logger.Infof("[Engine.fillTokenURIs]: token %s, chain id % has no token uri", s.SrcTokenAddr, s.SrcChainID)
		}

		s.TokenURI = uri
	}

	return nil
}

// fillDestination fills swap destination tokens
func (e *Engine) fillDestination(ss []*erc721.Swap) error {
	// TODO: check db index
	for _, s := range ss {
		var sp erc721.SwapPair
		err := e.deps.DB.Where(
			"src_token_addr = ? and src_chain_id = ? and dst_chain_id = ? and available = ?",
			s.SrcTokenAddr,
			s.SrcChainID,
			s.DstChainID,
			true,
		).First(&sp).Error

		s.RequestTrackRetry += 1
		if err == gorm.ErrRecordNotFound {
			continue
		}
		if err != nil {
			return errors.Wrap(err, "[Engine.fillDestination]: failed to query available Swaps")
		}

		s.SrcTokenName = sp.SrcTokenName
		s.DstTokenAddr = sp.DstTokenAddr
		s.DstTokenName = sp.DstTokenName
	}

	return nil
}

func (e *Engine) filterRejectedSwapEvents(ss []*erc721.Swap) (pass []*erc721.Swap, rejected []*erc721.Swap) {
	for _, s := range ss {
		if s.DstTokenAddr == "" && s.RequestTrackRetry > e.conf.MaxTrackRetry {
			rejected = append(rejected, s)
			continue
		}

		pass = append(pass, s)
	}

	return
}

// filterConfirmedSwapEvents checks block confirmation of the chain this engine is responsible
func (e *Engine) filterConfirmedSwapEvents(ss []*erc721.Swap) (events []*erc721.Swap, err error) {
	for _, s := range ss {
		confirmed, err := e.hasBlockConfirmed(s.RequestTxHash, e.chainID())
		if err != nil {
			util.Logger.Warning(errors.Wrap(err, "[Engine.filterConfirmedSwapEvents]: failed to check block confirmation"))
			continue
		}
		if confirmed {
			events = append(events, s)
		}
	}

	return events, nil
}

func (e *Engine) hasBlockConfirmed(txHash, chainID string) (bool, error) {
	if _, ok := e.deps.Recorder[chainID]; !ok {
		return false, errors.Errorf("[Engine.hasBlockConfirmed]: chain id %s is not supported", chainID)
	}

	block := e.deps.Recorder[chainID].LatestBlockCached()
	if block == nil {
		util.Logger.Infof("[Engine.hasBlockConfirmed]:: no latest block cache found for chain id %s", chainID)

		return false, nil
	}

	ctx := context.Background()
	txRecipient, err := e.deps.Client[chainID].TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return false, errors.Wrap(err, "[Engine.hasBlockConfirmed]: failed to get tx receipt")
	}
	if block.Height < txRecipient.BlockNumber.Int64()+e.conf.ConfirmNum {
		return false, nil
	}

	return true, nil
}
