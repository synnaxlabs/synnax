// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { PagerDuty } from "@/feature/pagerduty";

const ROUTING_KEY = uuid.create();

describe("PagerDuty Alert Task Types", () => {
  describe("config schema", () => {
    it("should accept a valid config", () => {
      const config = {
        routingKey: ROUTING_KEY,
        autoStart: true,
        alerts: [
          {
            key: "alert-1",
            status: "my-status",
            errorsCritical: true,
            component: "sensor",
            group: "hw",
            class: "temp",
            disabled: false,
          },
        ],
      };
      const result = PagerDuty.Task.ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.routingKey).toBe(ROUTING_KEY);
        expect(result.data.autoStart).toBe(true);
        expect(result.data.alerts).toHaveLength(1);
        expect(result.data.alerts[0].status).toBe("my-status");
        expect(result.data.alerts[0].errorsCritical).toBe(true);
      }
    });

    it("should reject a routing key that is not a secret key", () => {
      const config = {
        routingKey: "R022XIJR9M266DX570EVE6EXP1AFBN6D",
        alerts: [{ key: "a", status: "s", disabled: false }],
      };
      expect(PagerDuty.Task.ALERT_SCHEMAS.config.safeParse(config).success).toBe(false);
    });

    it("should require a routing key at deploy", () => {
      const result = PagerDuty.Task.deployAlertTaskConfigZ.safeParse({
        ...PagerDuty.Task.ALERT_SCHEMAS.config.parse({}),
        alerts: [{ key: "a", status: "s", disabled: false }],
      });
      expect(result.error?.issues).toEqual([
        expect.objectContaining({
          path: ["routingKey"],
          message: "Routing key is required",
        }),
      ]);
    });

    it("should default autoStart to false", () => {
      const config = {
        routingKey: ROUTING_KEY,
        alerts: [{ key: "a", status: "s", disabled: false }],
      };
      const result = PagerDuty.Task.ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(true);
      if (result.success) expect(result.data.autoStart).toBe(false);
    });

    it("should default alerts to an empty array", () => {
      const config = { routingKey: ROUTING_KEY };
      const result = PagerDuty.Task.ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(true);
      if (result.success) expect(result.data.alerts).toEqual([]);
    });

    it("should reject an empty alert list at deploy", () => {
      const config = { routingKey: ROUTING_KEY, alerts: [] };
      const result = PagerDuty.Task.deployAlertTaskConfigZ.safeParse(config);
      // Empty array fails the refine (at least one enabled alert required)
      expect(result.success).toBe(false);
    });

    it("should reject when no alerts are enabled", () => {
      const config = {
        routingKey: ROUTING_KEY,
        alerts: [
          { key: "a", status: "s1", disabled: true },
          { key: "b", status: "s2", disabled: true },
        ],
      };
      expect(PagerDuty.Task.deployAlertTaskConfigZ.validate(config)).toBe(false);
    });
  });

  describe("alert config schema", () => {
    it("should reject an empty status key", () => {
      const config = {
        routingKey: ROUTING_KEY,
        alerts: [{ key: "a", status: "", disabled: false }],
      };
      expect(PagerDuty.Task.deployAlertTaskConfigZ.validate(config)).toBe(false);
    });

    it("should default optional fields", () => {
      const config = {
        routingKey: ROUTING_KEY,
        alerts: [{ key: "a", status: "my-status" }],
      };
      const result = PagerDuty.Task.ALERT_SCHEMAS.config.safeParse(config);
      expect(result.success).toBe(true);
      if (result.success) {
        const alert = result.data.alerts[0];
        expect(alert.errorsCritical).toBe(false);
        expect(alert.component).toBe("");
        expect(alert.group).toBe("");
        expect(alert.class).toBe("");
        expect(alert.disabled).toBe(false);
      }
    });
  });
});

describe("draft configs", () => {
  // Drafts persist server-side before configuration, so the shape schema must
  // accept every zero config; retrieve parses with it.
  it("should accept the zero alert config", () => {
    expect(
      PagerDuty.Task.ALERT_SCHEMAS.config.safeParse(
        PagerDuty.Task.ALERT_SCHEMAS.config.parse({}),
      ).success,
    ).toBe(true);
  });
});
