package observer

import (
	"github.com/jinzhu/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

type Observer struct {
	DB *gorm.DB

	StartHeight int64
	ConfirmNum  int64

	Config   *util.Config
	Executor Executor
}

// NewObserver returns the observer instance
func NewObserver(db *gorm.DB, startHeight, confirmNum int64, cfg *util.Config, executor Executor) *Observer {
	return &Observer{
		DB: db,

		StartHeight: startHeight,
		ConfirmNum:  confirmNum,

		Config:   cfg,
		Executor: executor,
	}
}

// Start starts the routines of observer
func (ob *Observer) Start() {
	go ob.Fetch(ob.StartHeight)
	go ob.Prune()
	go ob.Alert()
}
