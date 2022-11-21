package ranger_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"strconv"
)

var _ = Describe("Writer", func() {
	It("should write", func() {
		db := MustSucceed(ranger.Open(MustSucceed(fs.OSDirFS("testdata"))))
		v := make([]byte, 0, 1000*8)
		for i := 0; i < 1000; i++ {
			v = append(v, strconv.Itoa(i)...)
		}
		for i := 0; i < 100000; i += 2 {
			w := MustSucceed(db.NewWriter(&ranger.WriterConfig{
				Start: telem.SecondTS * telem.TimeStamp(i),
			}))
			MustSucceed(w.Write(v))
			Expect(w.Commit(telem.TimeStamp(i+1) * telem.SecondTS)).To(Succeed())
			Expect(w.Close()).To(Succeed())
		}
		it := db.NewIterator(&ranger.IteratorConfig{
			Bounds: telem.TimeRange{
				Start: 50 * telem.SecondTS,
				End:   100 * telem.SecondTS,
			},
		})
		i := 0
		for it.SeekFirst(); it.Valid(); it.Next() {
			logrus.Info(it.Range())
			r := MustSucceed(it.NewReader())
			b := make([]byte, 1000*8)
			n, _ := r.Read(b)
			logrus.Info(n)
			i += 2
		}
		Expect(i).To(Equal(1000))
	})
})
