// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";

const createSlave = (
  key: string,
  properties: Partial<EtherCAT.Device.SlaveProperties>,
  { configured = true, connected = false, rack = 1 } = {},
): EtherCAT.Device.SlaveOnRack => ({
  key,
  name: key,
  rack,
  location: "test-location",
  make: EtherCAT.Device.MAKE,
  model: EtherCAT.Device.SLAVE_MODEL,
  configured,
  connected,
  properties: {
    ...EtherCAT.Device.ZERO_SLAVE_PROPERTIES,
    network: "eth0",
    vendorId: 0xdead,
    productCode: 7,
    ...properties,
  },
});

const NEW = { configured: false, connected: true };

const keysOf = (replacements: EtherCAT.Device.Replacement[]) =>
  replacements.map(({ slave, old }) => [slave.key, old.map((o) => o.key)]);

describe("findReplacements", () => {
  it("should pair a new slave with an old slave of its model on its network", () => {
    const slaves = [
      createSlave("old", { position: 3, serial: 0 }),
      createSlave("new", { position: 5, serial: 77, revision: 2 }, NEW),
    ];
    expect(keysOf(EtherCAT.Device.findReplacements(slaves))).toEqual([
      ["new", ["old"]],
    ]);
  });

  it("should give the old slaves in order of position", () => {
    const slaves = [
      createSlave("third", { position: 3 }),
      createSlave("first", { position: 1 }),
      createSlave("new", { position: 9 }, NEW),
    ];
    expect(keysOf(EtherCAT.Device.findReplacements(slaves))).toEqual([
      ["new", ["first", "third"]],
    ]);
  });

  it("should not pair slaves of another product, vendor, network, or rack", () => {
    const slaves = [
      createSlave("product", { productCode: 8 }),
      createSlave("vendor", { vendorId: 0xbeef }),
      createSlave("network", { network: "eth1" }),
      createSlave("rack", {}, { rack: 2 }),
      createSlave("new", {}, NEW),
    ];
    expect(EtherCAT.Device.findReplacements(slaves)).toEqual([]);
  });

  it("should not pair an old slave that is connected, disabled, or not configured", () => {
    const slaves = [
      createSlave("connected", {}, { connected: true }),
      createSlave("disabled", { enabled: false }),
      createSlave("unconfigured", {}, { configured: false }),
      createSlave("new", {}, NEW),
    ];
    expect(EtherCAT.Device.findReplacements(slaves)).toEqual([]);
  });

  it("should not pair a new slave that is disconnected", () => {
    const slaves = [
      createSlave("old", {}),
      createSlave("new", {}, { configured: false, connected: false }),
    ];
    expect(EtherCAT.Device.findReplacements(slaves)).toEqual([]);
  });
});
