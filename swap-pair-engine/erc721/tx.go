package engine

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

func (e *Engine) retrieveTx(txHash, chainID string) (*types.Transaction, bool, error) {
	if e.deps.Client[chainID] == nil {
		return nil, false, errors.Errorf("[Engine.retrieveTx]: client for chain id %s is not supported", chainID)
	}

	ctx := context.Background()
	tx, isPending, err := e.deps.Client[chainID].TransactionByHash(ctx, common.HexToHash(txHash))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, isPending, nil
		}

		return nil, isPending, errors.Wrap(err, "[Engine.retrieveTx]: failed to get a transaction")
	}

	return tx, isPending, nil
}

func (e *Engine) retrieveTxReceipt(txHash, chainID string) (*types.Receipt, error) {
	if e.deps.Client[chainID] == nil {
		return nil, errors.Errorf("[Engine.retrieveTxReceipt]: client for chain id %s is not supported", chainID)
	}

	ctx := context.Background()
	txRecipient, err := e.deps.Client[chainID].TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}

		return nil, errors.Wrap(err, "[Engine.retrieveTxReceipt]: failed to get a transaction receipt")
	}

	return txRecipient, nil
}

func (e *Engine) retrieveDstTokenAddr(height uint64, fromTokenAddr, registerTxHash, chainID string) (string, error) {
	if e.deps.Client[chainID] == nil {
		return "", errors.Errorf("[Engine.retrieveDstTokenAddr]: client for chain id %s is not supported", chainID)
	}

	ctx := context.Background()
	opts := bind.FilterOpts{
		Start:   height,
		End:     &height,
		Context: ctx,
	}
	txHash := [32]byte(common.HexToHash(registerTxHash))
	iter, err := e.deps.SwapAgent[chainID].FilterSwapPairCreated(&opts, [][32]byte{txHash}, nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "[Engine.retrieveDstTokenAddr]: failed to filter logs")
	}
	defer func() {
		if err := iter.Close(); err != nil {
			util.Logger.Errorf("[Engine.retrieveDstTokenAddr]: failed to close iterator, %s", err.Error())
		}
	}()

	for iter.Next() {
		if iter.Event.FromTokenAddr.String() == fromTokenAddr {
			return iter.Event.MirroredTokenAddr.String(), nil
		}
	}

	if err := iter.Error(); err != nil {
		return "", errors.Wrap(err, "[Recorder.retrieveDstTokenAddr]: failed to iterate events")
	}

	return "", nil
}

func (e *Engine) buildSignedTransaction(txInput []byte) (*types.Transaction, error) {
	contractAddr, ok := e.conf.SwapAgentAddresses[e.conf.ChainID.String()]
	if !ok {
		return nil, errors.Errorf("[Engine.buildSignedTransaction]: chain id %s is not supported", e.conf.ChainID.String())
	}

	ethClient, ok := e.deps.Client[e.conf.ChainID.String()]
	if !ok {
		return nil, errors.Errorf("[Engine.buildSignedTransaction]: chain id %s is not supported", e.conf.ChainID.String())
	}

	pKey, _, err := util.BuildKeys(e.conf.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.buildSignedTransaction]: failed to build keys")
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(pKey, e.conf.ChainID)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.buildSignedTransaction]: failed to create new transactor")
	}

	nonce, err := ethClient.PendingNonceAt(context.Background(), txOpts.From)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.buildSignedTransaction]: failed to get nonce")
	}

	gasPrice, err := ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.buildSignedTransaction]: failed to gas price")
	}

	value := big.NewInt(0)
	msg := ethereum.CallMsg{From: txOpts.From, To: &contractAddr, GasPrice: gasPrice, Value: value, Data: txInput}
	gasLimit, err := ethClient.EstimateGas(context.Background(), msg)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.buildSignedTransaction]: failed to estimate gas needed")
	}

	rawTx := types.NewTx(&types.AccessListTx{
		ChainID:  e.conf.ChainID,
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &contractAddr,
		Value:    value,
		Data:     txInput,
	})
	signedTx, err := txOpts.Signer(txOpts.From, rawTx)
	if err != nil {
		return nil, errors.Wrap(err, "[Engine.buildSignedTransaction]: failed to sign tx")
	}

	return signedTx, nil
}
