// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides helpers for testing imex exporters and importers.
package testutil

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/testutil"
)

// LoadEnvelope reads the wire-format envelope fixture at path and unmarshals it into an
// imex.Envelope, binding the codec Decode needs.
func LoadEnvelope(path string) imex.Envelope {
	ginkgo.GinkgoHelper()
	raw := testutil.MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	gomega.Expect(json.Unmarshal(raw, &env)).To(gomega.Succeed())
	return env
}

// WireRoundTrip marshals env to JSON and back, binding the codec Decode needs. Exported
// envelopes carry a body but no codec, so a decode must first pass through the wire.
func WireRoundTrip(env imex.Envelope) imex.Envelope {
	ginkgo.GinkgoHelper()
	b := testutil.MustSucceed(json.Marshal(env))
	var out imex.Envelope
	gomega.Expect(json.Unmarshal(b, &out)).To(gomega.Succeed())
	return out
}

// LoadBundle reads the bundle directory rooted at dir into the zip.Files an import
// takes, keyed by slash-separated path from the root. Every regular file in the tree is
// a member, directories included; an empty directory contributes nothing, matching what
// an archive carries.
func LoadBundle(dir string) zip.Files {
	ginkgo.GinkgoHelper()
	files := zip.Files{}
	gomega.Expect(filepath.WalkDir(
		dir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files[strings.ReplaceAll(rel, string(filepath.Separator), "/")] = data
			return nil
		},
	)).To(gomega.Succeed())
	return files
}
