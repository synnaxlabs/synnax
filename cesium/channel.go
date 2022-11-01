package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/array"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

func Keys(channels []Channel) []ChannelKey {
	return lo.Map(channels, func(ch Channel, i int) ChannelKey {
		return ch.Key
	})
}

func (d *db) retrieveChannel(key ChannelKey) (Channel, error) {
	return d.channels.GetChannel(key)
}

func (d *db) retrieveChannels(keys ...ChannelKey) ([]Channel, error) {
	return d.channels.GetChannels(keys...)
}

func (d *db) createChannel(ch *Channel) error {
	if err := d.validateChannel(ch); err != nil {
		return err
	}
	if err := d.applyChannelDefaults(ch); err != nil {
		return err
	}
	if err := d.maybeCreateNewIndexes(ch); err != nil {
		return err
	}
	return d.channels.SetChannel(*ch)
}

func (d *db) validateChannel(ch *Channel) error {
	v := validate.New("cesium")
	if ch.Index != 0 {
		v.Exec(func() error { return d.validateIndexExists(ch.Index) })
	} else if !ch.IsIndex {
		validate.Positive(v, "rate", ch.Rate)
	}
	if ch.Key != 0 {
		v.Exec(func() error { return d.validateChannelKeyNotAssigned(ch.Key) })
	}
	if ch.IsIndex {
		v.Exec(func() error { return d.validateNewIndexChannel(ch) })
	} else {
		validate.Positive(v, "density", ch.Density)
	}
	return v.Error()
}

func (d *db) applyChannelDefaults(ch *Channel) error {
	if ch.Index != 0 || ch.IsIndex {
		ch.Rate = index.IrregularRate
	}
	if ch.Key == 0 {
		key, err := d.channels.NextChannelKey()
		if err != nil {
			return err
		}
		ch.Key = key
	}
	return nil
}

func (d *db) validateIndexExists(idxKey ChannelKey) error {
	ch, err := d.channels.GetChannel(idxKey)
	if errors.Is(err, query.NotFound) {
		return errors.Wrapf(
			validate.Error,
			"[cesium] - provided index %s does not exist",
			idxKey,
		)
	}
	if err != nil {
		return err
	}
	if !ch.IsIndex {
		return errors.Wrapf(
			validate.Error,
			"[cesium] - provided channel %s is not an index",
			idxKey,
		)
	}
	return nil
}

func (d *db) validateChannelKeyNotAssigned(chKey ChannelKey) error {
	found, err := d.channels.ChannelsExist(chKey)
	if err != nil || !found {
		return err
	}
	return errors.Wrapf(
		validate.Error,
		"[cesium] - provided key %s already assigned",
		chKey,
	)
}

func (d *db) validateNewIndexChannel(ch *Channel) error {
	if ch.Index != 0 {
		return errors.Wrapf(
			validate.Error,
			"[cesium] - index channel can not be indexed",
		)
	}
	if ch.Density != telem.TimeStampDensity {
		return errors.Wrap(
			validate.Error,
			"[cesium] - index channel must use int64 timestamps",
		)
	}
	return nil
}

func (d *db) maybeCreateNewIndexes(ch *Channel) error {
	if !ch.IsIndex {
		return nil
	}
	var (
		idxWriter   index.CompoundWriter
		idxSearcher index.CompoundSearcher
	)
	i1 := &index.BinarySearch{
		Every: 100,
		Array: array.Searchable[index.Alignment]{
			Array: array.NewRolling[index.Alignment](10000),
		},
	}
	idxWriter = append(idxWriter, i1)
	idxSearcher = append(idxSearcher, i1)
	d.indexes.memWriters[ch.Key] = idxWriter
	d.indexes.memSearchers[ch.Key] = idxSearcher
	return nil
}
