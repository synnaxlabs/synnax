// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/telem"
)

type Frame struct {
	Keys []string `json:"keys" msgpack:"keys"`
	telem.Frame
}

func newFrameFromDistribution(frame framer.Frame) Frame {
	return Frame{
		Keys:  frame.Keys().Strings(),
		Frame: frame.Frame,
	}
}

func toDistributionFrame(f Frame) (framer.Frame, errors.Typed) {
	keys, err := channel.ParseKeys(f.Keys)
	if err != nil {
		return framer.Frame{}, errors.Parse(err)
	}
	return framer.NewFrame(keys, f.Arrays), errors.Nil
}

type FrameService struct {
	alamos.Instrumentation
	authProvider
	Internal *framer.Service
}

func NewSegmentService(p Provider) *FrameService {
	return &FrameService{
		Instrumentation: p.Instrumentation,
		Internal:        p.Config.Framer,
		authProvider:    p.auth,
	}
}
