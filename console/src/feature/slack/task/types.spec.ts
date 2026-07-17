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

describe("Slack Alert Task Types", () => {
  describe("config schema", () => {
    it("should accept a valid config", () => {
      const config = {
        device: "dev-1",
        channel: "#alerts",
        statuses: ["s1", "s2"],
        autoStart: true,
      };
      const result = Slack.Task.ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.device).toBe("dev-1");
        expect(result.data.channel).toBe("#alerts");
        expect(result.data.statuses).toEqual(["s1", "s2"]);
        expect(result.data.autoStart).toBe(true);
      }
    });

    it("should reject a missing device", () => {
      const result = Slack.Task.ALERT_SCHEMAS.config.safeParse({
        device: "",
        channel: "#alerts",
        statuses: ["s1"],
      });
      expect(result.success).toBe(false);
    });

    it("should reject a missing channel", () => {
      const result = Slack.Task.ALERT_SCHEMAS.config.safeParse({
        device: "dev-1",
        channel: "",
        statuses: ["s1"],
      });
      expect(result.success).toBe(false);
    });

    it("should reject an empty statuses list", () => {
      const result = Slack.Task.ALERT_SCHEMAS.config.safeParse({
        device: "dev-1",
        channel: "#alerts",
        statuses: [],
      });
      expect(result.success).toBe(false);
    });

    it("should reject an empty status key", () => {
      const result = Slack.Task.ALERT_SCHEMAS.config.safeParse({
        device: "dev-1",
        channel: "#alerts",
        statuses: [""],
      });
      expect(result.success).toBe(false);
    });

    it("should default autoStart to false", () => {
      const result = Slack.Task.ALERT_SCHEMAS.config.safeParse({
        device: "dev-1",
        channel: "#alerts",
        statuses: ["s1"],
      });
      expect(result.success).toBe(true);
      if (result.success) expect(result.data.autoStart).toBe(false);
    });
  });
});
