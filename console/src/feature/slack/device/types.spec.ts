// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Slack } from "@/feature/slack";

describe("Slack Device Types", () => {
  describe("properties schema", () => {
    it("should accept valid properties", () => {
      const result = Slack.Device.SCHEMAS.properties.safeParse({
        botToken: "xoxb-token",
        version: 1,
      });
      expect(result.success).toBe(true);
    });

    it("should reject an empty bot token", () => {
      const result = Slack.Device.SCHEMAS.properties.safeParse({
        botToken: "",
        version: 1,
      });
      expect(result.success).toBe(false);
    });
  });
});
