package kv

import (
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/errutil"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/version"
	"io"
)

type Variant uint32

const (
	Set Variant = iota
	Delete
)

type gossipState byte

const (
	infected gossipState = iota
	recovered
)

type Operation struct {
	Key         []byte
	Value       []byte
	Variant     Variant
	Version     version.Counter
	Leaseholder node.ID
	state       gossipState
}

func (o Operation) Digest() Digest {
	return Digest{
		Key:         o.Key,
		Version:     o.Version,
		Leaseholder: o.Leaseholder,
		Variant:     o.Variant,
	}
}

func (o Operation) apply(b kvx.Writer) error {
	if o.Variant == Delete {
		return b.Delete(o.Key)
	} else {
		return b.Set(o.Key, o.Value)
	}
}

type Digest struct {
	Key         []byte
	Version     version.Counter
	Leaseholder node.ID
	Variant     Variant
}

func (d Digest) apply(w kvx.Writer) error {
	key, err := digestKey(d.Key)
	if err != nil {
		return err
	}
	return kvx.Flush(w, key, d)
}

type Digests []Digest

func (d Digests) toRequest() BatchRequest {
	bd := BatchRequest{Operations: make([]Operation, len(d))}
	for i, d := range d {
		bd.Operations[i] = d.Operation()
	}
	return bd
}

type (
	segment = confluence.Segment[BatchRequest, BatchRequest]
	source  = confluence.Source[BatchRequest]
	sink    = confluence.Sink[BatchRequest]
)

// Load implements the kv.Loader interface.
func (d *Digest) Load(r io.Reader) error {
	c := errutil.NewCatchRead(r)
	c.Read(&d.Version)
	c.Read(&d.Leaseholder)
	c.Read(&d.Variant)
	return c.Error()
}

// Flush implements the kv.Flusher interface.
func (d Digest) Flush(w io.Writer) error {
	c := errutil.NewCatchWrite(w)
	c.Write(d.Version)
	c.Write(d.Leaseholder)
	c.Write(d.Variant)
	return c.Error()
}

func (d Digest) Operation() Operation {
	return Operation{
		Key:         d.Key,
		Version:     d.Version,
		Leaseholder: d.Leaseholder,
		Variant:     d.Variant,
	}
}

func digestKey(key []byte) (opKey []byte, err error) {
	return kvx.CompositeKey("--dig/", key)
}
