module github.com/synnaxlabs/synnax

go 1.21

require (
	github.com/blevesearch/bleve v1.0.14
	github.com/cockroachdb/cmux v0.0.0-20170110192607-30d10be49292
	github.com/cockroachdb/errors v1.9.1
	github.com/cockroachdb/pebble v0.0.0-20230411220144-fa2c2ec6669a
	github.com/go-playground/validator/v10 v10.12.0
	github.com/gofiber/fiber/v2 v2.43.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo/v2 v2.11.0
	github.com/onsi/gomega v1.27.10
	github.com/samber/lo v1.38.1
	github.com/spf13/cobra v1.7.0
	github.com/spf13/viper v1.15.0
	github.com/synnaxlabs/alamos v0.0.0-00010101000000-000000000000
	github.com/synnaxlabs/aspen v0.0.0-20220804103056-48505d5ea44e
	github.com/synnaxlabs/cesium v0.0.0-20220722114246-333fea6b09d0
	github.com/synnaxlabs/freighter v0.0.0-20220810182625-b66219353383
	github.com/synnaxlabs/x v0.0.0-20220801122519-e4a5e96a532d
	github.com/uptrace/uptrace-go v1.15.0
	go.opentelemetry.io/otel v1.15.0
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.11.0
	golang.org/x/sys v0.10.0
	google.golang.org/grpc v1.54.0
	google.golang.org/protobuf v1.30.0
)

replace github.com/synnaxlabs/aspen => ../aspen

replace github.com/synnaxlabs/cesium => ../cesium

replace github.com/synnaxlabs/freighter => ../freighter/go

replace github.com/synnaxlabs/x => ../x/go

replace github.com/synnaxlabs/alamos => ../alamos/go

require (
	github.com/DataDog/zstd v1.5.2 // indirect
	github.com/RoaringBitmap/roaring v0.4.23 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/mmap-go v1.0.2 // indirect
	github.com/blevesearch/segment v0.9.0 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/zap/v11 v11.0.14 // indirect
	github.com/blevesearch/zap/v12 v12.0.14 // indirect
	github.com/blevesearch/zap/v13 v13.0.6 // indirect
	github.com/blevesearch/zap/v14 v14.0.5 // indirect
	github.com/blevesearch/zap/v15 v15.0.3 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/couchbase/vellum v1.0.2 // indirect
	github.com/fasthttp/websocket v1.5.2 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/getsentry/sentry-go v0.20.0 // indirect
	github.com/glycerine/go-unsnap-stream v0.0.0-20181221182339-f9677308dec2 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gofiber/websocket/v2 v2.1.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.16.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.3 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.7 // indirect
	github.com/philhofer/fwd v1.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/savsgio/dictpool v0.0.0-20221023140959-7bf2e61cea94 // indirect
	github.com/savsgio/gotils v0.0.0-20230208104028-c358bd845dee // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/steveyen/gtreap v0.1.0 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/tinylib/msgp v1.1.8 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.45.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/willf/bitset v1.1.10 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.41.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.15.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.15.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.15.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.15.0 // indirect
	go.opentelemetry.io/otel/metric v0.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.15.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.15.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/tools v0.9.3 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
