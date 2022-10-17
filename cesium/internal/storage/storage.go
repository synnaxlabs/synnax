package storage

import (
	"github.com/synnaxlabs/cesium/internal/file"
	"github.com/synnaxlabs/cesium/internal/persist"
	"github.com/synnaxlabs/x/confluence"
)

type Storage struct {
	persist *persist.Persist[file.Key]
	ops     confluence.Inlet[[]persist.Operation[file.Key]]
}

type Config = persist.Config

func Open(fs file.FS, cfg Config) (*Storage, error) {
	pst, err := persist.New[file.Key](fs, cfg)
	if err != nil {
		return nil, err
	}
	ops := confluence.NewStream[[]persist.Operation[file.Key]](cfg.NumWorkers)
	pst.InFrom(ops)
	return &Storage{
		persist: pst,
		ops:     ops,
	}, nil
}

func NewReader[R ReadRequest](s *Storage) Reader[R] {
	return &reader[R]{ops: s.ops}
}

func NewWriter[R WriteRequest](s *Storage) Writer[R] {
	return &writer[R]{ops: s.ops}
}
