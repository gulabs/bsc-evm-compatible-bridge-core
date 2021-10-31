package erc721

import (
	"time"

	"gorm.io/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/block"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

type SwapPairRegisterEvent struct {
	ID string `gorm:"size:26;primary_key"`

	BlockHash  string     `gorm:"not null"`
	BlockLogID *string    `gorm:"size:26;index:foreign_key_block_log_id"`
	BlockLog   *block.Log `gorm:"foreignKey:BlockLogID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	SrcChainID string `gorm:"not null;index:source_chain_id"`
	DstChainID string `gorm:"not null"`

	Height int64  `gorm:"not null"`
	TxHash string `gorm:"not null;index:tx_hash,unique"`

	Log string

	State SwapPairRegisterEventState `gorm:"not null;index:state"`

	Sponsor string `gorm:"not null"`
	Symbol  string `gorm:"not null"`

	SrcTokenAddr string `gorm:"not null"`
	SrcTokenName string `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (SwapPairRegisterEvent) TableName() string {
	return "erc721_swap_pair_register_events"
}

func (s *SwapPairRegisterEvent) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = util.ULID()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	return nil
}
