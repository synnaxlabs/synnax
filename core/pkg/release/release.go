// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package release downloads the files of a Synnax release and checks their signatures.
package release

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"aead.dev/minisign"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Config sets where Download gets release files and the key that checks them.
type Config struct {
	// URL is the base URL of the release files: <URL>/<product>/v<version>/<name> holds
	// a File, and the same URL plus .sig holds its signature.
	URL string
	// PublicKey is the minisign public key that checks each signature.
	PublicKey string
	// StallTimeout is how long Download waits for data before it fails.
	StallTimeout time.Duration
}

// DefaultConfig gets the files of the Synnax releases on GitHub. Its key is the one
// that the desktop Console updater checks.
var DefaultConfig = Config{
	URL:          "https://github.com/synnaxlabs/synnax/releases/download",
	PublicKey:    "RWQgxxfjVfpFyJKxsLvQ4hcTpvkSs4DJq0kIvcvdk49PTCCkovyHWZr1",
	StallTimeout: time.Minute,
}

// Validate returns an error when c has no URL, a public key that does not parse, or a
// stall timeout that is not positive.
func (c Config) Validate() error {
	v := validate.New("release.Config")
	v.NotEmptyString("url", c.URL)
	var key minisign.PublicKey
	v.Ternary(
		"public_key",
		key.UnmarshalText([]byte(c.PublicKey)) != nil,
		"must be a minisign public key",
	)
	v.Positive("stall_timeout", c.StallTimeout)
	return v.Error()
}

// Product is a Synnax product that has release files.
type Product string

const (
	// Core is the Synnax Core.
	Core Product = "core"
	// Driver is the Synnax Driver.
	Driver Product = "driver"
)

// Platform is an operating system that release files run on.
type Platform string

const (
	// Linux is Linux on x86-64.
	Linux Platform = "linux"
	// MacOS is macOS.
	MacOS Platform = "macos"
	// Windows is Windows on x86-64.
	Windows Platform = "windows"
	// NILinuxRT is NI Linux Real-Time.
	NILinuxRT Platform = "nilinuxrt"
)

// File is the binary of a product release for one platform.
type File struct {
	// Product is the product that the file belongs to.
	Product Product
	// Version is the release version without a "v" prefix, such as 0.61.0.
	Version string
	// Platform is the operating system that the file runs on.
	Platform Platform
}

// Name returns the name of the file in its release, such as synnax-v0.61.0-linux or
// synnax-driver-v0.61.0-windows.exe.
func (f File) Name() string {
	prefix := "synnax"
	if f.Product == Driver {
		prefix = "synnax-driver"
	}
	name := fmt.Sprintf("%s-v%s-%s", prefix, f.Version, f.Platform)
	if f.Platform == Windows {
		name += ".exe"
	}
	return name
}

var (
	// ErrNotFound is returned when a release has no file of the given name.
	ErrNotFound = errors.Wrap(query.ErrNotFound, "release file not found")
	// ErrSignature is returned when a file does not match its signature.
	ErrSignature = errors.New("release file does not match its signature")
)

// maxSignatureSize is far above the size of a signature file, about 500 bytes.
const maxSignatureSize = 4096

// Download downloads file to dst, with mode 0600, and checks it against its signature.
// A signature file holds the base64 text of a minisign signature, as the Tauri signer
// writes it, and its trusted comment must name the file. Download returns the error of
// cfg.Validate first. On any error, Download removes dst.
func Download(ctx context.Context, cfg Config, file File, dst string) (err error) {
	if err = cfg.Validate(); err != nil {
		return err
	}
	var key minisign.PublicKey
	if err = key.UnmarshalText([]byte(cfg.PublicKey)); err != nil {
		return err
	}
	name := file.Name()
	url := fmt.Sprintf("%s/%s/v%s/%s", cfg.URL, file.Product, file.Version, name)
	sigBody, err := fetch(ctx, url+".sig", cfg.StallTimeout)
	if err != nil {
		return err
	}
	encoded, err := io.ReadAll(io.LimitReader(sigBody, maxSignatureSize))
	if err = errors.Combine(err, sigBody.Close()); err != nil {
		return err
	}
	signature, err := base64.StdEncoding.DecodeString(
		strings.TrimSpace(string(encoded)),
	)
	if err != nil || !names(signature, name) {
		return errors.Wrapf(ErrSignature, "%s", url)
	}
	body, err := fetch(ctx, url, cfg.StallTimeout)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, body.Close()) }()
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Combine(err, os.Remove(dst))
		}
	}()
	r := minisign.NewReader(body)
	_, err = io.Copy(f, r)
	if err = errors.Combine(err, f.Close()); err != nil {
		return err
	}
	if !r.Verify(key, signature) {
		return errors.Wrapf(ErrSignature, "%s", url)
	}
	return nil
}

// names reports whether the trusted comment of signature names the file name, so a
// signed file of another product, platform, or version fails. Verify authenticates the
// comment.
func names(signature []byte, name string) bool {
	var s minisign.Signature
	if s.UnmarshalText(signature) != nil {
		return false
	}
	return slices.Contains(strings.Split(s.TrustedComment, "\t"), "file:"+name)
}

// fetch returns the body of url. The request fails when no data arrives for stall.
func fetch(
	ctx context.Context,
	url string,
	stall time.Duration,
) (io.ReadCloser, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	timer := time.AfterFunc(stall, func() {
		cancel(errors.Newf("failed to download %s: no data for %s", url, stall))
	})
	r := &stallReader{cancel: cancel, timer: timer, stall: stall}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Combine(err, r.Close())
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Combine(err, r.Close())
	}
	r.ReadCloser = res.Body
	switch res.StatusCode {
	case http.StatusOK:
		return r, nil
	case http.StatusNotFound:
		err = errors.Wrapf(ErrNotFound, "%s", url)
	default:
		err = errors.Newf("failed to download %s: %s", url, res.Status)
	}
	return nil, errors.Combine(err, r.Close())
}

// stallReader reads a response body and cancels its request when no data arrives for
// stall.
type stallReader struct {
	io.ReadCloser
	cancel context.CancelCauseFunc
	timer  *time.Timer
	stall  time.Duration
}

func (r *stallReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.timer.Reset(r.stall)
	}
	return n, err
}

func (r *stallReader) Close() (err error) {
	r.timer.Stop()
	r.cancel(nil)
	if r.ReadCloser != nil {
		err = r.ReadCloser.Close()
	}
	return err
}
