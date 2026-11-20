// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"strconv"

	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// lessThanCheck is the template view of an lt rule: a check on an object schema.
type lessThanCheck struct {
	Field   string
	Other   string
	Message string
}

// lessThanChecks returns the lt checks of fields, one per field name. A schema made by
// extending a parent schema keeps that parent's checks, so the fields in carried get
// none.
func lessThanChecks(
	fields []resolution.Field,
	carried set.Set[string],
	data *templateData,
) []lessThanCheck {
	var checks []lessThanCheck
	seen := carried.Copy()
	for _, f := range fields {
		lt := validation.Parse(f.Domains["validate"]).LessThan
		if lt == nil || seen.Contains(f.Name) {
			continue
		}
		seen.Add(f.Name)
		message := lt.Message
		if message == "" {
			message = "must be less than " + lt.Field
		}
		checks = append(checks, lessThanCheck{
			Field:   FieldName(f.Name),
			Other:   FieldName(lt.Field),
			Message: strconv.Quote(message),
		})
	}
	if len(checks) > 0 {
		addXImport(data, xImport{name: "zod", submodule: "zod"})
	}
	return checks
}

// fieldNames returns the names of the unified fields of the type that ref names.
func fieldNames(ref resolution.TypeRef, table *resolution.Table) set.Set[string] {
	names := set.New[string]()
	if t, _, ok := resolution.Dealias(ref, table); ok {
		for _, f := range resolution.UnifiedFields(t, table) {
			names.Add(f.Name)
		}
	}
	return names
}
