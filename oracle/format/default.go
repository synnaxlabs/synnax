// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format

// Default returns the canonical Registry mapping each generated file
// extension to its formatter chain. The license-header formatter runs
// first, then the language-specific formatter(s):
//
//	.go              license, gofmt -s
//	.cpp / .h        license, clang-format
//	.ts / .tsx       license, prettier, eslint --fix
//	.py              license, ruff format, ruff check --fix
//	.proto           license, buf format
//
// Files with extensions not listed pass through with header only (or
// unchanged if the extension is also unknown to the License formatter).
func Default(repoRoot string) (*Registry, error) {
	license, err := NewLicense(repoRoot)
	if err != nil {
		return nil, err
	}
	r := NewRegistry()
	clang := NewClang()
	for _, ext := range []string{".cpp", ".cc", ".cxx", ".h", ".hpp"} {
		r.Register(ext, license)
		r.Register(ext, clang)
	}
	r.Register(".go", license)
	r.Register(".go", NewGo())
	prettier := NewPrettier()
	eslint := NewESLint()
	for _, ext := range []string{".ts", ".tsx"} {
		r.Register(ext, license)
		r.Register(ext, prettier)
		r.Register(ext, eslint)
	}
	r.Register(".py", license)
	r.Register(".py", NewRuff())
	r.Register(".proto", license)
	r.Register(".proto", NewBufFormat())
	return r, nil
}
