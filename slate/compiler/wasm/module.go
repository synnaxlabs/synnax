package wasm

import (
	"bytes"
	"encoding/binary"
	"math"
)

// FunctionType represents a function signature
type FunctionType struct {
	Params  []ValueType
	Results []ValueType
}

// Function represents a WASM function
type Function struct {
	Name    string
	TypeIdx uint32
	Locals  []ValueType
	Body    []byte
	Export  bool
}

// Import represents an imported function
type Import struct {
	Module  string
	Name    string
	TypeIdx uint32
}

// Module represents a complete WASM module
type Module struct {
	types     []FunctionType
	imports   []Import
	functions []Function
	memory    bool
	buf       bytes.Buffer
}

// NewModule creates a new WASM module
func NewModule() *Module {
	return &Module{
		types:     make([]FunctionType, 0),
		imports:   make([]Import, 0),
		functions: make([]Function, 0),
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

// AddFunction adds a function to the module
func (m *Module) AddFunction(f Function) uint32 {
	// Function index is imports + local functions
	idx := uint32(len(m.imports) + len(m.functions))
	m.functions = append(m.functions, f)
	return idx
}

// EnableMemory enables memory for the module
func (m *Module) EnableMemory() {
	m.memory = true
}

// Generate generates the WASM binary
func (m *Module) Generate() []byte {
	m.buf.Reset()

	// Write magic number and version (must be exact bytes)
	m.buf.WriteByte(0x00)
	m.buf.WriteByte(0x61)
	m.buf.WriteByte(0x73)
	m.buf.WriteByte(0x6d)
	m.buf.WriteByte(0x01)
	m.buf.WriteByte(0x00)
	m.buf.WriteByte(0x00)
	m.buf.WriteByte(0x00)

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
	if m.hasExports() {
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

	// Count exports
	exportCount := 0
	for _, fn := range m.functions {
		if fn.Export {
			exportCount++
		}
	}
	if m.memory {
		exportCount++
	}

	WriteLEB128(&section, uint64(exportCount))

	// Write memory export if enabled
	if m.memory {
		WriteLEB128(&section, uint64(len("memory")))
		section.WriteString("memory")
		section.WriteByte(byte(ExportMemory))
		WriteLEB128(&section, 0)
	}

	// Write function exports
	for i, fn := range m.functions {
		if fn.Export {
			WriteLEB128(&section, uint64(len(fn.Name)))
			section.WriteString(fn.Name)
			section.WriteByte(byte(ExportFunc))
			// Function index includes imports
			WriteLEB128(&section, uint64(len(m.imports)+i))
		}
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
			WriteLEB128(&code, 1) // 1 group
			WriteLEB128(&code, uint64(len(fn.Locals)))
			code.WriteByte(byte(fn.Locals[0])) // Assume all same type for now
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

func (m *Module) hasExports() bool {
	if m.memory {
		return true
	}
	for _, fn := range m.functions {
		if fn.Export {
			return true
		}
	}
	return false
}

func (m *Module) writeU32(v uint32) {
	binary.Write(&m.buf, binary.LittleEndian, v)
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

// writeSignedLEB128 writes a signed LEB128 integer
func writeSignedLEB128(w *bytes.Buffer, v int64) {
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if (v == 0 && b&0x40 == 0) || (v == -1 && b&0x40 != 0) {
			w.WriteByte(b)
			break
		}
		w.WriteByte(b | 0x80)
	}
}

// WriteF64 writes a float64 constant instruction
func WriteF64Const(w *bytes.Buffer, v float64) {
	w.WriteByte(byte(OpF64Const))
	bits := math.Float64bits(v)
	binary.Write(w, binary.LittleEndian, bits)
}

// WriteF32 writes a float32 constant instruction
func WriteF32Const(w *bytes.Buffer, v float32) {
	w.WriteByte(byte(OpF32Const))
	bits := math.Float32bits(v)
	binary.Write(w, binary.LittleEndian, bits)
}

// WriteI32 writes an i32 constant instruction
func WriteI32Const(w *bytes.Buffer, v int32) {
	w.WriteByte(byte(OpI32Const))
	writeSignedLEB128(w, int64(v))
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
