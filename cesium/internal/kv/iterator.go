package kv

import (
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
)

var RangeHasNoData = errors.New("[cesium.kv] - range has no data")

// Iterator is used to iterate over one or more channel's data.
// Iterator stores its values as a bounded range of segment headers.
// Iterator is not goroutine safe, but it is safe to open several iterators over the
// same data. Iterator segment metadata
type Iterator interface {
	// First moves the Iterator to the first segment in its Bounds. Returns true if:
	//
	//  	1. The Iterator is pointing at a valid segment within its Bounds.
	//
	//		2. Iterator.Error is nil.
	//
	// If true is returned, sets Iterator.View to the range of the first segment.
	First() bool
	// Last moves the Iterator to the last segment in its Bounds. Returns true if:
	//
	//		1. The Iterator is pointing at a valid segment within its Bounds.
	//
	//		2. Iterator.Error is nil
	//
	// If true is returned, sets Iterator.View to the range of the last segment.
	Last() bool
	// Next moves the Iterator to the next segment in its Bounds. Returns true if:
	//
	// 		1. The Iterator is pointing at a valid segment within its Bounds i.e.
	//		the Iterator bounds partially or completely overlap with the segment's
	//		range.
	//
	//		2. Iterator.Error is nil
	//
	// If true is returned, sets Iterator.View to the range of the segment. If the
	// segment range exceeds the iterator bounds, sets Value().Bounds.End to the
	// ends of the Iterator bounds. Constrains Iterator.View in the same manner.
	Next() bool
	// Prev moves the Iterator to the previous segment in its Bounds. Returns true if:
	//
	//		1. The Iterator is pointing at a valid segment within its Bounds i.e.
	//		the Iterator bounds partially or completely overlap with the segment's
	//		range.
	//
	//		2. Iterator.Error is nil.
	//
	// If true is returned, sets Iterator.View to the range of the segment. If the
	// segment range exceeds the iterator Bounds, sets Value().Bounds.Start to the
	// start of the Iterator.Bounds. Constrains Iterator.View in the same manner.
	Prev() bool
	// NextSpan moves the Iterator across the provided span, loading any segments it
	// encounters. Returns true if:
	//
	//		1. Iterator.View is pointing to a non-zero span within its Bounds.
	//	 	The span does not have to contain data for this condition to be met.
	//
	// 		2. Iterator.Error is nil.
	//
	NextSpan(span telem.TimeSpan) bool
	// PrevSpan moves the iterator backwards across the provided span, loading any
	// segments it encounters. Returns true if:
	//
	//		1. Iterator.View is pointing to a non-zero span within
	//	 	the Iterator.Bounds. The span does not have to contain data for
	//	 	this condition to be met.
	//
	//		2. Iterator.Error is nil.
	//
	PrevSpan(span telem.TimeSpan) bool
	// SetRange calls Seek to the start of the time range and calls NextSpan on the
	// span of the range. Returns true if both Seek and NextSpan return true.
	SetRange(tr telem.TimeRange) bool
	// SeekFirst seeks the Iterator to the first valid segment. Returns true if:
	//
	// 		1. The Iterator is pointing at a valid segment within its Bounds.
	//
	//		2. Iterator.Error is nil.
	//
	// If true is returned, sets Iterator.View to a zero-span range starting and ending
	// at the start of the first segment (constrained by Iterator.Bounds). Calls to
	// Iterator.Range will return an invalid Range. Next, NextSpan,
	// or SetRange must be called to load segments into Iterator.Range.
	SeekFirst() bool
	// SeekLast seeks the Iterator to the last valid segment.
	// Resets Iterator.Error.  Returns true if the Iterator is pointing at a valid
	// segment within its bounds.
	//
	// If true is returned, sets Iterator.View to a zero-span range starting and ending
	// at the end of the last segment (constrained by the Iterator.Bounds). Calls
	// to Iterator.Range will return an invalid Range. Prev, PrevSpan, or SetRange
	// must be called to load segments into Iterator.Range.
	SeekLast() bool
	// SeekLT seeks the Iterator to the first segment whose start timestamp is before
	// the provided stamp. Returns true if the Iterator is pointing to a valid
	// segment within Iterator.Bounds. Resets Iterator.Error.
	//
	// If true is returned, sets Iterator.View to a zero-span range starting and ending
	// at the start of the segment (constrained by Iterator.Bounds). Calls to Iterator.
	// Range will return an invalid Range. Next, NextSpan, Prev, PrevSpan or SetRange
	// must be called to load segments into Iterator.Range.
	SeekLT(stamp telem.TimeStamp) bool
	// SeekGE seeks the iterator to the first segment whose start timestamp
	// is after the provided stamp. Returns true if the Iterator is pointing
	// to a valid segment within Iterator.Bounds. Resets Iterator.Error.
	//
	// If true is returned, sets Iterator.View to a zero-span range starting and ending
	// at the start of the segment (constrained by Iterator.Bounds). Calls to Iterator.
	// Range will return an invalid Range. Calls to Iterator. Valid will return false.
	// Next, NextSpan, Prev, PrevSpan or SetRange must be called to load segments into
	// Iterator.Range and validate the Iterator.
	SeekGE(stamp telem.TimeStamp) bool
	// Seek seeks the Iterator to the provided timestamp. Returns true if stamp
	// is within the Iterator bounds. Sets the iterator view to a zero spanned
	// range starting and ending at the provided stamp. Calls to Iterator.Range
	// will return an invalid Range. Valid will return false. Next, NextSpan, Prev,
	// PrevSpan, or SetRange must be called to load segments into Iterator.Range and
	// validate the Iterator.
	Seek(stamp telem.TimeStamp) bool
	// Ranges returns a set of Range containing segments that overlap the current
	// Iterator.View. The segments in the range are not guaranteed to be contiguous or
	// contain data, but they are guaranteed to be sequential and overlap with the
	// Iterator.View. The bounds of reach range will be constrained by Iterator.View.
	//
	// It's important to note that the TimeRange returned by Range().TimeRange() will
	// either be equal to or contained in the current View. On the other hande,
	// Range().UnboundedRange() is either equal or contains Range().TimeRange() (meaning
	// it may be larger than the current View().
	Ranges() []*segment.Range
	// Range returns the Range for the first channel in the iterator. Returns
	// a nil range if the channel does not exist in the iterator. This is useful for
	// iterating over a single channel. See Ranges for details on what the returned
	// Range contains.
	Range() *segment.Range
	// GetValue returns the range for the provided channel key. Returns a nil range
	// if the channel does not exist in the iterator. Passing a channel key of 0
	// is identical to calling Range(). See Ranges for details on what the returned
	// Range contains.
	GetValue(key channel.Key) *segment.Range
	// View returns a telem.TimeRange representing the iterators current 'view' of the
	// data i.e. what is the potential set of segments currently loaded into Range().
	View() telem.TimeRange
	// Error returns any errors accumulated by the iterator. Error is reset after any
	// absolute call (First(), Last(), SeekFirst(), SeekLast(), SeekLT(),
	// SeekGE(), Seek(), SetRange()). If Error is non-nil, any relative calls (Next(),
	// Prev(), NextSpan(), PrevSpan(), or Valid()) will return false.
	Error() error
	// Valid returns true if the iterator View has a non-zero span and Iterator.Error
	// is not nil. This will return false after any seeking operations (Seek(),
	// SeekFirst(), SeekLast(), SeekLT(), SeekGE()).
	Valid() bool
	// Close closes the iterator. Returns any errors accumulated during iterator.
	Close() error
	// Bounds returns the bounds of the Iterator.
	Bounds() telem.TimeRange
}

// NewIterator opens a new unaryIterator over the specified time range of a channel.
func NewIterator(kve kv.DB, rng telem.TimeRange, keys ...channel.Key) (iter Iterator, err error) {
	if len(keys) == 0 {
		return nil, errors.New("[cesium] - iterator opened without keys")
	}
	if len(keys) == 1 {
		return newUnaryIterator(kve, rng, keys[0])
	}
	compound := make(compoundIterator, len(keys))
	for i, k := range keys {
		compound[i], err = newUnaryIterator(kve, rng, k)
		if err != nil {
			return nil, err
		}
	}
	return compound, err
}

type unaryIterator struct {
	internal *gorp.KVIterator[segment.Header]
	// view represents the iterators current 'view' of the time range (i.e. what
	// sub-range will calls to Value return). view is guaranteed to be within
	// the constraint range, but is not guaranteed to contain valid data.
	// view may have zero span (i.e. view.IsZero() is true).
	view                telem.TimeRange
	channel             channel.Channel
	rng                 *segment.Range
	bounds              telem.TimeRange
	_error              error
	_forceInternalValid bool
}

func newGorpHeaderIter(
	kve kv.Reader,
	opts kv.IteratorOptions,
) *gorp.KVIterator[segment.Header] {
	return gorp.WrapKVIter[segment.Header](
		kve.NewIterator(opts),
		gorp.WithEncoderDecoder(headerEncoderDecoder),
	)

}

// NewIterator opens a new unaryIterator over the specified time range of a channel.
func newUnaryIterator(kve kv.DB, rng telem.TimeRange, key channel.Key) (iter *unaryIterator, err error) {
	iter = &unaryIterator{bounds: telem.TimeRangeMax}

	chs, err := NewChannelService(kve).Get(key)
	if err != nil {
		iter.maybeSetError(err)
		return iter, iter.Error()
	}
	if len(chs) == 0 {
		iter.maybeSetError(query.NotFound)
		return iter, iter.Error()
	}

	iter.channel = chs[0]

	iter.internal = newGorpHeaderIter(kve, kv.PrefixIter(segment.NewKeyPrefix(key)))

	start, end := iter.key(rng.Start).Bytes(), iter.key(rng.End).Bytes()

	// Seek to the first segment that starts before the start of the range. If the
	// segment range overlaps with our desired range, we'll use it as the starting
	// point for the iterator. Otherwise, we'll seek to the first segment that starts
	// after the start of the range. If this rng overlaps with our desired range,
	// we'll use it as the starting point for the iterator. Otherwise, return an
	// error that the range has no data.
	if iter.SeekLT(rng.Start) && iter.Next() && iter.Range().Range().OverlapsWith(
		rng) {
		start = iter.key(iter.Range().UnboundedRange().Start).Bytes()
	} else if iter.SeekGE(rng.Start) && iter.Next() && iter.Range().Range().OverlapsWith(rng) {
		start = iter.key(iter.Range().UnboundedRange().Start).Bytes()
	} else {
		return nil, errors.CombineErrors(RangeHasNoData, iter.internal.Close())
	}

	if iter.SeekGE(rng.End) && iter.Prev() && iter.Range().Range().OverlapsWith(rng) {
		end = iter.key(iter.Range().UnboundedRange().End).Bytes()
	} else if iter.SeekLT(rng.End) && iter.Next() && iter.Range().Range().OverlapsWith(rng) {
		end = iter.key(iter.Range().UnboundedRange().End).Bytes()
	} else {
		return nil, errors.CombineErrors(RangeHasNoData, iter.internal.Close())
	}

	iter.bounds = rng
	if err := iter.internal.Close(); err != nil {
		return nil, err
	}
	iter.internal = newGorpHeaderIter(kve, kv.RangeIter(start, end))
	return iter, nil
}

// First implements Iterator.
func (i *unaryIterator) First() bool {
	i.hardReset()

	if !i.internal.First() {
		return false
	}

	i.appendHeader()
	i.setView(i.Range().UnboundedRange())
	i.boundRangeToView()

	return i.Valid()
}

// Last implements Iterator.
func (i *unaryIterator) Last() bool {
	i.hardReset()

	if !i.internal.Last() {
		return false
	}

	i.appendHeader()
	i.setView(i.Range().UnboundedRange())
	i.boundRangeToView()

	return i.Valid()
}

// Next implements Iterator.
func (i *unaryIterator) Next() bool {
	if i.error() != nil {
		return false
	}
	i.resetValue()
	i.resetForceInternalValid()

	// If we're at a zero view in the iterator, it means we just completed a seek operation. Instead, we
	// just load the rng at the current view and update our view to span it. BUT if we're at the
	// end of the range, still call next to invalidate it.
	if (!i.View().IsZero() || i.View().End == i.bounds.End) && !i.internal.Next() {
		return false
	}

	i.appendHeader()
	i.setView(i.rng.UnboundedRange())
	i.boundRangeToView()

	return i.Valid()
}

// Prev implements Iterator.
func (i *unaryIterator) Prev() bool {
	if i.error() != nil {
		return false
	}
	i.resetValue()
	i.resetForceInternalValid()

	if (!i.View().IsZero() || i.View().Start == i.bounds.Start) && !i.internal.Prev() {
		return false
	}

	i.appendHeader()
	i.setView(i.rng.UnboundedRange())
	i.boundRangeToView()

	return i.Valid()
}

// NextSpan implements Iterator.
func (i *unaryIterator) NextSpan(span telem.TimeSpan) bool {
	if span < 0 {
		return i.PrevSpan(-1 * span)
	}
	if i.error() != nil {
		return false
	}
	i.resetForceInternalValid()

	rng := i.View().End.SpanRange(span)
	prevRange := i.rng.UnboundedRange()
	i.resetValue()

	// If the last segment from the previous rng had relevant data, we need to load it back in.
	if rng.OverlapsWith(prevRange) || (i.internal.Valid() && i.internal.Value().End(i.channel.Rate, i.channel.Density).After(rng.Start)) {
		i.appendHeader()
	}

	// If the current rng can't satisfy the range, we need to continue loading in segments until it does.
	if !i.Range().UnboundedRange().End.After(rng.End) {
		for i.internal.Next(); i.Range().UnboundedRange().End.Before(rng.End) && i.
			internal.Valid(); i.internal.Next() {
			i.appendHeader()
		}
	}

	i.setView(rng)
	i.boundRangeToView()

	// In the edge case that we're crossing over the global range boundary, we need to run some special logic ->
	// If we don't have a non-internal-_map error, we weren't already at the end before this call, and we have
	// a valid rng otherwise, then we need to force the internal kv.Write to be valid until the next movement
	// call.
	if rng.End.After(i.bounds.End) &&
		i.Error() == nil &&
		prevRange.End != i.bounds.End &&
		!i.Range().Empty() {
		i.forceInternalValid()
	}

	return i.Valid()
}

// PrevSpan implements Iterator.
func (i *unaryIterator) PrevSpan(span telem.TimeSpan) bool {
	if span < 0 {
		return i.NextSpan(-1 * span)
	}
	if i.error() != nil {
		return false
	}
	i.resetForceInternalValid()
	i.resetValue()

	rng := i.View().Start.SpanRange(-span)
	prevView := i.View()

	if !i.SeekLT(i.View().Start) {
		return false
	}

	if !i.Range().UnboundedRange().OverlapsWith(rng) {
		i.resetValue()
	}

	if i.Range().UnboundedRange().Start.After(rng.Start) {
		for i.internal.Prev(); i.Range().UnboundedRange().Start.After(rng.Start) && i.internal.Valid(); i.internal.Prev() {
			i.prependHeader()
		}
	}

	i.setView(rng)
	i.boundRangeToView()

	// If our iterator isn't valid, it means we've reached the first segment in the global range. If our unbounded
	// view from the previous iteration ISN'T already at the start, we need to force the iterator to be valid.
	if !i.internal.Valid() &&
		i.Error() == nil &&
		prevView.Start != i.bounds.Start &&
		!i.Range().Empty() {
		i.forceInternalValid()
	}

	return i.Valid()
}

// SetRange implements Iterator.
func (i *unaryIterator) SetRange(tr telem.TimeRange) bool {
	tr = tr.BoundBy(i.bounds)

	// First we try blindly seeking to the start of the range. If that works,
	// we can simply return all the data in that span.
	if i.Seek(tr.Start) {
		return i.NextSpan(tr.Span())
	}

	// If not we seek the first segment whose values may fall within the range.
	if !i.SeekGE(tr.Start) {
		return false
	}

	// If the range of the rng we found doesn't overlap with the range we're
	// looking for then we return false.
	if !i.rng.UnboundedRange().OverlapsWith(tr) {
		return false
	}

	if !i.NextSpan(telem.TimeRange{Start: i.View().Start, End: tr.End}.Span()) {
		return false
	}

	i.setView(tr)
	i.boundRangeToView()

	return true
}

// SeekFirst implements Iterator.
func (i *unaryIterator) SeekFirst() bool {
	i.hardReset()
	if !i.internal.First() {
		return false
	}
	i.appendHeader()
	i.setView(i.rng.UnboundedRange().Start.SpanRange(0))
	i.boundRangeToView()
	return true
}

// SeekLast implements Iterator.
func (i *unaryIterator) SeekLast() bool {
	i.hardReset()
	if !i.internal.Last() {
		return false
	}
	i.appendHeader()
	i.setView(i.rng.UnboundedRange().End.SpanRange(0))
	i.boundRangeToView()
	return true
}

// SeekLT implements Iterator.
func (i *unaryIterator) SeekLT(stamp telem.TimeStamp) bool {
	i.hardReset()
	if !i.internal.SeekLT(i.key(stamp).Bytes()) {
		return false
	}
	i.appendHeader()
	i.setView(i.rng.UnboundedRange().Start.SpanRange(0))
	i.boundRangeToView()
	return true
}

// SeekGE implements Iterator.
func (i *unaryIterator) SeekGE(stamp telem.TimeStamp) bool {
	i.hardReset()
	if !i.internal.SeekGE(i.key(stamp).Bytes()) {
		return false
	}
	i.appendHeader()
	i.setView(i.rng.UnboundedRange().Start.SpanRange(0))
	i.boundRangeToView()
	return true
}

func (i *unaryIterator) hardReset() {
	i.resetError()
	i.resetForceInternalValid()
	i.resetValue()
}

// Seek implements Iterator.
func (i *unaryIterator) Seek(stamp telem.TimeStamp) bool {
	// We need to seek the first rng whose data might contain our timestamp. If it does, we're pointing at a valid
	// segment and can return true. If the seekLT is invalid, it means we definitely can't find a valid segment.
	if ok := i.SeekLT(stamp); !ok {
		return false
	}

	// If the range of the rng we looked for doesn't contain our stamp, it means a
	// segment with data at that timestamp doesn't exist, so we return false.
	if !i.rng.UnboundedRange().ContainsStamp(stamp) {
		return false
	}

	i.setView(stamp.SpanRange(0))
	i.boundRangeToView()

	return true
}

// Range implements Iterator.
func (i *unaryIterator) Range() *segment.Range {
	return i.rng
}

// Ranges implements Iterator.
func (i *unaryIterator) Ranges() []*segment.Range { return []*segment.Range{i.rng} }

// GetValue implements Iterator.
func (i *unaryIterator) GetValue(key channel.Key) *segment.Range {
	if key != 0 && key != i.channel.Key {
		return nil
	}
	return i.Range()
}

// View implements Iterator.
func (i *unaryIterator) View() telem.TimeRange { return i.view }

// Error implements Iterator.
func (i *unaryIterator) Error() error {
	if i.error() != nil {
		return i.error()
	}
	return i.internal.Error()
}

func (i *unaryIterator) Bounds() telem.TimeRange { return i.bounds }

// Valid implements Iterator.
func (i *unaryIterator) Valid() bool {
	return !i.View().IsZero() && (i.internal.Valid() || i._forceInternalValid) && i.Error() == nil
}

// Close closes the iterator.
func (i *unaryIterator) Close() error {
	i.resetValue()
	return i.internal.Close()
}

func (i *unaryIterator) key(stamp telem.TimeStamp) segment.Key {
	return segment.NewKey(i.channel.Key, stamp)
}

func (i *unaryIterator) setView(rng telem.TimeRange) { i.view = rng.BoundBy(i.bounds) }

func (i *unaryIterator) appendHeader() {
	i.rng.Headers = append(i.rng.Headers, i.internal.Value())
}

func (i *unaryIterator) prependHeader() {
	i.rng.Headers = append([]segment.Header{i.internal.Value()}, i.rng.Headers...)
}

func (i *unaryIterator) boundRangeToView() { i.rng.Bounds = i.View() }

func (i *unaryIterator) resetValue() {
	i.rng = new(segment.Range)
	i.rng.Bounds = telem.TimeRangeMax
	i.rng.Channel = i.channel
}

func (i *unaryIterator) forceInternalValid() { i._forceInternalValid = true }

func (i *unaryIterator) resetForceInternalValid() { i._forceInternalValid = false }

func (i *unaryIterator) maybeSetError(err error) {
	if err != nil {
		i._error = err
	}
}

func (i *unaryIterator) resetError() { i._error = nil }

func (i *unaryIterator) error() error { return i._error }

type compoundIterator []*unaryIterator

// First implements Iterator.
func (i compoundIterator) First() bool {
	return i._map(func(it *unaryIterator) bool { return it.First() })
}

// Last implements Iterator.
func (i compoundIterator) Last() bool {
	return i._map(func(it *unaryIterator) bool { return it.Last() })
}

// Next implements Iterator.
func (i compoundIterator) Next() bool {
	return i._map(func(it *unaryIterator) bool { return it.Next() })
}

// Prev implements Iterator.
func (i compoundIterator) Prev() bool {
	return i._map(func(it *unaryIterator) bool { return it.Prev() })
}

// NextSpan implements Iterator.
func (i compoundIterator) NextSpan(span telem.TimeSpan) bool {
	return i._map(func(it *unaryIterator) bool { return it.NextSpan(span) })
}

// PrevSpan implements Iterator.
func (i compoundIterator) PrevSpan(span telem.TimeSpan) bool {
	return i._map(func(it *unaryIterator) bool { return it.PrevSpan(span) })
}

// SetRange implements Iterator.
func (i compoundIterator) SetRange(rng telem.TimeRange) bool {
	return i._map(func(it *unaryIterator) bool { return it.SetRange(rng) })
}

// SeekFirst implements Iterator.
func (i compoundIterator) SeekFirst() bool {
	return i._map(func(it *unaryIterator) bool { return it.SeekFirst() })
}

// SeekLast implements Iterator.
func (i compoundIterator) SeekLast() bool {
	return i._map(func(it *unaryIterator) bool { return it.SeekLast() })
}

// SeekLT implements Iterator.
func (i compoundIterator) SeekLT(stamp telem.TimeStamp) bool {
	return i._map(func(it *unaryIterator) bool { return it.SeekLT(stamp) })
}

// SeekGE implements Iterator.
func (i compoundIterator) SeekGE(stamp telem.TimeStamp) bool {
	return i._map(func(it *unaryIterator) bool { return it.SeekGE(stamp) })
}

// Seek implements Iterator.
func (i compoundIterator) Seek(stamp telem.TimeStamp) bool {
	return i._map(func(it *unaryIterator) bool { return it.Seek(stamp) })
}

// Range implements Iterator.
func (i compoundIterator) Range() *segment.Range { return i[0].Range() }

// Ranges implements Iterator.
func (i compoundIterator) Ranges() []*segment.Range {
	values := make([]*segment.Range, len(i))
	for i, it := range i {
		values[i] = it.Range()
	}
	return values
}

// GetValue implements Iterator.
func (i compoundIterator) GetValue(key channel.Key) *segment.Range {
	for _, it := range i {
		if it.channel.Key == key {
			return it.Range()
		}
	}
	return nil
}

// View implements Iterator.
func (i compoundIterator) View() telem.TimeRange {
	view := telem.TimeRange{Start: telem.TimeStampMax, End: telem.TimeStampMin}
	for _, it := range i {
		if !it.Valid() {
			return telem.TimeRangeZero
		}
		if it.View().Start.Before(view.Start) {
			view.Start = it.View().Start
		}
		if it.View().End.After(view.End) {
			view.End = it.View().End
		}
	}
	return view
}

// Error implements Iterator.
func (i compoundIterator) Error() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, it := range i {
		c.Exec(it.Error)
	}
	return c.Error()

}

// Valid implements Iterator.
func (i compoundIterator) Valid() bool {
	return i._map(func(it *unaryIterator) bool { return it.Valid() })
}

// Close implements Iterator.
func (i compoundIterator) Close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, it := range i {
		c.Exec(it.Close)
	}
	return c.Error()
}

// Bounds implements Iterator.
func (i compoundIterator) Bounds() telem.TimeRange { return i[0].Bounds() }

func (i compoundIterator) _map(f func(iter *unaryIterator) bool) bool {
	ok := true
	for _, it := range i {
		if !f(it) {
			ok = false
		}
	}
	return ok
}
