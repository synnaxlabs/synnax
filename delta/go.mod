module github.com/arya-analytics/delta

go 1.19

require (
	github.com/arya-analytics/aspen v0.0.0-20220804103056-48505d5ea44e
	github.com/arya-analytics/cesium v0.0.0-20220722114246-333fea6b09d0
	github.com/arya-analytics/client-go v0.0.0-00010101000000-000000000000
	github.com/arya-analytics/freighter v0.0.0-20220810182625-b66219353383
	github.com/arya-analytics/x v0.0.0-20220801122519-e4a5e96a532d
	github.com/cockroachdb/cmux v0.0.0-20170110192607-30d10be49292
	github.com/cockroachdb/errors v1.9.0
	github.com/cockroachdb/pebble v0.0.0-20220513193540-b8c9a560bed5
	github.com/go-playground/validator/v10 v10.11.0
	github.com/gofiber/fiber/v2 v2.36.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.20.0
	github.com/samber/lo v1.27.0
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.5.0
	github.com/spf13/viper v1.12.0
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f
	google.golang.org/grpc v1.46.2
	google.golang.org/protobuf v1.28.0
)

replace (
	github.com/arya-analytics/aspen => ../aspen
	github.com/arya-analytics/cesium => ../cesium
	github.com/arya-analytics/client-go => ../client/go
	github.com/arya-analytics/freighter => ../freighter/go
	github.com/arya-analytics/x => ../x
)

require (
	github.com/DataDog/zstd v1.4.5 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/fasthttp/websocket v1.5.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/getsentry/sentry-go v0.12.0 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/gofiber/websocket/v2 v2.0.24 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/klauspost/compress v1.15.7 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/savsgio/gotils v0.0.0-20211223103454-d0aaa54c5899 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.38.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/exp v0.0.0-20220303212507-bbda1eaf7a17 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20220519153652-3a47de7e79bd // indirect
	google.golang.org/grpc/examples v0.0.0-20220721223107-fdc5d2f3da85 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
