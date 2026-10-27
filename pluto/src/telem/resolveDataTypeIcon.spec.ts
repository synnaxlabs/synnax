// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Telem } from "@/telem";

describe("resolveDataTypeIcon", () => {
  it("should return the Boolean icon for BOOLEAN data type", () => {
    expect(Telem.resolveDataTypeIcon(DataType.BOOLEAN)).toBe(Icon.Boolean);
  });

  it("should return the JSON icon for JSON data type", () => {
    expect(Telem.resolveDataTypeIcon(DataType.JSON)).toBe(Icon.JSON);
  });

  it("should return the Binary icon for BYTES data type", () => {
    expect(Telem.resolveDataTypeIcon(DataType.BYTES)).toBe(Icon.Binary);
  });

  it("should return the Binary icon for integer data types", () => {
    expect(Telem.resolveDataTypeIcon(DataType.INT8)).toBe(Icon.Binary);
    expect(Telem.resolveDataTypeIcon(DataType.INT16)).toBe(Icon.Binary);
    expect(Telem.resolveDataTypeIcon(DataType.INT32)).toBe(Icon.Binary);
    expect(Telem.resolveDataTypeIcon(DataType.INT64)).toBe(Icon.Binary);
    expect(Telem.resolveDataTypeIcon(DataType.UINT8)).toBe(Icon.Binary);
    expect(Telem.resolveDataTypeIcon(DataType.UINT16)).toBe(Icon.Binary);
    expect(Telem.resolveDataTypeIcon(DataType.UINT32)).toBe(Icon.Binary);
    expect(Telem.resolveDataTypeIcon(DataType.UINT64)).toBe(Icon.Binary);
  });

  it("should return the Decimal icon for float data types", () => {
    expect(Telem.resolveDataTypeIcon(DataType.FLOAT32)).toBe(Icon.Decimal);
    expect(Telem.resolveDataTypeIcon(DataType.FLOAT64)).toBe(Icon.Decimal);
  });

  it("should return the String icon for STRING data type", () => {
    expect(Telem.resolveDataTypeIcon(DataType.STRING)).toBe(Icon.String);
  });

  it("should return the String icon for UUID data type", () => {
    expect(Telem.resolveDataTypeIcon(DataType.UUID)).toBe(Icon.String);
  });

  it("should return the Time icon for TIMESTAMP data type", () => {
    expect(Telem.resolveDataTypeIcon(DataType.TIMESTAMP)).toBe(Icon.Time);
  });

  it("should return undefined for UNKNOWN data type", () => {
    expect(Telem.resolveDataTypeIcon(DataType.UNKNOWN)).toBeUndefined();
  });
});
