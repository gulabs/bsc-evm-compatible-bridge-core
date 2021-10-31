package engine

import (
	"context"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
	"gorm.io/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/block"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/erc721"
)

var (
	querySwapPairLimit      = 50
	watchRegisterEventDelay = time.Duration(3) * time.Second
)

func (e *Engine) manageOngoingRegistration() {
	fromChainID := e.chainID()
	ss, err := e.querySwapPair(fromChainID, []erc721.SwapPairState{
		erc721.SwapPairStateRegistrationOngoing,
	})
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageOngoingRegistration]: failed to query onging SwapPairs"))
		return
	}

	ss, err = e.filterConfirmedRegisterEvents(ss)
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageOngoingRegistration]: failed to filter confirmed SwapPairs"))
		return
	}

	if len(ss) == 0 {
		return
	}

	ids := make([]string, len(ss))
	for idx, s := range ss {
		ids[idx] = s.ID
	}

	if err := e.deps.DB.Model(&ss).Where("id in ?", ids).Updates(map[string]interface{}{
		"state": erc721.SwapPairStateRegistrationConfirmed,
	}).Error; err != nil {
		util.Logger.Error(
			errors.Wrapf(err, "[Engine.manageOngoingRegistration]: failed to update state '%s'", erc721.SwapPairStateRegistrationConfirmed),
		)
	}

	for _, s := range ss {
		util.Logger.Infof("[Engine.manageOngoingRegistration]: updated SwapPair %s state to '%s'", s.ID, s.State)
	}
}

func (e *Engine) manageConfirmedRegitration() {
	fromChainID := e.chainID()
	ss, err := e.querySwapPair(fromChainID, []erc721.SwapPairState{
		erc721.SwapPairStateRegistrationConfirmed,
	})
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageConfirmedRegitration]: failed to query confirmed SwapPairs"))
		return
	}

	for _, s := range ss {
		txHash, err := e.generateTxHash(s)
		if err != nil {
			// this error might comes from gas estimation, so it means we cannot send the real tx to the chain
			util.Logger.Warningf("[Engine.manageConfirmedRegitration]: failed to dry run tx of SwapPair %s", s.ID)

			s.State = erc721.SwapPairStateCreationTxDryRunFailed
			s.MessageLog = err.Error()
			if err := e.deps.DB.Save(s).Error; err != nil {
				util.Logger.Error(
					errors.Wrapf(err, "[Engine.manageConfirmedRegitration]: failed to update SwapPair %s to '%s' state", s.ID, s.State),
				)
			}

			continue
		}

		// We save the tx as our checkpoint to probe the stats later
		// It tells that this tx might be sent or might not, but it is okay
		// We will set the state to failed later
		s.State = erc721.SwapPairStateCreationTxCreated
		s.CreateTxHash = txHash
		s.CreateHeight = math.MaxInt64
		if err := e.deps.DB.Save(s).Error; err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageConfirmedRegitration]: failed to update SwapPair %s to '%s' state", s.ID, s.State),
			)

			continue
		}

		util.Logger.Infof(
			"[Engine.manageConfirmedRegitration]: sent dry run tx to chain id %s, %s",
			e.chainID(),
			txHash,
		)

		request, err := e.sendCreatePairRequest(s, false)
		if err != nil {
			// retry when a transaction is attempted to be replaced
			//  with a different one without the required price bump.
			if errors.Cause(err).Error() == core.ErrReplaceUnderpriced.Error() {
				s.State = erc721.SwapPairStateRegistrationConfirmed
				s.MessageLog = err.Error()
				if dbErr := e.deps.DB.Save(s).Error; dbErr != nil {
					util.Logger.Error(
						errors.Wrapf(dbErr, "[Engine.manageConfirmedRegitration]: failed to update SwapPair %s to '%s' state", s.ID, s.State),
					)

					continue
				}
			}

			s.State = erc721.SwapPairStateCreationTxFailed
			s.MessageLog = err.Error()
			if dbErr := e.deps.DB.Save(s).Error; dbErr != nil {
				util.Logger.Error(
					errors.Wrapf(dbErr, "[Engine.manageConfirmedRegitration]: failed to update SwapPair %s to '%s' state", s.ID, s.State),
				)

				continue
			}

			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageConfirmedRegitration]: failed to send a real tx %s of SwapPair %s", s.CreateTxHash, s.ID),
			)

			continue
		}

		util.Logger.Infof(
			"[Engine.manageConfirmedRegitration]: sent tx to chain id %s, %s/%s",
			e.chainID(),
			e.conf.ExplorerURL,
			request.Hash().String(),
		)

		// update tx hash again in case there are some parameters might change tx hash
		// for example, gas limit which comes from estimation
		s.CreateTxHash = request.Hash().String()
		if dbErr := e.deps.DB.Save(s).Error; dbErr != nil {
			util.Logger.Error(
				errors.Wrapf(dbErr, "[Engine.manageConfirmedRegitration]: failed to update SwapPair %s creation tx hash %s right after sending out", s.ID, s.CreateTxHash),
			)

			continue
		}
	}
}

func (e *Engine) manageTxCreatedRegistration() {
	fromChainID := e.chainID()
	ss, err := e.querySwapPair(fromChainID, []erc721.SwapPairState{
		erc721.SwapPairStateCreationTxCreated,
	})
	if err != nil {
		util.Logger.Error(errors.Wrap(err, "[Engine.manageTxCreatedRegistration]: failed to query tx_created SwapPairs"))
		return
	}

	for _, s := range ss {
		ethTx, isPending, err := e.retrieveTx(s.CreateTxHash, s.DstChainID)
		if err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageTxCreatedRegistration]: failed to get SwapPair creation tx %s", s.CreateTxHash),
			)

			continue
		}
		if isPending {
			util.Logger.Infof("[Engine.manageTxCreatedRegistration]: the tx %s is pending in mempools, skip", s.CreateTxHash)
			continue
		}

		receipt, err := e.retrieveTxReceipt(s.CreateTxHash, s.DstChainID)
		if err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageTxCreatedRegistration]: failed to get SwapPair creation receipt for tx %s", s.CreateTxHash),
			)

			continue
		}

		if ethTx == nil {
			util.Logger.Infof("[Engine.manageTxCreatedRegistration]: the tx is not found while cheking tx %s", s.CreateTxHash)
		}

		if receipt == nil {
			util.Logger.Infof("[Engine.manageTxCreatedRegistration]: the receipt is not found while cheking tx %s", s.CreateTxHash)
		}

		if ethTx == nil || receipt == nil {
			s.CreateTrackRetry += 1
			if s.CreateTrackRetry > e.conf.MaxTrackRetry {
				s.State = erc721.SwapPairStateCreationTxMissing
				s.MessageLog = "[Engine.manageTxCreatedRegistration]: tx is missing"
				if err := e.deps.DB.Save(s).Error; err != nil {
					util.Logger.Error(
						errors.Wrapf(err, "[Engine.manageTxCreatedRegistration]: failed to update SwapPair %s to '%s' state", s.ID, s.State),
					)

					continue
				}
			}

			continue
		}

		var b block.Log
		err = e.deps.DB.Where(
			"chain_id = ? and block_hash = ?",
			s.DstChainID,
			receipt.BlockHash.String(),
		).Select(
			"id",
		).First(
			&b,
		).Error
		if err == gorm.ErrRecordNotFound {
			util.Logger.Infof("[Engine.manageTxCreatedRegistration]: wait for the system to catch up the block %s in chain id %s", receipt.BlockHash.String(), e.chainID())

			continue
		}
		if err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageTxCreatedRegistration]: failed to update SwapPair %s to '%s' state", s.ID, s.State),
			)

			continue
		}

		createBlockHeight := receipt.BlockNumber.Int64()
		dstTokenAddr, err := e.retrieveDstTokenAddr(uint64(createBlockHeight), s.SrcTokenAddr, s.RegisterTxHash, s.DstChainID)
		if err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageTxCreatedRegistration]: failed to get destination token address for SwapPair %s", s.ID),
			)

			continue
		}

		if dstTokenAddr == "" {
			s.State = erc721.SwapPairStateCreationTxFailed
			s.MessageLog = "[Engine.manageTxCreatedRegistration]: destination token address was not found"
			if err := e.deps.DB.Save(s).Error; err != nil {
				util.Logger.Error(
					errors.Wrapf(err, "[Engine.manageTxCreatedRegistration]: failed to update SwapPair %s to '%s' state", s.ID, s.State),
				)

				continue
			}
		}

		gasPrice := big.NewInt(0)
		gasPrice.SetString(ethTx.GasPrice().String(), 10)
		s.DstTokenAddr = dstTokenAddr
		s.CreateGasPrice = gasPrice.String()
		s.CreateConsumedFeeAmount = big.NewInt(1).Mul(gasPrice, big.NewInt(s.CreateGasUsed)).String()
		s.CreateGasUsed = int64(receipt.GasUsed)
		s.CreateHeight = createBlockHeight
		s.CreateBlockHash = receipt.BlockHash.String()
		s.CreateBlockLogID = &b.ID
		s.State = erc721.SwapPairStateCreationTxSent
		if err := e.deps.DB.Save(s).Error; err != nil {
			util.Logger.Error(
				errors.Wrapf(err, "[Engine.manageTxCreatedRegistration]: failed to update SwapPair %s basic info", s.ID),
			)

			continue
		}

		util.Logger.Infof("[Engine.manageTxCreatedRegistration]: updated SwapPair %s after sending out with tx hash %s", s.ID, s.CreateTxHash)
	}
}

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

// querySwapPair queries SwapPair this engine is responsible
func (e *Engine) querySwapPair(fromChainID string, states []erc721.SwapPairState) ([]*erc721.SwapPair, error) {
	// TODO: check the index
	var ss []*erc721.SwapPair
	err := e.deps.DB.Where(
		"state in ? and src_chain_id = ?",
		states,
		fromChainID,
	).Order(
		"register_height asc",
	).Limit(
		querySwapPairLimit,
	).Find(&ss).Error
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.querySwapPair]: failed to query SwapPair")
	}

	return ss, nil
}

// filterConfirmedRegisterEvents checks block confirmation of the chain this engine is responsible
func (e *Engine) filterConfirmedRegisterEvents(ss []*erc721.SwapPair) (events []*erc721.SwapPair, err error) {
	for _, s := range ss {
		confirmed, err := e.hasBlockConfirmed(s.RegisterTxHash, e.chainID())
		if err != nil {
			util.Logger.Warning(errors.Wrap(err, "[Engine.filterConfirmedRegisterEvents]: failed to check block confirmation"))
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

func (e *Engine) generateTxHash(s *erc721.SwapPair) (string, error) {
	request, err := e.sendCreatePairRequest(s, true)
	if err != nil {
		return "", errors.Wrap(err, "[Engine.generateTxHash]: failed to dry run sending a request")
	}

	return request.Hash().String(), nil
}

// sendCreatePairRequest sends transaction to create a swap pair on destination chain
func (e *Engine) sendCreatePairRequest(s *erc721.SwapPair, dryRun bool) (*types.Transaction, error) {
	dstChainID := s.DstChainID
	dstChainIDInt := util.StrToBigInt(dstChainID)
	if _, ok := e.deps.Client[dstChainID]; !ok {
		return nil, errors.Errorf("[Engine.sendCreatePairRequest]: client for chain id %s is not supported", dstChainID)
	}
	if _, ok := e.deps.SwapAgent[dstChainID]; !ok {
		return nil, errors.Errorf("[Engine.sendCreatePairRequest]: swap agent for chain id %s is not supported", dstChainID)
	}

	ctx := context.Background()
	txOpts, err := util.TxOpts(ctx, e.deps.Client[dstChainID], e.conf.PrivateKey, dstChainIDInt)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.sendCreatePairRequest]: failed to create tx opts")
	}

	txOpts.NoSend = dryRun

	tx, err := e.deps.SwapAgent[dstChainID].CreateSwapPair(
		txOpts,
		common.HexToHash(s.RegisterTxHash),
		common.HexToAddress(s.SrcTokenAddr),
		util.StrToBigInt(s.SrcChainID),
		s.SrcTokenName,
		s.Symbol,
	)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.sendCreatePairRequest]: failed to send swap pair creation tx")
	}

	return tx, nil
}
