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

	corecommon "github.com/synycboom/bsc-evm-compatible-bridge-core/common"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

func (e *Engine) retrieveTx(txHash, chainID string) (*types.Transaction, bool, error) {
	if _, ok := e.deps.Client[chainID]; !ok {
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
	if _, ok := e.deps.Client[chainID]; !ok {
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

func (e *Engine) verifyForwardSwapFillEvent(height uint64, requestTxHash, chainID string) (bool, error) {
	if _, ok := e.deps.Client[chainID]; !ok {
		return false, errors.Errorf("[Engine.verifyForwardSwapFillEvent]: client for chain id %s is not supported", chainID)
	}

	ctx := context.Background()
	opts := bind.FilterOpts{
		Start:   height,
		End:     &height,
		Context: ctx,
	}
	txHash := [32]byte(common.HexToHash(requestTxHash))
	iter, err := e.deps.SwapAgent[chainID].FilterSwapFilled(&opts, [][32]byte{txHash}, nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "[Engine.verifyForwardSwapFillEvent]: failed to filter logs")
	}
	defer func() {
		if err := iter.Close(); err != nil {
			util.Logger.Errorf("[Engine.verifyForwardSwapFillEvent]: failed to close iterator, %s", err.Error())
		}
	}()

	return iter.Next(), nil
}

func (e *Engine) verifyBackwardSwapFillEvent(height uint64, requestTxHash, chainID string) (bool, error) {
	if _, ok := e.deps.Client[chainID]; !ok {
		return false, errors.Errorf("[Engine.verifyBackwardSwapFillEvent]: client for chain id %s is not supported", chainID)
	}

	ctx := context.Background()
	opts := bind.FilterOpts{
		Start:   height,
		End:     &height,
		Context: ctx,
	}
	txHash := [32]byte(common.HexToHash(requestTxHash))
	iter, err := e.deps.SwapAgent[chainID].FilterBackwardSwapFilled(&opts, [][32]byte{txHash}, nil, nil)
	if err != nil {
		return false, errors.Wrap(err, "[Engine.verifyBackwardSwapFillEvent]: failed to filter logs")
	}
	defer func() {
		if err := iter.Close(); err != nil {
			util.Logger.Errorf("[Engine.verifyBackwardSwapFillEvent]: failed to close iterator, %s", err.Error())
		}
	}()

	return iter.Next(), nil
}

func (e *Engine) retrieveTokenURI(tokenAddr, tokenID, chainID string) (string, error) {
	token, ok := e.deps.Token[chainID]
	if !ok {
		return "", errors.Errorf("[Engine.retrieveTokenURI]: unsupported chain id %s", chainID)
	}

	opts := &bind.CallOpts{
		Pending: true,
	}
	tID := util.StrToBigInt(tokenID)
	uri, err := token.TokenURI(opts, tokenAddr, tID)
	if err != nil {
		if strings.Contains(err.Error(), corecommon.ErrFunctionNotFound.Error()) {
			return "", nil
		}

		return "", errors.Wrap(err, "[Recorder.retrieveBaseURI]: failed to retrieve token URI")
	}

	return uri, nil
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
