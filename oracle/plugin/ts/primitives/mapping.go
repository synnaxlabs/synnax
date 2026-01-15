// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package primitives provides TypeScript-specific primitive type mappings for Oracle code generation.
package primitives

import "github.com/synnaxlabs/oracle/plugin/primitives"

// zodMapping contains TypeScript-specific primitive type mappings for Zod schemas.
var zodMapping = map[string]primitives.Mapping{
	"uuid":    {TargetType: "z.string().uuid()"},
	"string":  {TargetType: "z.string()"},
	"bool":    {TargetType: "z.boolean()"},
	"int8":    {TargetType: "z.number().int()"},
	"int16":   {TargetType: "z.number().int()"},
	"int32":   {TargetType: "z.number().int()"},
	"int64":   {TargetType: "z.bigint()"},
	"uint8":   {TargetType: "z.number().int().nonnegative()"},
	"uint12":  {TargetType: "z.number().int().nonnegative()"},
	"uint16":  {TargetType: "z.number().int().nonnegative()"},
	"uint20":  {TargetType: "z.number().int().nonnegative()"},
	"uint32":  {TargetType: "z.number().int().nonnegative()"},
	"uint64":  {TargetType: "z.bigint().nonnegative()"},
	"float32": {TargetType: "z.number()"},
	"float64": {TargetType: "z.number()"},
	"json":    {TargetType: "z.record(z.string(), z.unknown())"},
	"bytes":   {TargetType: "z.instanceof(Uint8Array)"},
	"any":     {TargetType: "z.unknown()"},
}

// typeMapping contains TypeScript-specific primitive type mappings for type annotations.
var typeMapping = map[string]primitives.Mapping{
	"uuid":    {TargetType: "string"},
	"string":  {TargetType: "string"},
	"bool":    {TargetType: "boolean"},
	"int8":    {TargetType: "number"},
	"int16":   {TargetType: "number"},
	"int32":   {TargetType: "number"},
	"int64":   {TargetType: "bigint"},
	"uint8":   {TargetType: "number"},
	"uint12":  {TargetType: "number"},
	"uint16":  {TargetType: "number"},
	"uint20":  {TargetType: "number"},
	"uint32":  {TargetType: "number"},
	"uint64":  {TargetType: "bigint"},
	"float32": {TargetType: "number"},
	"float64": {TargetType: "number"},
	"json":    {TargetType: "Record<string, unknown>"},
	"bytes":   {TargetType: "Uint8Array"},
	"any":     {TargetType: "unknown"},
}
