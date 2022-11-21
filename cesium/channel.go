package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/legindex"
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

func (d *db) createChannels(chs *[]Channel) error {
	for i, ch := range *chs {
		if err := d.createChannel(&ch); err != nil {
			return err
		}
		(*chs)[i] = ch
	}
	return nil
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
		ch.Rate = legindex.IrregularRate
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
			"[cesium] - provided ch %s is not an index",
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
			"[cesium] - index ch can not be indexed",
		)
	}
	if ch.Density != telem.TimeStampDensity {
		return errors.Wrap(
			validate.Error,
			"[cesium] - index ch must use int64 timestamps",
		)
	}
	return nil
}

func (d *db) maybeCreateNewIndexes(ch *Channel) error {
	if !ch.IsIndex {
		return nil
	}
	i1 := &legindex.BinarySearch{
		Every: 100,
		Array: array.Searchable[legindex.Alignment]{
			Array: array.NewRolling[legindex.Alignment](10000),
		},
		ChannelKey: ch.Key,
	}
	d.indexes.indexes[ch.Key] = i1
	return nil
}
