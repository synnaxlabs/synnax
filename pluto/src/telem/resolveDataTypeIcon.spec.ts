// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Icon } from "@/icon";
import { resolveDataTypeIcon } from "@/telem/resolveDataTypeIcon";

describe("resolveDataTypeIcon", () => {
  it("should return the JSON icon for JSON data type", () => {
    expect(resolveDataTypeIcon(DataType.JSON)).toBe(Icon.JSON);
  });

  it("should return the Binary icon for BYTES data type", () => {
    expect(resolveDataTypeIcon(DataType.BYTES)).toBe(Icon.Binary);
  });

  it("should return the Binary icon for integer data types", () => {
    expect(resolveDataTypeIcon(DataType.INT8)).toBe(Icon.Binary);
    expect(resolveDataTypeIcon(DataType.INT16)).toBe(Icon.Binary);
    expect(resolveDataTypeIcon(DataType.INT32)).toBe(Icon.Binary);
    expect(resolveDataTypeIcon(DataType.INT64)).toBe(Icon.Binary);
    expect(resolveDataTypeIcon(DataType.UINT8)).toBe(Icon.Binary);
    expect(resolveDataTypeIcon(DataType.UINT16)).toBe(Icon.Binary);
    expect(resolveDataTypeIcon(DataType.UINT32)).toBe(Icon.Binary);
    expect(resolveDataTypeIcon(DataType.UINT64)).toBe(Icon.Binary);
  });

  it("should return the Decimal icon for float data types", () => {
    expect(resolveDataTypeIcon(DataType.FLOAT32)).toBe(Icon.Decimal);
    expect(resolveDataTypeIcon(DataType.FLOAT64)).toBe(Icon.Decimal);
  });

  it("should return the String icon for STRING data type", () => {
    expect(resolveDataTypeIcon(DataType.STRING)).toBe(Icon.String);
  });

  it("should return the String icon for UUID data type", () => {
    expect(resolveDataTypeIcon(DataType.UUID)).toBe(Icon.String);
  });

  it("should return the Time icon for TIMESTAMP data type", () => {
    expect(resolveDataTypeIcon(DataType.TIMESTAMP)).toBe(Icon.Time);
  });

  it("should return undefined for UNKNOWN data type", () => {
    expect(resolveDataTypeIcon(DataType.UNKNOWN)).toBeUndefined();
  });
});
