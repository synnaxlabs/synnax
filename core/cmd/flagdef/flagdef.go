// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package flagdef parses CLI flag definitions from JSON and registers them with cobra
// commands. The JSON files in each cmd subpackage are the single source of truth for
// flag names, defaults, types, and help text shared between the Go CLI and the docs
// site's CLI reference.
package flagdef

import (
	"encoding/json"
	"time"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/x/errors"
)

// Definition describes a single CLI flag. The fields mirror the columns in the docs
// site CLI reference table so that the same JSON file can drive both flag registration
// in Go and table rendering in MDX.
type Definition struct {
	// Name is the long-form flag name (without leading dashes).
	Name string `json:"name"`
	// Short is the optional one-character shorthand. Empty when the flag has no
	// shorthand.
	Short string `json:"short,omitempty"`
	// Type is one of "string", "bool", "int", "duration", or "stringSlice".
	Type string `json:"type"`
	// Default is the default value. For "duration" the value is a Go duration string
	// (e.g. "2.5s") that is parsed at registration time.
	Default any `json:"default"`
	// Description is the help text shown in CLI output and in the docs site flag
	// table.
	Description string `json:"description"`
	// Persistent registers the flag on the command's persistent flag set, so child
	// commands inherit it.
	Persistent bool `json:"persistent,omitempty"`
}

// Parse unmarshals a JSON byte slice into a slice of [Definition].
func Parse(data []byte) ([]Definition, error) {
	var defs []Definition
	if err := json.Unmarshal(data, &defs); err != nil {
		return nil, errors.Wrap(err, "parse flag definitions")
	}
	return defs, nil
}

// MustParse is the panicking variant of [Parse], intended for use with embedded JSON
// at package init.
func MustParse(data []byte) []Definition {
	defs, err := Parse(data)
	if err != nil {
		panic(err)
	}
	return defs
}

// Register registers each definition in defs with cmd.
func Register(cmd *cobra.Command, defs []Definition) error {
	for _, d := range defs {
		if err := registerOne(cmd, d); err != nil {
			return errors.Wrapf(err, "register flag %s", d.Name)
		}
	}
	return nil
}

// MustRegister is the panicking variant of [Register].
func MustRegister(cmd *cobra.Command, defs []Definition) {
	if err := Register(cmd, defs); err != nil {
		panic(err)
	}
}

func registerOne(cmd *cobra.Command, d Definition) error {
	fs := cmd.Flags()
	if d.Persistent {
		fs = cmd.PersistentFlags()
	}
	switch d.Type {
	case "string":
		v, err := asString(d.Default)
		if err != nil {
			return err
		}
		if d.Short != "" {
			fs.StringP(d.Name, d.Short, v, d.Description)
		} else {
			fs.String(d.Name, v, d.Description)
		}
	case "bool":
		v, err := asBool(d.Default)
		if err != nil {
			return err
		}
		if d.Short != "" {
			fs.BoolP(d.Name, d.Short, v, d.Description)
		} else {
			fs.Bool(d.Name, v, d.Description)
		}
	case "int":
		v, err := asInt(d.Default)
		if err != nil {
			return err
		}
		fs.Int(d.Name, v, d.Description)
	case "duration":
		s, err := asString(d.Default)
		if err != nil {
			return err
		}
		dur, err := time.ParseDuration(s)
		if err != nil {
			return errors.Wrapf(err, "parse duration %q", s)
		}
		fs.Duration(d.Name, dur, d.Description)
	case "stringSlice":
		v, err := asStringSlice(d.Default)
		if err != nil {
			return err
		}
		if d.Short != "" {
			fs.StringSliceP(d.Name, d.Short, v, d.Description)
		} else {
			fs.StringSlice(d.Name, v, d.Description)
		}
	default:
		return errors.Newf("unsupported flag type %q", d.Type)
	}
	return nil
}

func asString(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", errors.Newf("expected string default, got %T", v)
	}
	return s, nil
}

func asBool(v any) (bool, error) {
	if v == nil {
		return false, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, errors.Newf("expected bool default, got %T", v)
	}
	return b, nil
}

func asInt(v any) (int, error) {
	if v == nil {
		return 0, nil
	}
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	default:
		return 0, errors.Newf("expected numeric default, got %T", v)
	}
}

func asStringSlice(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, errors.Newf("expected array default, got %T", v)
	}
	out := make([]string, len(arr))
	for i, e := range arr {
		s, ok := e.(string)
		if !ok {
			return nil, errors.Newf("expected string element at index %d, got %T", i, e)
		}
		out[i] = s
	}
	return out, nil
}
