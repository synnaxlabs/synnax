// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package config implements standardization utilities for managing service
// configurations.
package config

// Config is a configuration for a service that can be validated and override. Config is
// a recursive type, meaning that the type argument to C must be the config itself.
type Config[C any] interface {
	// Override sets all non-zero values from override on the config and returns the
	// merged result.
	Override(other C) C
	// Validate checks if the configuration is valid. Returns an error if it is not.
	Validate() error
}

// New creates a new configuration from a base configuration and a set of overrides. The
// overrides are applied in order, with the last override taking precedence. After the
// overrides are applied, the configuration is validated, returning an error if the
// configuration is invalid.
func New[C Config[C]](base C, overrides ...C) (C, error) {
	for _, override := range overrides {
		base = base.Override(override)
	}
	if err := base.Validate(); err != nil {
		var c C
		return c, err
	}
	return base, nil
}

// Bool returns a pointer to a boolean.
func Bool(b bool) *bool { return &b }

// True returns a pointer to a true boolean.
func True() *bool { return Bool(true) }

// False returns a pointer to a false boolean.
func False() *bool { return Bool(false) }
