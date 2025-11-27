// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package templates provides embedded templates for jerky code generation.
package templates

import (
	"embed"
	"text/template"
)

//go:embed *.tmpl
var FS embed.FS

// funcMap provides custom functions for templates.
var funcMap = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
}

// Load loads all embedded templates.
func Load() (*template.Template, error) {
	return template.New("").Funcs(funcMap).ParseFS(FS, "*.tmpl")
}

// MustLoad loads all embedded templates and panics on error.
func MustLoad() *template.Template {
	t, err := Load()
	if err != nil {
		panic(err)
	}
	return t
}
