// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FLAG_PORTAL } from "astro:env/client";
import { describe, expect, it } from "vitest";

import { enabled } from "@/flags";

describe("flags", () => {
  it("should read each flag from the environment schema of the site", () => {
    expect(typeof FLAG_PORTAL).toBe("boolean");
  });

  it("should reject a flag that is not registered", () => {
    expect(() => enabled("dashboards")).toThrow("unknown flag: dashboards");
  });
});
