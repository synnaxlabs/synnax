// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { flag } from "@/flags";

describe("flag", () => {
  it("should enable a flag in production only for the exact value true", () => {
    expect(flag("true", false)).toBe(true);
    expect(flag("TRUE", false)).toBe(false);
    expect(flag("1", false)).toBe(false);
    expect(flag("", false)).toBe(false);
    expect(flag(undefined, false)).toBe(false);
  });

  it("should enable every flag in a dev build", () => {
    expect(flag(undefined, true)).toBe(true);
    expect(flag("false", true)).toBe(true);
  });
});
