package engine

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

// fillRequiredInfo fills swap destination tokens
func (e *Engine) fillRequiredInfo(ss []*erc721.Swap) error {
	// TODO: check db index
	for _, s := range ss {
		if s.IsRequiredInfoValid() {
			continue
		}

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
			return errors.Wrap(err, "[Engine.fillRequiredInfo]: failed to query available Swaps")
		}

		tokenURI, err := e.retrieveTokenURI(s.SrcTokenAddr, s.TokenID, s.SrcChainID)
		if err != nil {
			return errors.Wrapf(err, "[Engine.fillRequiredInfo]: failed to retrieve token uri of token %s, chain id %", s.SrcTokenAddr, s.SrcChainID)
		}
		if tokenURI == "" {
			util.Logger.Infof("[Engine.fillRequiredInfo]: token %s, chain id % has no token uri", s.SrcTokenAddr, s.SrcChainID)
		}

		s.SetRequiredInfo(&sp, tokenURI)
	}

	return nil
}

func (e *Engine) separateSwapEvents(ss []*erc721.Swap) (pass []*erc721.Swap, pending []*erc721.Swap, rejected []*erc721.Swap) {
	for _, s := range ss {
		if !s.IsRequiredInfoValid() {
			if s.RequestTrackRetry > e.conf.MaxTrackRetry {
				rejected = append(rejected, s)
			} else {
				pending = append(pending, s)
			}

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
