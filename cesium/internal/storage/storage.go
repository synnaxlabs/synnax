package storage

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/telem"
	"io"
)

type Reader interface {
	Read(md []core.SegmentMD) ([]core.SugaredSegment, error)
}

type Writer interface {
	Write(ss []core.SugaredSegment) ([]core.SegmentMD, error)
}

type Storage struct {
	fs core.FS
}

func (s *Storage) NewReader() Reader {
	return &reader{fs: s.fs}
}

func (s *Storage) NewWriter() Writer {
	return &writer{fs: s.fs}
}

func Wrap(fs core.FS) *Storage { return &Storage{fs: fs} }

type reader struct {
	fs core.FS
}

func (r *reader) Read(md []core.SegmentMD) ([]core.SugaredSegment, error) {
	segments := make([]core.SugaredSegment, len(md))
	for i, md := range md {
		seg, err := r.read(md)
		if err != nil {
			return nil, err
		}
		segments[i] = seg
	}
	return segments, nil
}

func (r *reader) read(md core.SegmentMD) (seg core.SugaredSegment, err error) {
	seg.SegmentMD = md
	f, err := r.fs.Acquire(seg.FileKey)
	if err != nil {
		return seg, err
	}
	defer r.fs.Release(md.FileKey)
	seg.Data = make([]byte, md.Size)
	_, err = f.ReadAt(seg.Data, int64(md.Offset))
	return seg, err
}

type writer struct {
	fs core.FS
}

func (w *writer) Write(ss []core.SugaredSegment) ([]core.SegmentMD, error) {
	mds := make([]core.SegmentMD, len(ss))
	for i, seg := range ss {
		md, err := w.write(seg)
		if err != nil {
			return mds, err
		}
		mds[i] = md
	}
	return mds, nil
}

func (w *writer) write(seg core.SugaredSegment) (core.SegmentMD, error) {
	md := seg.SegmentMD
	f, err := w.fs.Acquire(seg.FileKey)
	if err != nil {
		return md, err
	}
	defer w.fs.Release(seg.FileKey)
	off, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return md, err
	}
	md.Offset = telem.Offset(off)
	_, err = f.Write(seg.Data)
	md.Size = telem.Size(len(seg.Data))
	return md, err
}
