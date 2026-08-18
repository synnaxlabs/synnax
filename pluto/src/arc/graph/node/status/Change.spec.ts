// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { configZ } from "@/arc/graph/node/status/config";

const TYPE = "status.set" as const;

describe("Change config schema", () => {
  it("Should parse a valid configuration", () => {
    expect(
      configZ.parse({
        type: TYPE,
        key_or_name: "alarm",
        variant: "success",
        message: "ok",
      }),
    ).toEqual({ type: TYPE, key_or_name: "alarm", variant: "success", message: "ok" });
  });

  it("Should accept each allowed variant", () => {
    for (const variant of status.VARIANTS)
      expect(
        configZ.parse({ type: TYPE, key_or_name: "k", variant, message: "m" }),
      ).toEqual({ type: TYPE, key_or_name: "k", variant, message: "m" });
  });

  it("Should reject an unknown variant", () => {
    const result = configZ.safeParse({
      type: TYPE,
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
    const result = configZ.safeParse({ type: TYPE, variant: "info", message: "m" });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toContainEqual(
      expect.objectContaining({ code: "invalid_type", path: ["key_or_name"] }),
    );
  });

  it("Should reject a missing message", () => {
    const result = configZ.safeParse({ type: TYPE, key_or_name: "k", variant: "info" });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toContainEqual(
      expect.objectContaining({ code: "invalid_type", path: ["message"] }),
    );
  });

  it("Should reject a missing variant", () => {
    const result = configZ.safeParse({ type: TYPE, key_or_name: "k", message: "m" });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toContainEqual(
      expect.objectContaining({ code: "invalid_value", path: ["variant"] }),
    );
  });
});
