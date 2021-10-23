package model

import (
	"time"

	"github.com/jinzhu/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/common"
)

type ERC721SwapPair struct {
	gorm.Model

	SourceTokenAddr string `gorm:"not null"`
	SourceChainName string `gorm:"not null"`
	SourceTokenName string `gorm:"not null"`

	DestinationTokenAddr string `gorm:"not null"`
	DestinationChainName string `gorm:"not null"`
	DestinationTokenName string `gorm:"not null"`

	Sponsor string `gorm:"not null;index:erc721_sponsor"`
	Symbol  string `gorm:"not null;index:erc721_symbol"`

	Available  bool   `gorm:"not null;index:erc721_available"`
	LowBound   string `gorm:"not null"`
	UpperBound string `gorm:"not null"`
	IconUrl    string

	RecordHash string `gorm:"not null"`
}

func (ERC721SwapPair) TableName() string {
	return "erc721_swap_pairs"
}

type ERC721SwapPairRegisterTxLog struct {
	Id int64

	SourceTokenAddr string `gorm:"not null"`
	SourceTokenName string `gorm:"not null"`
	SourceChainName string `gorm:"not null;index:erc721_swappair_register_tx_log_source_chain_name"`

	DestinationTokenAddr string `gorm:"not null"`
	DestinationTokenName string `gorm:"not null"`
	DestinationChainName string `gorm:"not null;index:erc721_swappair_register_tx_log_destination_chain_name"`

	Symbol  string `gorm:"not null;index:erc721_swappair_register_tx_log_symbol"`
	Sponsor string `gorm:"not null"`

	Status       TxStatus `gorm:"not null;index:erc721_swappair_register_tx_log_status"`
	TxHash       string   `gorm:"not null;index:erc721_swappair_register_tx_log_tx_hash"`
	BlockHash    string   `gorm:"not null"`
	Height       int64    `gorm:"not null"`
	ConfirmedNum int64    `gorm:"not null"`

	Phase TxPhase `gorm:"not null;index:erc721_swappair_register_tx_log_phase"`

	UpdateTime int64
	CreateTime int64
}

func (ERC721SwapPairRegisterTxLog) TableName() string {
	return "erc721_swap_pair_register_tx"
}

func (l *ERC721SwapPairRegisterTxLog) BeforeCreate() (err error) {
	l.CreateTime = time.Now().Unix()
	l.UpdateTime = time.Now().Unix()
	return nil
}

type ERC721SwapPairCreateTx struct {
	gorm.Model

	SwapPairRegisterTxHash string `gorm:"unique;not null"`
	SwapPairCreateTxHash   string `gorm:"unique;not null"`

	SourceTokenAddr string `gorm:"not null"`
	SourceTokenName string `gorm:"not null"`
	SourceChainName string `gorm:"not null"`

	DestinationTokenAddr string `gorm:"not null"`
	DestinationTokenName string `gorm:"not null"`
	DestinationChainName string `gorm:"not null"`

	Symbol string `gorm:"not null;index:erc721_swap_pair_create_tx_symbol"`

	GasPrice          string `gorm:"not null"`
	ConsumedFeeAmount string
	Height            int64
	Status            FillTxStatus `gorm:"not null"`
	TrackRetryCounter int64
}

func (ERC721SwapPairCreateTx) TableName() string {
	return "erc721_swap_pair_create_tx"
}

type ERC721SwapPairStateMachine struct {
	gorm.Model

	SourceTokenAddr string `gorm:"not null"`
	SourceChainName string `gorm:"not null"`
	SourceTokenName string `gorm:"not null"`

	DestinationTokenAddr string `gorm:"not null"`
	DestinationChainName string `gorm:"not null"`
	DestinationTokenName string `gorm:"not null"`

	Symbol string `gorm:"not null;index:erc721_swap_pair_sm_symbol"`

	Sponsor string                `gorm:"not null"`
	Status  common.SwapPairStatus `gorm:"not null;index:erc721_swap_pair_sm_status"`

	PairRegisterTxHash string `gorm:"not null"`
	PairCreateTxHash   string

	// used to log more message about how this erc721_swap_pair failed or invalid
	Log string

	RecordHash string `gorm:"not null"`
}

func (ERC721SwapPairStateMachine) TableName() string {
	return "erc721_swap_pair_sm"
}
