module github.com/synnaxlabs/arc

go 1.26.4

replace (
	github.com/synnaxlabs/alamos => ../../alamos/go
	github.com/synnaxlabs/freighter => ../../freighter/go
	github.com/synnaxlabs/x => ../../x/go
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.1
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/onsi/ginkgo/v2 v2.32.0
	github.com/onsi/gomega v1.42.0
	github.com/samber/lo v1.53.0
	github.com/synnaxlabs/alamos v0.0.0
	github.com/synnaxlabs/freighter v0.0.0
	github.com/synnaxlabs/x v0.0.0
	github.com/tetratelabs/wazero v1.12.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	go.lsp.dev/jsonrpc2 v0.10.0
	go.uber.org/zap v1.28.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/errors v1.14.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/getsentry/sentry-go v0.47.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/pprof v0.0.0-20260604005048-7023385849c0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.3.4 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.lsp.dev/uri v1.0.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260622175928-b703f567277d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260622175928-b703f567277d // indirect
)
