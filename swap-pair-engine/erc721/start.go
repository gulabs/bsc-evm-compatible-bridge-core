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
	go e.run(e.manageOngoingRegistration, watchRegisterEventDelay)
	go e.run(e.manageConfirmedRegitration, watchRegisterEventDelay)
	go e.run(e.manageTxCreatedRegistration, watchRegisterEventDelay)
	go e.run(e.manageTxSentRegistration, watchRegisterEventDelay)
}

func (e *Engine) run(fn func(), delay time.Duration) {
	fnName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if delay.Seconds() == 0 {
		delay = watchEventDelay
	}

	for {
		time.Sleep(watchEventDelay)

		if e.deps.Recorder.LatestBlockCached() == nil {
			util.Logger.Infof("[Engine.run][%s]: no latest block cache found", fnName)

			continue
		}

		fn()
	}
}
