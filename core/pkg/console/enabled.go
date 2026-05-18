// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build console

package console

import (
	"embed"
	"io/fs"

	"github.com/samber/lo"
)

//go:embed all:dist
var embeddedAssets embed.FS

var defaultFS fs.FS

func init() { defaultFS = lo.Must(fs.Sub(embeddedAssets, "dist")) }
