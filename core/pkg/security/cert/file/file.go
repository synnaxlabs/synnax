// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package file provides a certificate source that serves a PEM cert/key pair from disk,
// hot-reloading on change so rotation needs no restart.
package file

import (
	"crypto/tls"
	"io"
	"os"
	"sync"
	"time"

	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
)

// SourceType is the configuration token selecting the file source.
const SourceType = "file"

// NewSource builds a file source serving the PEM pair at certPath and keyPath, reading
// them through fs. It returns validate.ErrValidation if either path is empty.
func NewSource(fs xfs.FS, certPath, keyPath string) (cert.Source, error) {
	if certPath == "" || keyPath == "" {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"file source requires both a cert and a key path",
		)
	}
	if fs == nil {
		fs = xfs.Default
	}
	return &source{fs: fs, certPath: certPath, keyPath: keyPath}, nil
}

var _ cert.Source = (*source)(nil)

// source serves a certificate from PEM files, reloading them when they change so
// certificate rotation needs no restart.
type source struct {
	fs       xfs.FS
	certPath string
	keyPath  string
	mu       sync.Mutex
	cached   *tls.Certificate
	certMod  time.Time
	keyMod   time.Time
}

// GetCertificate implements cert.Source.
func (s *source) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	certMod, err := s.modTime(s.certPath)
	if err != nil {
		return nil, err
	}
	keyMod, err := s.modTime(s.keyPath)
	if err != nil {
		return nil, err
	}
	if s.cached != nil && certMod.Equal(s.certMod) && keyMod.Equal(s.keyMod) {
		return s.cached, nil
	}
	c, err := s.load()
	if err != nil {
		return nil, err
	}
	s.cached, s.certMod, s.keyMod = c, certMod, keyMod
	return c, nil
}

func (s *source) modTime(path string) (time.Time, error) {
	info, err := s.fs.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (s *source) load() (*tls.Certificate, error) {
	certPEM, err := s.readAll(s.certPath)
	if err != nil {
		return nil, err
	}
	keyPEM, err := s.readAll(s.keyPath)
	if err != nil {
		return nil, err
	}
	c, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *source) readAll(path string) (b []byte, err error) {
	f, err := s.fs.Open(path, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	return io.ReadAll(f)
}
