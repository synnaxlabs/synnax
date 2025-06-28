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
	"github.com/synnaxlabs/x/zyn"
)

const (
	TelemSinkType   = "telem.sink"
	TelemSourceType = "telem.source"
)

type TelemConfig struct {
	Channel channel.Key
}

func (t *TelemConfig) Parse(data any) error {
	return telemConfigZ.Parse(data, t)
}

var telemConfigZ = zyn.Object(map[string]zyn.Schema{
	"channel": zyn.Uint32().Coerce(),
})

func telemSource(ctx context.Context, cfg Config, n Node) (NodeSchema, bool, error) {
	var ns NodeSchema
	if n.Type != TelemSourceType {
		return ns, false, nil
	}
	tCfg := &TelemConfig{}
	if err := tCfg.Parse(cfg); err != nil {
		return ns, true, err
	}
	var ch channel.Channel
	if err := cfg.Channel.NewRetrieve().
		WhereKeys(tCfg.Channel).
		Entry(&ch).
		Exec(ctx, nil); err != nil {
		return ns, true, err
	}
	ns.Outputs = []Output{{Key: "Value", DataType: zyn.DataType(ch.DataType)}}
	ns.Type = TelemSourceType
	return ns, true, nil
}

func telemSink(ctx context.Context, cfg Config, n Node) (NodeSchema, bool, error) {
	var ns NodeSchema
	if n.Type != TelemSinkType {
		return ns, false, nil
	}
	tCfg := &TelemConfig{}
	if err := tCfg.Parse(cfg); err != nil {
		return ns, true, err
	}
	var ch channel.Channel
	if err := cfg.Channel.NewRetrieve().WhereKeys(tCfg.Channel).Entry(&ch).Exec(ctx, nil); err != nil {
		return ns, true, err
	}
	ns.Inputs = []Input{{Key: "Value", AcceptsDataType: zyn.Literal(ch.DataType)}}
	ns.Type = TelemSinkType
	return ns, true, nil
}
