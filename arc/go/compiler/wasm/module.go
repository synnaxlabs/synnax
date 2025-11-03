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
	"encoding/binary"
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

// Module represents a complete WASM module
type Module struct {
	types     []FunctionType
	imports   []Import
	functions []Function
	exports   []Export
	memory    bool
	buf       bytes.Buffer
}

// NewModule creates a new WASM module
func NewModule() *Module {
	return &Module{
		types:     make([]FunctionType, 0),
		imports:   make([]Import, 0),
		functions: make([]Function, 0),
		exports:   make([]Export, 0),
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

// EnableMemory enables memory for the module
func (m *Module) EnableMemory() {
	m.memory = true
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
	return m.buf.Bytes()
}

func (m *Module) writeTypeSection() {
	var section bytes.Buffer
	WriteLEB128(&section, uint64(len(m.types)))
	for _, ft := range m.types {
		section.WriteByte(byte(FuncType))
		WriteLEB128(&section, uint64(len(ft.Params)))
		for _, param := range ft.Params {
			section.WriteByte(byte(param))
		}
		WriteLEB128(&section, uint64(len(ft.Results)))
		for _, result := range ft.Results {
			section.WriteByte(byte(result))
		}
	}

	m.writeSection(SectionType, section.Bytes())
}

func (m *Module) writeImportSection() {
	var section bytes.Buffer

	WriteLEB128(&section, uint64(len(m.imports)))

	for _, imp := range m.imports {
		WriteLEB128(&section, uint64(len(imp.Module)))
		section.WriteString(imp.Module)
		WriteLEB128(&section, uint64(len(imp.Name)))
		section.WriteString(imp.Name)
		section.WriteByte(byte(ExportFunc))
		WriteLEB128(&section, uint64(imp.TypeIdx))
	}

	m.writeSection(SectionImport, section.Bytes())
}

func (m *Module) writeFunctionSection() {
	var section bytes.Buffer
	WriteLEB128(&section, uint64(len(m.functions)))
	for _, fn := range m.functions {
		WriteLEB128(&section, uint64(fn.TypeIdx))
	}
	m.writeSection(SectionFunc, section.Bytes())
}

func (m *Module) writeMemorySection() {
	var section bytes.Buffer
	// Number of memories (1)
	section.WriteByte(1)
	// Memory limits (min 1 page, no max)
	section.WriteByte(0)     // no max
	WriteLEB128(&section, 1) // min 1 page
	m.writeSection(SectionMemory, section.Bytes())
}

func (m *Module) writeExportSection() {
	var section bytes.Buffer
	WriteLEB128(&section, uint64(len(m.exports)))
	for _, exp := range m.exports {
		WriteLEB128(&section, uint64(len(exp.Name)))
		section.WriteString(exp.Name)
		section.WriteByte(byte(exp.Kind))
		WriteLEB128(&section, uint64(exp.Index))
	}
	m.writeSection(SectionExport, section.Bytes())
}

func (m *Module) writeCodeSection() {
	var section bytes.Buffer
	WriteLEB128(&section, uint64(len(m.functions)))
	for _, fn := range m.functions {
		var code bytes.Buffer
		// Write local declarations
		if len(fn.Locals) > 0 {
			// Group locals by type for efficiency
			grouped := groupLocalsByType(fn.Locals)
			WriteLEB128(&code, uint64(len(grouped)))
			for _, group := range grouped {
				WriteLEB128(&code, uint64(group.count))
				code.WriteByte(byte(group.typ))
			}
		} else {
			WriteLEB128(&code, 0) // no locals
		}
		code.Write(fn.Body)
		code.WriteByte(byte(OpEnd))
		WriteLEB128(&section, uint64(code.Len()))
		section.Write(code.Bytes())
	}
	m.writeSection(SectionCode, section.Bytes())
}

func (m *Module) writeSection(sectionType byte, data []byte) {
	m.buf.WriteByte(sectionType)
	WriteLEB128(&m.buf, uint64(len(data)))
	m.buf.Write(data)
}

// WriteLEB128 writes an unsigned LEB128 integer
func WriteLEB128(w *bytes.Buffer, v uint64) {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, v)
	w.Write(buf[:n])
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
