// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { validateAction } from "@/validate";

describe("validate", () => {
  it("should throw an error if an action is undefined", () => {
    expect(() => validateAction({})).toThrowError();
  });
  it("should throw an error if an action type is undefined or an empty string", () => {
    expect(() => validateAction({ action: { type: "" } })).toThrowError();
    // @ts-expect-error - expect this to fail
    expect(() => validateAction({ action: { type: undefined } })).toThrowError();
  });
  it("should not throw an error for a valid action", () => {
    expect(() => validateAction({ action: { type: "type" } })).not.toThrowError();
  });
});
