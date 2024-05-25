// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/testdata"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"os"
	"strconv"
)

var _ = Describe("Migration Test", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db          *cesium.DB
				fs          xfs.FS
				cleanUp     func() error
				jsonEncoder = binary.JSONEncoderDecoder{}
			)
			BeforeEach(func() { fs, cleanUp = makeFS() })
			AfterEach(func() { Expect(cleanUp()).To(Succeed()) })
			Specify("V1 to V2", func() {
				By("Making a copy of a V1 database")
				sourceFS := MustSucceed(xfs.Default.Sub("../testdata/v1/data"))
				destFS := fs
				Expect(testutil.ReplicateFS(sourceFS, destFS)).To(Succeed())

				By("Opening the V1 database in V2")
				db = MustSucceed(cesium.Open("", cesium.WithFS(fs)))

				By("Asserting that the version got migrated, the meta file got changed, and the format is correct")
				for _, ch := range testdata.Channels {
					chInDB := MustSucceed(db.RetrieveChannel(ctx, ch.Key))
					Expect(chInDB.Version).To(Equal(uint8(2)))

					var (
						channelFS = MustSucceed(fs.Sub(strconv.Itoa(int(ch.Key))))
						r         = MustSucceed(channelFS.Open("meta.json", os.O_RDONLY))
						s         = MustSucceed(r.Stat()).Size()
						buf       = make([]byte, s)
						chInMeta  cesium.Channel
					)

					_, err := r.Read(buf)
					Expect(err).ToNot(HaveOccurred())
					Expect(r.Close()).To(Succeed())

					err = jsonEncoder.Decode(ctx, buf, &chInMeta)
					Expect(err).ToNot(HaveOccurred())
					Expect(chInMeta).To(Equal(chInDB))

					if !ch.Virtual {
						Expect(MustSucceed(channelFS.Exists("tombstone.domain"))).To(BeTrue())
						r = MustSucceed(channelFS.Open("index.domain", os.O_RDONLY))
						buf = make([]byte, 4)
						_, err = r.Read(buf)
						if len(buf) != 0 {
							Expect(err).ToNot(HaveOccurred())
						}
						s = MustSucceed(r.Stat()).Size()
						Expect(s).To(Equal(int64(telem.ByteOrder.Uint32(buf)*26 + 4)))

						buf = make([]byte, s-4)
						_, err = r.ReadAt(buf, 4)
						Expect(err).ToNot(HaveOccurred())
						Expect(r.Close()).To(Succeed())

						var (
							oldIndex     = make([]byte, s-4)
							oldIndexFile = MustSucceed(sourceFS.Open(strconv.Itoa(int(ch.Key))+"/index.domain", os.O_RDONLY))
						)
						_, err = oldIndexFile.Read(oldIndex)
						Expect(oldIndexFile.Close()).To(Succeed())

						Expect(buf).To(Equal(oldIndex))
					}
				}

				Expect(db.Close()).To(Succeed())
			})
		})
	}
})
