// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ALERT_SCHEMAS, ZERO_ALERT_TASK_CONFIG } from "@/hardware/pagerduty/task/types";

describe("PagerDuty Alert Task Types", () => {
  describe("config schema", () => {
    it("should accept a valid config", () => {
      const config = {
        routingKey: "a".repeat(32),
        autoStart: true,
        alerts: [
          {
            key: "alert-1",
            status: "my-status",
            treatErrorAsCritical: true,
            component: "sensor",
            group: "hw",
            class: "temp",
            enabled: true,
          },
        ],
      };
      const result = ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.routingKey).toBe("a".repeat(32));
        expect(result.data.autoStart).toBe(true);
        expect(result.data.alerts).toHaveLength(1);
        expect(result.data.alerts[0].status).toBe("my-status");
        expect(result.data.alerts[0].treatErrorAsCritical).toBe(true);
      }
    });

    it("should reject a routing key that is not 32 characters", () => {
      const config = {
        routingKey: "tooshort",
        alerts: [{ key: "a", status: "s", enabled: true }],
      };
      const result = ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should reject an empty routing key", () => {
      const result = ALERT_SCHEMAS.config.safeParse({
        ...ZERO_ALERT_TASK_CONFIG,
        alerts: [{ key: "a", status: "s", enabled: true }],
      });
      expect(result.success).toBe(false);
    });

    it("should default autoStart to false", () => {
      const config = {
        routingKey: "b".repeat(32),
        alerts: [{ key: "a", status: "s", enabled: true }],
      };
      const result = ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(true);
      if (result.success) expect(result.data.autoStart).toBe(false);
    });

    it("should default alerts to an empty array", () => {
      const config = { routingKey: "c".repeat(32) };
      const result = ALERT_SCHEMAS.config.safeParse(config);
      // Empty array fails the refine (at least one enabled alert required)
      expect(result.success).toBe(false);
    });

    it("should reject when no alerts are enabled", () => {
      const config = {
        routingKey: "d".repeat(32),
        alerts: [
          { key: "a", status: "s1", enabled: false },
          { key: "b", status: "s2", enabled: false },
        ],
      };
      const result = ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });
  });

  describe("alert config schema", () => {
    it("should reject an empty status key", () => {
      const config = {
        routingKey: "e".repeat(32),
        alerts: [{ key: "a", status: "", enabled: true }],
      };
      const result = ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(false);
    });

    it("should default optional fields", () => {
      const config = {
        routingKey: "f".repeat(32),
        alerts: [{ key: "a", status: "my-status" }],
      };
      const result = ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(true);
      if (result.success) {
        const alert = result.data.alerts[0];
        expect(alert.treatErrorAsCritical).toBe(false);
        expect(alert.component).toBe("");
        expect(alert.group).toBe("");
        expect(alert.class).toBe("");
        expect(alert.enabled).toBe(true);
      }
    });
  });
});
