// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/testdata"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration Test", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db        *cesium.DB
				fs        xfs.FS
				cleanUp   func() error
				jsonCodec = binary.JSONCodec{}
			)
			BeforeEach(func() { fs, cleanUp = makeFS() })
			AfterEach(func() { Expect(cleanUp()).To(Succeed()) })
			Specify("V1 to V2", func() {
				By("Making a copy of an unversioned database")
				sourceFS := MustSucceed(xfs.Default.Sub("../testdata/v1/db-data"))
				destFS := fs
				Expect(testutil.CopyFS(sourceFS, destFS)).To(Succeed())

				By("Opening the V1 database in V2")
				db = MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fs), cesium.WithInstrumentation(PanicLogger())))

				By("Asserting that the version got migrated, the meta file got changed, and the format is correct")
				for _, ch := range testdata.Channels {
					chInDB, err := db.RetrieveChannel(ctx, ch.Key)
					if ch.Key == testdata.LegacyRateKey {
						Expect(err).To(HaveOccurredAs(query.ErrNotFound))
						continue
					} else {
						Expect(err).ToNot(HaveOccurred())
					}
					Expect(chInDB.Version).To(Equal(uint8(2)))

					var (
						channelFS = MustSucceed(fs.Sub(strconv.Itoa(int(ch.Key))))
						r         = MustSucceed(channelFS.Open("meta.json", os.O_RDONLY))
						s         = MustSucceed(r.Stat()).Size()
						buf       = make([]byte, s)
						chInMeta  cesium.Channel
					)

					_, err = r.Read(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(r.Close()).To(Succeed())

					err = jsonCodec.Decode(ctx, buf, &chInMeta)
					Expect(err).ToNot(HaveOccurred())
					Expect(chInMeta).To(Equal(chInDB))

				}

				Expect(db.Close()).To(Succeed())
			})
		})
	}
})
