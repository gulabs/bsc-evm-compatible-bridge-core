package observer

import (
	"time"

	"gorm.io/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/common"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/block"
)

type Recorder interface {
	Block(height int64) (*common.Block, error)
	ChainID() string
	Record(tx *gorm.DB, block *block.Log) error
	Delete(tx *gorm.DB, height int64) error
}

type Config struct {
	StartHeight        int64
	ConfirmNum         int64
	FetchInterval      time.Duration
	BlockUpdateTimeout time.Duration
}

type Dependencies struct {
	DB       *gorm.DB
	Recorder Recorder
}

type Observer struct {
	conf *Config
	deps *Dependencies
}

// NewObserver returns the observer instance
func NewObserver(c *Config, d *Dependencies) *Observer {
	return &Observer{
		conf: c,
		deps: d,
	}
}

// Start starts the routines of observer
func (o *Observer) Start() {
	go o.Update()
	go o.Prune()
	go o.Alert()
}
