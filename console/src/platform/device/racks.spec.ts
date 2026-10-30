// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, status } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Device } from "@/platform/device";

const createRack = (key: rack.Key, integrations: string[], running: boolean) =>
  rack.payloadZ.parse({
    key,
    name: `Rack ${key}`,
    integrations,
    status: running
      ? status.create<typeof rack.statusDetailsZ>({
          variant: "success",
          message: "Running",
          details: { rack: key },
        })
      : undefined,
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
