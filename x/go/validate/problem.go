// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// Problem is one rule a checked value breaks.
type Problem struct {
	// Path is the dotted location of the bad value, such as "messages.0.name". It is
	// empty for a problem with the whole value.
	Path string `json:"path" msgpack:"path"`
	// Message says what is wrong.
	Message string `json:"message" msgpack:"message"`
}

// Problems are the problems a check finds in one value, in the order the check found
// them.
type Problems []Problem

// Addf adds a problem at the path joined from segments.
func (p *Problems) Addf(path []string, format string, args ...any) {
	*p = append(*p, Problem{
		Path:    strings.Join(path, "."),
		Message: fmt.Sprintf(format, args...),
	})
}

// AddError adds each error that err holds as a problem. err may join several errors.
// A PathError becomes a problem at its path, and any other error a problem with the
// whole value.
func (p *Problems) AddError(err error) {
	if err == nil {
		return
	}
	if members := errors.UnwrapMulti(err); members != nil {
		for _, m := range members {
			p.AddError(m)
		}
		return
	}
	var pathErr PathError
	if errors.As(err, &pathErr) {
		*p = append(*p, Problem{
			Path:    strings.Join(pathErr.Path, "."),
			Message: pathErr.Err.Error(),
		})
		return
	}
	*p = append(*p, Problem{Message: err.Error()})
}
