// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"io"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

type ExportService struct {
	dbProvider
	accessProvider
	framerProvider
	channelProvider
}

func NewExportService(p Provider) *ExportService {
	return &ExportService{
		dbProvider:      p.db,
		accessProvider:  p.access,
		framerProvider:  p.framer,
		channelProvider: p.channel,
	}
}

type ExportCSVRequest struct {
	Keys      channel.Keys    `json:"keys"`
	TimeRange telem.TimeRange `json:"time_range"`
}

type ExportCSVResponse = io.Reader

func (s *ExportService) CSV(ctx context.Context, req ExportCSVRequest) (ExportCSVResponse, error) {
	// TODO: cleanup code
	keys := req.Keys.Unique()
	iter, err := s.framer.Iterator.Open(ctx, framer.IteratorConfig{
		Keys:   keys,
		Bounds: req.TimeRange,
	})
	if err != nil {
		return nil, err
	}
	r, w := io.Pipe()

	go func() {
		defer iter.Close()
		channels := make([]channel.Channel, len(keys))
		if err := s.WithTx(ctx, func(tx gorp.Tx) error {
			return (*s.channel).NewRetrieve().WhereKeys(keys...).Entries(&channels).Exec(ctx, tx)
		}); err != nil {
			w.CloseWithError(err)
			return
		}
		headerRecords := binary.NewCSVRecords(1, len(keys))
		headerRecords[0] = lo.Map(channels, func(c channel.Channel, _ int) string {
			return c.Name
		})
		if err := (&binary.CSVCodec{}).EncodeStream(ctx, w, headerRecords); err != nil {
			w.CloseWithError(err)
			return
		}
		for ok := iter.SeekFirst() && iter.Next(iterator.AutoSpan); ok; ok = iter.Next(iterator.AutoSpan) {
			if err := (&binary.CSVCodec{}).EncodeStream(ctx, w, iter.Value()); err != nil {
				w.CloseWithError(err)
				return
			}
		}
		if err := iter.Error(); err != nil {
			w.CloseWithError(err)
			return
		}
		w.Close()
	}()

	return r, nil
}
