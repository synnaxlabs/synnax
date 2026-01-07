// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package svc

import (
	"context"

	"github.com/cockroachdb/errors"
)

// ErrNotSupported is returned when service operations are attempted on non-Windows platforms.
var ErrNotSupported = errors.New("Windows Service is only supported on Windows")

// Service metadata constants (defined for API compatibility, not used on non-Windows).
const (
	Name        = "SynnaxServer"
	DisplayName = "Synnax Server"
	Description = "Synnax telemetry engine for hardware systems"
)

// Uninstall is not supported on non-Windows platforms.
func Uninstall() error { return ErrNotSupported }

// Start is not supported on non-Windows platforms.
func Start() error { return ErrNotSupported }

// Stop is not supported on non-Windows platforms.
func Stop() error { return ErrNotSupported }

// RunAsService is not supported on non-Windows platforms.
func RunAsService(func(context.Context) error) error { return ErrNotSupported }

// ParseServiceArgs is a no-op on non-Windows platforms.
func ParseServiceArgs(_ []string) error { return nil }
