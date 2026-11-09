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
	"strings"

	"aead.dev/minisign"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Config sets where Download gets release files and the key that checks them.
type Config struct {
	// URL is the base URL of the release files: <URL>/<product>/v<version>/<name>
	// holds a file, and the same URL plus .sig holds its signature.
	URL string
	// PublicKey is the minisign public key that checks each signature.
	PublicKey string
}

// DefaultConfig gets the files of the Synnax releases on GitHub. Its key is the one
// that the desktop Console updater checks.
var DefaultConfig = Config{
	URL:       "https://github.com/synnaxlabs/synnax/releases/download",
	PublicKey: "RWQgxxfjVfpFyJKxsLvQ4hcTpvkSs4DJq0kIvcvdk49PTCCkovyHWZr1",
}

var (
	// ErrNotFound is returned when a release has no file of the given name.
	ErrNotFound = errors.Wrap(query.ErrNotFound, "release file not found")
	// ErrSignature is returned when a file does not match its signature.
	ErrSignature = errors.New("release file does not match its signature")
)

// maxSignatureSize is far above the size of a signature file, about 500 bytes.
const maxSignatureSize = 4096

// Download downloads the file name of the release of product at version to dst, with
// mode 0600, and checks the file against its signature. A signature file holds the
// base64 text of a minisign signature, as the Tauri signer writes it. On any error,
// Download removes dst.
func Download(
	ctx context.Context,
	cfg Config,
	product, version, name, dst string,
) (err error) {
	var key minisign.PublicKey
	if err = key.UnmarshalText([]byte(cfg.PublicKey)); err != nil {
		return errors.Wrap(err, "invalid release public key")
	}
	url := fmt.Sprintf("%s/%s/v%s/%s", cfg.URL, product, version, name)
	sigBody, err := fetch(ctx, url+".sig")
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
	if err != nil {
		return errors.Wrapf(ErrSignature, "%s", url)
	}
	body, err := fetch(ctx, url)
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

func fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	switch res.StatusCode {
	case http.StatusOK:
		return res.Body, nil
	case http.StatusNotFound:
		err = errors.Wrapf(ErrNotFound, "%s", url)
	default:
		err = errors.Newf("failed to download %s: %s", url, res.Status)
	}
	return nil, errors.Combine(err, res.Body.Close())
}
