package erc721

import (
	"time"

	"gorm.io/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

type SwapPairCreateTx struct {
	ID string `gorm:"size:26;primary_key"`

	SwapPairRegisterEventTxHash string                 `gorm:"index:swap_pair_register_event_tx_hash"`
	SwapPairRegisterEventID     *string                `gorm:"size:26;index:foreign_key_swap_pair_register_event"`
	SwapPairRegisterEvent       *SwapPairRegisterEvent `gorm:"foreignKey:SwapPairRegisterEventID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	SrcChainID string `gorm:"not null;index:filter_src_chain_state,priority:1"`
	DstChainID string `gorm:"not null"`

	Height            int64  `gorm:"not null;index:height"`
	TxHash            string `gorm:"not null;index:tx_hash"`
	GasPrice          string `gorm:"not null"`
	ConsumedFeeAmount string

	Log string

	State             SwapPairCreateState `gorm:"not null;index:filter_src_chain_state,priority:2"`
	TrackRetryCounter int                 `gorm:"not null;index:filter_src_chain_state,priority:3"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (SwapPairCreateTx) TableName() string {
	return "erc721_swap_pair_create_txs"
}

func (s *SwapPairCreateTx) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = util.ULID()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	return nil
}
