// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Stream only type check", func() {
	It("Should refuse a stream only series of a stored index of the wrong data type",
		func(ctx SpecContext) {
			db := mustOpenDBOnFS(ctx, fs.NewMem())
			idx, data := GenerateChannelKey(), GenerateChannelKey()
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: data, Name: "data", Index: idx, DataType: telem.Float32T,
				},
			)).To(Succeed())
			keys := channel.Keys{channel.Key(idx), channel.Key(data)}
			w := MustSucceed(db.svc.Open(context.WithoutCancel(ctx), writer.Config{
				ControlSubject: xcontrol.Subject{Key: "a", Name: "a"},
				Keys:           keys,
				Start:          10 * telem.SecondTS,
				Mode:           writer.ModeStreamOnly,
				Sync:           new(true),
			}))
			w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesV[float32](1, 2, 3),
				telem.NewSeriesV[float32](1, 2, 3),
			}))
			Expect(w.Close()).To(MatchError(validate.ErrValidation))
		},
	)
})
