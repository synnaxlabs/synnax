module github.com/synnaxlabs/synnax

go 1.26.4

replace (
	github.com/synnaxlabs/alamos => ../alamos/go
	github.com/synnaxlabs/arc => ../arc/go
	github.com/synnaxlabs/aspen => ../aspen
	github.com/synnaxlabs/cesium => ../cesium
	github.com/synnaxlabs/freighter => ../freighter/go
	github.com/synnaxlabs/x => ../x/go
)

require (
	github.com/PagerDuty/go-pagerduty v1.8.0
	github.com/blevesearch/bleve/v2 v2.6.0
	github.com/cockroachdb/cmux v0.0.0-20250514152509-914d3bf9ec58
	github.com/cockroachdb/pebble/v2 v2.1.6
	github.com/gofiber/contrib/v3/monitor v1.1.0
	github.com/gofiber/fiber/v3 v3.3.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/onsi/ginkgo/v2 v2.32.0
	github.com/onsi/gomega v1.42.0
	github.com/samber/lo v1.53.0
	github.com/shirou/gopsutil/v4 v4.26.5
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	github.com/synnaxlabs/alamos v0.0.0
	github.com/synnaxlabs/arc v0.0.0
	github.com/synnaxlabs/aspen v0.0.0
	github.com/synnaxlabs/cesium v0.0.0
	github.com/synnaxlabs/freighter v0.0.0
	github.com/synnaxlabs/x v0.0.0
	github.com/tetratelabs/wazero v1.12.0
	github.com/uptrace/uptrace-go v1.43.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	go.uber.org/zap v1.28.0
	golang.org/x/crypto v0.53.0
	golang.org/x/sys v0.46.0
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/RaduBerinde/axisds v0.1.0 // indirect
	github.com/RaduBerinde/btreemap v0.0.0-20260105202824-d3184786f603 // indirect
	github.com/RoaringBitmap/roaring/v2 v2.18.2 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.5 // indirect
	github.com/blevesearch/bleve_index_api v1.3.12 // indirect
	github.com/blevesearch/geo v0.2.5 // indirect
	github.com/blevesearch/go-faiss v1.1.5 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.2.0 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.4.7 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.2.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.3 // indirect
	github.com/blevesearch/zapx/v12 v12.4.3 // indirect
	github.com/blevesearch/zapx/v13 v13.4.3 // indirect
	github.com/blevesearch/zapx/v14 v14.4.3 // indirect
	github.com/blevesearch/zapx/v15 v15.4.3 // indirect
	github.com/blevesearch/zapx/v16 v16.3.4 // indirect
	github.com/blevesearch/zapx/v17 v17.1.7 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/crlib v0.0.0-20251122031428-fe658a2dbda1 // indirect
	github.com/cockroachdb/errors v1.14.0 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/pebble v1.1.5 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/cockroachdb/swiss v0.0.0-20251224182025-b0f6560f979b // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/fasthttp/websocket v1.5.12 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/getsentry/sentry-go v0.47.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20260601182631-00ed12fed2a6 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gofiber/contrib/v3/websocket v1.2.0 // indirect
	github.com/gofiber/schema v1.8.0 // indirect
	github.com/gofiber/utils/v2 v2.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20260604005048-7023385849c0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20260330125221-c963978e514e // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/minio/minlz v1.1.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml/v2 v2.4.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.69.0 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/savsgio/gotils v0.0.0-20250924091648-bce9a52d7761 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.71.0 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.etcd.io/bbolt v1.5.0 // indirect
	go.lsp.dev/jsonrpc2 v1.0.0 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.lsp.dev/uri v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.69.0 // indirect
	go.opentelemetry.io/contrib/processors/minsev v0.16.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0 // indirect
	go.opentelemetry.io/otel/log v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260622175928-b703f567277d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260622175928-b703f567277d // indirect
)
