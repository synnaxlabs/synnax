// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
)

type TelemService struct {
	alamos.Instrumentation
	authProvider
	Framer *framer.Service
}

func NewTelemService(p Provider) *TelemService {
	return &TelemService{
		Instrumentation: p.Instrumentation,
		Framer:          p.Config.Framer,
		authProvider:    p.auth,
	}
}
