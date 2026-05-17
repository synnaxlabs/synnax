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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

var (
	gatewayRequestsAddr    address.Address = "requests"
	gatewayResponsesAddr   address.Address = "responses"
	gatewayTSWriterAddr    address.Address = "ts_writer"
	gatewayAutoIndexerAddr address.Address = "auto_indexer"
)

// newGateway opens a new StreamWriter that writes to the store on the gateway node.
// channelMap must contain channel metadata for at least every key in cfg.Keys; entries
// for keys owned by other leaseholders are ignored. When cfg.AutoIndexing is enabled,
// an autoIndexer is inserted before the storage writer so timestamps and authority
// propagation for indexes owned by the gateway are produced against the gateway's
// local state (the gateway as a leaseholder for those indexes).
func (s *Service) newGateway(
	ctx context.Context,
	cfg Config,
	channelMap map[channel.Key]channel.Channel,
) (StreamWriter, error) {
	w, err := s.cfg.TS.NewStreamWriter(ctx, cfg.toStorage())
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	plumber.SetSegment(pipe, gatewayTSWriterAddr, w)
	reqT := &confluence.LinearTransform[Request, ts.WriterRequest]{}
	reqT.Transform = newRequestTranslator()
	resT := &confluence.LinearTransform[ts.WriterResponse, Response]{}
	resT.Transform = newResponseTranslator(s.cfg.HostResolver.HostKey())
	plumber.SetSegment(pipe, gatewayRequestsAddr, reqT)
	plumber.SetSegment(pipe, gatewayResponsesAddr, resT)

	entry := gatewayRequestsAddr
	if cfg.AutoIndexing != nil && *cfg.AutoIndexing {
		channels := make([]channel.Channel, 0, len(cfg.Keys))
		for _, k := range cfg.Keys {
			if ch, ok := channelMap[k]; ok {
				channels = append(channels, ch)
			}
		}
		ai := newAutoIndexer(channels, cfg)
		plumber.SetSegment(pipe, gatewayAutoIndexerAddr, ai)
		plumber.MustConnect[Request](pipe, gatewayAutoIndexerAddr, gatewayRequestsAddr, 1)
		entry = gatewayAutoIndexerAddr
	}
	plumber.MustConnect[ts.WriterRequest](pipe, gatewayRequestsAddr, gatewayTSWriterAddr, 1)
	plumber.MustConnect[ts.WriterResponse](pipe, gatewayTSWriterAddr, gatewayResponsesAddr, 1)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo(entry))
	lo.Must0(seg.RouteOutletFrom(gatewayResponsesAddr))
	return seg, nil
}
