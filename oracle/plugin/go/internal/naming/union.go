// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package naming

import "github.com/synnaxlabs/oracle/resolution"

// variantField is the field of a Go union that holds the active variant. A method of
// the same name cannot exist.
const variantField = "Variant"

// UnionTagMethod returns the method of the Go union that returns the tag of the active
// variant. It returns false when the union has no such method.
func UnionTagMethod(form resolution.UnionForm) (string, bool) {
	method := ToPascalCase(form.Discriminator)
	return method, method != variantField
}
