// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";

const float = (...values: number[]): Task.Payload.ChannelField => ({
  type: "float",
  enumValues: values.map((value, i) => ({ label: `E${i}`, value })),
});

const integer = (...values: string[]): Task.Payload.ChannelField => ({
  type: "integer",
  enumValues: values.map((value, i) => ({ label: `E${i}`, value })),
});

const unsigned = (...values: string[]): Task.Payload.ChannelField => ({
  type: "unsigned",
  enumValues: values.map((value, i) => ({ label: `E${i}`, value })),
});

describe("kindOptions", () => {
  it("should give a timestamp the ISO 8601 time format", () => {
    expect(Task.Payload.kindOptions(float(1), "timestamp")).toEqual({
      timeFormat: "iso8601",
    });
  });

  it.each(["string", "boolean", "json", "uuid", "bytes"] as const)(
    "should give %s no options",
    (kind) => {
      expect(Task.Payload.kindOptions(float(1), kind)).toEqual({});
    },
  );

  it("should keep a float entry as a float", () => {
    expect(Task.Payload.kindOptions(float(0.5), "float")).toEqual({
      enumValues: [{ label: "E0", value: 0.5 }],
    });
  });

  it("should keep whole float entries as integer text and drop the rest", () => {
    expect(Task.Payload.kindOptions(float(-3, 0.5), "integer")).toEqual({
      enumValues: [{ label: "E0", value: "-3" }],
    });
  });

  it("should drop a negative entry as unsigned", () => {
    expect(Task.Payload.kindOptions(integer("-1", "7"), "unsigned")).toEqual({
      enumValues: [{ label: "E1", value: "7" }],
    });
  });

  it("should keep an integer up to 2^53 as a float and drop a larger one", () => {
    expect(
      Task.Payload.kindOptions(
        integer("9007199254740992", "9007199254740993"),
        "float",
      ),
    ).toEqual({ enumValues: [{ label: "E0", value: 9007199254740992 }] });
  });

  it("should drop an unsigned entry past the int64 range as an integer", () => {
    expect(
      Task.Payload.kindOptions(
        unsigned("9223372036854775807", "9223372036854775808"),
        "integer",
      ),
    ).toEqual({ enumValues: [{ label: "E0", value: "9223372036854775807" }] });
  });

  it("should carry no entries from a field that is not numeric", () => {
    const field: Task.Payload.ChannelField = {
      type: "string",
      enumValues: [{ label: "ON", value: "1" }],
    };
    expect(Task.Payload.kindOptions(field, "integer")).toEqual({ enumValues: [] });
  });
});
