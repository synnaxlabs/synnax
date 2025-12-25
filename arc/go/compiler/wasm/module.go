// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"bytes"

	"github.com/synnaxlabs/x/binary"
)

// FunctionType represents a function signature
type FunctionType struct {
	Params  []ValueType
	Results []ValueType
}

// Function represents a WASM function
type Function struct {
	TypeIdx uint32
	Locals  []ValueType
	Body    []byte
}

// Import represents an imported function
type Import struct {
	Module  string
	Name    string
	TypeIdx uint32
}

// Export represents an exported item
type Export struct {
	Name  string
	Kind  ExportKind
	Index uint32
}

// DataSegment represents a data segment to be placed in linear memory
type DataSegment struct {
	Offset uint32
	Bytes  []byte
}

// Module represents a complete WASM module
type Module struct {
	types      []FunctionType
	imports    []Import
	functions  []Function
	exports    []Export
	data       []DataSegment
	dataOffset uint32
	memory     bool
	buf        bytes.Buffer
}

// NewModule creates a new WASM module
func NewModule() *Module {
	return &Module{
		types:     make([]FunctionType, 0),
		imports:   make([]Import, 0),
		functions: make([]Function, 0),
		exports:   make([]Export, 0),
		data:      make([]DataSegment, 0),
	}
}

// AddType adds a function type and returns its index
func (m *Module) AddType(ft FunctionType) uint32 {
	for i, existing := range m.types {
		if typesEqual(existing, ft) {
			return uint32(i)
		}
	}
	idx := uint32(len(m.types))
	m.types = append(m.types, ft)
	return idx
}

// AddImport adds an imported function
func (m *Module) AddImport(module, name string, ft FunctionType) uint32 {
	typeIdx := m.AddType(ft)
	m.imports = append(m.imports, Import{
		Module:  module,
		Name:    name,
		TypeIdx: typeIdx,
	})
	return uint32(len(m.imports) - 1)
}

// AddFunction adds a function to the module and returns its index
func (m *Module) AddFunction(typeIdx uint32, locals []ValueType, body []byte) uint32 {
	// Function index is imports + local functions
	idx := uint32(len(m.imports) + len(m.functions))
	m.functions = append(m.functions, Function{
		TypeIdx: typeIdx,
		Locals:  locals,
		Body:    body,
	})
	return idx
}

// AddExport adds an export to the module
func (m *Module) AddExport(name string, kind ExportKind, index uint32) {
	m.exports = append(m.exports, Export{
		Name:  name,
		Kind:  kind,
		Index: index,
	})
}

// Debug returns debug info about the module
func (m *Module) Debug() (types, functions, exports int) {
	return len(m.types), len(m.functions), len(m.exports)
}

// ImportNames returns the names of all imported functions in order.
func (m *Module) ImportNames() []string {
	names := make([]string, len(m.imports))
	for i, imp := range m.imports {
		names[i] = imp.Name
	}
	return names
}

// EnableMemory enables memory for the module
func (m *Module) EnableMemory() {
	m.memory = true
}

// AddData adds a data segment to the module and returns its offset in linear memory.
// The data will be placed at the current dataOffset, which is then incremented.
func (m *Module) AddData(bytes []byte) uint32 {
	offset := m.dataOffset
	m.data = append(m.data, DataSegment{
		Offset: offset,
		Bytes:  bytes,
	})
	m.dataOffset += uint32(len(bytes))
	return offset
}

// Generate generates the WASM binary
func (m *Module) Generate() []byte {
	m.buf.Reset()
	m.buf.Write(MagicNumber)
	m.buf.Write(Version)
	if len(m.types) > 0 {
		m.writeTypeSection()
	}
	if len(m.imports) > 0 {
		m.writeImportSection()
	}
	if len(m.functions) > 0 {
		m.writeFunctionSection()
	}
	if m.memory {
		m.writeMemorySection()
	}
	if len(m.exports) > 0 {
		m.writeExportSection()
	}
	if len(m.functions) > 0 {
		m.writeCodeSection()
	}
	if len(m.data) > 0 {
		m.writeDataSection()
	}
	return m.buf.Bytes()
}

func (m *Module) writeTypeSection() {
	var section bytes.Buffer
	binary.WriteLEB128Unsigned(&section, uint64(len(m.types)))
	for _, ft := range m.types {
		section.WriteByte(byte(FuncType))
		binary.WriteLEB128Unsigned(&section, uint64(len(ft.Params)))
		for _, param := range ft.Params {
			section.WriteByte(byte(param))
		}
		binary.WriteLEB128Unsigned(&section, uint64(len(ft.Results)))
		for _, result := range ft.Results {
			section.WriteByte(byte(result))
		}
	}

	m.writeSection(SectionType, section.Bytes())
}

func (m *Module) writeImportSection() {
	var section bytes.Buffer

	binary.WriteLEB128Unsigned(&section, uint64(len(m.imports)))

	for _, imp := range m.imports {
		binary.WriteLEB128Unsigned(&section, uint64(len(imp.Module)))
		section.WriteString(imp.Module)
		binary.WriteLEB128Unsigned(&section, uint64(len(imp.Name)))
		section.WriteString(imp.Name)
		section.WriteByte(byte(ExportFunc))
		binary.WriteLEB128Unsigned(&section, uint64(imp.TypeIdx))
	}

	m.writeSection(SectionImport, section.Bytes())
}

func (m *Module) writeFunctionSection() {
	var section bytes.Buffer
	binary.WriteLEB128Unsigned(&section, uint64(len(m.functions)))
	for _, fn := range m.functions {
		binary.WriteLEB128Unsigned(&section, uint64(fn.TypeIdx))
	}
	m.writeSection(SectionFunc, section.Bytes())
}

func (m *Module) writeMemorySection() {
	var section bytes.Buffer
	// Number of memories (1)
	section.WriteByte(1)
	// Memory limits (min 1 page, no max)
	section.WriteByte(0)                    // no max
	binary.WriteLEB128Unsigned(&section, 1) // min 1 page
	m.writeSection(SectionMemory, section.Bytes())
}

func (m *Module) writeExportSection() {
	var section bytes.Buffer
	binary.WriteLEB128Unsigned(&section, uint64(len(m.exports)))
	for _, exp := range m.exports {
		binary.WriteLEB128Unsigned(&section, uint64(len(exp.Name)))
		section.WriteString(exp.Name)
		section.WriteByte(byte(exp.Kind))
		binary.WriteLEB128Unsigned(&section, uint64(exp.Index))
	}
	m.writeSection(SectionExport, section.Bytes())
}

func (m *Module) writeCodeSection() {
	var section bytes.Buffer
	binary.WriteLEB128Unsigned(&section, uint64(len(m.functions)))
	for _, fn := range m.functions {
		var code bytes.Buffer
		// Write local declarations
		if len(fn.Locals) > 0 {
			// Group locals by type for efficiency
			grouped := groupLocalsByType(fn.Locals)
			binary.WriteLEB128Unsigned(&code, uint64(len(grouped)))
			for _, group := range grouped {
				binary.WriteLEB128Unsigned(&code, uint64(group.count))
				code.WriteByte(byte(group.typ))
			}
		} else {
			binary.WriteLEB128Unsigned(&code, 0) // no locals
		}
		code.Write(fn.Body)
		code.WriteByte(byte(OpEnd))
		binary.WriteLEB128Unsigned(&section, uint64(code.Len()))
		section.Write(code.Bytes())
	}
	m.writeSection(SectionCode, section.Bytes())
}

func (m *Module) writeDataSection() {
	var section bytes.Buffer
	// Number of data segments
	binary.WriteLEB128Unsigned(&section, uint64(len(m.data)))
	for _, seg := range m.data {
		// Memory index (always 0 for single memory)
		section.WriteByte(0)
		// Offset expression: i32.const <offset>, end
		section.WriteByte(byte(OpI32Const))
		binary.WriteLEB128Signed(&section, int64(seg.Offset))
		section.WriteByte(byte(OpEnd))
		// Data length and bytes
		binary.WriteLEB128Unsigned(&section, uint64(len(seg.Bytes)))
		section.Write(seg.Bytes)
	}
	m.writeSection(SectionData, section.Bytes())
}

func (m *Module) writeSection(sectionType byte, data []byte) {
	m.buf.WriteByte(sectionType)
	binary.WriteLEB128Unsigned(&m.buf, uint64(len(data)))
	m.buf.Write(data)
}

func typesEqual(a, b FunctionType) bool {
	if len(a.Params) != len(b.Params) || len(a.Results) != len(b.Results) {
		return false
	}
	for i := range a.Params {
		if a.Params[i] != b.Params[i] {
			return false
		}
	}
	for i := range a.Results {
		if a.Results[i] != b.Results[i] {
			return false
		}
	}
	return true
}

type localGroup struct {
	count uint32
	typ   ValueType
}

// groupLocalsByType groups consecutive locals of the same type
func groupLocalsByType(locals []ValueType) []localGroup {
	if len(locals) == 0 {
		return nil
	}
	var (
		groups       []localGroup
		currentType  = locals[0]
		currentCount = uint32(1)
	)
	for i := 1; i < len(locals); i++ {
		if locals[i] == currentType {
			currentCount++
		} else {
			groups = append(groups, localGroup{count: currentCount, typ: currentType})
			currentType = locals[i]
			currentCount = 1
		}
	}
	groups = append(groups, localGroup{count: currentCount, typ: currentType})
	return groups
}
