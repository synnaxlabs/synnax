// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { config } from "@/arc/functions/status/Change";

describe("Change config schema", () => {
  it("Should accept a valid configuration", () => {
    expect(() =>
      config.parse({ key_or_name: "alarm", variant: "success", message: "ok" }),
    ).not.toThrow();
  });

  it("Should accept each allowed variant", () => {
    for (const variant of [
      "success",
      "info",
      "warning",
      "error",
      "loading",
      "disabled",
    ]) 
      expect(() =>
        config.parse({ key_or_name: "k", variant, message: "m" }),
      ).not.toThrow();
    
  });

  it("Should reject an unknown variant", () => {
    expect(() =>
      config.parse({ key_or_name: "k", variant: "bogus", message: "m" }),
    ).toThrow();
  });

  it("Should reject a missing key_or_name", () => {
    expect(() => config.parse({ variant: "info", message: "m" })).toThrow();
  });

  it("Should reject a missing message", () => {
    expect(() => config.parse({ key_or_name: "k", variant: "info" })).toThrow();
  });

  it("Should reject a missing variant", () => {
    expect(() => config.parse({ key_or_name: "k", message: "m" })).toThrow();
  });
});
