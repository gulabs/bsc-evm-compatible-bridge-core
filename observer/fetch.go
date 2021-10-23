package observer

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

// GetCurrentBlockLog returns the highest block log
func (ob *Observer) GetCurrentBlockLog() (*model.BlockLog, error) {
	blockLog := model.BlockLog{}
	err := ob.DB.Where("chain = ?", ob.Executor.GetChainName()).Order("height desc").First(&blockLog).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(err, "[Observer.GetCurrentBlockLog]: failed to get data from block log")
	}

	return &blockLog, nil
}

// Fetch starts the main routine for fetching blocks of BSC
func (ob *Observer) Fetch(startHeight int64) {
	for {
		curBlockLog, err := ob.GetCurrentBlockLog()
		if err != nil {
			util.Logger.Errorf("get current block log from db error: %s", err.Error())
			ob.fetchSleep()
			continue
		}

		nextHeight := curBlockLog.Height + 1
		if curBlockLog.Height == 0 && startHeight != 0 {
			nextHeight = startHeight
		}

		util.Logger.Debugf("fetch %s block, height=%d", ob.Executor.GetChainName(), nextHeight)
		err = ob.fetchBlock(curBlockLog.Height, nextHeight, curBlockLog.BlockHash)
		if err != nil {
			util.Logger.Debugf("fetch %s block error, err=%s", ob.Executor.GetChainName(), err.Error())
			ob.fetchSleep()
		}
	}
}

func (ob *Observer) fetchSleep() {
	if ob.Executor.GetChainName() == ob.Config.ChainConfig.SourceChainName {
		time.Sleep(time.Duration(ob.Config.ChainConfig.SourceChainObserverFetchInterval) * time.Second)
	} else if ob.Executor.GetChainName() == ob.Config.ChainConfig.DestinationChainName {
		time.Sleep(time.Duration(ob.Config.ChainConfig.DestinationChainObserverFetchInterval) * time.Second)
	}
}

// fetchBlock fetches the next block of BSC and saves it to database. if the next block hash
// does not match to the parent hash, the current block will be deleted for there is a fork.
func (ob *Observer) fetchBlock(curHeight, nextHeight int64, curBlockHash string) error {
	blockAndEventLogs, err := ob.Executor.GetBlockAndTxEvents(nextHeight)
	if err != nil {
		return errors.Wrapf(err, "[Observer.fetchBlock]: get block info error, height=%d", nextHeight)
	}

	parentHash := blockAndEventLogs.ParentBlockHash
	if curHeight != 0 && parentHash != curBlockHash {
		// return ob.DeleteBlockAndTxEvents(curHeight)
	} else {
		nextBlockLog := model.BlockLog{
			BlockHash:  blockAndEventLogs.BlockHash,
			ParentHash: parentHash,
			Height:     blockAndEventLogs.Height,
			BlockTime:  blockAndEventLogs.BlockTime,
			Chain:      blockAndEventLogs.Chain,
		}

		err := ob.SaveBlockAndTxEvents(&nextBlockLog, blockAndEventLogs.Events)
		if err != nil {
			return err
		}

		// err = ob.UpdateSwapStartConfirmedNum(nextBlockLog.Height)
		// if err != nil {
		// 	return err
		// }
		// err = ob.UpdateSwapPairRegisterConfirmedNum(nextBlockLog.Height)
		// if err != nil {
		// 	return err
		// }
	}

	return nil
}

func (ob *Observer) SaveBlockAndTxEvents(blockLog *model.BlockLog, packages []interface{}) error {
	tx := ob.DB.Begin()
	if err := tx.Error; err != nil {
		return errors.Wrap(err, "[Observer.SaveBlockAndTxEvents]: cannot begin tx")
	}

	if err := tx.Create(blockLog).Error; err != nil {
		tx.Rollback()

		return errors.Wrap(err, "[Observer.SaveBlockAndTxEvents]: cannot create block log")
	}

	for _, pack := range packages {
		if err := tx.Create(pack).Error; err != nil {
			tx.Rollback()

			return errors.Wrap(err, "[Observer.SaveBlockAndTxEvents]: cannot create other events")
		}
	}

	return tx.Commit().Error
}
