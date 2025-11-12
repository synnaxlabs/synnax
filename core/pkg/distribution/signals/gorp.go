// Copyright 2025 Synnax Labs, Inc.
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
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/binary"
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
type GorpPublisherConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// DB is the DB to subscribe to.
	DB *gorp.DB
	// SetDataType is the data type of the key used by the DB.
	SetDataType telem.DataType
	// DeleteDataType is the data type of the key used by the DB.
	DeleteDataType telem.DataType
	// MarshalSet is a function that marshals the key used by the DB into a byte slice.
	MarshalSet func(entry E) ([]byte, error)
	// MarshalDelete is a function that marshals the key used by the DB into a byte slice.
	MarshalDelete func(K) ([]byte, error)
	// SetName is the name of the set channel.
	SetName string
	// DeleteName is the name of the delete channel.
	DeleteName string
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
	g.DB = override.Nil(g.DB, other.DB)
	g.SetDataType = override.String(g.SetDataType, other.SetDataType)
	g.DeleteDataType = override.String(g.DeleteDataType, other.DeleteDataType)
	g.MarshalSet = override.Nil(g.MarshalSet, other.MarshalSet)
	g.MarshalDelete = override.Nil(g.MarshalDelete, other.MarshalDelete)
	g.SetName = override.String(g.SetName, other.SetName)
	g.DeleteName = override.String(g.DeleteName, other.DeleteName)
	return g
}

func (g GorpPublisherConfig[K, E]) Validate() error {
	v := validate.New("cdc.GorpPublisherConfig")
	validate.NotEmptyString(v, "SetName", g.SetName)
	validate.NotEmptyString(v, "DeleteName", g.DeleteName)
	validate.NotNil(v, "DB", g.DB)
	validate.NotEmptyString(v, "SetDataType", g.SetDataType)
	validate.NotEmptyString(v, "DeleteDataType", g.DeleteDataType)
	validate.NotNil(v, "MarshalSet", g.MarshalSet)
	validate.NotNil(v, "MarshalDelete", g.MarshalDelete)
	return v.Error()
}

var jsonEcd = binary.JSONCodec{}

func MarshalJSON[K gorp.Key, E gorp.Entry[K]](e E) ([]byte, error) {
	b, err := jsonEcd.Encode(context.TODO(), e)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// GorpPublisherConfigUUID is a helper function for creating a Signals pipeline that propagates
// changes to UUID keyed gorp entries written to the provided DB. The returned
// configuration should be passed to PublishFromGorp.
func GorpPublisherConfigUUID[E gorp.Entry[uuid.UUID]](db *gorp.DB) GorpPublisherConfig[uuid.UUID, E] {
	return GorpPublisherConfig[uuid.UUID, E]{
		DB:             db,
		DeleteDataType: telem.UUIDT,
		SetDataType:    telem.JSONT,
		MarshalDelete:  func(k uuid.UUID) ([]byte, error) { return k[:], nil },
		MarshalSet:     MarshalJSON[uuid.UUID, E],
	}
}

func GorpPublisherConfigPureNumeric[K types.Numeric, E gorp.Entry[K]](db *gorp.DB, dt telem.DataType) GorpPublisherConfig[K, E] {
	return GorpPublisherConfig[K, E]{
		DB:             db,
		DeleteDataType: dt,
		SetDataType:    dt,
		MarshalDelete: func(k K) (b []byte, err error) {
			b = make([]byte, dt.Density())
			data := xunsafe.CastSlice[byte, K](b)
			data[0] = k
			return b, nil
		},
		MarshalSet: func(e E) (b []byte, err error) {
			b = make([]byte, dt.Density())
			data := xunsafe.CastSlice[byte, K](b)
			data[0] = e.GorpKey()
			return b, nil
		},
	}
}

func GorpPublisherConfigString[E gorp.Entry[string]](db *gorp.DB) GorpPublisherConfig[string, E] {
	return GorpPublisherConfig[string, E]{
		DB:             db,
		DeleteDataType: telem.StringT,
		SetDataType:    telem.JSONT,
		MarshalDelete:  func(k string) ([]byte, error) { return append([]byte(k), '\n'), nil },
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
	var (
		obs = observe.Translator[gorp.TxReader[K, E], []change.Change[[]byte, struct{}]]{
			Observable: gorp.Observe[K, E](cfg.DB),
			Translate: func(r gorp.TxReader[K, E]) []change.Change[[]byte, struct{}] {
				out := make([]change.Change[[]byte, struct{}], 0, r.Count())
				for c, ok := r.Next(ctx); ok; c, ok = r.Next(ctx) {
					oc := change.Change[[]byte, struct{}]{Variant: c.Variant}
					if c.Variant == change.Set {
						v, err := cfg.MarshalSet(c.Value)
						if err != nil {
							svc.L.Error("failed to marshal set", zap.Error(err), zap.String("channel", cfg.SetName))
						}
						oc.Key = v
					} else {
						k, err := cfg.MarshalDelete(c.Key)
						if err != nil {
							svc.L.Error("failed to marshal delete", zap.Error(err), zap.String("channel", cfg.DeleteName))
						}
						oc.Key = k
					}
					out = append(out, oc)
				}
				return out
			},
		}
		obsCfg = ObservablePublisherConfig{
			Name:          fmt.Sprintf("gorp_%s", strings.ToLower(types.Name[E]())),
			Observable:    obs,
			SetChannel:    channel.Channel{Name: cfg.SetName, DataType: cfg.SetDataType, Internal: true},
			DeleteChannel: channel.Channel{Name: cfg.DeleteName, DataType: cfg.DeleteDataType, Internal: true},
		}
	)
	return svc.PublishFromObservable(ctx, obsCfg)
}
