// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// TableConfig configures the set and delete channels of a Gorp table. A set sample is
// the JSON of the entry. A delete sample is the key: 16 bytes for a UUID key, the raw
// bytes for a numeric key, and a variable sample for a string key.
type TableConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Changes decodes the changes that a commit made to the table, such as
	// gorp.Table.Changes.
	//
	// [REQUIRED]
	Changes func(gorp.Commit) gorp.TxReader[K, E]
	// Resource returns the resource of the entry with the given key. A streamer sends a
	// sample of the set or delete channel only to a subject that can retrieve it.
	//
	// [REQUIRED]
	Resource func(K) resource.ID
	// SetName is the name of the set channel.
	//
	// [OPTIONAL] - Defaults to sy_<type>_set.
	SetName string
	// DeleteName is the name of the delete channel.
	//
	// [OPTIONAL] - Defaults to sy_<type>_delete.
	DeleteName string
	// MarshalSet encodes an entry as one sample of the set channel.
	//
	// [OPTIONAL] - Defaults to MarshalJSON.
	MarshalSet func(E) ([]byte, error)
	// SetKey returns the key of the entry that a sample of the set channel holds.
	//
	// [OPTIONAL] - Defaults to the key field of the JSON sample.
	SetKey func(sample []byte) (K, error)
	// SetCheck returns the check of the set of an entry. It runs once for each set. A
	// streamer sends the set only to a subject that passes it, in place of a check of
	// Resource.
	//
	// [OPTIONAL] - Defaults to a check of Resource.
	SetCheck func(E) (check, error)
	// DeleteChecks returns the lookup of the check of each entry that c deleted, by
	// key.
	// It runs once for each commit with deletes. A streamer sends a delete only to a
	// subject that passes its check, in place of a check of Resource.
	//
	// [OPTIONAL] - Defaults to a check of Resource.
	DeleteChecks func(c gorp.Commit) func(key K) (check, bool)
	// DeleteSets returns the lookup of the sample that the set channel carries at the
	// delete in c of each entry, with the check of the subjects that receive it, by
	// key.
	// It returns false for a delete that needs no such sample. It runs once for each
	// commit with deletes, and needs SetCheck.
	//
	// [OPTIONAL] - Defaults to no sample.
	DeleteSets func(c gorp.Commit) func(key K) (sample []byte, sampleCheck check, ok bool)
	// SetDisabled is true when an action channel describes the sets of the table, so
	// the table has no set channel.
	//
	// [OPTIONAL] - Defaults to false.
	SetDisabled *bool
}

var _ config.Config[TableConfig[string, gorp.Entry[string]]] = TableConfig[
	string,
	gorp.Entry[string],
]{}

// DefaultTableConfig returns the default channel names and set encoding for a table of
// E.
func DefaultTableConfig[K gorp.Key, E gorp.Entry[K]]() TableConfig[K, E] {
	t := strings.ToLower(types.Name[E]())
	return TableConfig[K, E]{
		SetName:     fmt.Sprintf("sy_%s_set", t),
		DeleteName:  fmt.Sprintf("sy_%s_delete", t),
		MarshalSet:  MarshalJSON[K, E],
		SetKey:      unmarshalKey[K],
		SetDisabled: new(false),
	}
}

// Override implements config.Config.
func (c TableConfig[K, E]) Override(other TableConfig[K, E]) TableConfig[K, E] {
	c.Changes = override.Nil(c.Changes, other.Changes)
	c.Resource = override.Nil(c.Resource, other.Resource)
	c.SetName = override.String(c.SetName, other.SetName)
	c.DeleteName = override.String(c.DeleteName, other.DeleteName)
	c.MarshalSet = override.Nil(c.MarshalSet, other.MarshalSet)
	c.SetKey = override.Nil(c.SetKey, other.SetKey)
	c.SetCheck = override.Nil(c.SetCheck, other.SetCheck)
	c.DeleteChecks = override.Nil(c.DeleteChecks, other.DeleteChecks)
	c.DeleteSets = override.Nil(c.DeleteSets, other.DeleteSets)
	c.SetDisabled = override.Nil(c.SetDisabled, other.SetDisabled)
	return c
}

// Validate implements config.Config.
func (c TableConfig[K, E]) Validate() error {
	v := validate.New("signals.table_config")
	v.NotNil("changes", c.Changes)
	v.NotNil("resource", c.Resource)
	v.NotEmptyString("delete_name", c.DeleteName)
	v.NotNil("set_disabled", c.SetDisabled)
	if v.Error() != nil {
		return v.Error()
	}
	if !*c.SetDisabled {
		v.NotEmptyString("set_name", c.SetName)
		v.NotNil("marshal_set", c.MarshalSet)
		v.NotNil("set_key", c.SetKey)
	}
	// The sample that DeleteSets adds is checked as a set.
	if c.DeleteSets != nil {
		v.NotNil("set_check", c.SetCheck)
	}
	return v.Error()
}

// PublishTable creates the set and delete channels of a table and adds them to the
// channels that Start writes. Call it before Start.
func (p *Provider) PublishTable[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	cfgs ...TableConfig[K, E],
) error {
	cfg, err := config.New(DefaultTableConfig[K, E](), cfgs...)
	if err != nil {
		return err
	}
	f := &tableFeed[K, E]{
		cfg:          cfg,
		L:            p.cfg.L,
		del:          channel.Channel{Name: cfg.DeleteName, DataType: keyDataType[K]()},
		setChecks:    newCommitChecks(p.checkCommits),
		deleteChecks: newCommitChecks(p.checkCommits),
	}
	channels := []*channel.Channel{&f.del}
	if !*cfg.SetDisabled {
		f.set = &channel.Channel{Name: cfg.SetName, DataType: telem.JSONT}
		channels = append(channels, f.set)
	}
	return p.add(ctx, f, channels...)
}

// tableFeed writes the set and delete samples of one table.
type tableFeed[K gorp.Key, E gorp.Entry[K]] struct {
	cfg TableConfig[K, E]
	L   *alamos.Logger
	// set is nil when the table has no set channel.
	set *channel.Channel
	del channel.Channel
	// setChecks and deleteChecks hold the check of each set and delete, by the bytes of
	// its sample.
	setChecks    *commitChecks
	deleteChecks *commitChecks
}

// commitSamples holds the samples and checks that one commit adds to the channels of a
// table feed.
type commitSamples[K gorp.Key] struct {
	sets, deletes           telem.Series
	setChecks, deleteChecks map[string]check
	deleteCheck             func(K) (check, bool)
	deleteSet               func(K) ([]byte, check, bool)
}

func (s *commitSamples[K]) addSetCheck(sample []byte, c check) {
	if s.setChecks == nil {
		s.setChecks = make(map[string]check)
	}
	s.setChecks[string(sample)] = c
}

func (f *tableFeed[K, E]) append(c gorp.Commit, fr framer.Frame) framer.Frame {
	// The loop body of an iterator is a closure, so each local it writes would move to
	// the heap on every commit, for every feed. One struct moves instead.
	s := commitSamples[K]{
		sets:    telem.Series{DataType: telem.JSONT},
		deletes: telem.Series{DataType: f.del.DataType},
	}
	for ch := range f.cfg.Changes(c) {
		f.add(c, &s, ch)
	}
	if s.setChecks != nil {
		f.setChecks.add(s.setChecks)
	}
	if s.deleteChecks != nil {
		f.deleteChecks.add(s.deleteChecks)
	}
	if len(s.sets.Data) > 0 {
		fr = fr.Append(f.set.Key(), s.sets)
	}
	if len(s.deletes.Data) > 0 {
		fr = fr.Append(f.del.Key(), s.deletes)
	}
	return fr
}

// add adds the samples and checks of ch, a change in c, to s.
func (f *tableFeed[K, E]) add(
	c gorp.Commit,
	s *commitSamples[K],
	ch gorp.EntryChange[K, E],
) {
	if ch.Kind == gorp.ChangeDelete {
		f.addDelete(c, s, ch.Key)
		return
	}
	if f.set == nil {
		return
	}
	b, err := f.cfg.MarshalSet(ch.Value)
	if err != nil {
		f.L.Error(
			"failed to marshal a set",
			zap.String("channel", f.set.Name),
			zap.Error(err),
		)
		return
	}
	if f.cfg.SetCheck != nil {
		sample, n := telem.UnmarshalVariableSample(b)
		if n != len(b) {
			f.L.DPanic("a set is not one sample", zap.String("channel", f.set.Name))
			return
		}
		check, err := f.cfg.SetCheck(ch.Value)
		if err != nil {
			f.L.Error(
				"failed to check a set",
				zap.String("channel", f.set.Name),
				zap.Error(err),
			)
			return
		}
		s.addSetCheck(sample, check)
	}
	s.sets.Data = append(s.sets.Data, b...)
}

// addDelete adds the samples and checks of the delete of key in c to s.
func (f *tableFeed[K, E]) addDelete(c gorp.Commit, s *commitSamples[K], key K) {
	sample := keySample(key)
	s.deletes.Data = append(s.deletes.Data, sample...)
	if f.set != nil && f.cfg.DeleteSets != nil {
		if s.deleteSet == nil {
			s.deleteSet = f.cfg.DeleteSets(c)
		}
		if set, check, ok := s.deleteSet(key); ok {
			s.sets.Data = append(s.sets.Data, telem.MarshalVariableSample(set)...)
			s.addSetCheck(set, check)
		}
	}
	if f.cfg.DeleteChecks == nil {
		return
	}
	if s.deleteCheck == nil {
		s.deleteCheck = f.cfg.DeleteChecks(c)
		s.deleteChecks = make(map[string]check)
	}
	check, ok := s.deleteCheck(key)
	if !ok {
		f.L.DPanic("a delete has no check", zap.String("channel", f.del.Name))
		return
	}
	if f.del.DataType.IsVariable() {
		unwrapped, n := telem.UnmarshalVariableSample(sample)
		if n != len(sample) {
			f.L.DPanic("a delete is not one sample", zap.String("channel", f.del.Name))
			return
		}
		sample = unwrapped
	}
	s.deleteChecks[string(sample)] = check
}

func (f *tableFeed[K, E]) records() map[channel.Key]Record {
	del := Record{Of: recordOf(unmarshalKeySample[K], f.cfg.Resource), Deleted: true}
	if f.cfg.DeleteChecks != nil {
		del.Check = func(sample []byte) (check, error) {
			return f.deleteChecks.get(sample)
		}
	}
	records := map[channel.Key]Record{f.del.Key(): del}
	if f.set != nil {
		r := Record{Of: recordOf(f.cfg.SetKey, f.cfg.Resource)}
		if f.cfg.SetCheck != nil {
			r.Check = func(sample []byte) (check, error) {
				return f.setChecks.get(sample)
			}
		}
		records[f.set.Key()] = r
	}
	return records
}

// MarshalJSON encodes e as one variable JSON sample.
func MarshalJSON[K gorp.Key, E gorp.Entry[K]](e E) ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return telem.MarshalVariableSample(b), nil
}

// keyDataType returns the data type of the delete channel of a table keyed by K.
func keyDataType[K gorp.Key]() telem.DataType {
	t := reflect.TypeFor[K]()
	switch t.Kind() {
	case reflect.String:
		return telem.StringT
	case reflect.Array:
		return telem.UUIDT
	case reflect.Uint8:
		return telem.Uint8T
	case reflect.Uint16:
		return telem.Uint16T
	case reflect.Uint32:
		return telem.Uint32T
	case reflect.Uint64:
		return telem.Uint64T
	case reflect.Int8:
		return telem.Int8T
	case reflect.Int16:
		return telem.Int16T
	case reflect.Int32:
		return telem.Int32T
	case reflect.Int64:
		return telem.Int64T
	}
	panic(fmt.Sprintf("[signals] - no delete channel data type for key type %s", t))
}

// keySample returns k as one sample of the delete channel of its table.
func keySample[K gorp.Key](k K) []byte {
	if v := reflect.ValueOf(k); v.Kind() == reflect.String {
		return telem.MarshalVariableSample([]byte(v.String()))
	}
	b := make([]byte, unsafe.Sizeof(k))
	*(*K)(unsafe.Pointer(&b[0])) = k
	return b
}
