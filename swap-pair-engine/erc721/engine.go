package engine

import (
	"math/big"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/agent"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/client"
	corecommon "github.com/synycboom/bsc-evm-compatible-bridge-core/common"
)

type Recorder interface {
	LatestBlockCached() *corecommon.Block
}

type Config struct {
	ExplorerURL        string
	PrivateKey         string
	ChainID            *big.Int
	ConfirmNum         int64
	MaxTrackRetry      int64
	SwapAgentAddresses map[string]common.Address
}

type Dependencies struct {
	Client    map[string]client.ETHClient
	DB        *gorm.DB
	Recorder  Recorder
	SwapAgent map[string]agent.SwapAgent
}

type Engine struct {
	conf *Config
	deps *Dependencies
}

func NewEngine(c *Config, d *Dependencies) *Engine {
	return &Engine{
		conf: c,
		deps: d,
	}
}
