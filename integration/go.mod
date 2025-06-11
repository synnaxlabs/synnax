module github.com/synnaxlabs/integration

go 1.24.4

require golang.org/x/sync v0.15.0

require github.com/synnaxlabs/x v0.0.0-20220801122519-e4a5e96a532d

require github.com/synnaxlabs/synnax v0.0.0-20250101015019-dde1732a3d6e

require (
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/RoaringBitmap/roaring/v2 v2.4.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/blevesearch/bleve/v2 v2.5.0 // indirect
	github.com/blevesearch/bleve_index_api v1.2.7 // indirect
	github.com/blevesearch/geo v0.1.20 // indirect
	github.com/blevesearch/go-faiss v1.0.25 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.0.4 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.3.9 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.1.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.1 // indirect
	github.com/blevesearch/zapx/v12 v12.4.1 // indirect
	github.com/blevesearch/zapx/v13 v13.4.1 // indirect
	github.com/blevesearch/zapx/v14 v14.4.1 // indirect
	github.com/blevesearch/zapx/v15 v15.4.1 // indirect
	github.com/blevesearch/zapx/v16 v16.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/cmux v0.0.0-20250514152509-914d3bf9ec58 // indirect
	github.com/cockroachdb/crlib v0.0.0-20250521014800-1789bc709bcb // indirect
	github.com/cockroachdb/errors v1.12.0 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/pebble v1.1.5 // indirect
	github.com/cockroachdb/pebble/v2 v2.0.5 // indirect
	github.com/cockroachdb/redact v1.1.6 // indirect
	github.com/cockroachdb/swiss v0.0.0-20250327203710-2932b022f6df // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/getsentry/sentry-go v0.33.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/geo v0.0.0-20230421003525-6adc56603217 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/gomega v1.37.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.64.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/samber/lo v1.50.0 // indirect
	github.com/synnaxlabs/alamos v0.0.0-00010101000000-000000000000 // indirect
	github.com/synnaxlabs/aspen v0.0.0-20220804103056-48505d5ea44e // indirect
	github.com/synnaxlabs/cesium v0.0.0-20220722114246-333fea6b09d0 // indirect
	github.com/synnaxlabs/freighter v0.0.0-20220810182625-b66219353383 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20250606033433-dcc06ee1d476 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/synnaxlabs/x => ../x/go

replace github.com/synnaxlabs/synnax => ../synnax

replace github.com/synnaxlabs/freighter => ../freighter/go

replace github.com/synnaxlabs/aspen => ../aspen

replace github.com/synnaxlabs/alamos => ../alamos/go

replace github.com/synnaxlabs/cesium => ../cesium
