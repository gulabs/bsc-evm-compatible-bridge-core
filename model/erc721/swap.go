package erc721

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/block"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

type SwapState string

const (
	SwapStateRequestOngoing     SwapState = "request_ongoing"
	SwapStateRequestRejected    SwapState = "request_rejected"
	SwapStateRequestConfirmed   SwapState = "request_confirmed"
	SwapStateFillTxDryRunFailed SwapState = "fill_tx_dry_run_failed"
	SwapStateFillTxCreated      SwapState = "fill_tx_created"
	SwapStateFillTxSent         SwapState = "fill_tx_sent"
	SwapStateFillTxConfirmed    SwapState = "fill_tx_confirmed"
	SwapStateFillTxFailed       SwapState = "fill_tx_failed"
	SwapStateFillTxMissing      SwapState = "fill_tx_missing"
)

type Swap struct {
	ID string `gorm:"size:26;primary_key"`

	// Basic Token Information
	SrcChainID   string `gorm:"not null"`
	DstChainID   string `gorm:"not null"`
	SrcTokenAddr string `gorm:"not null"`
	DstTokenAddr string
	SrcTokenName string
	DstTokenName string
	Sender       string `gorm:"not null"`
	Receiver     string `gorm:"not null"`
	TokenID      string `gorm:"not null"`
	Signature    string `gorm:"not null"`

	// Swap State
	State SwapState `gorm:"not null"`

	// Request Transaction Information
	RequestTxHash     string     `gorm:"not null"`
	RequestHeight     int64      `gorm:"not null"`
	RequestBlockHash  string     `gorm:"not null"`
	RequestBlockLogID *string    `gorm:"size:26;index:foreign_key_request_block_log_id"`
	RequestBlockLog   *block.Log `gorm:"foreignKey:RequestBlockLogID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	// Fill Transaction Information
	FillConsumedFeeAmount string
	FillGasPrice          string
	FillGasUsed           int64
	FillHeight            int64
	FillTxHash            string
	FillTrackRetry        int64
	FillBlockHash         string     `gorm:"not null"`
	FillBlockLogID        *string    `gorm:"size:26;index:foreign_key_fill_block_log_id"`
	FillBlockLog          *block.Log `gorm:"foreignKey:FillBlockLogID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	MessageLog string

	// Timestamp
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Swap) TableName() string {
	return "erc721_swap"
}

func (s *Swap) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = util.ULID()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Swap) SignaturePayload() string {
	return fmt.Sprintf("%v#%v#%v#%v#%v#%v#%v#%v#%v#%v#%v#%v#%v#%v",
		s.State,
		s.SrcChainID,
		s.DstChainID,
		s.SrcTokenAddr,
		s.DstTokenAddr,
		s.SrcTokenName,
		s.DstTokenName,
		s.Sender,
		s.Receiver,
		s.TokenID,
		s.RequestTxHash,
		s.RequestHeight,
		s.FillTxHash,
		s.FillHeight,
	)
}

func (s *Swap) VerifySignature(hmacKey string) bool {
	oldSig := s.Signature
	s.UpdateSignature(hmacKey)
	newSig := s.Signature

	return oldSig == newSig
}

func (s *Swap) UpdateSignature(hmacKey string) {
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(s.SignaturePayload()))
	s.Signature = hex.EncodeToString(mac.Sum(nil))
}
