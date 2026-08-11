// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Identity is the Content-Encoding token for a body that is not compressed.
const Identity = "identity"

// ErrUnsupportedContentEncoding is returned when a message declares a Content-Encoding
// that no available Compression handles.
var ErrUnsupportedContentEncoding = errors.Wrap(
	validate.ErrValidation, "unsupported content encoding",
)

// ErrBodyTooLarge is returned when a compressed body expands past the caller's limit.
// It guards against a small request body that decompresses into an arbitrarily large
// one.
var ErrBodyTooLarge = errors.Wrap(
	validate.ErrValidation, "decompressed body exceeds the maximum size",
)

// Compression compresses and decompresses HTTP message bodies under a single
// Content-Encoding token. Implementations are safe for concurrent use.
type Compression interface {
	// ContentEncoding returns the token this Compression reads and writes in the
	// Content-Encoding and Accept-Encoding headers.
	ContentEncoding() string
	// Compress returns src compressed. The returned slice does not alias src.
	Compress(src []byte) ([]byte, error)
	// Decompress returns src decompressed. It returns ErrBodyTooLarge if the
	// decompressed body would exceed maxSize bytes; a maxSize of zero applies no
	// limit.
	Decompress(src []byte, maxSize int) ([]byte, error)
}

var (
	// Gzip is the gzip (RFC 1952) Compression. Every HTTP client understands it, which
	// makes it the safe last resort, but its 32KB window costs it an order of
	// magnitude of ratio on bodies with repetition spread wider than that.
	Gzip Compression = &gzipCompression{
		writers: sync.Pool{New: func() any {
			w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
			return w
		}},
		readers: sync.Pool{New: func() any { return new(gzip.Reader) }},
	}
	// Brotli is the brotli (RFC 7932) Compression. It reaches the best ratio of the
	// three and every current browser accepts it, but it is the slowest by far.
	Brotli Compression = &brotliCompression{
		writers: sync.Pool{New: func() any {
			return brotli.NewWriterLevel(io.Discard, brotliDefaultLevel)
		}},
	}
	// Zstd is the zstandard (RFC 8878) Compression. It reaches within a few percent of
	// Brotli's ratio at five to ten times the speed, making it the first choice for any
	// peer that accepts it.
	Zstd Compression = newZstdCompression()
	// Deflate is the zlib (RFC 1950) Compression that HTTP names "deflate". It offers
	// nothing over Gzip beyond a few bytes of framing, and exists so a peer whose
	// runtime only exposes deflate is still understood.
	Deflate Compression = &deflateCompression{
		writers: sync.Pool{New: func() any {
			w, _ := zlib.NewWriterLevel(io.Discard, zlib.DefaultCompression)
			return w
		}},
	}
)

// brotliDefaultLevel trades a small amount of ratio for a large amount of speed.
// Brotli's own default (6) costs roughly four times the CPU of gzip for a few percent
// of size on JSON bodies.
const brotliDefaultLevel = 4

type gzipCompression struct {
	// writers pools gzip.Writer instances, which allocate a 32KB window each.
	writers sync.Pool
	// readers pools gzip.Reader instances.
	readers sync.Pool
}

func (*gzipCompression) ContentEncoding() string { return "gzip" }

func (c *gzipCompression) Compress(src []byte) ([]byte, error) {
	w := c.writers.Get().(*gzip.Writer)
	defer c.writers.Put(w)
	var buf bytes.Buffer
	buf.Grow(len(src) / 3)
	w.Reset(&buf)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *gzipCompression) Decompress(src []byte, maxSize int) ([]byte, error) {
	r := c.readers.Get().(*gzip.Reader)
	defer c.readers.Put(r)
	if err := r.Reset(bytes.NewReader(src)); err != nil {
		return nil, err
	}
	return readAllLimited(r, maxSize)
}

type deflateCompression struct {
	// writers pools zlib.Writer instances, which allocate a compression window each.
	writers sync.Pool
}

func (*deflateCompression) ContentEncoding() string { return "deflate" }

func (c *deflateCompression) Compress(src []byte) ([]byte, error) {
	w := c.writers.Get().(*zlib.Writer)
	defer c.writers.Put(w)
	var buf bytes.Buffer
	buf.Grow(len(src) / 3)
	w.Reset(&buf)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (*deflateCompression) Decompress(src []byte, maxSize int) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return readAllLimited(r, maxSize)
}

type brotliCompression struct {
	// writers pools brotli.Writer instances, which allocate encoder state each.
	writers sync.Pool
}

func (*brotliCompression) ContentEncoding() string { return "br" }

func (c *brotliCompression) Compress(src []byte) ([]byte, error) {
	w := c.writers.Get().(*brotli.Writer)
	defer c.writers.Put(w)
	var buf bytes.Buffer
	buf.Grow(len(src) / 3)
	w.Reset(&buf)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (*brotliCompression) Decompress(src []byte, maxSize int) ([]byte, error) {
	return readAllLimited(brotli.NewReader(bytes.NewReader(src)), maxSize)
}

// zstdCompression holds one shared encoder, whose EncodeAll is safe for concurrent use
// and pools its own state, alongside a pool of decoders. Decoders are pooled rather
// than shared because bounding the output requires reading a decoder as a stream, and a
// streaming decoder carries per-body state.
type zstdCompression struct {
	encoder  *zstd.Encoder
	decoders sync.Pool
}

func newZstdCompression() *zstdCompression {
	encoder, err := zstd.NewWriter(
		nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(1),
	)
	if err != nil {
		panic(err)
	}
	return &zstdCompression{
		encoder: encoder,
		decoders: sync.Pool{New: func() any {
			d, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
			if err != nil {
				panic(err)
			}
			return d
		}},
	}
}

func (*zstdCompression) ContentEncoding() string { return "zstd" }

func (c *zstdCompression) Compress(src []byte) ([]byte, error) {
	return c.encoder.EncodeAll(src, make([]byte, 0, len(src)/3)), nil
}

func (c *zstdCompression) Decompress(src []byte, maxSize int) ([]byte, error) {
	d := c.decoders.Get().(*zstd.Decoder)
	// Closing the decoder would retire it permanently, so it is Reset and returned to
	// the pool instead. Resetting onto an empty reader drops any reference to src.
	defer func() {
		if err := d.Reset(nil); err == nil {
			c.decoders.Put(d)
		}
	}()
	if err := d.Reset(bytes.NewReader(src)); err != nil {
		return nil, err
	}
	return readAllLimited(d, maxSize)
}

// readAllLimited reads r to completion, returning ErrBodyTooLarge once the output
// passes maxSize bytes. A maxSize of zero or less applies no limit.
func readAllLimited(r io.Reader, maxSize int) ([]byte, error) {
	if maxSize <= 0 {
		b, err := io.ReadAll(r)
		if err != nil {
			// io.ReadAll hands back what it managed to read alongside the error.
			// A half-decompressed body is never useful to a caller, and returning
			// it invites treating it as a whole one.
			return nil, err
		}
		return b, nil
	}
	// Read one byte past the limit so a body sitting exactly on it still succeeds
	// while the first byte over it is caught.
	b, err := io.ReadAll(io.LimitReader(r, int64(maxSize)+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxSize {
		return nil, errors.Wrapf(ErrBodyTooLarge, "limit is %d bytes", maxSize)
	}
	return b, nil
}

// ResolveCompression returns the Compression in options matching contentEncoding, for
// decoding a body whose encoding the sender has already chosen. It returns a nil
// Compression and a nil error when the body is not compressed, meaning contentEncoding
// is empty or identity. It returns ErrUnsupportedContentEncoding when no option handles
// the token.
func ResolveCompression(
	contentEncoding string,
	options []Compression,
) (Compression, error) {
	token := strings.ToLower(strings.TrimSpace(contentEncoding))
	if token == "" || token == Identity {
		return nil, nil
	}
	for _, o := range options {
		if o.ContentEncoding() == token {
			return o, nil
		}
	}
	return nil, errors.Wrapf(ErrUnsupportedContentEncoding, "%q", contentEncoding)
}

// NegotiateCompression returns the Compression in options that acceptEncoding ranks
// highest, for choosing an encoding the receiver has told us it accepts. Ties resolve
// to the earlier entry in options, so callers order options by preference. It returns
// false when the receiver accepts none of them, in which case the body must be sent
// uncompressed. An absent or empty Accept-Encoding never selects a Compression: a
// sender that does not advertise cannot be assumed to decode.
func NegotiateCompression(
	acceptEncoding string,
	options []Compression,
) (Compression, bool) {
	if len(options) == 0 || strings.TrimSpace(acceptEncoding) == "" {
		return nil, false
	}
	qualities, wildcard, hasWildcard := parseAcceptEncoding(acceptEncoding)
	var (
		best     Compression
		bestQ    float64
		bestSeen bool
	)
	for _, o := range options {
		q, ok := qualities[o.ContentEncoding()]
		if !ok {
			if !hasWildcard {
				continue
			}
			q = wildcard
		}
		// A quality of zero is an explicit rejection, not a low preference.
		if q <= 0 || (bestSeen && q <= bestQ) {
			continue
		}
		best, bestQ, bestSeen = o, q, true
	}
	return best, bestSeen
}

// parseAcceptEncoding splits an Accept-Encoding header into its per-token quality
// values, plus the quality of the "*" wildcard and whether the wildcard was present.
// Tokens without an explicit q parameter take a quality of 1.
func parseAcceptEncoding(header string) (map[string]float64, float64, bool) {
	qualities := make(map[string]float64)
	var (
		wildcard    float64
		hasWildcard bool
	)
	for part := range strings.SplitSeq(header, ",") {
		token, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			continue
		}
		q := 1.0
		if key, value, ok := strings.Cut(params, "="); ok &&
			strings.EqualFold(strings.TrimSpace(key), "q") {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			// A malformed q parameter is ignored rather than rejected, leaving the
			// token at its default quality.
			if err == nil {
				q = parsed
			}
		}
		if token == "*" {
			wildcard, hasWildcard = q, true
			continue
		}
		qualities[token] = q
	}
	return qualities, wildcard, hasWildcard
}
