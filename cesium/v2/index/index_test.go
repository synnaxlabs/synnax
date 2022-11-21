package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/cesium/testutil"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"os"
)

var _ = Describe("rangerIndex", func() {
	It("Should correctly get the distance between two time stamps", func() {
		db := MustSucceed(ranger.Open(MustSucceed(fs.OSDirFS("testdata"))))
		w := MustSucceed(db.NewWriter(&ranger.WriterConfig{
			Start: 10 * telem.SecondTS,
		}))
		for i := 0; i < 1000; i++ {
			MustSucceed(w.Write(testutil.MarshalTimeStamps([]telem.TimeStamp{
				telem.TimeStamp(i) * telem.SecondTS * 10,
			})))
		}
		Expect(w.Commit(1000 * 10 * telem.SecondTS)).To(Succeed())
		Expect(w.Close()).To(Succeed())
		idx := index.rangerIndex{DB: db}
		dist, err := idx.Distance(10*telem.SecondTS, 5000*telem.SecondTS)
		logrus.Info(dist, err)
		os.RemoveAll("testdata")
	})

})
