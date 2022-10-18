package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/array"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

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
	return d.channels.SetChannel(*ch)
}

func (d *db) validateChannel(ch *Channel) error {
	v := validate.New("cesium")
	if ch.Index != 0 {
		v.Exec(func() error { return d.validateIndexExists(ch.Index) })
	} else {
		validate.Positive(v, "rate", ch.Rate)
	}
	if ch.Key != 0 {
		v.Exec(func() error { return d.validateChannelExists(ch.Key) })
	}
	if ch.IsIndex {
		v.Exec(func() error { return d.validateNewIndexChannel(ch) })
	} else {
		validate.Positive(v, "density", ch.Density)
	}
	return v.Error()
}

func (d *db) applyChannelDefaults(ch *Channel) error {
	if ch.Index != 0 {
		ch.Rate = index.IrregularRate
	}
	if ch.Key != 0 {
		key, err := d.channels.NextChannelKey()
		if err != nil {
			return err
		}
		ch.Key = key
	}
	return nil
}

func (d *db) validateIndexExists(idxKey ChannelKey) error {
	found, err := d.channels.ChannelsExist(idxKey)
	if err != nil || found {
		return err
	}
	return errors.Wrapf(
		validate.ValidationError,
		"[cesium] - provided index %s does not exist",
		idxKey,
	)
}

func (d *db) validateChannelExists(chKey ChannelKey) error {
	found, err := d.channels.ChannelsExist(chKey)
	if err != nil || found {
		return err
	}
	return errors.Wrapf(
		validate.ValidationError,
		"[cesium] - provided channel %s does not exist",
		chKey,
	)
}

func (d *db) validateNewIndexChannel(ch *Channel) error {
	if ch.Density != telem.TimeStampDensity {
		return errors.Wrapf(
			validate.ValidationError,
			"[cesium] - index channel must use int64 timestamps",
			telem.TimeStampDensity,
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
