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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
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
	Changes func(context.Context, gorp.Commit) gorp.TxReader[K, E]
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
		SetDisabled: new(false),
	}
}

// Override implements config.Config.
func (c TableConfig[K, E]) Override(other TableConfig[K, E]) TableConfig[K, E] {
	c.Changes = override.Nil(c.Changes, other.Changes)
	c.SetName = override.String(c.SetName, other.SetName)
	c.DeleteName = override.String(c.DeleteName, other.DeleteName)
	c.MarshalSet = override.Nil(c.MarshalSet, other.MarshalSet)
	c.SetDisabled = override.Nil(c.SetDisabled, other.SetDisabled)
	return c
}

// Validate implements config.Config.
func (c TableConfig[K, E]) Validate() error {
	v := validate.New("signals.table_config")
	v.NotNil("changes", c.Changes)
	v.NotEmptyString("delete_name", c.DeleteName)
	v.NotNil("set_disabled", c.SetDisabled)
	if v.Error() != nil {
		return v.Error()
	}
	if !*c.SetDisabled {
		v.NotEmptyString("set_name", c.SetName)
		v.NotNil("marshal_set", c.MarshalSet)
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
		cfg: cfg,
		L:   p.cfg.L,
		del: channel.Channel{Name: cfg.DeleteName, DataType: keyDataType[K]()},
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
}

func (f *tableFeed[K, E]) append(
	ctx context.Context,
	c gorp.Commit,
	fr framer.Frame,
) framer.Frame {
	var (
		sets    = telem.Series{DataType: telem.JSONT}
		deletes = telem.Series{DataType: f.del.DataType}
	)
	for ch := range f.cfg.Changes(ctx, c) {
		if ch.Kind == gorp.ChangeDelete {
			deletes.Data = append(deletes.Data, keySample(ch.Key)...)
			continue
		}
		if f.set == nil {
			continue
		}
		b, err := f.cfg.MarshalSet(ch.Value)
		if err != nil {
			f.L.Error(
				"failed to marshal a set",
				zap.String("channel", f.set.Name),
				zap.Error(err),
			)
			continue
		}
		sets.Data = append(sets.Data, b...)
	}
	if len(sets.Data) > 0 {
		fr = fr.Append(f.set.Key(), sets)
	}
	if len(deletes.Data) > 0 {
		fr = fr.Append(f.del.Key(), deletes)
	}
	return fr
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
