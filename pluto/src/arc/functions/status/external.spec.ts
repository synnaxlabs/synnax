// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { SPEC, SYMBOLS } from "@/arc/functions/status/external";

describe("status.set SPEC", () => {
  it("Should expose the canonical key", () => {
    expect(SPEC.key).toBe("status.set");
  });

  it("Should sit at zIndex 100", () => {
    expect(SPEC.zIndex).toBe(100);
  });

  it("Should default to the post-RFC-0037 config shape", () => {
    expect(SPEC.defaultProps()).toEqual({
      key_or_name: "",
      variant: "success",
      message: "Notification",
    });
  });

  it("Should expose Form, Symbol, and a Preview that reuses Symbol", () => {
    expect(SPEC.Form).toBeDefined();
    expect(SPEC.Symbol).toBeDefined();
    expect(SPEC.Preview).toBe(SPEC.Symbol);
  });
});

describe("SYMBOLS map", () => {
  it("Should register SPEC under its key", () => {
    expect(SYMBOLS["status.set"]).toBe(SPEC);
  });
});
