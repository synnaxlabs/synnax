// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHexOrDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
		wantErr  bool
	}{
		{"123", 123, false},
		{"0x1A00", 0x1A00, false},
		{"0X1A00", 0x1A00, false},
		{"#x1A00", 0x1A00, false},
		{"#X1A00", 0x1A00, false},
		{"0xDEBE50F7", 0xDEBE50F7, false},
		{"  0x1A00  ", 0x1A00, false},
		{"", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseHexOrDecimal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHexOrDecimal(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("parseHexOrDecimal(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMapDataType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"BOOL", "telem::UINT8_T"},
		{"BOOLEAN", "telem::UINT8_T"},
		{"BIT", "telem::UINT8_T"},
		{"SINT", "telem::INT8_T"},
		{"INT8", "telem::INT8_T"},
		{"USINT", "telem::UINT8_T"},
		{"UINT8", "telem::UINT8_T"},
		{"BYTE", "telem::UINT8_T"},
		{"INT", "telem::INT16_T"},
		{"INT16", "telem::INT16_T"},
		{"UINT", "telem::UINT16_T"},
		{"UINT16", "telem::UINT16_T"},
		{"WORD", "telem::UINT16_T"},
		{"DINT", "telem::INT32_T"},
		{"INT32", "telem::INT32_T"},
		{"UDINT", "telem::UINT32_T"},
		{"UINT32", "telem::UINT32_T"},
		{"DWORD", "telem::UINT32_T"},
		{"LINT", "telem::INT64_T"},
		{"INT64", "telem::INT64_T"},
		{"ULINT", "telem::UINT64_T"},
		{"UINT64", "telem::UINT64_T"},
		{"LWORD", "telem::UINT64_T"},
		{"REAL", "telem::FLOAT32_T"},
		{"REAL32", "telem::FLOAT32_T"},
		{"FLOAT", "telem::FLOAT32_T"},
		{"LREAL", "telem::FLOAT64_T"},
		{"REAL64", "telem::FLOAT64_T"},
		{"DOUBLE", "telem::FLOAT64_T"},
		{"unknown", "telem::UINT8_T"},
		{"", "telem::UINT8_T"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapDataType(tt.input)
			if got != tt.expected {
				t.Errorf("mapDataType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidatePDOIndex(t *testing.T) {
	tests := []struct {
		index   uint16
		isInput bool
		valid   bool
	}{
		// TxPDO (input) valid range: 0x1A00-0x1BFF
		{0x1A00, true, true},
		{0x1A50, true, true},
		{0x1BFF, true, true},
		{0x1600, true, false},
		{0x1C00, true, false},
		// RxPDO (output) valid range: 0x1600-0x17FF
		{0x1600, false, true},
		{0x1700, false, true},
		{0x17FF, false, true},
		{0x1800, false, false},
		{0x1A00, false, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := validatePDOIndex(tt.index, tt.isInput)
			if got != tt.valid {
				t.Errorf("validatePDOIndex(0x%04X, %v) = %v, want %v",
					tt.index, tt.isInput, got, tt.valid)
			}
		})
	}
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{`hello "world"`, `hello \"world\"`},
		{`path\to\file`, `path\\to\\file`},
		{`"quoted" and \back`, `\"quoted\" and \\back`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeString(tt.input)
			if got != tt.expected {
				t.Errorf("escapeString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDetectDuplicates(t *testing.T) {
	devices := []ParsedDevice{
		{VendorID: 1, ProductCode: 1, Revision: 1, Name: "Device A", SourceFile: "a.xml"},
		{VendorID: 1, ProductCode: 1, Revision: 1, Name: "Device A Copy", SourceFile: "b.xml"},
		{VendorID: 1, ProductCode: 2, Revision: 1, Name: "Device B", SourceFile: "a.xml"},
		{VendorID: 2, ProductCode: 1, Revision: 1, Name: "Device C", SourceFile: "c.xml"},
	}

	deduplicated, count := detectDuplicates(devices, false)

	if count != 1 {
		t.Errorf("detectDuplicates() returned count = %d, want 1", count)
	}
	if len(deduplicated) != 3 {
		t.Errorf("detectDuplicates() returned %d devices, want 3", len(deduplicated))
	}

	if deduplicated[0].SourceFile != "a.xml" {
		t.Errorf("deduplicated[0].SourceFile = %q, want %q", deduplicated[0].SourceFile, "a.xml")
	}
}

func TestCollectVendors(t *testing.T) {
	devices := []ParsedDevice{
		{VendorID: 1, VendorName: "Vendor A"},
		{VendorID: 1, VendorName: "Vendor A"},
		{VendorID: 2, VendorName: "Vendor B"},
	}

	vendors := collectVendors(devices)

	if len(vendors) != 2 {
		t.Errorf("collectVendors() returned %d vendors, want 2", len(vendors))
	}
	if vendors[1] != "Vendor A" {
		t.Errorf("vendors[1] = %q, want %q", vendors[1], "Vendor A")
	}
	if vendors[2] != "Vendor B" {
		t.Errorf("vendors[2] = %q, want %q", vendors[2], "Vendor B")
	}
}

func TestGenerateStats(t *testing.T) {
	devices := []ParsedDevice{
		{
			VendorID:   1,
			VendorName: "Vendor A",
			InputPDOs:  []ParsedPDOEntry{{DataType: "telem::UINT8_T"}, {DataType: "telem::INT32_T"}},
			OutputPDOs: []ParsedPDOEntry{{DataType: "telem::UINT16_T"}},
		},
		{
			VendorID:   2,
			VendorName: "Vendor B",
			InputPDOs:  []ParsedPDOEntry{{DataType: "telem::INT32_T"}},
		},
	}

	stats := generateStats(devices, 5)

	if stats.DeviceCount != 2 {
		t.Errorf("stats.DeviceCount = %d, want 2", stats.DeviceCount)
	}
	if stats.VendorCount != 2 {
		t.Errorf("stats.VendorCount = %d, want 2", stats.VendorCount)
	}
	if stats.PDOEntryCount != 4 {
		t.Errorf("stats.PDOEntryCount = %d, want 4", stats.PDOEntryCount)
	}
	if stats.InputPDOCount != 3 {
		t.Errorf("stats.InputPDOCount = %d, want 3", stats.InputPDOCount)
	}
	if stats.OutputPDOCount != 1 {
		t.Errorf("stats.OutputPDOCount = %d, want 1", stats.OutputPDOCount)
	}
	if stats.DuplicatesFound != 5 {
		t.Errorf("stats.DuplicatesFound = %d, want 5", stats.DuplicatesFound)
	}
	if stats.DataTypes["telem::INT32_T"] != 2 {
		t.Errorf("stats.DataTypes[INT32_T] = %d, want 2", stats.DataTypes["telem::INT32_T"])
	}
}

func TestGenerateCPP(t *testing.T) {
	devices := []ParsedDevice{
		{
			VendorID:    0x12345678,
			VendorName:  "Test Vendor",
			ProductCode: 1,
			Revision:    1,
			Name:        "Test Device",
			InputPDOs: []ParsedPDOEntry{
				{PDOIndex: 0x1A00, Index: 0x6000, SubIndex: 1, BitLength: 16,
					Name: "Input", DataType: "telem::UINT16_T"},
			},
			OutputPDOs: []ParsedPDOEntry{
				{PDOIndex: 0x1600, Index: 0x7000, SubIndex: 1, BitLength: 8,
					Name: "Output", DataType: "telem::UINT8_T"},
			},
		},
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "known_devices.h")

	err := generateCPP(devices, outputPath)
	if err != nil {
		t.Fatalf("generateCPP() error = %v", err)
	}

	// Check header file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Header file was not created")
	}

	// Check cpp file exists
	cppPath := filepath.Join(tmpDir, "known_devices.cpp")
	if _, err := os.Stat(cppPath); os.IsNotExist(err) {
		t.Error("CPP file was not created")
	}

	// Read header and check for key contents
	hdrContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read header: %v", err)
	}
	hdrStr := string(hdrContent)

	expectedStrings := []string{
		"DEVICE_COUNT = 1",
		"VENDOR_COUNT = 1",
		"lookup_device_pdos",
		"vendor_name",
		"is_device_known",
	}
	for _, s := range expectedStrings {
		if !containsString(hdrStr, s) {
			t.Errorf("Header missing expected content: %q", s)
		}
	}

	// Read cpp and check for key contents
	cppContent, err := os.ReadFile(cppPath)
	if err != nil {
		t.Fatalf("Failed to read cpp: %v", err)
	}
	cppStr := string(cppContent)

	expectedCppStrings := []string{
		"constexpr",
		"DEVICE_INDEX",
		"VENDORS",
		"0x12345678",
		"Test Vendor",
	}
	for _, s := range expectedCppStrings {
		if !containsString(cppStr, s) {
			t.Errorf("CPP missing expected content: %q", s)
		}
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
