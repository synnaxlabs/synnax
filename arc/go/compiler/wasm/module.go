// Copyright 2026 Synnax Labs, Inc.
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
)

// FunctionType represents a function signature
type FunctionType struct {
	Params  []ValueType
	Results []ValueType
}

// function represents a WASM function
type function struct {
	Locals  []ValueType
	Body    []byte
	TypeIdx uint32
}

// Import represents an imported function
type Import struct {
	Module  string
	Name    string
	TypeIdx uint32
}

// export represents an exported item
type export struct {
	Name  string
	Kind  ExportKind
	Index uint32
}

// dataSegment represents a data segment to be placed in linear memory
type dataSegment struct {
	Bytes  []byte
	Offset uint32
}

// Module represents a complete WASM module
type Module struct {
	types      []FunctionType
	imports    []Import
	functions  []function
	exports    []export
	data       []dataSegment
	buf        bytes.Buffer
	dataOffset uint32
	memory     bool
}

// NewModule creates a new WASM module
func NewModule() *Module {
	return &Module{
		types:     make([]FunctionType, 0),
		imports:   make([]Import, 0),
		functions: make([]function, 0),
		exports:   make([]export, 0),
		data:      make([]dataSegment, 0),
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
	m.functions = append(m.functions, function{
		TypeIdx: typeIdx,
		Locals:  locals,
		Body:    body,
	})
	return idx
}

// AddExport adds an export to the module
func (m *Module) AddExport(name string, kind ExportKind, index uint32) {
	m.exports = append(m.exports, export{
		Name:  name,
		Kind:  kind,
		Index: index,
	})
}

// ImportCount returns the number of imported functions in the module.
// This is used to calculate function indices for call instructions.
func (m *Module) ImportCount() uint32 {
	return uint32(len(m.imports))
}

// EnableMemory enables memory for the module
func (m *Module) EnableMemory() {
	m.memory = true
}

// AddData adds a data segment to the module and returns its offset in linear memory.
// The data will be placed at the current dataOffset, which is then incremented.
func (m *Module) AddData(bytes []byte) uint32 {
	offset := m.dataOffset
	m.data = append(m.data, dataSegment{
		Offset: offset,
		Bytes:  bytes,
	})
	m.dataOffset += uint32(len(bytes))
	return offset
}

// Generate generates the WASM binary
func (m *Module) Generate() []byte {
	m.buf.Reset()
	m.buf.Write(magicNumber)
	m.buf.Write(version)
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
	writeUnsignedLEB128(&section, uint64(len(m.types)))
	for _, ft := range m.types {
		section.WriteByte(byte(funcType))
		writeUnsignedLEB128(&section, uint64(len(ft.Params)))
		for _, param := range ft.Params {
			section.WriteByte(byte(param))
		}
		writeUnsignedLEB128(&section, uint64(len(ft.Results)))
		for _, result := range ft.Results {
			section.WriteByte(byte(result))
		}
	}

	m.writeSection(sectionType, section.Bytes())
}

func (m *Module) writeImportSection() {
	var section bytes.Buffer

	writeUnsignedLEB128(&section, uint64(len(m.imports)))

	for _, imp := range m.imports {
		writeUnsignedLEB128(&section, uint64(len(imp.Module)))
		section.WriteString(imp.Module)
		writeUnsignedLEB128(&section, uint64(len(imp.Name)))
		section.WriteString(imp.Name)
		section.WriteByte(byte(ExportKindFunc))
		writeUnsignedLEB128(&section, uint64(imp.TypeIdx))
	}

	m.writeSection(sectionImport, section.Bytes())
}

func (m *Module) writeFunctionSection() {
	var section bytes.Buffer
	writeUnsignedLEB128(&section, uint64(len(m.functions)))
	for _, fn := range m.functions {
		writeUnsignedLEB128(&section, uint64(fn.TypeIdx))
	}
	m.writeSection(sectionFunc, section.Bytes())
}

// MaxMemoryPages is the most 64 KiB pages that the linear memory of a module can grow
// to.
const MaxMemoryPages = 4096

func (m *Module) writeMemorySection() {
	var section bytes.Buffer
	// Number of memories (1)
	section.WriteByte(1)
	section.WriteByte(1) // limits carry a max
	writeUnsignedLEB128(&section, 1)
	writeUnsignedLEB128(&section, MaxMemoryPages)
	m.writeSection(sectionMemory, section.Bytes())
}

func (m *Module) writeExportSection() {
	var section bytes.Buffer
	writeUnsignedLEB128(&section, uint64(len(m.exports)))
	for _, exp := range m.exports {
		writeUnsignedLEB128(&section, uint64(len(exp.Name)))
		section.WriteString(exp.Name)
		section.WriteByte(byte(exp.Kind))
		writeUnsignedLEB128(&section, uint64(exp.Index))
	}
	m.writeSection(sectionExport, section.Bytes())
}

func (m *Module) writeCodeSection() {
	var section bytes.Buffer
	writeUnsignedLEB128(&section, uint64(len(m.functions)))
	for _, fn := range m.functions {
		var code bytes.Buffer
		// Write local declarations
		if len(fn.Locals) > 0 {
			// Group locals by type for efficiency
			grouped := groupLocalsByType(fn.Locals)
			writeUnsignedLEB128(&code, uint64(len(grouped)))
			for _, group := range grouped {
				writeUnsignedLEB128(&code, uint64(group.count))
				code.WriteByte(byte(group.typ))
			}
		} else {
			writeUnsignedLEB128(&code, 0) // no locals
		}
		code.Write(fn.Body)
		code.WriteByte(byte(OpEnd))
		writeUnsignedLEB128(&section, uint64(code.Len()))
		section.Write(code.Bytes())
	}
	m.writeSection(sectionCode, section.Bytes())
}

func (m *Module) writeDataSection() {
	var section bytes.Buffer
	// Number of data segments
	writeUnsignedLEB128(&section, uint64(len(m.data)))
	for _, seg := range m.data {
		// Memory index (always 0 for single memory)
		section.WriteByte(0)
		// Offset expression: i32.const <offset>, end
		section.WriteByte(byte(OpI32Const))
		writeSignedLEB128(&section, int64(seg.Offset))
		section.WriteByte(byte(OpEnd))
		// Data length and bytes
		writeUnsignedLEB128(&section, uint64(len(seg.Bytes)))
		section.Write(seg.Bytes)
	}
	m.writeSection(sectionData, section.Bytes())
}

func (m *Module) writeSection(sectionType byte, data []byte) {
	m.buf.WriteByte(sectionType)
	writeUnsignedLEB128(&m.buf, uint64(len(data)))
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
