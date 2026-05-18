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
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	xunsafe "github.com/synnaxlabs/x/unsafe"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// GorpPublisherConfig is the configuration for opening a Signals pipeline that subscribes
// changes to a particular entry type in a gorp.DB. It's not typically necessary
// to instantiate this configuration directly, instead use a helper function
// such as GorpPublisherConfigUUID.
//
// SetName and DeleteName default to "sy_<type>_set" / "sy_<type>_delete" when the
// caller does not override them. To opt out of a channel entirely (e.g. when the
// caller is publishing the set events through a separate, custom pipeline) set
// DisableSet or DisableDelete; the corresponding events are then dropped and the
// channel is not created. At least one of the two channels must remain enabled.
type GorpPublisherConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Observable is the observable to subscribe to for entry changes.
	Observable observe.Observable[gorp.TxReader[K, E]]
	// SetDataType is the data type of the set channel.
	SetDataType telem.DataType
	// DeleteDataType is the data type of the delete channel.
	DeleteDataType telem.DataType
	// MarshalSet is a function that marshals an entry into the set channel's payload.
	MarshalSet func(entry E) ([]byte, error)
	// MarshalDelete is a function that marshals a deleted entry's key into the delete
	// channel's payload.
	MarshalDelete func(K) ([]byte, error)
	// SetName is the name of the set channel.
	SetName string
	// DeleteName is the name of the delete channel.
	DeleteName string
	// DisableSet drops VariantSet events and skips creating the set channel. Use
	// when the caller propagates set events through a separate pipeline.
	DisableSet bool
	// DisableDelete drops VariantDelete events and skips creating the delete channel.
	DisableDelete bool
}

var _ config.Config[GorpPublisherConfig[uuid.UUID, gorp.Entry[uuid.UUID]]] = GorpPublisherConfig[uuid.UUID, gorp.Entry[uuid.UUID]]{}

func DefaultGorpPublisherConfig[K gorp.Key, E gorp.Entry[K]]() GorpPublisherConfig[K, E] {
	t := types.Name[E]()
	return GorpPublisherConfig[K, E]{
		SetName:    fmt.Sprintf("sy_%s_set", strings.ToLower(t)),
		DeleteName: fmt.Sprintf("sy_%s_delete", strings.ToLower(t)),
	}
}

func (g GorpPublisherConfig[K, E]) Override(other GorpPublisherConfig[K, E]) GorpPublisherConfig[K, E] {
	g.Observable = override.Nil(g.Observable, other.Observable)
	g.SetDataType = override.String(g.SetDataType, other.SetDataType)
	g.DeleteDataType = override.String(g.DeleteDataType, other.DeleteDataType)
	g.MarshalSet = override.Nil(g.MarshalSet, other.MarshalSet)
	g.MarshalDelete = override.Nil(g.MarshalDelete, other.MarshalDelete)
	g.SetName = override.String(g.SetName, other.SetName)
	g.DeleteName = override.String(g.DeleteName, other.DeleteName)
	g.DisableSet = g.DisableSet || other.DisableSet
	g.DisableDelete = g.DisableDelete || other.DisableDelete
	g.Observable = override.Nil(g.Observable, other.Observable)
	return g
}

func (g GorpPublisherConfig[K, E]) Validate() error {
	v := validate.New("cdc.gorp_publisher_config")
	setEnabled := !g.DisableSet && g.SetName != ""
	deleteEnabled := !g.DisableDelete && g.DeleteName != ""
	v.Ternary(
		"channels",
		!setEnabled && !deleteEnabled,
		"at least one of the set or delete channel must be enabled",
	)
	if setEnabled {
		validate.NotEmptyString(v, "set_data_type", g.SetDataType)
		validate.NotNil(v, "marshal_set", g.MarshalSet)
	}
	if deleteEnabled {
		validate.NotEmptyString(v, "delete_data_type", g.DeleteDataType)
		validate.NotNil(v, "marshal_delete", g.MarshalDelete)
	}
	validate.NotNil(v, "observable", g.Observable)
	return v.Error()
}

func MarshalJSON[K gorp.Key, E gorp.Entry[K]](e E) ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return telem.MarshalVariableSample(b), nil
}

// GorpPublisherConfigUUID is a helper function for creating a Signals pipeline that propagates
// changes to UUID keyed gorp entries written to the provided DB. The returned
// configuration should be passed to PublishFromGorp.
func GorpPublisherConfigUUID[E gorp.Entry[uuid.UUID]](obs observe.Observable[gorp.TxReader[uuid.UUID, E]]) GorpPublisherConfig[uuid.UUID, E] {
	return GorpPublisherConfig[uuid.UUID, E]{
		Observable:     obs,
		DeleteDataType: telem.UUIDT,
		SetDataType:    telem.JSONT,
		MarshalDelete:  func(k uuid.UUID) ([]byte, error) { return k[:], nil },
		MarshalSet:     MarshalJSON[uuid.UUID, E],
	}
}

func GorpPublisherConfigPureNumeric[K types.SizedNumeric, E gorp.Entry[K]](obs observe.Observable[gorp.TxReader[K, E]], dt telem.DataType) GorpPublisherConfig[K, E] {
	return GorpPublisherConfig[K, E]{
		Observable:     obs,
		DeleteDataType: dt,
		SetDataType:    dt,
		MarshalDelete: func(k K) (b []byte, err error) {
			return xunsafe.CastToBytes(k), nil
		},
		MarshalSet: func(e E) (b []byte, err error) {
			return xunsafe.CastToBytes(e.GorpKey()), nil
		},
	}
}

func GorpPublisherConfigNumeric[K types.SizedNumeric, E gorp.Entry[K]](obs observe.Observable[gorp.TxReader[K, E]], dt telem.DataType) GorpPublisherConfig[K, E] {
	return GorpPublisherConfig[K, E]{
		Observable:     obs,
		DeleteDataType: dt,
		SetDataType:    telem.JSONT,
		MarshalDelete: func(k K) (b []byte, err error) {
			return xunsafe.CastToBytes(k), nil
		},
		MarshalSet: MarshalJSON[K, E],
	}
}

func GorpPublisherConfigString[E gorp.Entry[string]](obs observe.Observable[gorp.TxReader[string, E]]) GorpPublisherConfig[string, E] {
	return GorpPublisherConfig[string, E]{
		Observable:     obs,
		DeleteDataType: telem.StringT,
		SetDataType:    telem.JSONT,
		MarshalDelete:  func(k string) ([]byte, error) { return telem.MarshalVariableSample([]byte(k)), nil },
		MarshalSet:     MarshalJSON[string, E],
	}
}

// PublishFromGorp opens a Signals pipeline that subscribes to the sets and deletes of a
// particular entry type in the configured gorp.DB. The returned io.Closer should be
// closed to stop the Signals pipeline.
func PublishFromGorp[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	svc *Provider,
	cfgs ...GorpPublisherConfig[K, E],
) (io.Closer, error) {
	cfg, err := config.New(DefaultGorpPublisherConfig[K, E](), cfgs...)
	if err != nil {
		return nil, err
	}
	setEnabled := !cfg.DisableSet && cfg.SetName != ""
	deleteEnabled := !cfg.DisableDelete && cfg.DeleteName != ""
	obs := observe.Translator[gorp.TxReader[K, E], []change.Change[[]byte, struct{}]]{
		Observable: cfg.Observable,
		Translate: func(ctx context.Context, r gorp.TxReader[K, E]) ([]change.Change[[]byte, struct{}], bool) {
			var out []change.Change[[]byte, struct{}]
			for c := range r {
				oc := change.Change[[]byte, struct{}]{Variant: c.Variant}
				if c.Variant == change.VariantSet {
					if !setEnabled {
						continue
					}
					v, err := cfg.MarshalSet(c.Value)
					if err != nil {
						svc.L.Error("failed to marshal set", zap.Error(err), zap.String("channel", cfg.SetName))
					}
					oc.Key = v
				} else {
					if !deleteEnabled {
						continue
					}
					k, err := cfg.MarshalDelete(c.Key)
					if err != nil {
						svc.L.Error("failed to marshal delete", zap.Error(err), zap.String("channel", cfg.DeleteName))
					}
					oc.Key = k
				}
				out = append(out, oc)
			}
			return out, len(out) > 0
		},
	}
	obsCfg := ObservablePublisherConfig{
		Name:       fmt.Sprintf("gorp_%s", strings.ToLower(types.Name[E]())),
		Observable: obs,
	}
	if setEnabled {
		obsCfg.SetChannel = channel.Channel{Name: cfg.SetName, DataType: cfg.SetDataType, Internal: true}
	}
	if deleteEnabled {
		obsCfg.DeleteChannel = channel.Channel{Name: cfg.DeleteName, DataType: cfg.DeleteDataType, Internal: true}
	}
	return svc.PublishFromObservable(ctx, obsCfg)
}
