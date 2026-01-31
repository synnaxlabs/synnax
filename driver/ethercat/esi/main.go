// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// ESI Parser - Parses EtherCAT Slave Information (ESI) XML files and generates C++ code
// containing hardcoded PDO definitions for known devices.
//
// Usage: go run main.go <esi-directory> <output.h>
package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

// Parsed device info for C++ generation

type ParsedDevice struct {
	VendorID    uint32
	VendorName  string
	ProductCode uint32
	Revision    uint32
	Name        string
	InputPDOs   []ParsedPDOEntry
	OutputPDOs  []ParsedPDOEntry
}

type ParsedPDOEntry struct {
	PDOIndex  uint16
	Index     uint16
	SubIndex  uint8
	BitLength uint8
	Name      string
	DataType  string
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

func mapDataType(esiType string) string {
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

func parseESIFile(path string) ([]ParsedDevice, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var esi EtherCATInfo
	if err := xml.Unmarshal(data, &esi); err != nil {
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
			fmt.Fprintf(os.Stderr, "Warning: failed to parse product code %q in %s: %v\n",
				dev.Type.ProductCode, path, err)
			continue
		}

		revision, err := parseHexOrDecimal(dev.Type.RevisionNo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse revision %q in %s: %v\n",
				dev.Type.RevisionNo, path, err)
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
		}

		// Parse TxPDO (inputs - slave to master)
		for _, pdo := range dev.TxPdo {
			pdoIndex, err := parseHexOrDecimal(pdo.Index)
			if err != nil {
				continue
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
					DataType:  mapDataType(entry.DataType),
				})
			}
		}

		// Parse RxPDO (outputs - master to slave)
		for _, pdo := range dev.RxPdo {
			pdoIndex, err := parseHexOrDecimal(pdo.Index)
			if err != nil {
				continue
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
					DataType:  mapDataType(entry.DataType),
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

func generateCPP(devices []ParsedDevice, outputPath string) error {
	// Generate header file
	var hdr strings.Builder
	hdr.WriteString(`// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// AUTO-GENERATED FILE - DO NOT EDIT
// Generated by: go run driver/ethercat/esi/main.go
// To regenerate: go run driver/ethercat/esi/main.go <esi-directory> <output.h>

#pragma once

#include <cstdint>

#include "driver/ethercat/master/slave_info.h"

namespace ethercat::esi {

bool lookup_device_pdos(
    uint32_t vendor_id,
    uint32_t product_code,
    uint32_t revision,
    SlaveInfo &slave
);

}
`)

	// Generate implementation file with POD arrays for fast compilation
	var impl strings.Builder
	impl.WriteString(`// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// AUTO-GENERATED FILE - DO NOT EDIT
// Generated by: go run driver/ethercat/esi/main.go

#include "driver/ethercat/esi/known_devices.h"

#include "x/cpp/telem/telem.h"

namespace ethercat::esi {

namespace {

struct PDOData {
    uint16_t pdo_index;
    uint16_t index;
    uint8_t subindex;
    uint8_t bit_length;
    const char* name;
    telem::DataType data_type;
};

struct DeviceData {
    uint32_t vendor_id;
    uint32_t product_code;
    uint32_t revision;
    const char* name;
    const PDOData* inputs;
    size_t input_count;
    const PDOData* outputs;
    size_t output_count;
};

`)

	// Generate PDO arrays for each device
	for i, dev := range devices {
		if len(dev.InputPDOs) > 0 {
			impl.WriteString(fmt.Sprintf("const PDOData DEVICE_%d_INPUTS[] = {\n", i))
			for _, pdo := range dev.InputPDOs {
				impl.WriteString(fmt.Sprintf(
					"    {0x%04X, 0x%04X, %d, %d, \"%s\", %s},\n",
					pdo.PDOIndex, pdo.Index, pdo.SubIndex, pdo.BitLength,
					escapeString(pdo.Name), pdo.DataType,
				))
			}
			impl.WriteString("};\n\n")
		}

		if len(dev.OutputPDOs) > 0 {
			impl.WriteString(fmt.Sprintf("const PDOData DEVICE_%d_OUTPUTS[] = {\n", i))
			for _, pdo := range dev.OutputPDOs {
				impl.WriteString(fmt.Sprintf(
					"    {0x%04X, 0x%04X, %d, %d, \"%s\", %s},\n",
					pdo.PDOIndex, pdo.Index, pdo.SubIndex, pdo.BitLength,
					escapeString(pdo.Name), pdo.DataType,
				))
			}
			impl.WriteString("};\n\n")
		}
	}

	// Generate device table
	impl.WriteString("const DeviceData KNOWN_DEVICES[] = {\n")
	for i, dev := range devices {
		inputsPtr := "nullptr"
		outputsPtr := "nullptr"
		if len(dev.InputPDOs) > 0 {
			inputsPtr = fmt.Sprintf("DEVICE_%d_INPUTS", i)
		}
		if len(dev.OutputPDOs) > 0 {
			outputsPtr = fmt.Sprintf("DEVICE_%d_OUTPUTS", i)
		}

		impl.WriteString(fmt.Sprintf(
			"    {0x%08X, %d, 0x%08X, \"%s\", %s, %d, %s, %d},\n",
			dev.VendorID, dev.ProductCode, dev.Revision, escapeString(dev.Name),
			inputsPtr, len(dev.InputPDOs), outputsPtr, len(dev.OutputPDOs),
		))
	}
	impl.WriteString("};\n\n")

	impl.WriteString(fmt.Sprintf("constexpr size_t KNOWN_DEVICES_COUNT = %d;\n\n", len(devices)))

	impl.WriteString(`}

bool lookup_device_pdos(
    const uint32_t vendor_id,
    const uint32_t product_code,
    const uint32_t revision,
    SlaveInfo &slave
) {
    const DeviceData* match = nullptr;

    for (size_t i = 0; i < KNOWN_DEVICES_COUNT; ++i) {
        const auto &dev = KNOWN_DEVICES[i];
        if (dev.vendor_id == vendor_id && dev.product_code == product_code) {
            if (dev.revision == revision) {
                match = &dev;
                break;
            }
            if (match == nullptr) {
                match = &dev;
            }
        }
    }

    if (match == nullptr) return false;

    slave.input_pdos.clear();
    slave.input_pdos.reserve(match->input_count);
    for (size_t i = 0; i < match->input_count; ++i) {
        const auto &p = match->inputs[i];
        slave.input_pdos.emplace_back(
            p.pdo_index, p.index, p.subindex, p.bit_length,
            true, p.name, p.data_type
        );
    }

    slave.output_pdos.clear();
    slave.output_pdos.reserve(match->output_count);
    for (size_t i = 0; i < match->output_count; ++i) {
        const auto &p = match->outputs[i];
        slave.output_pdos.emplace_back(
            p.pdo_index, p.index, p.subindex, p.bit_length,
            false, p.name, p.data_type
        );
    }

    slave.pdos_discovered = true;
    return true;
}

}
`)

	// Write header
	if err := os.WriteFile(outputPath, []byte(hdr.String()), 0644); err != nil {
		return err
	}

	// Write implementation (same name but .cpp)
	implPath := strings.TrimSuffix(outputPath, ".h") + ".cpp"
	return os.WriteFile(implPath, []byte(impl.String()), 0644)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <esi-directory> <output.h>\n", os.Args[0])
		os.Exit(1)
	}

	esiDir := os.Args[1]
	outputPath := os.Args[2]

	var allDevices []ParsedDevice

	err := filepath.Walk(esiDir, func(path string, info os.FileInfo, err error) error {
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
		devices, err := parseESIFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			return nil
		}

		allDevices = append(allDevices, devices...)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
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

	fmt.Printf("Found %d devices with PDO definitions\n", len(allDevices))

	if err := generateCPP(allDevices, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating C++ code: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outputPath)
}
