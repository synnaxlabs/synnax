// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, status } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Device } from "@/platform/device";

const createRack = (key: rack.Key, integrations: string[], running: boolean) =>
  rack.payloadZ.parse({
    key,
    name: `Rack ${key}`,
    role: uuid.create(),
    integrations,
    status: running
      ? status.create<typeof rack.statusDetailsZ>({
          variant: "success",
          message: "Running",
          details: { running: true },
        })
      : undefined,
  });

const createRackWithStatus = (variant: status.Variant, running: boolean) =>
  rack.payloadZ.parse({
    key: 1,
    name: "Rack 1",
    role: uuid.create(),
    integrations: ["opc"],
    status: status.create<typeof rack.statusDetailsZ>({
      variant,
      message: "Heartbeat",
      details: { running },
    }),
  });

describe("isRunning", () => {
  it("should return true for a warning from a running Driver", () => {
    expect(Device.isRunning(createRackWithStatus("warning", true))).toBe(true);
  });

  it("should return false for a success without a running Driver", () => {
    expect(Device.isRunning(createRackWithStatus("success", false))).toBe(false);
  });
});

describe("isReady", () => {
  it("should return false for an error from a running Driver", () => {
    expect(Device.isReady(createRackWithStatus("error", true))).toBe(false);
  });
});

const OPC = { integration: "opc", name: "OPC UA" };

describe("getBlockedReason", () => {
  describe("across every rack", () => {
    it("should return null when one running rack has the integration", () => {
      const racks = [createRack(1, [], true), createRack(2, ["opc"], true)];
      expect(Device.getBlockedReason(racks, OPC)).toBeNull();
    });

    it("should block when the only rack with the integration is not running", () => {
      const racks = [createRack(1, ["opc"], false), createRack(2, [], true)];
      expect(Device.getBlockedReason(racks, OPC)).toBe(
        "No running Driver has OPC UA enabled",
      );
    });
  });

  describe("scoped to one rack", () => {
    it("should block when that rack lacks the integration, even if another has it", () => {
      const racks = [createRack(1, [], true), createRack(2, ["opc"], true)];
      expect(Device.getBlockedReason(racks, { ...OPC, rack: 1 })).toBe(
        "Rack 1 does not have OPC UA enabled",
      );
    });

    it("should block when that rack is not running", () => {
      const racks = [createRack(1, ["opc"], false), createRack(2, ["opc"], true)];
      expect(Device.getBlockedReason(racks, { ...OPC, rack: 1 })).toBe(
        "Rack 1 is not running",
      );
    });

    it("should name the error of a rack whose Driver reports one", () => {
      const r = rack.payloadZ.parse({
        key: 1,
        name: "Rack 1",
        role: uuid.create(),
        integrations: ["opc"],
        status: status.create<typeof rack.statusDetailsZ>({
          variant: "error",
          message: "Driver 0.59.3 cannot work with Core 0.60.1",
          details: { running: true },
        }),
      });
      expect(Device.getBlockedReason([r], { ...OPC, rack: 1 })).toBe(
        "Driver 0.59.3 cannot work with Core 0.60.1",
      );
    });

    it("should return null when that rack runs the integration", () => {
      const racks = [createRack(1, ["opc"], true), createRack(2, [], false)];
      expect(Device.getBlockedReason(racks, { ...OPC, rack: 1 })).toBeNull();
    });

    it("should block when that rack no longer exists", () => {
      const racks = [createRack(2, ["opc"], true)];
      expect(Device.getBlockedReason(racks, { ...OPC, rack: 1 })).toBe(
        "This Driver no longer exists",
      );
    });
  });
});
