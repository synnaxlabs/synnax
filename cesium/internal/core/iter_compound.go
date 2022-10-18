package core

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

type baseCompoundIterator[I BaseIterator] struct {
	iters []I
}

func (c *baseCompoundIterator[I]) Close() error {
	return c.execErr(func(i I) error { return i.Close() })
}

func (c *baseCompoundIterator[I]) Error() error {
	return c.execErr(func(i I) error { return i.Error() })
}

func (c *baseCompoundIterator[I]) Valid() bool {
	return c.exec(func(i I) bool { return i.Valid() })
}

func (c *baseCompoundIterator[I]) exec(f func(iter I) bool) bool {
	ok := false
	for _, iter := range c.iters {
		if f(iter) {
			ok = true
		}
	}
	return ok
}

func (c *baseCompoundIterator[I]) execErr(f func(iter I) error) (err error) {
	for _, iter := range c.iters {
		if iErr := f(iter); iErr != nil {
			err = iErr
		}
	}
	return err
}

type compoundPositionIterator struct {
	baseCompoundIterator[PositionIterator]
}

func NewCompoundPositionIterator(iters ...PositionIterator) PositionIterator {
	return &compoundPositionIterator{
		baseCompoundIterator: baseCompoundIterator[PositionIterator]{iters: iters},
	}
}

var _ PositionIterator = (*compoundPositionIterator)(nil)

func (c *compoundPositionIterator) Next(span position.Span) bool {
	return c.exec(func(i PositionIterator) bool { return i.Next(span) })
}

func (c *compoundPositionIterator) Prev(span position.Span) bool {
	return c.exec(func(i PositionIterator) bool { return i.Prev(span) })
}

func (c *compoundPositionIterator) SeekFirst() bool {
	return c.exec(func(i PositionIterator) bool { return i.SeekFirst() })
}

func (c *compoundPositionIterator) SeekLast() bool {
	return c.exec(func(i PositionIterator) bool { return i.SeekLast() })
}

func (c *compoundPositionIterator) SeekGE(pos position.Position) bool {
	return c.exec(func(i PositionIterator) bool { return i.SeekGE(pos) })
}

func (c *compoundPositionIterator) SeekLE(pos position.Position) bool {
	return c.exec(func(i PositionIterator) bool { return i.SeekLE(pos) })
}

func (c *compoundPositionIterator) View() position.Range {
	starts := make([]position.Position, len(c.iters))
	ends := make([]position.Position, len(c.iters))
	for i, w := range c.iters {
		r := w.View()
		starts[i] = r.Start
		ends[i] = r.End
	}
	return position.Range{Start: lo.Min(starts), End: lo.Max(ends)}
}

func (c *compoundPositionIterator) SetBounds(bounds position.Range) {
	for _, w := range c.iters {
		w.SetBounds(bounds)
	}
}

func (c *compoundPositionIterator) Bounds() position.Range {
	return c.iters[0].Bounds()
}

func (c *compoundPositionIterator) Value() []SegmentMD {
	var values []SegmentMD
	for _, w := range c.iters {
		values = append(values, w.Value()...)
	}
	return values
}

type compoundTimeIterator struct {
	baseCompoundIterator[TimeIterator]
}

func NewCompoundMDStampIterator(iters ...TimeIterator) TimeIterator {
	return &compoundTimeIterator{
		baseCompoundIterator: baseCompoundIterator[TimeIterator]{iters: iters},
	}
}

var _ TimeIterator = (*compoundTimeIterator)(nil)

func (c *compoundTimeIterator) Next(span telem.TimeSpan) bool {
	return c.exec(func(i TimeIterator) bool { return i.Next(span) })
}

func (c *compoundTimeIterator) Prev(span telem.TimeSpan) bool {
	return c.exec(func(i TimeIterator) bool { return i.Prev(span) })
}

func (c *compoundTimeIterator) SeekFirst() bool {
	return c.exec(func(i TimeIterator) bool { return i.SeekFirst() })
}

func (c *compoundTimeIterator) SeekLast() bool {
	return c.exec(func(i TimeIterator) bool { return i.SeekLast() })
}

func (c *compoundTimeIterator) SeekGE(stamp telem.TimeStamp) bool {
	return c.exec(func(i TimeIterator) bool { return i.SeekGE(stamp) })
}

func (c *compoundTimeIterator) SeekLE(stamp telem.TimeStamp) bool {
	return c.exec(func(i TimeIterator) bool { return i.SeekLE(stamp) })
}

func (c *compoundTimeIterator) View() telem.TimeRange {
	starts := make([]telem.TimeStamp, len(c.iters))
	ends := make([]telem.TimeStamp, len(c.iters))
	for i, w := range c.iters {
		r := w.View()
		starts[i] = r.Start
		ends[i] = r.End
	}
	return telem.TimeRange{Start: lo.Min(starts), End: lo.Max(ends)}
}

func (c *compoundTimeIterator) SetBounds(bounds telem.TimeRange) bool {
	return c.exec(func(i TimeIterator) bool {
		return i.SetBounds(bounds)
	})
}

func (c *compoundTimeIterator) Bounds() telem.TimeRange {
	return c.iters[0].Bounds()
}

func (c *compoundTimeIterator) Value() []SegmentMD {
	var values []SegmentMD
	for _, w := range c.iters {
		values = append(values, w.Value()...)
	}
	return values
}
