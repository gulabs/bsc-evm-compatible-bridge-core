package observer

import (
	"time"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/common"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/model"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

// Prune prunes the outdated blocks
func (ob *Observer) Prune() {
	for {
		curBlockLog, err := ob.GetCurrentBlockLog()
		if err != nil {
			util.Logger.Errorf("get current block log error, err=%s", err.Error())
			time.Sleep(common.ObserverPruneInterval)

			continue
		}
		err = ob.DB.Where("chain = ? and height < ?", ob.Executor.GetChainName(), curBlockLog.Height-common.ObserverMaxBlockNumber).Delete(model.BlockLog{}).Error
		if err != nil {
			util.Logger.Infof("prune block logs error, err=%s", err.Error())
		}
		time.Sleep(common.ObserverPruneInterval)
	}
}
