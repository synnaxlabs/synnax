// Copyright 2025 Synnax Labs, Inc.
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
	Name     string
	FuncName string
	Op       string
	IsComp   bool
}

type UnaryOperation struct {
	Name     string
	FuncName string
	Op       string
}

type ReductionOperation struct {
	Name     string
	FuncName string
}

var operations = []Operation{
	// Comparison operations (return uint8/bool)
	{Name: "GreaterThan", FuncName: "Gt", Op: ">", IsComp: true},
	{Name: "GreaterThanOrEqual", FuncName: "Gte", Op: ">=", IsComp: true},
	{Name: "LessThan", FuncName: "Lt", Op: "<", IsComp: true},
	{Name: "LessThanOrEqual", FuncName: "Lte", Op: "<=", IsComp: true},
	{Name: "Equal", FuncName: "Eq", Op: "==", IsComp: true},
	{Name: "NotEqual", FuncName: "Neq", Op: "!=", IsComp: true},
	// Arithmetic operations (return same type)
	{Name: "Add", FuncName: "Add", Op: "+", IsComp: false},
	{Name: "Subtract", FuncName: "Sub", Op: "-", IsComp: false},
	{Name: "Multiply", FuncName: "Mul", Op: "*", IsComp: false},
	{Name: "Divide", FuncName: "Div", Op: "/", IsComp: false},
}

// Modulo operations - uses % for integers, math.Mod for floats
var moduloIntOp = Operation{Name: "Modulo", FuncName: "Mod", Op: "%", IsComp: false}

// Logical operations only for uint8 (boolean) types
var logicalOperations = []Operation{
	{Name: "And", FuncName: "And", Op: "&", IsComp: false},
	{Name: "Or", FuncName: "Or", Op: "|", IsComp: false},
}

var reductionOperations = []ReductionOperation{
	{Name: "Avg", FuncName: "Avg"},
	{Name: "Min", FuncName: "Min"},
	{Name: "Max", FuncName: "Max"},
}

// Scalar arithmetic operations (series op scalar -> same type)
var scalarArithmeticOps = []Operation{
	{Name: "AddScalar", FuncName: "AddS", Op: "+", IsComp: false},
	{Name: "SubtractScalar", FuncName: "SubS", Op: "-", IsComp: false},
	{Name: "MultiplyScalar", FuncName: "MulS", Op: "*", IsComp: false},
	{Name: "DivideScalar", FuncName: "DivS", Op: "/", IsComp: false},
}

// Modulo scalar operation - uses % for integers, math.Mod for floats
var moduloScalarIntOp = Operation{Name: "ModuloScalar", FuncName: "ModS", Op: "%", IsComp: false}

// Scalar comparison operations (series op scalar -> uint8)
var scalarComparisonOps = []Operation{
	{Name: "GreaterThanScalar", FuncName: "GtS", Op: ">", IsComp: true},
	{Name: "GreaterThanOrEqualScalar", FuncName: "GteS", Op: ">=", IsComp: true},
	{Name: "LessThanScalar", FuncName: "LtS", Op: "<", IsComp: true},
	{Name: "LessThanOrEqualScalar", FuncName: "LteS", Op: "<=", IsComp: true},
	{Name: "EqualScalar", FuncName: "EqS", Op: "==", IsComp: true},
	{Name: "NotEqualScalar", FuncName: "NeqS", Op: "!=", IsComp: true},
}

const headerTemplate = `
package op

import (
	"math"

	"github.com/synnaxlabs/x/telem"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

var _ = math.Mod // Ensure math is used
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

func main() {
	tmpl := template.Must(template.New("funcs").Parse(funcTemplate))
	unaryTmpl := template.Must(template.New("unary").Parse(unaryFuncTemplate))
	reductionTmpl := template.Must(template.New("reduction").Parse(reductionFuncTemplate))
	scalarArithTmpl := template.Must(template.New("scalarArith").Parse(scalarArithFuncTemplate))
	scalarCompTmpl := template.Must(template.New("scalarComp").Parse(scalarCompFuncTemplate))
	floatModuloTmpl := template.Must(template.New("floatModulo").Parse(floatModuloFuncTemplate))
	floatModuloScalarTmpl := template.Must(template.New("floatModuloScalar").Parse(floatModuloScalarFuncTemplate))

	var buf strings.Builder
	buf.WriteString(headerTemplate)

	// Generate regular operations for all types
	for _, typ := range types {
		err := tmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Operations": operations,
		})
		if err != nil {
			panic(err)
		}
	}

	// Generate modulo operations - integer types use %, float types use math.Mod
	for _, typ := range types {
		if typ.IsFloat {
			// Float types use math.Mod
			err := floatModuloTmpl.Execute(&buf, map[string]interface{}{
				"Type": typ,
			})
			if err != nil {
				panic(err)
			}
		} else {
			// Integer types use %
			err := tmpl.Execute(&buf, map[string]interface{}{
				"Type":       typ,
				"Operations": []Operation{moduloIntOp},
			})
			if err != nil {
				panic(err)
			}
		}
	}

	// Generate logical operations for uint8 only
	uint8Type := TypeInfo{Name: "U8", GoType: "uint8", Size: 1, IsUnsigned: true}
	err := tmpl.Execute(&buf, map[string]interface{}{
		"Type":       uint8Type,
		"Operations": logicalOperations,
	})
	if err != nil {
		panic(err)
	}

	// Generate Not operation for uint8 only
	notOp := []UnaryOperation{{Name: "Not", FuncName: "Not", Op: "^"}}
	err = unaryTmpl.Execute(&buf, map[string]interface{}{
		"Type":     uint8Type,
		"UnaryOps": notOp,
	})
	if err != nil {
		panic(err)
	}

	// Generate Negate operation for signed and float types only
	negateOp := []UnaryOperation{{Name: "Negate", FuncName: "Neg", Op: "-"}}
	for _, typ := range types {
		if typ.IsSigned || typ.IsFloat {
			err = unaryTmpl.Execute(&buf, map[string]interface{}{
				"Type":     typ,
				"UnaryOps": negateOp,
			})
			if err != nil {
				panic(err)
			}
		}
	}

	// Generate reduction operations for all types
	for _, typ := range types {
		err := reductionTmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Reductions": reductionOperations,
		})
		if err != nil {
			panic(err)
		}
	}

	// Generate scalar arithmetic operations for all types
	for _, typ := range types {
		err := scalarArithTmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Operations": scalarArithmeticOps,
		})
		if err != nil {
			panic(err)
		}
	}

	// Generate modulo scalar operations - integer types use %, float types use math.Mod
	for _, typ := range types {
		if typ.IsFloat {
			// Float types use math.Mod
			err := floatModuloScalarTmpl.Execute(&buf, map[string]interface{}{
				"Type": typ,
			})
			if err != nil {
				panic(err)
			}
		} else {
			// Integer types use %
			err := scalarArithTmpl.Execute(&buf, map[string]interface{}{
				"Type":       typ,
				"Operations": []Operation{moduloScalarIntOp},
			})
			if err != nil {
				panic(err)
			}
		}
	}

	// Generate scalar comparison operations for all types
	for _, typ := range types {
		err := scalarCompTmpl.Execute(&buf, map[string]interface{}{
			"Type":       typ,
			"Operations": scalarComparisonOps,
		})
		if err != nil {
			panic(err)
		}
	}

	output := buf.String()
	err = os.WriteFile("op.go", []byte(output), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("Generated op.go successfully")
}
