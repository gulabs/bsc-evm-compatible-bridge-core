package recorder

import (
	"math/big"

	"gorm.io/gorm"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/agent"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/client"
	corecommon "github.com/synycboom/bsc-evm-compatible-bridge-core/common"
)

type Config struct {
	ChainID   *big.Int
	ChainName string
	HMACKey   string
}

type Dependencies struct {
	Client    map[string]client.ETHClient
	DB        *gorm.DB
	SwapAgent map[string]agent.SwapAgent
}

type Recorder struct {
	latestBlockCached *corecommon.Block
	conf              *Config
	deps              *Dependencies
}

func NewRecorder(c *Config, d *Dependencies) *Recorder {
	return &Recorder{
		conf: c,
		deps: d,
	}
}
