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

import { Task } from "@/platform/task";

describe("readOptionsOf", () => {
  it("should keep the options of a field of the same kind", () => {
    const previous: Task.Payload.ReadOptions = {
      type: "integer",
      dataType: DataType.INT64,
      enumValues: [{ label: "ON", value: "1" }],
      fill: "-1",
    };
    expect(Task.Payload.readOptionsOf("int16", previous)).toEqual({
      ...previous,
      dataType: DataType.INT16,
    });
  });

  it("should carry the entries and fill that a new numeric kind holds", () => {
    expect(
      Task.Payload.readOptionsOf("uint8", {
        type: "float",
        enumValues: [
          { label: "ON", value: 1 },
          { label: "HALF", value: 0.5 },
          { label: "NEG", value: -1 },
        ],
        fill: 2,
      }),
    ).toEqual({
      type: "unsigned",
      dataType: DataType.UINT8,
      enumValues: [{ label: "ON", value: "1" }],
      fill: "2",
    });
  });

  it("should drop an entry and a fill that do not fit the new width", () => {
    expect(
      Task.Payload.readOptionsOf("uint8", {
        type: "float",
        enumValues: [
          { label: "MAX", value: 255 },
          { label: "BIG", value: 256 },
        ],
        fill: 300,
      }),
    ).toEqual({
      type: "unsigned",
      dataType: DataType.UINT8,
      enumValues: [{ label: "MAX", value: "255" }],
      fill: undefined,
    });
    expect(
      Task.Payload.readOptionsOf("float32", {
        type: "integer",
        enumValues: [{ label: "ONE", value: "1" }],
        fill: "-1",
      }),
    ).toEqual({
      type: "float",
      dataType: DataType.FLOAT32,
      enumValues: [{ label: "ONE", value: 1 }],
      fill: -1,
    });
  });

  it("should drop an integer entry that a float does not hold exactly", () => {
    expect(
      Task.Payload.readOptionsOf("float64", {
        type: "integer",
        enumValues: [
          { label: "MAX", value: "9223372036854775807" },
          { label: "ONE", value: "1" },
          { label: "PARTIAL", value: "" },
        ],
        fill: "9007199254740993",
      }),
    ).toEqual({
      type: "float",
      dataType: DataType.FLOAT64,
      enumValues: [{ label: "ONE", value: 1 }],
      fill: undefined,
    });
  });

  it("should drop a NaN fill on an integer kind", () => {
    expect(
      Task.Payload.readOptionsOf("int32", { type: "float", enumValues: [], fill: NaN }),
    ).toEqual({
      type: "integer",
      dataType: DataType.INT32,
      enumValues: [],
      fill: undefined,
    });
  });

  it("should start a field of another kind with no options", () => {
    expect(
      Task.Payload.readOptionsOf("string", {
        type: "unsigned",
        enumValues: [{ label: "ON", value: "1" }],
      }),
    ).toEqual({ type: "string", enumValues: [] });
  });
});
