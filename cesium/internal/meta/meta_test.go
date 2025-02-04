// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package meta_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"os"
	"strconv"
)

var _ = Describe("Meta", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		var (
			fs      xfs.FS
			cleanUp func() error
		)
		BeforeEach(func() {
			fs, cleanUp = makeFS()
		})
		AfterEach(func() { Expect(cleanUp()).To(Succeed()) })
		Context("FS: "+fsName, Ordered, func() {
			Specify("Corrupted meta.json", func() {
				db := MustSucceed(cesium.Open("", cesium.WithFS(fs), cesium.WithInstrumentation(PanicLogger())))
				key := GenerateChannelKey()

				Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Name: "Faraday", Rate: 1 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
				Expect(db.Close()).To(Succeed())

				f, err := fs.Open(strconv.Itoa(int(key))+"/meta.json", os.O_WRONLY)
				Expect(err).ToNot(HaveOccurred())
				_, err = f.Write([]byte("heheheha"))
				Expect(err).ToNot(HaveOccurred())
				Expect(f.Close()).To(Succeed())

				db, err = cesium.Open("", cesium.WithFS(fs), cesium.WithInstrumentation(PanicLogger()))
				Expect(err).To(MatchError(ContainSubstring("error decoding meta in folder for channel %d", key)))
			})

			Describe("Impossible meta configurations", func() {
				var (
					jsonCodec = &binary.JSONCodec{}
					key       = GenerateChannelKey()
				)

				DescribeTable("meta configs", func(badCh cesium.Channel, badField string) {
					db := MustSucceed(cesium.Open("", cesium.WithFS(fs), cesium.WithInstrumentation(PanicLogger())))
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
					Expect(db.Close()).To(Succeed())

					f := MustSucceed(fs.Open(strconv.Itoa(int(key))+"/meta.json", os.O_WRONLY))
					encoded := MustSucceed(jsonCodec.Encode(ctx, badCh))

					_, err := f.WriteAt(encoded, 0)
					Expect(err).ToNot(HaveOccurred())
					Expect(f.Close()).To(Succeed())

					db, err = cesium.Open("", cesium.WithFS(fs), cesium.WithInstrumentation(PanicLogger()))
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring(badField)))
				},
					Entry("datatype not set", cesium.Channel{Key: key, Rate: 1 * telem.Hz}, "dataType"),
					Entry("virtual with rate", cesium.Channel{Key: key, Virtual: true, Rate: 1 * telem.Hz, DataType: telem.Int64T}, "virtual channel cannot have a rate"),
					Entry("virtual indexed", cesium.Channel{Key: key, Virtual: true, Index: key + 100, DataType: telem.Int64T}, "virtual channel cannot be indexed"),
					Entry("index not type timestamp", cesium.Channel{Key: key, IsIndex: true, DataType: telem.Float32T}, "index channel must be of type timestamp"),
				)
			})
		})
	}
})
