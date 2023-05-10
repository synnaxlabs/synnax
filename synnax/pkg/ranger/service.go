// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
}

var _ config.Config[Config] = Config{}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("ranger")
	validate.NotNil(v, "cesium", c.DB)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

type Service struct{ Config }

type Writer struct {
	tx  gorp.Tx
	otg ontology.Writer
}

func (w Writer) Create(
	ctx context.Context,
	name string,
	tr telem.TimeRange,
) (r Range, err error) {
	r.Key = uuid.New()
	r.Name = name
	r.TimeRange = tr
	if err = gorp.NewCreate[uuid.UUID, Range]().Entry(&r).Exec(ctx, w.tx); err != nil {
		return
	}
	if err = w.otg.DefineResource(ctx, OntologyID(r.Key)); err != nil {
		return
	}
	return r, err
}

func (w Writer) Delete(ctx context.Context, key uuid.UUID) error {
	if err := gorp.NewDelete[uuid.UUID, Range]().WhereKeys(key).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResource(ctx, OntologyID(key))
}
