// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  convertSeriesToSupportedGL,
  resolveGLDataType,
} from "@/telem/aether/convertSeries";

describe("convertSeriesToSupportedGL", () => {
  it("returns the original series for variable data types", () => {
    const series = new Series({
      data: ["a", "b", "c"],
      dataType: DataType.STRING,
    });
    expect(convertSeriesToSupportedGL(series)).toBe(series);
  });

  it("returns the original series for UINT8", () => {
    const series = new Series({
      data: new Uint8Array([1, 2, 3]),
      dataType: DataType.UINT8,
    });
    expect(convertSeriesToSupportedGL(series)).toBe(series);
  });

  it("converts a FLOAT64 series to FLOAT32 with no offset", () => {
    const series = new Series({
      data: new Float64Array([1.5, 2.5, 3.5]),
      dataType: DataType.FLOAT64,
    });
    const result = convertSeriesToSupportedGL(series);
    expect(result.dataType.equals(DataType.FLOAT32)).toBe(true);
    expect(result.at(0)).toBe(1.5);
    expect(result.at(1)).toBe(2.5);
    expect(result.at(2)).toBe(3.5);
  });

  it("defaults the offset to the first sample for INT64 to preserve precision above 2^53", () => {
    const first = 1778020940471336960n;
    const series = new Series({
      data: [first, first + 1n, first + 2n],
      dataType: DataType.INT64,
    });
    const result = convertSeriesToSupportedGL(series);
    expect(result.dataType.equals(DataType.FLOAT32)).toBe(true);
    expect(result.sampleOffset).toBe(first);
    expect(result.at(0)).toBe(first);
    expect(result.at(1)).toBe(first + 1n);
    expect(result.at(2)).toBe(first + 2n);
  });

  it("defaults the offset to the first sample for UINT64 to preserve precision above 2^53", () => {
    const first = 1778020940471336960n;
    const series = new Series({
      data: [first, first + 1n, first + 2n],
      dataType: DataType.UINT64,
    });
    const result = convertSeriesToSupportedGL(series);
    expect(result.dataType.equals(DataType.FLOAT32)).toBe(true);
    expect(result.sampleOffset).toBe(first);
    expect(result.at(0)).toBe(first);
  });

  it("defaults the offset to the first sample for TIMESTAMP to preserve precision above 2^53", () => {
    const first = 1778020940471336960n;
    const series = new Series({
      data: [first, first + 1n, first + 2n],
      dataType: DataType.TIMESTAMP,
    });
    const result = convertSeriesToSupportedGL(series);
    expect(result.dataType.equals(DataType.FLOAT32)).toBe(true);
    expect(result.sampleOffset).toBe(first);
    expect(result.at(0)).toBe(first);
  });

  it("respects an explicitly provided offset for bigint series", () => {
    const offset = 1778020940471336000n;
    const first = 1778020940471336960n;
    const series = new Series({
      data: [first, first + 1n, first + 2n],
      dataType: DataType.INT64,
    });
    const result = convertSeriesToSupportedGL(series, offset);
    expect(result.sampleOffset).toBe(offset);
  });
});

describe("resolveGLDataType", () => {
  it("returns variable data types unchanged", () => {
    expect(resolveGLDataType(DataType.STRING).equals(DataType.STRING)).toBe(true);
    expect(resolveGLDataType(DataType.JSON).equals(DataType.JSON)).toBe(true);
  });

  it("returns UINT8 unchanged", () => {
    expect(resolveGLDataType(DataType.UINT8).equals(DataType.UINT8)).toBe(true);
  });

  it("returns FLOAT32 for any other fixed-density data type", () => {
    expect(resolveGLDataType(DataType.FLOAT64).equals(DataType.FLOAT32)).toBe(true);
    expect(resolveGLDataType(DataType.INT64).equals(DataType.FLOAT32)).toBe(true);
    expect(resolveGLDataType(DataType.UINT64).equals(DataType.FLOAT32)).toBe(true);
    expect(resolveGLDataType(DataType.TIMESTAMP).equals(DataType.FLOAT32)).toBe(true);
    expect(resolveGLDataType(DataType.INT32).equals(DataType.FLOAT32)).toBe(true);
  });
});
