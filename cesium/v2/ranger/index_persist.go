package ranger

import (
	"encoding/binary"
	"github.com/synnaxlabs/x/telem"
	"io"
	"os"
)

type indexPersist struct {
	Config
	indexEncoder
	io.ReadWriteSeeker
	idx *index
}

const indexFile = "index"

func openIndexPersist(idx *index, cfg Config) (*indexPersist, error) {
	f, err := cfg.FS.OpenFile(fileName(indexFile), os.O_CREATE|os.O_RDWR, 0644)
	ip := &indexPersist{ReadWriteSeeker: f, idx: idx}
	idx.OnChange(ip.onChange)
	return ip, err
}

func (f *indexPersist) onChange(update indexUpdate) {
	var encoded []byte
	f.idx.read(func() {
		encoded = f.encode(update.afterIndex, f.idx.mu.pointers)
	})
	if _, err := f.Seek(int64(update.afterIndex*pointerByteSize), io.SeekStart); err != nil {
		f.Logger.Error(err.Error())
	}
	_, err := f.Write(encoded)
	if err != nil {
		f.Logger.Error(err.Error())
	}
}

func (f *indexPersist) load() ([]pointer, error) {
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	b := make([]byte, size)
	if _, err := f.Read(b); err != nil {
		return nil, err
	}
	return f.decode(b), nil
}

type indexEncoder struct{}

var byteOrder = binary.LittleEndian

func (f *indexEncoder) encode(start int, pointers []pointer) []byte {
	b := make([]byte, (len(pointers)-start)*pointerByteSize)
	for i := start; i < len(pointers); i++ {
		ptr := pointers[i]
		base := (i - start) * pointerByteSize
		byteOrder.PutUint64(b[base:base+8], uint64(ptr.Start))
		byteOrder.PutUint64(b[base+8:base+16], uint64(ptr.End))
		byteOrder.PutUint16(b[base+16:base+18], ptr.fileKey)
		byteOrder.PutUint32(b[base+18:base+22], ptr.offset)
		byteOrder.PutUint32(b[base+22:base+26], ptr.length)
	}
	return b
}

func (f *indexEncoder) decode(b []byte) []pointer {
	pointers := make([]pointer, len(b)/pointerByteSize)
	for i := 0; i < len(pointers); i++ {
		base := i * pointerByteSize
		pointers[i] = pointer{
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(byteOrder.Uint64(b[base : base+8])),
				End:   telem.TimeStamp(byteOrder.Uint64(b[base+8 : base+16])),
			},
			fileKey: byteOrder.Uint16(b[base+16 : base+18]),
			offset:  byteOrder.Uint32(b[base+18 : base+22]),
			length:  byteOrder.Uint32(b[base+22 : base+26]),
		}
	}
	return pointers
}
