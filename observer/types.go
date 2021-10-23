package observer

import "github.com/synycboom/bsc-evm-compatible-bridge-core/common"

type Executor interface {
	GetBlockAndTxEvents(height int64) (*common.BlockAndEventLogs, error)
	GetChainName() string
}
