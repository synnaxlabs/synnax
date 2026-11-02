// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// virtualKeys returns the keys of the virtual channels among keys.
func virtualKeys(
	ctx context.Context,
	db *ts.DB,
	keys []ts.ChannelKey,
) (set.Set[ts.ChannelKey], error) {
	channels, err := db.RetrieveChannels(ctx, keys...)
	if err != nil {
		return nil, err
	}
	virtual := make(set.Set[ts.ChannelKey])
	for _, ch := range channels {
		if ch.Virtual {
			virtual.Add(ch.Key)
		}
	}
	return virtual, nil
}

// stampVirtual stamps each series of a virtual channel with one instant from the host's
// clock, since no index places its samples.
func stampVirtual(fr ts.Frame, virtual set.Set[ts.ChannelKey]) ts.Frame {
	now := telem.Now()
	for rawI, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) || !virtual.Contains(k) {
			continue
		}
		s := fr.RawSeriesAt(rawI)
		s.TimeRange = now.SpanRange(telem.Nanosecond)
		fr.SetRawSeriesAt(rawI, s)
	}
	return fr
}

func newRequestTranslator(
	virtual set.Set[ts.ChannelKey],
) func(ctx context.Context, in Request) (ts.WriterRequest, bool, error) {
	return func(ctx context.Context, in Request) (ts.WriterRequest, bool, error) {
		fr := in.Frame.ToStorage()
		if in.Command == CommandWrite && len(virtual) > 0 {
			fr = stampVirtual(fr, virtual)
		}
		return ts.WriterRequest{
			Command: ts.WriterCommand(in.Command),
			Frame:   fr,
			Config:  in.Config.toStorage(),
			SeqNum:  in.SeqNum,
		}, true, nil
	}
}

func newResponseTranslator(
	host node.Key,
) func(ctx context.Context, in ts.WriterResponse) (Response, bool, error) {
	return func(ctx context.Context, in ts.WriterResponse) (Response, bool, error) {
		return Response{
			Command:    Command(in.Command),
			SeqNum:     in.SeqNum,
			NodeKey:    host,
			End:        in.End,
			Authorized: in.Authorized,
			Err:        in.Err,
		}, true, nil
	}
}
