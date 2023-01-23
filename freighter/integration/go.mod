module github.com/synnaxlabs/freighter/integration

go 1.19

replace (
	github.com/synnaxlabs/freighter => ../go
	github.com/synnaxlabs/x => ../../x/go
)

require (
	github.com/cockroachdb/errors v1.9.0
	github.com/gofiber/fiber/v2 v2.36.0
	github.com/synnaxlabs/freighter v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.22.0
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/fasthttp/websocket v1.5.0 // indirect
	github.com/getsentry/sentry-go v0.12.0 // indirect
	github.com/gofiber/websocket/v2 v2.0.24 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/klauspost/compress v1.15.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/samber/lo v1.27.0 // indirect
	github.com/savsgio/gotils v0.0.0-20211223103454-d0aaa54c5899 // indirect
	github.com/synnaxlabs/x v0.0.0-20220801122519-e4a5e96a532d // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.38.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/exp v0.0.0-20220303212507-bbda1eaf7a17 // indirect
	golang.org/x/sys v0.4.0 // indirect
)
