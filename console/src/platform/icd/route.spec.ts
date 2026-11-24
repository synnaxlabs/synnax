// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

describe("route", () => {
  const present = (pointer: string): { match: icd.JSONMatch } => ({
    match: { type: "present", pointer },
  });
  const fields = (...pointers: string[]) => pointers.map((pointer) => ({ pointer }));

  it("should give the message whose present rule is at a field", () => {
    const rpm = present("/rpm");
    expect(ICD.route(fields("/t", "/rpm"), [present("/fault"), rpm])).toBe(rpm);
  });

  it("should give the message whose present rule holds a field", () => {
    const status = present("/status");
    expect(ICD.route(fields("/status/code"), [status])).toBe(status);
    expect(ICD.route(fields("/statuses"), [status])).toBeUndefined();
  });

  it("should give the message with no rule when no present rule matches", () => {
    const rest = {};
    expect(ICD.route(fields("/t"), [present("/rpm"), rest])).toBe(rest);
  });

  it("should give no message when none matches and each has a rule", () => {
    expect(ICD.route(fields("/t"), [present("/rpm")])).toBeUndefined();
  });
});
