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
	"maps"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/testdata"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// telemetry returns the contents of every file under dir in fs other than channel
// metadata.
func telemetry(fs xfs.FS, dir string) map[string][]byte {
	GinkgoHelper()
	files := make(map[string][]byte)
	for _, info := range MustSucceed(fs.List(dir)) {
		name := path.Join(dir, info.Name())
		if info.IsDir() {
			maps.Copy(files, telemetry(fs, name))
		} else if info.Name() != "meta.json" {
			files[name] = MustSucceed(xfs.ReadFile(fs, name))
		}
	}
	return files
}

var _ = Describe("Migration Test", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db *cesium.DB
				fs xfs.FS
			)
			BeforeEach(func() { fs = openFS() })
			Specify("V1 to current", func(ctx SpecContext) {
				By("Making a copy of an unversioned database")
				sourceFS := MustSucceed(xfs.Default.Sub("../testdata/v1/db-data"))
				destFS := fs
				Expect(xfs.CopyDir(sourceFS, destFS, "")).To(Succeed())
				before := telemetry(fs, "")

				By("Opening the V1 database in V2")
				db = MustSucceed(
					cesium.Open(
						ctx,
						"",
						cesium.WithFS(fs),
						cesium.WithInstrumentation(PanicLogger()),
					),
				)

				By("Asserting that the version got migrated and the format is correct")
				for _, ch := range testdata.Channels {
					chInDB, err := db.RetrieveChannel(ctx, ch.Key)
					if ch.Key == testdata.LegacyRateKey {
						Expect(err).To(MatchError(query.ErrNotFound))
						continue
					} else {
						Expect(err).ToNot(HaveOccurred())
					}
					Expect(chInDB.Version).To(Equal(channel.VersionCurrent))
					Expect(chInDB.IsIndex).To(Equal(ch.IsIndex))
				}

				Expect(db.Close()).To(Succeed())

				By("Asserting that the migration left the telemetry untouched")
				after := telemetry(fs, "")
				for name, data := range before {
					Expect(after).To(HaveKeyWithValue(name, data), name)
				}
			})
		})
	}
})
