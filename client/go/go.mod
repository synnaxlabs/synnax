module github.com/arya-analytics/client-go

go 1.19

replace github.com/arya-analytics/x => ../../x

replace github.com/arya-analytics/delta => ../../delta

replace github.com/arya-analytics/freighter => ../../freighter/go

require (
	github.com/arya-analytics/delta v0.0.0-20220801132418-2a1462153dba
	github.com/arya-analytics/freighter v0.0.0-20220810182625-b66219353383
	github.com/arya-analytics/x v0.0.0-20220801122519-e4a5e96a532d
	github.com/cockroachdb/errors v1.9.0
	github.com/sirupsen/logrus v1.9.0
)

require (
	github.com/DataDog/zstd v1.4.5 // indirect
	github.com/arya-analytics/aspen v0.0.0-20220804103056-48505d5ea44e // indirect
	github.com/arya-analytics/cesium v0.0.0-20220722114246-333fea6b09d0 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/pebble v0.0.0-20220513193540-b8c9a560bed5 // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/getsentry/sentry-go v0.12.0 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.11.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/klauspost/compress v1.15.7 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/samber/lo v1.27.0 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	golang.org/x/exp v0.0.0-20220303212507-bbda1eaf7a17 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4 // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20220519153652-3a47de7e79bd // indirect
	google.golang.org/grpc v1.46.2 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
