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
	"github.com/synnaxlabs/x/errors"
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
	Keys         channel.Keys           `json:"keys" msgpack:"keys"`
	TimeRange    telem.TimeRange        `json:"time_range" msgpack:"time_range"`
	ChannelNames map[channel.Key]string `json:"channel_names" msgpack:"channel_names"`
}

type ExportCSVResponse = *io.PipeReader

func (es *ExportService) ExportCSV(ctx context.Context, req ExportCSVRequest) (ExportCSVResponse, error) {
	keys := req.Keys.Unique()
	indexKeys := make(channel.Keys, len(keys))
	if err := es.WithTx(ctx, func(tx gorp.Tx) error {
		var channels []channel.Channel
		if err := es.channel.NewRetrieve().WhereKeys(keys...).Entries(&channels).Exec(ctx, tx); err != nil {
			return err
		}
		for i, c := range channels {
			indexKeys[i] = c.Index()
		}
		return nil
	}); err != nil {
		return nil, err
	}
	allKeys := append(keys, indexKeys...).Unique()
	channels := make([]channel.Channel, len(allKeys))
	if err := es.WithTx(ctx, func(tx gorp.Tx) error {
		return es.channel.NewRetrieve().WhereKeys(allKeys...).Entries(&channels).Exec(ctx, tx)
	}); err != nil {
		return nil, err
	}
	headerRecords := make([][]string, 1)
	headerRecords[0] = lo.Map(channels, func(c channel.Channel, _ int) string {
		if name, ok := req.ChannelNames[c.Key()]; ok {
			return name
		}
		return c.Name
	})
	r, w := io.Pipe()
	go func() {
		if err := binary.CSVEncoder.EncodeStream(ctx, w, headerRecords); err != nil {
			w.CloseWithError(err)
			return
		}
		iter, err := es.framer.Iterator.Open(ctx, framer.IteratorConfig{
			Keys:   allKeys,
			Bounds: req.TimeRange,
		})
		if err != nil {
			w.CloseWithError(err)
			return
		}
		for ok := iter.SeekFirst() && iter.Next(iterator.AutoSpan); ok; ok = iter.Next(iterator.AutoSpan) {
			if err = binary.CSVEncoder.EncodeStream(ctx, w, iter.Value()); err != nil {
				w.CloseWithError(errors.Combine(err, iter.Close()))
				return
			}
		}
		// TODO: handle iter.Error()
		w.CloseWithError(iter.Close())
	}()

	return r, nil
}
