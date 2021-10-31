package model

import (
	"gorm.io/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/block"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/erc721"
)

func InitTables(db *gorm.DB) {
	db.AutoMigrate(&block.Log{})
	db.AutoMigrate(&erc721.SwapPair{})
	db.AutoMigrate(&erc721.SwapPairCreateTx{})
	db.AutoMigrate(&erc721.SwapPairRegisterEvent{})
}
