package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/validate"
)

var (
	ErrSegmentOverlap = errors.Wrap(validate.Error, "segment overlaps with a previous segment")
)

// indexProcessor receives a set of input segments, aligns them to the index,
// and returns the aligned indexes. In some cases, the processed segments may
// be split into multiple segments and may be used to update the index.
type indexProcessor interface {
	index.Keyed
	process(segment []core.SugaredSegment) ([]core.SugaredSegment, error)
	highWaterMark() position.Position
}

var (
	_ indexProcessor = (*rateIndexProcessor)(nil)
	_ indexProcessor = (*indexedIndexProcessor)(nil)
	_ indexProcessor = (*indexIndexProcessor)(nil)
)

type rateIndexProcessor struct {
	ch       Channel
	hwm      position.Position
	searcher index.Searcher
}

// Key implements the index.Keyed interface.
func (s *rateIndexProcessor) Key() ChannelKey { return s.searcher.Key() }

// highWaterMark implements the indexProcessor interface.
func (s *rateIndexProcessor) highWaterMark() position.Position { return s.hwm }

// process implements the indexProcessor interface.
func (s *rateIndexProcessor) process(segments []core.SugaredSegment) ([]core.SugaredSegment, error) {
	for i, seg := range segments {
		// Rate indexes are always exact.
		alignment, _ := s.searcher.SearchP(seg.Start, position.Uncertain)
		if alignment.Start.Before(s.hwm) {
			return nil, errors.Wrap(
				validate.Error,
				"segment overlaps with a previous segment",
			)
		}
		seg.Alignment = alignment.Start
		s.hwm = seg.End(s.ch.Density)
		segments[i] = seg
	}
	return segments, nil
}

type indexedIndexProcessor struct {
	ch       Channel
	hwm      position.Position
	searcher index.Searcher
	batch    core.MDBatch
}

// Key implements the index.Keyed interface.
func (s *indexedIndexProcessor) Key() ChannelKey { return s.searcher.Key() }

// highWaterMark implements the indexProcessor interface.
func (s *indexedIndexProcessor) highWaterMark() position.Position { return s.hwm }

// process implements the indexProcessor interface.
func (s *indexedIndexProcessor) process(segments []core.SugaredSegment) ([]core.SugaredSegment, error) {
	for i, seg := range segments {
		alignment, err := s.searcher.SearchP(seg.Start, position.Uncertain)
		if err != nil {
			return nil, err
		}
		//if !alignment.Exact() {
		//	return nil, errors.Wrap(
		//		validate.Error,
		//		"segment start does not align with a known position in the index",
		//	)
		//}
		//if alignment.Start.Before(s.hwm) {
		//	return nil, ErrSegmentOverlap
		//}
		//_, _ := s.batch.Retrieve(s.ch.Index, alignment.Start)
		//if errors.Is(err, query.NotFound) {
		//	return nil, errors.Wrap(
		//		validate.Error,
		//		"segment start timestamp must be aligned with the start of a segment in the index",
		//	)
		//}
		//if err != nil {
		//	return nil, err
		//}
		//if seg.Size() != md.Size {
		//	return nil, errors.Wrap(
		//		validate.Error,
		//		"segment size does not match the size of the segment in the index",
		//	)
		//}
		seg.Alignment = alignment.Start
		s.maybeUpdateHwm(seg)
		segments[i] = seg
	}
	return segments, nil
}

func (s *indexedIndexProcessor) maybeUpdateHwm(seg core.SugaredSegment) {
	maybeHwm := seg.End(s.ch.Density)
	if maybeHwm.After(s.hwm) {
		s.hwm = maybeHwm
	}
}

type indexIndexProcessor struct {
	channel  Channel
	searcher index.Searcher
	writer   index.Writer
	hwm      position.Position
}

func (p *indexIndexProcessor) Key() ChannelKey {
	return p.searcher.Key()
}

func (p *indexIndexProcessor) highWaterMark() position.Position { return p.hwm }

func (p *indexIndexProcessor) process(segments []core.SugaredSegment) ([]core.SugaredSegment, error) {
	for i, seg := range segments {
		alignment, err := p.searcher.SearchP(seg.Start, position.Uncertain)
		if err != nil {
			return nil, err
		}
		if alignment.Start.Before(p.hwm) {
			return nil, ErrSegmentOverlap
		}
		seg.Alignment = alignment.Start
		alignments, err := index.DecodeAlignments([]core.SugaredSegment{seg})
		if seg.Start != alignments[0].Stamp {
			return nil, errors.Wrapf(
				validate.Error,
				"segment start does not align with a known position in the index: %s != %s",
				seg.Start, alignments[0].Stamp,
			)
		}
		if err != nil {
			return nil, err
		}
		if err := p.writer.Write(alignments); err != nil {
			return nil, err
		}
		p.hwm = alignments[len(alignments)-1].Pos
		segments[i] = seg
	}
	return segments, nil
}
