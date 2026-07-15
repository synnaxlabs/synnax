// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

// String implements fmt.Stringer.
func (r ResourceType) String() string { return string(r) }

// String returns a string representation of the ID in the format "Type:Key".
func (i ID) String() string { return string(i.Type) + ":" + i.Key }

// IsZero returns true if the ID is the zero value (both Key and Type are empty).
func (i ID) IsZero() bool { return i.Key == "" && i.Type == "" }

// IsType returns true if the ID represents a type identifier (has a Type but no Key).
// Type IDs are used to identify resource types rather than specific resource instances.
func (i ID) IsType() bool { return i.Type != "" && i.Key == "" }
