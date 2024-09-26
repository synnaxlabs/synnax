// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/downsampler"
)

type Service struct {
	Internal *framer.Service // distribution layer frame service
}

// got these from api/framer.go calls to Internal
func (s *Service) OpenIterator(ctx context.Context, cfg framer.IteratorConfig) (*framer.Iterator, error) {
	return s.Internal.OpenIterator(ctx, cfg)
}

func (s *Service) NewStreamIterator(ctx context.Context, cfg framer.IteratorConfig) (framer.StreamIterator, error) {
	return s.Internal.NewStreamIterator(ctx, cfg)
}

func (s *Service) NewStreamWriter(ctx context.Context, cfg framer.WriterConfig) (framer.StreamWriter, error) {
	return s.Internal.NewStreamWriter(ctx, cfg)
}

func (s *Service) NewDeleter() framer.Deleter {
	return s.Internal.NewDeleter()
}

func (s *Service) NewStreamer(ctx context.Context, cfg framer.StreamerConfig) (framer.Streamer, error) {
	if cfg.DownsampleFactor >= 1 {
		return downsampler.NewStreamer(ctx, cfg, s.Internal)
	} else {
		return s.Internal.NewStreamer(ctx, cfg)
	}
}

func NewService(framerSvc *framer.Service) (*Service, error) {
	return &Service{
		Internal: framerSvc,
	}, nil
}
