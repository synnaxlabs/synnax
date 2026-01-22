// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/samber/lo"
)

type TypeInfo struct {
	Name       string
	GoType     string
	Size       int
	IsFloat    bool
	IsSigned   bool
	IsUnsigned bool
}

var types = []TypeInfo{
	{Name: "F64", GoType: "float64", Size: 8, IsFloat: true},
	{Name: "F32", GoType: "float32", Size: 4, IsFloat: true},
	{Name: "I64", GoType: "int64", Size: 8, IsSigned: true},
	{Name: "I32", GoType: "int32", Size: 4, IsSigned: true},
	{Name: "I16", GoType: "int16", Size: 2, IsSigned: true},
	{Name: "I8", GoType: "int8", Size: 1, IsSigned: true},
	{Name: "U64", GoType: "uint64", Size: 8, IsUnsigned: true},
	{Name: "U32", GoType: "uint32", Size: 4, IsUnsigned: true},
	{Name: "U16", GoType: "uint16", Size: 2, IsUnsigned: true},
	{Name: "U8", GoType: "uint8", Size: 1, IsUnsigned: true},
}

type Operation struct {
	Name   string
	Op     string
	IsComp bool
}

type UnaryOperation struct {
	Name string
	Op   string
}

type ReductionOperation struct {
	Name string
}

var operations = []Operation{
	// Comparison operations (return uint8/bool)
	{Name: "GreaterThan", Op: ">", IsComp: true},
	{Name: "GreaterThanOrEqual", Op: ">=", IsComp: true},
	{Name: "LessThan", Op: "<", IsComp: true},
	{Name: "LessThanOrEqual", Op: "<=", IsComp: true},
	{Name: "Equal", Op: "==", IsComp: true},
	{Name: "NotEqual", Op: "!=", IsComp: true},
	// Arithmetic operations (return same type)
	{Name: "Add", Op: "+"},
	{Name: "Subtract", Op: "-"},
	{Name: "Multiply", Op: "*"},
	{Name: "Divide", Op: "/"},
}

// Modulo operations - uses % for integers, math.Mod for floats
var moduloIntOp = Operation{Name: "Modulo", Op: "%"}

// Logical operations only for uint8 (boolean) types
var logicalOperations = []Operation{
	{Name: "And", Op: "&"},
	{Name: "Or", Op: "|"},
}

var reductionOperations = []ReductionOperation{
	{Name: "Avg"},
	{Name: "Min"},
	{Name: "Max"},
}

// Scalar arithmetic operations (series op scalar -> same type)
var scalarArithmeticOps = []Operation{
	{Name: "AddScalar", Op: "+"},
	{Name: "SubtractScalar", Op: "-"},
	{Name: "MultiplyScalar", Op: "*"},
	{Name: "DivideScalar", Op: "/"},
}

// Reverse scalar arithmetic operations (scalar op series -> same type)
// Used for non-commutative operations where scalar is on the left
var reverseScalarArithmeticOps = []Operation{
	{Name: "ReverseSubtractScalar", Op: "-"}, // scalar - series
	{Name: "ReverseDivideScalar", Op: "/"},   // scalar / series
}

// Reverse modulo for integers only (floats use math.Mod, handled separately)
var reverseModuloScalarIntOp = Operation{Name: "ReverseModuloScalar", Op: "%"}

// Modulo scalar operation - uses % for integers, math.Mod for floats
var moduloScalarIntOp = Operation{Name: "ModuloScalar", Op: "%"}

// Scalar comparison operations (series op scalar -> uint8)
var scalarComparisonOps = []Operation{
	{Name: "GreaterThanScalar", Op: ">", IsComp: true},
	{Name: "GreaterThanOrEqualScalar", Op: ">=", IsComp: true},
	{Name: "LessThanScalar", Op: "<", IsComp: true},
	{Name: "LessThanOrEqualScalar", Op: "<=", IsComp: true},
	{Name: "EqualScalar", Op: "==", IsComp: true},
	{Name: "NotEqualScalar", Op: "!=", IsComp: true},
}

const headerTemplate = `
package op

import (
	"math"

	"github.com/synnaxlabs/x/telem"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

// Blank identifier ensures math import is used even when no float types are generated,
// since only float modulo operations use math.Mod.
var _ = math.Mod
`

const unaryFuncTemplate = `{{range $.UnaryOps}}
func {{.Name}}{{$.Type.Name}}(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](input.Data)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	for i := int64(0); i < inputLen; i++ {
		outData[i] = {{.Op}}inData[i]
	}
}
{{end}}`

const reductionFuncTemplate = `{{range $.Reductions}}
func {{.Name}}{{$.Type.Name}}(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](input.Data)

	{{if eq .Name "Avg"}}
	// Compute sum of new input samples
	var newSum {{$.Type.GoType}}
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / {{$.Type.GoType}}(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg * {{$.Type.GoType}}(prevCount) + newSum) / {{$.Type.GoType}}(totalCount)
	}

	return prevCount + inputLen
	{{else if eq .Name "Min"}}
	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen
	{{else if eq .Name "Max"}}
	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen
	{{end}}
}
{{end}}`

const funcTemplate = `{{range $.Operations}}{{if .IsComp}}
func {{.Name}}{{$.Type.Name}}(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast {{$.Type.GoType}}
	if lhsLen > 0 {
		lhsLast = lhsData[lhsLen-1]
	}
	if rhsLen > 0 {
		rhsLast = rhsData[rhsLen-1]
	}

	for i := int64(0); i < maxLen; i++ {
		lhsVal := lhsLast
		if i < lhsLen {
			lhsVal = lhsData[i]
			lhsLast = lhsVal
		}
		rhsVal := rhsLast
		if i < rhsLen {
			rhsVal = rhsData[i]
			rhsLast = rhsVal
		}
		if lhsVal {{.Op}} rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}
{{else}}
func {{.Name}}{{$.Type.Name}}(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](rhs.Data)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	var lhsLast, rhsLast {{$.Type.GoType}}
	if lhsLen > 0 {
		lhsLast = lhsData[lhsLen-1]
	}
	if rhsLen > 0 {
		rhsLast = rhsData[rhsLen-1]
	}

	for i := int64(0); i < maxLen; i++ {
		lhsVal := lhsLast
		if i < lhsLen {
			lhsVal = lhsData[i]
			lhsLast = lhsVal
		}
		rhsVal := rhsLast
		if i < rhsLen {
			rhsVal = rhsData[i]
			rhsLast = rhsVal
		}
		outData[i] = lhsVal {{.Op}} rhsVal
	}
}
{{end}}{{end}}`

// Template for scalar arithmetic operations (series op scalar -> same type)
const scalarArithFuncTemplate = `{{range $.Operations}}
func {{.Name}}{{$.Type.Name}}(series telem.Series, scalar {{$.Type.GoType}}, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](series.Data)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] {{.Op}} scalar
	}
}
{{end}}`

// Template for scalar comparison operations (series op scalar -> uint8)
const scalarCompFuncTemplate = `{{range $.Operations}}
func {{.Name}}{{$.Type.Name}}(series telem.Series, scalar {{$.Type.GoType}}, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] {{.Op}} scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}
{{end}}`

// Template for reverse scalar arithmetic operations (scalar op series -> same type)
// Note: scalar is on the LEFT side of the operation
const reverseScalarArithFuncTemplate = `{{range $.Operations}}
func {{.Name}}{{$.Type.Name}}(series telem.Series, scalar {{$.Type.GoType}}, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](series.Data)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar {{.Op}} inData[i]
	}
}
{{end}}`

// Template for float modulo (binary) - uses math.Mod
const floatModuloFuncTemplate = `
func Modulo{{$.Type.Name}}(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](rhs.Data)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	var lhsLast, rhsLast {{$.Type.GoType}}
	if lhsLen > 0 {
		lhsLast = lhsData[lhsLen-1]
	}
	if rhsLen > 0 {
		rhsLast = rhsData[rhsLen-1]
	}

	for i := int64(0); i < maxLen; i++ {
		lhsVal := lhsLast
		if i < lhsLen {
			lhsVal = lhsData[i]
			lhsLast = lhsVal
		}
		rhsVal := rhsLast
		if i < rhsLen {
			rhsVal = rhsData[i]
			rhsLast = rhsVal
		}
		outData[i] = {{$.Type.GoType}}(math.Mod(float64(lhsVal), float64(rhsVal)))
	}
}
`

// Template for float modulo scalar - uses math.Mod
const floatModuloScalarFuncTemplate = `
func ModuloScalar{{$.Type.Name}}(series telem.Series, scalar {{$.Type.GoType}}, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](series.Data)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = {{$.Type.GoType}}(math.Mod(float64(inData[i]), float64(scalar)))
	}
}
`

// Template for float reverse modulo scalar - uses math.Mod with scalar on left
const floatReverseModuloScalarFuncTemplate = `
func ReverseModuloScalar{{$.Type.Name}}(series telem.Series, scalar {{$.Type.GoType}}, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](series.Data)
	outData := xunsafe.CastSlice[uint8, {{$.Type.GoType}}](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = {{$.Type.GoType}}(math.Mod(float64(scalar), float64(inData[i])))
	}
}
`

func main() {
	tmpl := template.Must(template.New("funcs").Parse(funcTemplate))
	unaryTmpl := template.Must(template.New("unary").Parse(unaryFuncTemplate))
	reductionTmpl := template.Must(template.New("reduction").Parse(reductionFuncTemplate))
	scalarArithTmpl := template.Must(template.New("scalarArith").Parse(scalarArithFuncTemplate))
	reverseScalarArithTmpl := template.Must(template.New("reverseScalarArith").Parse(reverseScalarArithFuncTemplate))
	scalarCompTmpl := template.Must(template.New("scalarComp").Parse(scalarCompFuncTemplate))
	floatModuloTmpl := template.Must(template.New("floatModulo").Parse(floatModuloFuncTemplate))
	floatModuloScalarTmpl := template.Must(template.New("floatModuloScalar").Parse(floatModuloScalarFuncTemplate))
	floatReverseModuloScalarTmpl := template.Must(template.New("floatReverseModuloScalar").Parse(floatReverseModuloScalarFuncTemplate))

	var buf strings.Builder
	buf.WriteString(headerTemplate)

	// Generate regular operations for all types
	for _, typ := range types {
		lo.Must0(tmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Operations": operations,
		}))
	}

	// Generate modulo operations - integer types use %, float types use math.Mod
	for _, typ := range types {
		if typ.IsFloat {
			// Float types use math.Mod
			lo.Must0(floatModuloTmpl.Execute(&buf, map[string]interface{}{
				"Type": typ,
			}))
		} else {
			// Integer types use %
			lo.Must0(tmpl.Execute(&buf, map[string]interface{}{
				"Type":       typ,
				"Operations": []Operation{moduloIntOp},
			}))
		}
	}

	// Generate logical operations for uint8 only
	uint8Type := TypeInfo{Name: "U8", GoType: "uint8", Size: 1, IsUnsigned: true}
	lo.Must0(tmpl.Execute(&buf, map[string]interface{}{
		"Type":       uint8Type,
		"Operations": logicalOperations,
	}))

	// Generate Not operation for uint8 only
	notOp := []UnaryOperation{{Name: "Not", Op: "^"}}
	lo.Must0(unaryTmpl.Execute(&buf, map[string]interface{}{
		"Type":     uint8Type,
		"UnaryOps": notOp,
	}))

	// Generate Negate operation for signed and float types only
	negateOp := []UnaryOperation{{Name: "Negate", Op: "-"}}
	for _, typ := range types {
		if typ.IsSigned || typ.IsFloat {
			lo.Must0(unaryTmpl.Execute(&buf, map[string]interface{}{
				"Type":     typ,
				"UnaryOps": negateOp,
			}))
		}
	}

	// Generate reduction operations for all types
	for _, typ := range types {
		lo.Must0(reductionTmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Reductions": reductionOperations,
		}))
	}

	// Generate scalar arithmetic operations for all types
	for _, typ := range types {
		lo.Must0(scalarArithTmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Operations": scalarArithmeticOps,
		}))
	}

	// Generate reverse scalar arithmetic operations for all types
	for _, typ := range types {
		lo.Must0(reverseScalarArithTmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Operations": reverseScalarArithmeticOps,
		}))
	}

	for _, typ := range types {
		if typ.IsFloat {
			lo.Must0(floatReverseModuloScalarTmpl.Execute(&buf, map[string]interface{}{
				"Type": typ,
			}))
		} else {
			lo.Must0(reverseScalarArithTmpl.Execute(&buf, map[string]interface{}{
				"Type":       typ,
				"Operations": []Operation{reverseModuloScalarIntOp},
			}))
		}
	}

	// Generate modulo scalar operations - integer types use %, float types use math.Mod
	for _, typ := range types {
		if typ.IsFloat {
			lo.Must0(floatModuloScalarTmpl.Execute(&buf, map[string]interface{}{
				"Type": typ,
			}))
		} else {
			lo.Must0(scalarArithTmpl.Execute(&buf, map[string]interface{}{
				"Type":       typ,
				"Operations": []Operation{moduloScalarIntOp},
			}))
		}
	}

	// Generate scalar comparison operations for all types
	for _, typ := range types {
		lo.Must0(scalarCompTmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Operations": scalarComparisonOps,
		}))
	}

	output := buf.String()
	lo.Must0(os.WriteFile("op.go", []byte(output), 0644))

	fmt.Println("Generated op.go successfully")
}
