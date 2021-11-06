package recorder

import (
	"gorm.io/gorm"

	"github.com/pkg/errors"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/model/block"
)

func (r *Recorder) Record(tx *gorm.DB, b *block.Log) error {
	if err := r.recordRegisterTx(tx, b); err != nil {
		return errors.Wrap(err, "[Recorder.Record]: failed to record register tx")
	}
	if err := r.recordSwapTx(tx, b); err != nil {
		return errors.Wrap(err, "[Recorder.Record]: failed to record swap tx")
	}
	if err := r.recordBackwardSwapTx(tx, b); err != nil {
		return errors.Wrap(err, "[Recorder.Record]: failed to record backward swap tx")
	}

	return nil
}
