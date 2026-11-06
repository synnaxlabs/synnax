// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11

import (
	"encoding/json"

	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// decodeElementConfig validates an opaque config payload against the element config
// union. It wraps validate.ErrValidation when the payload names no variant, names one
// the union does not, or carries fields that do not fit the variant it names.
func decodeElementConfig(raw msgpack.EncodedJSON) (ElementConfig, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return ElementConfig{}, err
	}
	var cfg ElementConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return ElementConfig{}, errors.Wrapf(
			validate.ErrValidation, "invalid element config: %s", err,
		)
	}
	// A null payload decodes to a nil variant without erroring, which would persist an
	// entry no client can read back.
	if cfg.Variant == nil {
		return ElementConfig{}, errors.Wrap(
			validate.ErrValidation, "element config names no variant",
		)
	}
	return cfg, nil
}

// elementConfigFields returns a config's wire fields as an opaque map for partial
// merging.
func elementConfigFields(cfg ElementConfig) (msgpack.EncodedJSON, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var m msgpack.EncodedJSON
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// DecodeWithDefaults decodes a normalized config over its variant's schema defaults. A
// field the config carries keeps its value even when it is zero; only an omitted field
// takes the default. It returns an error wrapping validate.ErrValidation when the
// config names no known variant.
func DecodeWithDefaults(stored msgpack.EncodedJSON) (ElementConfig, error) {
	defaults, err := defaultConfig(stored)
	if err != nil {
		return ElementConfig{}, err
	}
	fields, err := elementConfigFields(defaults)
	if err != nil {
		return ElementConfig{}, err
	}
	mergeFields(fields, stored)
	return decodeElementConfig(fields)
}

// defaultConfig returns the schema defaults of the variant a normalized config names,
// including those of its control state when the config carries one.
func defaultConfig(stored msgpack.EncodedJSON) (ElementConfig, error) {
	probe := msgpack.EncodedJSON{"variant": stored["variant"]}
	// The control state is the only optional field whose type declares defaults.
	if _, ok := stored["control"].(map[string]any); ok {
		probe["control"] = map[string]any{}
	}
	cfg, err := decodeElementConfig(probe)
	if err != nil {
		return ElementConfig{}, err
	}
	cfg.ApplyDefaults()
	return cfg, nil
}

// mergeFields writes every field of src into dst, merging nested objects by field.
func mergeFields(dst, src map[string]any) {
	for k, val := range src {
		if d, ok := dst[k].(map[string]any); ok {
			if s, ok := val.(map[string]any); ok {
				mergeFields(d, s)
				continue
			}
		}
		dst[k] = val
	}
}
