// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// ESI Parser - Parses EtherCAT Slave Information (ESI) XML files and generates C++ code
// containing a binary blob with PDO definitions for known devices.
//
// Usage:
//
//	go run main.go --catalog catalog/vendors.json --output-dir .
//	go run main.go <esi-directory> <output.h>  # legacy mode
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html/charset"
)

// ESI XML structures

type EtherCATInfo struct {
	XMLName      xml.Name     `xml:"EtherCATInfo"`
	Vendor       Vendor       `xml:"Vendor"`
	Descriptions Descriptions `xml:"Descriptions"`
}

type Vendor struct {
	ID   string `xml:"Id"`
	Name string `xml:"Name"`
}

type Descriptions struct {
	Devices Devices `xml:"Devices"`
}

type Devices struct {
	Device []Device `xml:"Device"`
}

type Device struct {
	Type    DeviceType `xml:"Type"`
	Name    []Name     `xml:"Name"`
	TxPdo   []PDO      `xml:"TxPdo"`
	RxPdo   []PDO      `xml:"RxPdo"`
	Profile Profile    `xml:"Profile"`
}

type DeviceType struct {
	ProductCode string `xml:"ProductCode,attr"`
	RevisionNo  string `xml:"RevisionNo,attr"`
	Value       string `xml:",chardata"`
}

type Name struct {
	LcId  string `xml:"LcId,attr"`
	Value string `xml:",chardata"`
}

type PDO struct {
	Sm    string     `xml:"Sm,attr"`
	Index string     `xml:"Index"`
	Name  string     `xml:"Name"`
	Entry []PDOEntry `xml:"Entry"`
}

type PDOEntry struct {
	Index    string `xml:"Index"`
	SubIndex string `xml:"SubIndex"`
	BitLen   string `xml:"BitLen"`
	Name     string `xml:"Name"`
	DataType string `xml:"DataType"`
}

type Profile struct {
	ChannelCount int `xml:"ChannelCount"`
}

// Vendor catalog structures

type VendorCatalog struct {
	Vendors []VendorInfo `json:"vendors"`
}

type VendorInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Directory  string   `json:"directory"`
	SourceURLs []string `json:"source_urls"`
}

// Parsed device info for C++ generation

type ParsedDevice struct {
	VendorID    uint32
	VendorName  string
	ProductCode uint32
	Revision    uint32
	Name        string
	InputPDOs   []ParsedPDOEntry
	OutputPDOs  []ParsedPDOEntry
	SourceFile  string
}

type ParsedPDOEntry struct {
	PDOIndex  uint16
	Index     uint16
	SubIndex  uint8
	BitLength uint8
	Name      string
	DataType  string
}

// DeviceKey for duplicate detection
type DeviceKey struct {
	VendorID    uint32
	ProductCode uint32
	Revision    uint32
}

// Statistics for output
type RegistryStats struct {
	DeviceCount      int            `json:"device_count"`
	VendorCount      int            `json:"vendor_count"`
	PDOEntryCount    int            `json:"pdo_entry_count"`
	InputPDOCount    int            `json:"input_pdo_count"`
	OutputPDOCount   int            `json:"output_pdo_count"`
	DataTypes        map[string]int `json:"data_types"`
	VendorDevices    map[string]int `json:"vendor_devices"`
	DuplicatesFound  int            `json:"duplicates_found"`
	ValidationErrors int            `json:"validation_errors"`
}

func parseHexOrDecimal(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#x") || strings.HasPrefix(s, "#X") {
		return strconv.ParseUint(s[2:], 16, 64)
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return strconv.ParseUint(s[2:], 16, 64)
	}
	return strconv.ParseUint(s, 10, 64)
}

// mapDataType converts ESI data type to a numeric ID for compact storage
func mapDataType(esiType string) uint8 {
	switch strings.ToUpper(esiType) {
	case "BOOL", "BOOLEAN", "BIT":
		return 1 // UINT8
	case "BIT2", "BIT3", "BIT4", "BIT5", "BIT6", "BIT7", "BIT8":
		return 1 // UINT8
	case "SINT", "INT8", "INTEGER8":
		return 2 // INT8
	case "USINT", "UINT8", "UNSIGNED8", "BYTE":
		return 1 // UINT8
	case "INT", "INT16", "INTEGER16":
		return 3 // INT16
	case "UINT", "UINT16", "UNSIGNED16", "WORD":
		return 4 // UINT16
	case "DINT", "INT32", "INTEGER32":
		return 5 // INT32
	case "UDINT", "UINT32", "UNSIGNED32", "DWORD":
		return 6 // UINT32
	case "LINT", "INT64", "INTEGER64":
		return 7 // INT64
	case "ULINT", "UINT64", "UNSIGNED64", "LWORD":
		return 8 // UINT64
	case "REAL", "REAL32", "FLOAT":
		return 9 // FLOAT32
	case "LREAL", "REAL64", "DOUBLE":
		return 10 // FLOAT64
	default:
		return 1 // UINT8
	}
}

// mapDataTypeString returns the telem:: type string for C++ code
func mapDataTypeString(esiType string) string {
	switch strings.ToUpper(esiType) {
	case "BOOL", "BOOLEAN", "BIT":
		return "telem::UINT8_T"
	case "BIT2", "BIT3", "BIT4", "BIT5", "BIT6", "BIT7", "BIT8":
		return "telem::UINT8_T"
	case "SINT", "INT8", "INTEGER8":
		return "telem::INT8_T"
	case "USINT", "UINT8", "UNSIGNED8", "BYTE":
		return "telem::UINT8_T"
	case "INT", "INT16", "INTEGER16":
		return "telem::INT16_T"
	case "UINT", "UINT16", "UNSIGNED16", "WORD":
		return "telem::UINT16_T"
	case "DINT", "INT32", "INTEGER32":
		return "telem::INT32_T"
	case "UDINT", "UINT32", "UNSIGNED32", "DWORD":
		return "telem::UINT32_T"
	case "LINT", "INT64", "INTEGER64":
		return "telem::INT64_T"
	case "ULINT", "UINT64", "UNSIGNED64", "LWORD":
		return "telem::UINT64_T"
	case "REAL", "REAL32", "FLOAT":
		return "telem::FLOAT32_T"
	case "LREAL", "REAL64", "DOUBLE":
		return "telem::FLOAT64_T"
	default:
		return "telem::UINT8_T"
	}
}

func getDeviceName(names []Name, fallback string) string {
	for _, n := range names {
		if n.LcId == "1033" { // English
			return strings.TrimSpace(n.Value)
		}
	}
	if len(names) > 0 {
		return strings.TrimSpace(names[0].Value)
	}
	return fallback
}

// validatePDOIndex checks if a PDO index is in the valid range
func validatePDOIndex(index uint16, isInput bool) bool {
	if isInput {
		// TxPDO: 0x1A00-0x1BFF
		return index >= 0x1A00 && index <= 0x1BFF
	}
	// RxPDO: 0x1600-0x17FF
	return index >= 0x1600 && index <= 0x17FF
}

func parseESIFile(path string, verbose bool) ([]ParsedDevice, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Create decoder with charset support for ISO-8859-1 and other encodings
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = func(label string, input io.Reader) (io.Reader, error) {
		return charset.NewReaderLabel(label, input)
	}

	var esi EtherCATInfo
	if err := decoder.Decode(&esi); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	vendorID, err := parseHexOrDecimal(esi.Vendor.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vendor ID in %s: %w", path, err)
	}

	var devices []ParsedDevice

	for _, dev := range esi.Descriptions.Devices.Device {
		productCode, err := parseHexOrDecimal(dev.Type.ProductCode)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse product code %q in %s: %v\n",
					dev.Type.ProductCode, path, err)
			}
			continue
		}

		revision, err := parseHexOrDecimal(dev.Type.RevisionNo)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse revision %q in %s: %v\n",
					dev.Type.RevisionNo, path, err)
			}
			continue
		}

		// Skip devices with no PDOs defined
		if len(dev.TxPdo) == 0 && len(dev.RxPdo) == 0 {
			continue
		}

		parsed := ParsedDevice{
			VendorID:    uint32(vendorID),
			VendorName:  esi.Vendor.Name,
			ProductCode: uint32(productCode),
			Revision:    uint32(revision),
			Name:        getDeviceName(dev.Name, dev.Type.Value),
			SourceFile:  filepath.Base(path),
		}

		// Parse TxPDO (inputs - slave to master)
		for _, pdo := range dev.TxPdo {
			pdoIndex, err := parseHexOrDecimal(pdo.Index)
			if err != nil {
				continue
			}

			if !validatePDOIndex(uint16(pdoIndex), true) && verbose {
				fmt.Fprintf(os.Stderr, "Warning: TxPDO index 0x%04X outside valid range in %s\n",
					pdoIndex, path)
			}

			for _, entry := range pdo.Entry {
				if entry.Index == "" || entry.SubIndex == "" {
					continue
				}

				index, err := parseHexOrDecimal(entry.Index)
				if err != nil {
					continue
				}

				subIndex, err := parseHexOrDecimal(entry.SubIndex)
				if err != nil {
					continue
				}

				bitLen, err := strconv.Atoi(entry.BitLen)
				if err != nil {
					continue
				}

				parsed.InputPDOs = append(parsed.InputPDOs, ParsedPDOEntry{
					PDOIndex:  uint16(pdoIndex),
					Index:     uint16(index),
					SubIndex:  uint8(subIndex),
					BitLength: uint8(bitLen),
					Name:      entry.Name,
					DataType:  mapDataTypeString(entry.DataType),
				})
			}
		}

		// Parse RxPDO (outputs - master to slave)
		for _, pdo := range dev.RxPdo {
			pdoIndex, err := parseHexOrDecimal(pdo.Index)
			if err != nil {
				continue
			}

			if !validatePDOIndex(uint16(pdoIndex), false) && verbose {
				fmt.Fprintf(os.Stderr, "Warning: RxPDO index 0x%04X outside valid range in %s\n",
					pdoIndex, path)
			}

			for _, entry := range pdo.Entry {
				if entry.Index == "" || entry.SubIndex == "" {
					continue
				}

				index, err := parseHexOrDecimal(entry.Index)
				if err != nil {
					continue
				}

				subIndex, err := parseHexOrDecimal(entry.SubIndex)
				if err != nil {
					continue
				}

				bitLen, err := strconv.Atoi(entry.BitLen)
				if err != nil {
					continue
				}

				parsed.OutputPDOs = append(parsed.OutputPDOs, ParsedPDOEntry{
					PDOIndex:  uint16(pdoIndex),
					Index:     uint16(index),
					SubIndex:  uint8(subIndex),
					BitLength: uint8(bitLen),
					Name:      entry.Name,
					DataType:  mapDataTypeString(entry.DataType),
				})
			}
		}

		if len(parsed.InputPDOs) > 0 || len(parsed.OutputPDOs) > 0 {
			devices = append(devices, parsed)
		}
	}

	return devices, nil
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// detectDuplicates checks for duplicate devices and returns deduplicated list
func detectDuplicates(devices []ParsedDevice, verbose bool) ([]ParsedDevice, int) {
	seen := make(map[DeviceKey]int)
	var deduplicated []ParsedDevice
	duplicateCount := 0

	for _, dev := range devices {
		key := DeviceKey{
			VendorID:    dev.VendorID,
			ProductCode: dev.ProductCode,
			Revision:    dev.Revision,
		}

		if existingIdx, exists := seen[key]; exists {
			duplicateCount++
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: duplicate device 0x%08X/0x%08X/0x%08X (%s) from %s, "+
					"keeping existing from %s\n",
					dev.VendorID, dev.ProductCode, dev.Revision, dev.Name, dev.SourceFile,
					deduplicated[existingIdx].SourceFile)
			}
			continue
		}

		seen[key] = len(deduplicated)
		deduplicated = append(deduplicated, dev)
	}

	return deduplicated, duplicateCount
}

// collectVendors extracts unique vendors from devices
func collectVendors(devices []ParsedDevice) map[uint32]string {
	vendors := make(map[uint32]string)
	for _, dev := range devices {
		if _, exists := vendors[dev.VendorID]; !exists {
			vendors[dev.VendorID] = dev.VendorName
		}
	}
	return vendors
}

// generateStats creates statistics about the registry
func generateStats(devices []ParsedDevice, duplicates int) RegistryStats {
	stats := RegistryStats{
		DeviceCount:     len(devices),
		DataTypes:       make(map[string]int),
		VendorDevices:   make(map[string]int),
		DuplicatesFound: duplicates,
	}

	vendors := make(map[uint32]string)
	for _, dev := range devices {
		vendors[dev.VendorID] = dev.VendorName
		stats.VendorDevices[dev.VendorName]++
		stats.InputPDOCount += len(dev.InputPDOs)
		stats.OutputPDOCount += len(dev.OutputPDOs)

		for _, pdo := range dev.InputPDOs {
			stats.DataTypes[pdo.DataType]++
		}
		for _, pdo := range dev.OutputPDOs {
			stats.DataTypes[pdo.DataType]++
		}
	}

	stats.VendorCount = len(vendors)
	stats.PDOEntryCount = stats.InputPDOCount + stats.OutputPDOCount

	return stats
}

// StringTable manages deduplicated strings with offsets
type StringTable struct {
	data    []byte
	offsets map[string]uint32
}

func NewStringTable() *StringTable {
	return &StringTable{
		data:    []byte{0}, // Start with null byte for empty strings
		offsets: map[string]uint32{"": 0},
	}
}

func (st *StringTable) Add(s string) uint32 {
	if offset, exists := st.offsets[s]; exists {
		return offset
	}
	offset := uint32(len(st.data))
	st.offsets[s] = offset
	st.data = append(st.data, []byte(s)...)
	st.data = append(st.data, 0) // null terminator
	return offset
}

func (st *StringTable) Data() []byte {
	return st.data
}

// Binary format constants
const (
	BinaryMagic   = 0x52495345 // "ESIR" in little endian
	BinaryVersion = 1
)

// dataTypeToID converts telem type string to numeric ID
func dataTypeToID(dt string) uint8 {
	switch dt {
	case "telem::UINT8_T":
		return 1
	case "telem::INT8_T":
		return 2
	case "telem::INT16_T":
		return 3
	case "telem::UINT16_T":
		return 4
	case "telem::INT32_T":
		return 5
	case "telem::UINT32_T":
		return 6
	case "telem::INT64_T":
		return 7
	case "telem::UINT64_T":
		return 8
	case "telem::FLOAT32_T":
		return 9
	case "telem::FLOAT64_T":
		return 10
	default:
		return 1
	}
}

func generateCPP(devices []ParsedDevice, outputPath string) error {
	vendors := collectVendors(devices)
	stringTable := NewStringTable()

	// Group devices by vendor_id + product_code for the index
	type IndexKey struct {
		VendorID    uint32
		ProductCode uint32
	}
	deviceGroups := make(map[IndexKey][]int)
	for i, dev := range devices {
		key := IndexKey{VendorID: dev.VendorID, ProductCode: dev.ProductCode}
		deviceGroups[key] = append(deviceGroups[key], i)
	}

	// Sort keys for binary search
	var sortedKeys []IndexKey
	for k := range deviceGroups {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		if sortedKeys[i].VendorID != sortedKeys[j].VendorID {
			return sortedKeys[i].VendorID < sortedKeys[j].VendorID
		}
		return sortedKeys[i].ProductCode < sortedKeys[j].ProductCode
	})

	// Sort vendor IDs
	var vendorIDs []uint32
	for id := range vendors {
		vendorIDs = append(vendorIDs, id)
	}
	sort.Slice(vendorIDs, func(i, j int) bool { return vendorIDs[i] < vendorIDs[j] })

	// Pre-register all strings
	for _, id := range vendorIDs {
		stringTable.Add(vendors[id])
	}
	for _, dev := range devices {
		stringTable.Add(dev.Name)
		for _, pdo := range dev.InputPDOs {
			stringTable.Add(pdo.Name)
		}
		for _, pdo := range dev.OutputPDOs {
			stringTable.Add(pdo.Name)
		}
	}

	// Count total PDOs
	totalPDOs := 0
	for _, dev := range devices {
		totalPDOs += len(dev.InputPDOs) + len(dev.OutputPDOs)
	}

	// Build binary blob
	var blob bytes.Buffer

	// Header (32 bytes)
	binary.Write(&blob, binary.LittleEndian, uint32(BinaryMagic))
	binary.Write(&blob, binary.LittleEndian, uint32(BinaryVersion))
	binary.Write(&blob, binary.LittleEndian, uint32(len(vendorIDs)))
	binary.Write(&blob, binary.LittleEndian, uint32(len(sortedKeys)))
	binary.Write(&blob, binary.LittleEndian, uint32(len(devices)))
	binary.Write(&blob, binary.LittleEndian, uint32(totalPDOs))
	binary.Write(&blob, binary.LittleEndian, uint32(0)) // string_table_offset placeholder
	binary.Write(&blob, binary.LittleEndian, uint32(0)) // string_table_size placeholder

	// Vendor table (8 bytes each)
	for _, id := range vendorIDs {
		binary.Write(&blob, binary.LittleEndian, uint32(id))
		binary.Write(&blob, binary.LittleEndian, stringTable.Add(vendors[id]))
	}

	// Device index (16 bytes each)
	deviceOffset := uint32(0)
	for _, key := range sortedKeys {
		count := uint32(len(deviceGroups[key]))
		binary.Write(&blob, binary.LittleEndian, key.VendorID)
		binary.Write(&blob, binary.LittleEndian, key.ProductCode)
		binary.Write(&blob, binary.LittleEndian, deviceOffset)
		binary.Write(&blob, binary.LittleEndian, count)
		deviceOffset += count
	}

	// Build ordered device list
	var orderedDevices []int
	for _, key := range sortedKeys {
		orderedDevices = append(orderedDevices, deviceGroups[key]...)
	}

	// Device table (16 bytes each)
	// We need to know PDO offsets, so first calculate them
	pdoOffsets := make([]uint32, len(devices))
	currentPDOOffset := uint32(0)
	for i, dev := range devices {
		pdoOffsets[i] = currentPDOOffset
		currentPDOOffset += uint32(len(dev.InputPDOs) + len(dev.OutputPDOs))
	}

	for _, origIdx := range orderedDevices {
		dev := devices[origIdx]
		binary.Write(&blob, binary.LittleEndian, dev.Revision)
		binary.Write(&blob, binary.LittleEndian, stringTable.Add(dev.Name))
		binary.Write(&blob, binary.LittleEndian, pdoOffsets[origIdx])
		binary.Write(&blob, binary.LittleEndian, uint16(len(dev.InputPDOs)))
		binary.Write(&blob, binary.LittleEndian, uint16(len(dev.OutputPDOs)))
	}

	// PDO table (12 bytes each)
	for _, origIdx := range orderedDevices {
		dev := devices[origIdx]
		for _, pdo := range dev.InputPDOs {
			binary.Write(&blob, binary.LittleEndian, pdo.PDOIndex)
			binary.Write(&blob, binary.LittleEndian, pdo.Index)
			binary.Write(&blob, binary.LittleEndian, pdo.SubIndex)
			binary.Write(&blob, binary.LittleEndian, pdo.BitLength)
			binary.Write(&blob, binary.LittleEndian, dataTypeToID(pdo.DataType))
			binary.Write(&blob, binary.LittleEndian, uint8(0)) // padding
			binary.Write(&blob, binary.LittleEndian, stringTable.Add(pdo.Name))
		}
		for _, pdo := range dev.OutputPDOs {
			binary.Write(&blob, binary.LittleEndian, pdo.PDOIndex)
			binary.Write(&blob, binary.LittleEndian, pdo.Index)
			binary.Write(&blob, binary.LittleEndian, pdo.SubIndex)
			binary.Write(&blob, binary.LittleEndian, pdo.BitLength)
			binary.Write(&blob, binary.LittleEndian, dataTypeToID(pdo.DataType))
			binary.Write(&blob, binary.LittleEndian, uint8(0)) // padding
			binary.Write(&blob, binary.LittleEndian, stringTable.Add(pdo.Name))
		}
	}

	// String table
	stringTableOffset := uint32(blob.Len())
	stringData := stringTable.Data()
	blob.Write(stringData)

	// Fix up header with string table info
	blobData := blob.Bytes()
	binary.LittleEndian.PutUint32(blobData[24:28], stringTableOffset)
	binary.LittleEndian.PutUint32(blobData[28:32], uint32(len(stringData)))

	// Generate just the blob as an .inc file (no synnax dependencies)
	var inc strings.Builder
	inc.WriteString(`// AUTO-GENERATED FILE - DO NOT EDIT
// Generated by ESI parser from EtherCAT Slave Information XML files
// This file contains only the binary blob data - no external dependencies
//
// To regenerate: go run main.go <esi-directory> <output-dir>

`)
	inc.WriteString(fmt.Sprintf("// Statistics: %d devices, %d vendors, %d PDO entries\n", len(devices), len(vendors), totalPDOs))
	inc.WriteString(fmt.Sprintf("// Blob size: %d bytes\n\n", len(blobData)))

	inc.WriteString("const uint8_t REGISTRY_BLOB[] = {\n")

	// Write blob as hex bytes, 16 per line
	for i, b := range blobData {
		if i%16 == 0 {
			inc.WriteString("    ")
		}
		inc.WriteString(fmt.Sprintf("0x%02X,", b))
		if i%16 == 15 || i == len(blobData)-1 {
			inc.WriteString("\n")
		}
	}

	inc.WriteString("};\n")

	// Write the .inc file
	incPath := filepath.Join(filepath.Dir(outputPath), "registry_blob.inc")
	return os.WriteFile(incPath, []byte(inc.String()), 0644)
}

func writeStats(stats RegistryStats, outputDir string) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "registry_stats.json"), data, 0644)
}

func loadVendorCatalog(catalogPath string) (*VendorCatalog, error) {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}
	var catalog VendorCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	return &catalog, nil
}

func main() {
	// Define command-line flags
	catalogPath := flag.String("catalog", "", "Path to vendors.json catalog file")
	outputDir := flag.String("output-dir", ".", "Output directory for generated files")
	failOnDuplicates := flag.Bool("fail-on-duplicates", false, "Exit with error if duplicates found")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	var allDevices []ParsedDevice
	var err error

	if *catalogPath != "" {
		// New catalog-based mode
		catalog, err := loadVendorCatalog(*catalogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading catalog: %v\n", err)
			os.Exit(1)
		}

		catalogDir := filepath.Dir(*catalogPath)

		for _, vendor := range catalog.Vendors {
			vendorDir := filepath.Join(catalogDir, vendor.Directory)
			if _, err := os.Stat(vendorDir); os.IsNotExist(err) {
				if *verbose {
					fmt.Printf("Skipping vendor %s: directory %s does not exist\n",
						vendor.Name, vendorDir)
				}
				continue
			}

			err := filepath.Walk(vendorDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				if !strings.HasSuffix(strings.ToLower(path), ".xml") {
					return nil
				}

				if *verbose {
					fmt.Printf("Parsing %s...\n", path)
				}
				devices, err := parseESIFile(path, *verbose)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
					return nil
				}

				allDevices = append(allDevices, devices...)
				return nil
			})

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error walking vendor directory %s: %v\n", vendorDir, err)
			}
		}
	} else if len(flag.Args()) >= 2 {
		// Legacy mode: go run main.go <esi-directory> <output.h>
		esiDir := flag.Args()[0]
		*outputDir = filepath.Dir(flag.Args()[1])

		err = filepath.Walk(esiDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(path), ".xml") {
				return nil
			}

			fmt.Printf("Parsing %s...\n", path)
			devices, parseErr := parseESIFile(path, true)
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", parseErr)
				return nil
			}

			allDevices = append(allDevices, devices...)
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s --catalog <vendors.json> --output-dir <dir>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s <esi-directory> <output.h>  (legacy mode)\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Detect and handle duplicates
	allDevices, duplicateCount := detectDuplicates(allDevices, *verbose)
	if duplicateCount > 0 && *failOnDuplicates {
		fmt.Fprintf(os.Stderr, "Error: %d duplicate devices found\n", duplicateCount)
		os.Exit(1)
	}

	// Sort by vendor ID, product code, revision for consistent output
	sort.Slice(allDevices, func(i, j int) bool {
		if allDevices[i].VendorID != allDevices[j].VendorID {
			return allDevices[i].VendorID < allDevices[j].VendorID
		}
		if allDevices[i].ProductCode != allDevices[j].ProductCode {
			return allDevices[i].ProductCode < allDevices[j].ProductCode
		}
		return allDevices[i].Revision < allDevices[j].Revision
	})

	fmt.Printf("Found %d devices with PDO definitions from %d vendors\n",
		len(allDevices), len(collectVendors(allDevices)))

	// Determine output directory
	if len(flag.Args()) >= 2 {
		*outputDir = flag.Args()[1]
	}

	// Create a dummy path for generateCPP (it extracts the directory)
	outputPath := filepath.Join(*outputDir, "registry_blob.inc")

	if err := generateCPP(allDevices, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating blob: %v\n", err)
		os.Exit(1)
	}

	// Generate statistics
	stats := generateStats(allDevices, duplicateCount)
	if err := writeStats(stats, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write stats: %v\n", err)
	}

	fmt.Printf("Generated %s\n", outputPath)
	fmt.Printf("Binary blob size: %d bytes (%.2f MB)\n",
		stats.PDOEntryCount*12+stats.DeviceCount*16+stats.VendorCount*8+32,
		float64(stats.PDOEntryCount*12+stats.DeviceCount*16+stats.VendorCount*8+32)/(1024*1024))
}
