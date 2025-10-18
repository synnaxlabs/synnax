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
	// Check if type already exists
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
	// Import functions come before local functions in the index space
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
	// Write magic number
	m.buf.Write(MagicNumber)
	// Write version (must be exact bytes)
	m.buf.Write([]byte{0x01, 0x00, 0x00, 0x00})
	// Write type section
	if len(m.types) > 0 {
		m.writeTypeSection()
	}
	// Write import section
	if len(m.imports) > 0 {
		m.writeImportSection()
	}
	// Write function section
	if len(m.functions) > 0 {
		m.writeFunctionSection()
	}
	// Write memory section
	if m.memory {
		m.writeMemorySection()
	}
	// Write export section
	if len(m.exports) > 0 {
		m.writeExportSection()
	}
	// Write code section
	if len(m.functions) > 0 {
		m.writeCodeSection()
	}

	return m.buf.Bytes()
}

func (m *Module) writeTypeSection() {
	var section bytes.Buffer

	// Write number of types
	WriteLEB128(&section, uint64(len(m.types)))

	for _, ft := range m.types {
		section.WriteByte(0x60) // func type

		// Write params
		WriteLEB128(&section, uint64(len(ft.Params)))
		for _, param := range ft.Params {
			section.WriteByte(byte(param))
		}

		// Write results
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
		// Write module name
		WriteLEB128(&section, uint64(len(imp.Module)))
		section.WriteString(imp.Module)

		// Write import name
		WriteLEB128(&section, uint64(len(imp.Name)))
		section.WriteString(imp.Name)

		// Import kind (0 = function)
		section.WriteByte(0x00)

		// Type index
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
	section.WriteByte(0x01)

	// Memory limits (min 1 page, no max)
	section.WriteByte(0x00)  // no max
	WriteLEB128(&section, 1) // min 1 page

	m.writeSection(SectionMemory, section.Bytes())
}

func (m *Module) writeExportSection() {
	var section bytes.Buffer

	WriteLEB128(&section, uint64(len(m.exports)))

	for _, exp := range m.exports {
		// Write export name
		WriteLEB128(&section, uint64(len(exp.Name)))
		section.WriteString(exp.Name)
		// Write export kind
		section.WriteByte(byte(exp.Kind))
		// Write export index
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

		// Write function body
		code.Write(fn.Body)

		// Function must end with 'end'
		code.WriteByte(byte(OpEnd))

		// Write size and code
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
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		w.WriteByte(b)
		if v == 0 {
			break
		}
	}
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

	var groups []localGroup
	currentType := locals[0]
	currentCount := uint32(1)

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
