package observer

import (
	"fmt"
	"time"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/common"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

// Alert sends alerts to tg group if there is no new block fetched in a specific time
func (ob *Observer) Alert() {
	for {
		curOtherChainBlockLog, err := ob.GetCurrentBlockLog()
		if err != nil {
			util.Logger.Errorf("get current block log error, err=%s", err.Error())
			time.Sleep(common.ObserverAlertInterval)

			continue
		}
		if curOtherChainBlockLog.Height > 0 {
			if time.Now().Unix()-curOtherChainBlockLog.CreateTime > ob.Config.AlertConfig.BlockUpdateTimeout {
				msg := fmt.Sprintf("last block fetched at %s, chain=%s, height=%d",
					time.Unix(curOtherChainBlockLog.CreateTime, 0).String(), ob.Executor.GetChainName(), curOtherChainBlockLog.Height)
				util.SendTelegramMessage(msg)
			}
		}

		time.Sleep(common.ObserverAlertInterval)
	}
}
