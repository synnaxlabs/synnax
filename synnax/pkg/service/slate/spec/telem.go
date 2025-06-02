// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package spec

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

const (
	TelemSinkType   = "telem.sink"
	TelemSourceType = "telem.source"
)

func telemSource(ctx context.Context, cfg Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != TelemSourceType {
		return ns, false, err
	}
	chKey, ok := schema.Get[float64](schema.Resource{Data: n.Data}, "channel")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{
			Field:   "channel",
			Message: "invalid channel",
		})
	}
	var ch channel.Channel
	if err = cfg.Channel.NewRetrieve().
		WhereKeys(channel.Key(chKey)).
		Entry(&ch).
		Exec(ctx, nil); err != nil {
		return ns, ok, err
	}
	ns.Outputs = []Output{
		{
			Key:      "value",
			DataType: string(ch.DataType),
		},
	}
	ns.Type = TelemSourceType
	return ns, true, nil
}

func telemSink(ctx context.Context, cfg Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != TelemSinkType {
		return ns, false, err
	}
	chKey, ok := schema.Get[float64](schema.Resource{Data: n.Data}, "channel")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{})
	}
	var ch channel.Channel
	if err = cfg.Channel.NewRetrieve().WhereKeys(channel.Key(chKey)).Entry(&ch).Exec(ctx, nil); err != nil {
		return ns, ok, err
	}
	ns.Inputs = []Input{
		{
			Key:             "value",
			AcceptsDataType: strictlyMatchDataType(string(ch.DataType)),
		},
	}
	ns.Type = TelemSinkType
	return ns, true, nil
}
