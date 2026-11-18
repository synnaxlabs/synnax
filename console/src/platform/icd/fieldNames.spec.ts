// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

describe("ICD.fieldNames", () => {
  it("should join the tokens of a pointer with an underscore", () => {
    expect(ICD.fieldNames(["/motor/speed"])).toEqual(["motor_speed"]);
  });

  it("should name a pointer with no name value", () => {
    expect(ICD.fieldNames(["", "/"])).toEqual(["value", "value_2"]);
  });

  it("should unescape the tokens of a pointer", () => {
    expect(ICD.fieldNames(["/a~1b", "/c~0d"])).toEqual(["a/b", "c~d"]);
  });

  it("should number a name that another pointer took", () => {
    expect(ICD.fieldNames(["/a/b", "/a_b", "/a_b"])).toEqual(["a_b", "a_b_2", "a_b_3"]);
  });

  it("should number a name whose channel name another name takes", () => {
    expect(ICD.fieldNames(["/a~1b", "/a_b"])).toEqual(["a/b", "a_b_2"]);
    expect(ICD.fieldNames(["/motor_speed"], ["motor-speed"])).toEqual([
      "motor_speed_2",
    ]);
  });

  it("should number a name that a field already holds", () => {
    expect(ICD.fieldNames(["/speed"], ["speed", "speed_2"])).toEqual(["speed_3"]);
  });
});
