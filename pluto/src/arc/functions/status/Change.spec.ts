// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { config } from "@/arc/functions/status/Change";

describe("Change config schema", () => {
  it("Should parse a valid configuration", () => {
    expect(
      config.parse({ key_or_name: "alarm", variant: "success", message: "ok" }),
    ).toEqual({ key_or_name: "alarm", variant: "success", message: "ok" });
  });

  it("Should accept each allowed variant", () => {
    for (const variant of status.VARIANTS)
      expect(config.parse({ key_or_name: "k", variant, message: "m" })).toEqual({
        key_or_name: "k",
        variant,
        message: "m",
      });
  });

  it("Should reject an unknown variant", () => {
    const result = config.safeParse({
      key_or_name: "k",
      variant: "bogus",
      message: "m",
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toContainEqual(
      expect.objectContaining({ code: "invalid_value", path: ["variant"] }),
    );
  });

  it("Should reject a missing key_or_name", () => {
    const result = config.safeParse({ variant: "info", message: "m" });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toContainEqual(
      expect.objectContaining({ code: "invalid_type", path: ["key_or_name"] }),
    );
  });

  it("Should reject a missing message", () => {
    const result = config.safeParse({ key_or_name: "k", variant: "info" });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toContainEqual(
      expect.objectContaining({ code: "invalid_type", path: ["message"] }),
    );
  });

  it("Should reject a missing variant", () => {
    const result = config.safeParse({ key_or_name: "k", message: "m" });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toContainEqual(
      expect.objectContaining({ code: "invalid_value", path: ["variant"] }),
    );
  });
});
