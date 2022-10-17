package core

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/segment"
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

type compoundMDPositionIterator struct {
	baseCompoundIterator[MDPositionIterator]
}

func NewCompoundMDPositionIterator(iters ...MDPositionIterator) MDPositionIterator {
	return &compoundMDPositionIterator{
		baseCompoundIterator: baseCompoundIterator[MDPositionIterator]{iters: iters},
	}
}

var _ MDPositionIterator = (*compoundMDPositionIterator)(nil)

func (c *compoundMDPositionIterator) Next(span position.Span) bool {
	return c.exec(func(i MDPositionIterator) bool { return i.Next(span) })
}

func (c *compoundMDPositionIterator) Prev(span position.Span) bool {
	return c.exec(func(i MDPositionIterator) bool { return i.Prev(span) })
}

func (c *compoundMDPositionIterator) SeekFirst() bool {
	return c.exec(func(i MDPositionIterator) bool { return i.SeekFirst() })
}

func (c *compoundMDPositionIterator) SeekLast() bool {
	return c.exec(func(i MDPositionIterator) bool { return i.SeekLast() })
}

func (c *compoundMDPositionIterator) SeekGE(pos position.Position) bool {
	return c.exec(func(i MDPositionIterator) bool { return i.SeekGE(pos) })
}

func (c *compoundMDPositionIterator) SeekLE(pos position.Position) bool {
	return c.exec(func(i MDPositionIterator) bool { return i.SeekLE(pos) })
}

func (c *compoundMDPositionIterator) View() position.Range {
	starts := make([]position.Position, len(c.iters))
	ends := make([]position.Position, len(c.iters))
	for i, w := range c.iters {
		r := w.View()
		starts[i] = r.Start
		ends[i] = r.End
	}
	return position.Range{Start: lo.Min(starts), End: lo.Max(ends)}
}

func (c *compoundMDPositionIterator) SetBounds(bounds position.Range) {
	for _, w := range c.iters {
		w.SetBounds(bounds)
	}
}

func (c *compoundMDPositionIterator) Bounds() position.Range {
	return c.iters[0].Bounds()
}

func (c *compoundMDPositionIterator) Value() []segment.MD {
	var values []segment.MD
	for _, w := range c.iters {
		values = append(values, w.Value()...)
	}
	return values
}

type compoundMDStampIterator struct {
	baseCompoundIterator[MDStampIterator]
}

func NewCompoundMDStampIterator(iters ...MDStampIterator) MDStampIterator {
	return &compoundMDStampIterator{
		baseCompoundIterator: baseCompoundIterator[MDStampIterator]{iters: iters},
	}
}

var _ MDStampIterator = (*compoundMDStampIterator)(nil)

func (c *compoundMDStampIterator) Next(span telem.TimeSpan) bool {
	return c.exec(func(i MDStampIterator) bool { return i.Next(span) })
}

func (c *compoundMDStampIterator) Prev(span telem.TimeSpan) bool {
	return c.exec(func(i MDStampIterator) bool { return i.Prev(span) })
}

func (c *compoundMDStampIterator) SeekFirst() bool {
	return c.exec(func(i MDStampIterator) bool { return i.SeekFirst() })
}

func (c *compoundMDStampIterator) SeekLast() bool {
	return c.exec(func(i MDStampIterator) bool { return i.SeekLast() })
}

func (c *compoundMDStampIterator) SeekGE(stamp telem.TimeStamp) bool {
	return c.exec(func(i MDStampIterator) bool { return i.SeekGE(stamp) })
}

func (c *compoundMDStampIterator) SeekLE(stamp telem.TimeStamp) bool {
	return c.exec(func(i MDStampIterator) bool { return i.SeekLE(stamp) })
}

func (c *compoundMDStampIterator) View() telem.TimeRange {
	starts := make([]telem.TimeStamp, len(c.iters))
	ends := make([]telem.TimeStamp, len(c.iters))
	for i, w := range c.iters {
		r := w.View()
		starts[i] = r.Start
		ends[i] = r.End
	}
	return telem.TimeRange{Start: lo.Min(starts), End: lo.Max(ends)}
}

func (c *compoundMDStampIterator) SetBounds(bounds telem.TimeRange) bool {
	return c.exec(func(i MDStampIterator) bool { return i.SetBounds(bounds) })
}

func (c *compoundMDStampIterator) Bounds() telem.TimeRange {
	return c.iters[0].Bounds()
}

func (c *compoundMDStampIterator) Value() []segment.MD {
	var values []segment.MD
	for _, w := range c.iters {
		values = append(values, w.Value()...)
	}
	return values
}
