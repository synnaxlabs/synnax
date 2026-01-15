// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import "github.com/synnaxlabs/oracle/resolution"

type Holder interface {
	GetDomains() map[string]resolution.Domain
}

type TypeHolder struct{ T resolution.Type }

func (h TypeHolder) GetDomains() map[string]resolution.Domain { return h.T.Domains }

type FieldHolder struct{ F resolution.Field }

func (h FieldHolder) GetDomains() map[string]resolution.Domain { return h.F.Domains }

func GetString(h Holder, domainName, exprName string) string {
	domain, ok := h.GetDomains()[domainName]
	if !ok {
		return ""
	}
	expr, ok := domain.Expressions.Find(exprName)
	if !ok || len(expr.Values) == 0 {
		return ""
	}
	if v := expr.Values[0].StringValue; v != "" {
		return v
	}
	return expr.Values[0].IdentValue
}

func GetStringFromType(t resolution.Type, domainName, exprName string) string {
	return GetString(TypeHolder{t}, domainName, exprName)
}

func GetStringFromField(f resolution.Field, domainName, exprName string) string {
	return GetString(FieldHolder{f}, domainName, exprName)
}

// GetAllStrings collects string values from ALL expressions with the given name.
// Unlike GetStrings which only finds the first matching expression, this function
// finds all expressions with the name and collects all their string values.
// This is useful for directives that can appear multiple times like @go field.
func GetAllStrings(h Holder, domainName, exprName string) []string {
	domain, ok := h.GetDomains()[domainName]
	if !ok {
		return nil
	}
	var result []string
	for _, expr := range domain.Expressions {
		if expr.Name != exprName {
			continue
		}
		for _, v := range expr.Values {
			if v.StringValue != "" {
				result = append(result, v.StringValue)
			} else if v.IdentValue != "" {
				result = append(result, v.IdentValue)
			}
		}
	}
	return result
}

func GetAllStringsFromType(t resolution.Type, domainName, exprName string) []string {
	return GetAllStrings(TypeHolder{t}, domainName, exprName)
}

func HasExpr(h Holder, domainName, exprName string) bool {
	domain, ok := h.GetDomains()[domainName]
	if !ok {
		return false
	}
	_, ok = domain.Expressions.Find(exprName)
	return ok
}

func HasExprFromType(t resolution.Type, domainName, exprName string) bool {
	return HasExpr(TypeHolder{t}, domainName, exprName)
}

func GetName(t resolution.Type, domainName string) string {
	if override := GetStringFromType(t, domainName, "name"); override != "" {
		return override
	}
	return t.Name
}

func GetType(t resolution.Type, domainName string) string {
	return GetStringFromType(t, domainName, "type")
}

func GetFieldType(f resolution.Field, domainName string) string {
	return GetStringFromField(f, domainName, "type")
}
