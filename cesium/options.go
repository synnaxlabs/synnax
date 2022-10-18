package cesium

import (
	"github.com/cockroachdb/pebble/vfs"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/kfs"
	"github.com/synnaxlabs/x/kv"
	"go.uber.org/zap"
	"time"
)

const (
	// cesiumDirectory is the directory in which cesium files are stored.
	cesiumDirectory = "cesium"
	// kvDirectory is the directory in which kv files are stored.
	kvDirectory = "kv"
)

type Option func(*options)

type options struct {
	dirname string
	fs      struct {

		// kfs.BaseFS will be embedded as an option here.
		opts []kfs.Option
		sync struct {
			interval time.Duration
			maxAge   time.Duration
		}
	}
	exp    alamos.Experiment
	logger *zap.Logger
	kv     struct {
		external bool
		engine   kv.DB
		// fs is the file system we use for key-value storage. We don't use pebble's
		// vfs for the time series engine because it doesn't implement seek handles
		// for its files.
		fs vfs.FS
	}
}

func (o *options) logArgs() []interface{} {
	return []interface{}{
		"dirname", o.dirname,
		"syncInterval", o.fs.sync.interval,
		"syncMaxAge", o.fs.sync.maxAge,
		"kvExternal", o.kv.external,
	}
}

func newOptions(dirname string, opts ...Option) *options {
	o := &options{dirname: dirname}
	for _, opt := range opts {
		opt(o)
	}
	mergeDefaultOptions(o)
	return o
}

func mergeDefaultOptions(o *options) {
	if o.fs.sync.interval == 0 {
		o.fs.sync.interval = 1 * time.Second
	}
	if o.fs.sync.maxAge == 0 {
		o.fs.sync.maxAge = 1 * time.Hour
	}

	// || LOGGER ||

	if o.logger == nil {
		o.logger = zap.NewNop()
	}
	o.fs.opts = append(o.fs.opts, kfs.WithExperiment(o.exp))
	o.fs.opts = append(o.fs.opts, kfs.WithExtensionConfig(".tof"))
}

func MemBacked() Option {
	return func(o *options) {
		o.dirname = ""
		WithFS(vfs.NewMem(), kfs.NewMem())(o)
	}
}

func WithFS(vfs vfs.FS, baseKFS kfs.BaseFS) Option {
	return func(o *options) {
		o.kv.fs = vfs
		o.fs.opts = append(o.fs.opts, kfs.WithFS(baseKFS))
	}
}

func WithKVEngine(kv kv.DB) Option {
	return func(o *options) {
		o.kv.external = true
		o.kv.engine = kv
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

func WithExperiment(exp alamos.Experiment) Option {
	return func(o *options) {
		o.exp = alamos.Sub(exp, "cesium")
	}
}
