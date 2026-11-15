// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Arc } from "@/components/arc";

describe("Arc.diagnostic", () => {
  it("should give the code, severity, and message of a kind", () => {
    expect(Arc.diagnostic("ARC4004")).toEqual({
      code: "ARC4004",
      severity: "warning",
      message: "declare variables at the top of a stage or sequence",
    });
  });

  it("should put the values in place of the verbs", () => {
    expect(Arc.diagnostic("ARC4003", ["<old>", "<new>"]).message).toEqual(
      "'<old>' is deprecated, use '<new>' instead",
    );
  });

  it("should put the value of a quoted verb in quotes", () => {
    expect(Arc.diagnostic("ARC5001", ["<m>"]).message).toEqual(
      'imported module "<m>" is unused',
    );
  });

  it("should give one message for each list of values", () => {
    expect(Arc.diagnostic("ARC6002", ["division"], ["modulo"]).message).toEqual([
      "integer division by zero",
      "integer modulo by zero",
    ]);
  });

  it("should give no message for a kind that takes it from an error", () => {
    expect(Arc.diagnostic("ARC3002")).toEqual({ code: "ARC3002", severity: "error" });
  });

  it("should throw for a code that no kind has", () => {
    expect(() => Arc.diagnostic("ARC9999")).toThrow(
      "no Arc diagnostic kind has the code ARC9999",
    );
  });

  it("should throw when a verb has no value", () => {
    expect(() => Arc.diagnostic("ARC4003", ["<old>"])).toThrow(
      "ARC4003 has a verb with no value",
    );
  });

  it("should throw when a value has no verb", () => {
    expect(() => Arc.diagnostic("ARC4004", ["<x>"])).toThrow(
      "ARC4004 has a value with no verb",
    );
  });
});
