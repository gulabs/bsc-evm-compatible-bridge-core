package model

import (
	"time"

	"github.com/jinzhu/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/common"
)

type ERC721SwapStartTxLog struct {
	Id int64

	DestinationChainName string `gorm:"not null;index:erc721_swap_start_tx_log_destination_chain_name"`
	DestinationTokenAddr string `gorm:"not null"`

	SourceChainName string `gorm:"not null;index:erc721_swap_start_tx_log_source_chain_name"`
	SourceTokenAddr string `gorm:"not null"`

	TokenID   string `gorm:"not null"`
	TokenURL  string `gorm:"not null"`
	FeeAmount string `gorm:"not null"`

	Status       TxStatus `gorm:"not null;index:erc721_swap_start_tx_log_status"`
	TxHash       string   `gorm:"not null;index:erc721_swap_start_tx_log_tx_hash"`
	BlockHash    string   `gorm:"not null"`
	Height       int64    `gorm:"not null"`
	ConfirmedNum int64    `gorm:"not null"`

	Phase TxPhase `gorm:"not null;index:erc721_swap_start_tx_log_phase"`

	UpdateTime int64
	CreateTime int64
}

func (ERC721SwapStartTxLog) TableName() string {
	return "erc721_swap_start_txs"
}

func (l *ERC721SwapStartTxLog) BeforeCreate() (err error) {
	l.CreateTime = time.Now().Unix()
	l.UpdateTime = time.Now().Unix()
	return nil
}

type ERC721SwapFillTx struct {
	gorm.Model

	Direction         common.SwapDirection `gorm:"not null"`
	StartSwapTxHash   string               `gorm:"not null;index:erc721_swap_fill_tx_start_swap_tx_hash"`
	FillSwapTxHash    string               `gorm:"not null;index:erc721_swap_fill_tx_fill_swap_tx_hash"`
	GasPrice          string               `gorm:"not null"`
	ConsumedFeeAmount string
	Height            int64
	Status            FillTxStatus `gorm:"not null"`
	TrackRetryCounter int64
}

func (ERC721SwapFillTx) TableName() string {
	return "erc721_swap_fill_txs"
}

type ERC721RetrySwap struct {
	gorm.Model

	Status               common.RetrySwapStatus `gorm:"not null"`
	SwapID               uint                   `gorm:"not null"`
	Direction            common.SwapDirection   `gorm:"not null"`
	StartTxHash          string                 `gorm:"not null;index:erc721_retry_swap_start_tx_hash"`
	FillTxHash           string                 `gorm:"not null"`
	Sponsor              string                 `gorm:"not null;index:erc721_retry_swap_sponsor"`
	SourceTokenAddr      string                 `gorm:"not null;index:erc721_retry_swap_source_token_addr"`
	SourceChainName      string                 `gorm:"not null;index:erc721_retry_swap_source_chain_name"`
	DestinationTokenAddr string                 `gorm:"not null;index:erc721_retry_swap_destination_token_addr"`
	DestinationChainName string                 `gorm:"not null;index:erc721_retry_swap_destination_chain_name"`
	Symbol               string                 `gorm:"not null"`
	TokenID              string                 `gorm:"not null"`
	TokenURL             string                 `gorm:"not null"`

	RecordHash string `gorm:"not null"`
	ErrorMsg   string
}

func (ERC721RetrySwap) TableName() string {
	return "erc721_retry_swaps"
}

type ERC721RetrySwapTx struct {
	gorm.Model

	RetrySwapID         uint                 `gorm:"not null;index:erc721_retry_swap_tx_retry_swap_id"`
	StartTxHash         string               `gorm:"not null;index:erc721_retry_swap_tx_start_tx_hash"`
	Direction           common.SwapDirection `gorm:"not null"`
	TrackRetryCounter   int64
	RetryFillSwapTxHash string            `gorm:"not null"`
	Status              FillRetryTxStatus `gorm:"not null"`
	ErrorMsg            string            `gorm:"not null"`
	GasPrice            string
	ConsumedFeeAmount   string
	Height              int64
}

func (ERC721RetrySwapTx) TableName() string {
	return "erc721_retry_swap_txs"
}

type ERC721Swap struct {
	gorm.Model

	Status common.SwapStatus `gorm:"not null;index:erc721_swap_status"`
	// the user addreess who start this swap
	Sponsor string `gorm:"not null;index:erc721_swap_sponsor"`

	SourceTokenAddr string `gorm:"not null;index:erc721_swap_source_token_addr"`
	SourceChainName string `gorm:"not null;index:erc721_swap_source_chain_name"`

	DestinationTokenAddr string `gorm:"not null;index:erc721_swap_destination_token_addr"`
	DestinationChainName string `gorm:"not null;index:erc721_swap_destination_chain_name"`

	Symbol   string `gorm:"not null"`
	TokenID  string `gorm:"not null"`
	TokenURL string `gorm:"not null"`

	Direction common.SwapDirection `gorm:"not null;index:erc721_swap_direction"`

	// The tx hash confirmed deposit
	StartTxHash string `gorm:"not null;index:erc721_swap_start_tx_hash"`
	// The tx hash confirmed withdraw
	FillTxHash string `gorm:"not null;index:erc721_swap_fill_tx_hash"`

	// used to log more message about how this swap failed or invalid
	Log string

	RecordHash string `gorm:"not null"`
}

func (ERC721Swap) TableName() string {
	return "erc721_swaps"
}
