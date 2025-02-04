// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

// newGateway opens a new StreamWriter that writes to the store on the gateway node.
func (s *Service) newGateway(ctx context.Context, cfg Config) (StreamWriter, error) {
	w, err := s.TS.NewStreamWriter(ctx, cfg.toStorage())
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	plumber.SetSegment[ts.WriterRequest, ts.WriterResponse](pipe, "ts-writer", w)
	reqT := &confluence.LinearTransform[Request, ts.WriterRequest]{}
	reqT.Transform = newRequestTranslator()
	resT := &confluence.LinearTransform[ts.WriterResponse, Response]{}
	resT.Transform = newResponseTranslator(s.HostResolver.HostKey())
	plumber.SetSegment[Request, ts.WriterRequest](pipe, "requests", reqT)
	plumber.SetSegment[ts.WriterResponse, Response](pipe, "responses", resT)
	plumber.MustConnect[ts.WriterRequest](pipe, "requests", "ts-writer", 1)
	plumber.MustConnect[ts.WriterResponse](pipe, "ts-writer", "responses", 1)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo("requests"))
	lo.Must0(seg.RouteOutletFrom("responses"))
	return seg, nil
}
