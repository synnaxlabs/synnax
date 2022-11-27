package cesium

import (
	"github.com/synnaxlabs/cesium/internal/unary"
	xfs "github.com/synnaxlabs/x/io/fs"
)

func Open(dirname string, opts ...Option) (DB, error) {
	o := newOptions(dirname, opts...)
	if err := openFS(o); err != nil {
		return nil, err
	}

	sugLog := o.logger.Sugar()
	sugLog.Infow("opening cesium time series engine", o.logArgs()...)

	info, err := o.fs.List()
	if err != nil {
		return nil, err
	}
	_db := &cesium{options: o, dbs: make(map[string]unary.DB, len(info))}
	for _, i := range info {
		if i.IsDir() {
			err := _db.openUnary(Channel{Key: i.Name()})
			if err != nil {
				return nil, err
			}
		}
	}
	return _db, nil
}

func openFS(opts *options) error {
	if opts.fs == nil {
		_fs, err := xfs.DefaultFS.Sub(opts.dirname)
		if err != nil {
			return err
		}
		opts.fs = _fs
	}
	return nil
}
