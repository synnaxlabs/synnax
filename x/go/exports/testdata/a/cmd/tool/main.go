// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"os"
	"text/template"

	"example.com/a/lib"
)

func Exported() {}

type data struct{ Imports lib.Imports }

func main() {
	Exported()
	t := template.Must(template.New("t").Parse("{{.Imports.Has}}"))
	if err := t.Execute(os.Stdout, data{}); err != nil {
		panic(err)
	}
}
