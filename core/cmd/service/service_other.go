// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !linux && !windows

package service

import "github.com/synnaxlabs/x/errors"

var errUnsupported = errors.New("the Core service runs only on Linux and Windows")

// Lookup returns false, because the Core service runs only on Linux and Windows.
func Lookup() (Installation, bool, error) { return Installation{}, false, nil }

// Start returns an error, because the Core service runs only on Linux and Windows.
func Start() error { return errUnsupported }

// Stop returns an error, because the Core service runs only on Linux and Windows.
func Stop() error { return errUnsupported }
