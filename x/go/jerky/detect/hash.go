// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package detect provides change detection for jerky schema evolution.
package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/synnaxlabs/x/jerky/parse"
	"github.com/synnaxlabs/x/jerky/state"
)

// ComputeStructHash computes a hash of a struct's fields for change detection.
// The hash is deterministic based on field names, types, and declaration order.
func ComputeStructHash(parsed parse.ParsedStruct) string {
	h := sha256.New()

	// Include struct name
	h.Write([]byte(parsed.Name))
	h.Write([]byte{0})

	// Include fields in declaration order
	for _, field := range parsed.Fields {
		h.Write([]byte(field.Name))
		h.Write([]byte{0})
		h.Write([]byte(field.GoType.String()))
		h.Write([]byte{0})
		h.Write([]byte(field.Tags.JSON))
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil))[:12]
}

// ComputeStructHashFromFields computes a hash from a map of field info (for state file).
func ComputeStructHashFromFields(typeName string, fields map[string]state.FieldInfo, fieldOrder []string) string {
	h := sha256.New()

	h.Write([]byte(typeName))
	h.Write([]byte{0})

	// Use explicit field order if provided
	if len(fieldOrder) > 0 {
		for _, name := range fieldOrder {
			field, ok := fields[name]
			if !ok {
				continue
			}
			h.Write([]byte(name))
			h.Write([]byte{0})
			h.Write([]byte(field.Type))
			h.Write([]byte{0})
			if tags, ok := field.Tags["json"]; ok {
				h.Write([]byte(tags))
			}
			h.Write([]byte{0})
		}
	} else {
		// Fall back to sorted keys for determinism
		sortedNames := make([]string, 0, len(fields))
		for name := range fields {
			sortedNames = append(sortedNames, name)
		}
		sort.Strings(sortedNames)

		for _, name := range sortedNames {
			field := fields[name]
			h.Write([]byte(name))
			h.Write([]byte{0})
			h.Write([]byte(field.Type))
			h.Write([]byte{0})
			if tags, ok := field.Tags["json"]; ok {
				h.Write([]byte(tags))
			}
			h.Write([]byte{0})
		}
	}

	return hex.EncodeToString(h.Sum(nil))[:12]
}

// ComputeCompositeHash computes a hash that includes both the struct hash and dependency hashes.
// This is used to detect when a type needs a new version due to changes in its dependencies.
func ComputeCompositeHash(structHash string, depHashes map[string]string) string {
	h := sha256.New()

	// Include struct hash
	h.Write([]byte(structHash))
	h.Write([]byte{0})

	// Include dependency hashes in sorted order for determinism
	if len(depHashes) > 0 {
		depKeys := make([]string, 0, len(depHashes))
		for k := range depHashes {
			depKeys = append(depKeys, k)
		}
		sort.Strings(depKeys)

		for _, k := range depKeys {
			h.Write([]byte(k))
			h.Write([]byte{0})
			h.Write([]byte(depHashes[k]))
			h.Write([]byte{0})
		}
	}

	return hex.EncodeToString(h.Sum(nil))[:12]
}
