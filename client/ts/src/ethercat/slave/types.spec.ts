// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ethercat } from "@/ethercat";

const SLAVE = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "EL3102",
  vendorId: 2,
  productCode: 0x0c1e3052,
  revision: 0x00110000,
  master: "33333333-3333-4333-8333-333333333333",
} as const;

describe("slaveZ", () => {
  it("should parse a slave and leave it unconfigured", () => {
    const slave = ethercat.slave.slaveZ.parse(SLAVE);
    expect(slave.serial).toEqual(0);
    expect(slave.identifier).toEqual("");
    expect(slave.disabled).toBe(false);
    expect(slave.pdos).toEqual({ inputs: [], outputs: [] });
    expect(slave.readBindings).toEqual([]);
    expect(slave.opModes).toEqual([]);
    expect(slave.opMode).toBeUndefined();
    expect(slave.switchedAt.equals(TimeStamp.ZERO)).toBe(true);
  });

  it("should parse the PDO entries and channel bindings", () => {
    const slave = ethercat.slave.slaveZ.parse({
      ...SLAVE,
      pdos: {
        inputs: [
          {
            name: "Value",
            index: 0x6000,
            subIndex: 0x11,
            bitLength: 16,
            dataType: "int16",
          },
        ],
      },
      readBindings: [{ index: 0x6000, subIndex: 0x11, channel: 7 }],
      writeBindings: [{ index: 0x7000, subIndex: 1, command: 8 }],
      startupParameters: [
        { index: 0x8000, subIndex: 1, dataType: "uint16", value: 500 },
      ],
    });
    expect(slave.pdos.inputs[0].pdoName).toEqual("");
    expect(slave.readBindings[0].scaled).toBe(false);
    expect(slave.writeBindings[0].state).toEqual(0);
    expect(slave.startupParameters[0]).toMatchObject({
      dataType: "uint16",
      value: 500,
    });
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["no master", { master: undefined }],
    ["no vendor ID", { vendorId: undefined }],
    ["no revision", { revision: undefined }],
    [
      "a write binding without a command",
      { writeBindings: [{ index: 0x7000, subIndex: 1 }] },
    ],
    [
      "a startup value that the data type cannot hold",
      {
        startupParameters: [
          { index: 0x8000, subIndex: 1, dataType: "uint8", value: 256 },
        ],
      },
    ],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(ethercat.slave.slaveZ.safeParse({ ...SLAVE, ...change }).success).toBe(
      false,
    );
  });
});
