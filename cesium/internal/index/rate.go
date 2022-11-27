package index

import (
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type Rate struct {
	Rate   telem.Rate
	Logger *zap.Logger
}

// Distance implements Index.
func (r Rate) Distance(tr telem.TimeRange, continuous bool) (count int64, err error) {
	r.Logger.Debug("idx distance",
		zap.Stringer("timeRange", tr),
		zap.Bool("continuous", continuous),
	)
	defer func() {
		r.Logger.Debug("idx distance done",
			zap.Stringer("timeRange", tr),
			zap.Bool("continuous", continuous),
			zap.Int64("count", count),
		)
	}()

	var sub int64 = 1
	// If we're above the end of the range slightly, we want to include the last sample.
	if telem.TimeSpan(tr.End)%r.Rate.Period() != 0 {
		sub = 0
	}

	count = int64(tr.Span()/r.Rate.Period()) - sub
	return
}

// Stamp implements Searcher.
func (r Rate) Stamp(ref telem.TimeStamp, sampleCount int64) (telem.TimeStamp, error) {
	end := ref.Add(r.Rate.Span(int(sampleCount)))
	r.Logger.Debug("idx stamp done", zap.Stringer("ref", ref), zap.Int64("sampleCount", sampleCount), zap.Stringer("end", end))
	return end, nil
}
