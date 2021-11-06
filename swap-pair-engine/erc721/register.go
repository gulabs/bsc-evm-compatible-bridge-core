package engine

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/erc721"
)

var (
	querySwapPairLimit      = 50
	watchRegisterEventDelay = time.Duration(3) * time.Second
)

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
		s.BaseURI,
		s.SrcTokenName,
		s.Symbol,
	)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.sendCreatePairRequest]: failed to send swap pair creation tx")
	}

	return tx, nil
}
