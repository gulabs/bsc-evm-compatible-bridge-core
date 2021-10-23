package model

import (
	"time"

	"github.com/jinzhu/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/common"
)

type ERC721SwapPair struct {
	gorm.Model
	Sponsor              string `gorm:"not null;index:sponsor"`
	Symbol               string `gorm:"not null;index:symbol"`
	Name                 string `gorm:"not null"`
	SourceTokenAddr      string `gorm:"not null"`
	DestinationTokenAddr string `gorm:"not null"`
	Available            bool   `gorm:"not null;index:available"`
	LowBound             string `gorm:"not null"`
	UpperBound           string `gorm:"not null"`
	IconUrl              string

	RecordHash string `gorm:"not null"`
}

func (ERC721SwapPair) TableName() string {
	return "erc721_swap_pairs"
}

type ERC721SwapPairRegisterTxLog struct {
	Id    int64
	Chain string `gorm:"not null;index:swappair_register_tx_log_chain"`

	Sponsor              string `gorm:"not null"`
	SourceTokenAddr      string `gorm:"not null"`
	DestinationTokenAddr string `gorm:"not null"`
	Symbol               string `gorm:"not null;index:swappair_register_tx_log_symbol"`
	Name                 string `gorm:"not null"`

	Status       TxStatus `gorm:"not null;index:swappair_register_tx_log_status"`
	TxHash       string   `gorm:"not null;index:swappair_register_tx_log_tx_hash"`
	BlockHash    string   `gorm:"not null"`
	Height       int64    `gorm:"not null"`
	ConfirmedNum int64    `gorm:"not null"`

	Phase TxPhase `gorm:"not null;index:swappair_register_tx_log_phase"`

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

	SourceTokenAddr      string `gorm:"not null"`
	DestinationTokenAddr string `gorm:"not null"`

	Symbol string `gorm:"not null;index:swap_pair_create_tx_symbol"`
	Name   string `gorm:"not null"`

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

	Status common.SwapPairStatus `gorm:"not null;index:swap_pair_sm_status"`

	DestinationTokenAddr string `gorm:"not null"`
	SourceTokenAddr      string

	Sponsor string `gorm:"not null"`
	Symbol  string `gorm:"not null;index:swap_pair_sm_symbol"`
	Name    string `gorm:"not null"`

	PairRegisterTxHash string `gorm:"not null"`
	PairCreateTxHash   string

	// used to log more message about how this erc721_swap_pair failed or invalid
	Log string

	RecordHash string `gorm:"not null"`
}

func (ERC721SwapPairStateMachine) TableName() string {
	return "erc721_swap_pair_sm"
}
