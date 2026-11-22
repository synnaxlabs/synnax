// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ethercat } from "@/ethercat";

const MASTER = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "EtherCAT eth0",
  interface: "eth0",
} as const;

describe("masterZ", () => {
  it("should parse a master and apply the defaults", () => {
    const master = ethercat.master.masterZ.parse(MASTER);
    expect(master.interface).toEqual("eth0");
    expect(master.slaves).toEqual([]);
    expect(master.networkName).toEqual("");
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["an empty interface", { interface: "" }],
    ["a slave key that is not a UUID", { slaves: ["slot-1"] }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(ethercat.master.masterZ.safeParse({ ...MASTER, ...change }).success).toBe(
      false,
    );
  });
});
