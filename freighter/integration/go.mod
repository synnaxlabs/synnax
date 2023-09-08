module github.com/synnaxlabs/freighter/integration

go 1.20

replace (
	github.com/synnaxlabs/alamos => ../../alamos/go
	github.com/synnaxlabs/freighter => ../go
	github.com/synnaxlabs/x => ../../x/go
)

require (
	github.com/cockroachdb/errors v1.9.1
	github.com/gofiber/fiber/v2 v2.43.0
	github.com/synnaxlabs/freighter v0.0.0-00010101000000-000000000000
	github.com/synnaxlabs/x v0.0.0-20220801122519-e4a5e96a532d
	go.uber.org/zap v1.24.0
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/fasthttp/websocket v1.5.2 // indirect
	github.com/getsentry/sentry-go v0.20.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gofiber/websocket/v2 v2.1.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2 // indirect
	github.com/klauspost/compress v1.16.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/onsi/ginkgo/v2 v2.9.2 // indirect
	github.com/onsi/gomega v1.27.6 // indirect
	github.com/philhofer/fwd v1.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/samber/lo v1.38.1 // indirect
	github.com/savsgio/dictpool v0.0.0-20221023140959-7bf2e61cea94 // indirect
	github.com/savsgio/gotils v0.0.0-20230208104028-c358bd845dee // indirect
	github.com/synnaxlabs/alamos v0.0.0-00010101000000-000000000000 // indirect
	github.com/tinylib/msgp v1.1.8 // indirect
	github.com/uptrace/uptrace-go v1.15.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.45.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.41.0 // indirect
	go.opentelemetry.io/otel v1.15.0 // indirect
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
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.54.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
