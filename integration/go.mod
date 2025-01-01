module github.com/synnaxlabs/integration

go 1.23.4

require golang.org/x/sync v0.10.0

require github.com/synnaxlabs/x v0.0.0-20220801122519-e4a5e96a532d

require github.com/synnaxlabs/synnax v0.0.0-20250101015019-dde1732a3d6e

require (
	github.com/DataDog/zstd v1.5.6 // indirect
	github.com/RoaringBitmap/roaring v1.9.4 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.20.0 // indirect
	github.com/blevesearch/bleve/v2 v2.4.4 // indirect
	github.com/blevesearch/bleve_index_api v1.2.0 // indirect
	github.com/blevesearch/geo v0.1.20 // indirect
	github.com/blevesearch/go-faiss v1.0.24 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.0.4 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.3.0 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.1.0 // indirect
	github.com/blevesearch/zapx/v11 v11.3.10 // indirect
	github.com/blevesearch/zapx/v12 v12.3.10 // indirect
	github.com/blevesearch/zapx/v13 v13.3.10 // indirect
	github.com/blevesearch/zapx/v14 v14.3.10 // indirect
	github.com/blevesearch/zapx/v15 v15.3.17 // indirect
	github.com/blevesearch/zapx/v16 v16.1.10 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/cmux v0.0.0-20170110192607-30d10be49292 // indirect
	github.com/cockroachdb/errors v1.11.3 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/pebble v1.1.2 // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20230807174530-cc333fc44b06 // indirect
	github.com/getsentry/sentry-go v0.30.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/geo v0.0.0-20230421003525-6adc56603217 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.61.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/samber/lo v1.47.0 // indirect
	github.com/synnaxlabs/alamos v0.0.0-00010101000000-000000000000 // indirect
	github.com/synnaxlabs/aspen v0.0.0-20220804103056-48505d5ea44e // indirect
	github.com/synnaxlabs/cesium v0.0.0-20220722114246-333fea6b09d0 // indirect
	github.com/synnaxlabs/freighter v0.0.0-20220810182625-b66219353383 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.etcd.io/bbolt v1.3.11 // indirect
	go.opentelemetry.io/otel v1.33.0 // indirect
	go.opentelemetry.io/otel/trace v1.33.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241230172942-26aa7a208def // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241230172942-26aa7a208def // indirect
	google.golang.org/grpc v1.69.2 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
)

replace github.com/synnaxlabs/x => ../x/go

replace github.com/synnaxlabs/synnax => ../synnax

replace github.com/synnaxlabs/freighter => ../freighter/go

replace github.com/synnaxlabs/aspen => ../aspen

replace github.com/synnaxlabs/alamos => ../alamos/go

replace github.com/synnaxlabs/cesium => ../cesium
