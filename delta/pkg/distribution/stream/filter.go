package stream

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/set"
)

type sampleFilter struct {
	keyCount int
	set      set.Set[channel.Key]
	confluence.LinearTransform[[]Sample, []Sample]
}

func newSampleFilter(keys channel.Keys) *sampleFilter {
	sf := &sampleFilter{set: keys.ToSet()}
	sf.TransformFunc.Transform = sf.filter
	sf.keyCount = len(keys)
	return sf
}

func (s *sampleFilter) filter(ctx context.Context, in []Sample) ([]Sample, bool, error) {
	out := make([]Sample, 0, s.keyCount)
	for _, sample := range in {
		if s.set.Contains(sample.ChannelKey) {
			out = append(out, sample)
		}
	}
	return out, true, nil
}
