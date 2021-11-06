package engine

import (
	"reflect"
	"runtime"
	"time"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

const (
	watchEventDelay = time.Duration(5) * time.Second
)

func (e *Engine) Start() {
	go e.run(e.manageOngoingRequest, watchSwapEventDelay)
	go e.run(e.manageConfirmedSwap, watchSwapEventDelay)
	go e.run(e.manageTxCreatedSwap, watchSwapEventDelay)
	go e.run(e.manageTxSentSwap, watchSwapEventDelay)
}

func (e *Engine) run(fn func(), delay time.Duration) {
	fnName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if delay.Seconds() == 0 {
		delay = watchEventDelay
	}

	for {
		time.Sleep(watchEventDelay)

		if e.deps.Recorder[e.chainID()].LatestBlockCached() == nil {
			util.Logger.Infof("[Engine.run][%s]: no latest block cache found for chain id %s", fnName, e.chainID())

			continue
		}

		fn()
	}
}
