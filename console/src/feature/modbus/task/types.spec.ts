// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Modbus } from "@/feature/modbus";

describe("Modbus Scan Task Types", () => {
  it("should parse null scan config as empty object", () => {
    const result = Modbus.Task.SCAN_SCHEMAS.config.safeParse(null);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should parse undefined scan config as empty object", () => {
    const result = Modbus.Task.SCAN_SCHEMAS.config.safeParse(undefined);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should accept null statusData", () => {
    expect(z.validate(Modbus.Task.SCAN_SCHEMAS.statusData, null)).toBe(true);
  });

  it("should accept undefined statusData", () => {
    expect(z.validate(Modbus.Task.SCAN_SCHEMAS.statusData, undefined)).toBe(true);
  });
});

describe("Modbus Read Task Types", () => {
  it("should accept a null statusData", () => {
    expect(z.validate(Modbus.Task.READ_SCHEMAS.statusData, null)).toBe(true);
  });

  const holding = (key: string, address: number, dataType: string) =>
    Modbus.Task.READ_CHANNEL_SCHEMAS.holding_register.parse({
      type: "holding_register",
      key,
      address,
      dataType,
    });

  describe("addressSpan", () => {
    it("should span one address for a coil", () => {
      const coil = Modbus.Task.READ_CHANNEL_SCHEMAS.coil.parse({
        type: "coil",
        address: 4,
      });
      expect(Modbus.Task.addressSpan(coil)).toEqual([4, 4]);
    });

    it("should span one address per 16 bits of a register's data type", () => {
      expect(Modbus.Task.addressSpan(holding("a", 10, "float64"))).toEqual([10, 13]);
    });
  });

  describe("findOverlaps", () => {
    it("should mark the registers that read a shared address", () => {
      const a = holding("a", 10, "float32");
      const b = holding("b", 11, "uint16");
      const overlaps = Modbus.Task.findOverlaps([a, b, holding("c", 20, "uint16")]);
      expect(overlaps.keys).toEqual(new Set(["a", "b"]));
      expect(overlaps.ranges).toEqual({ holding_register: [[11, 11]] });
    });

    it("should not mark registers that only touch", () => {
      const a = holding("a", 10, "float32");
      const b = holding("b", 12, "uint16");
      const overlaps = Modbus.Task.findOverlaps([a, b]);
      expect(overlaps.keys.size).toBe(0);
      expect(overlaps.ranges).toEqual({});
    });

    it("should not mark channels of another type at the same address", () => {
      const a = holding("a", 10, "uint16");
      const b = Modbus.Task.READ_CHANNEL_SCHEMAS.input_register.parse({
        type: "input_register",
        key: "b",
        address: 10,
        dataType: "uint16",
      });
      expect(Modbus.Task.findOverlaps([a, b]).keys.size).toBe(0);
    });

    it("should find one range for many channels on the same addresses", () => {
      const channels = ["a", "b", "c"].map((k) => holding(k, 10, "float32"));
      const overlaps = Modbus.Task.findOverlaps(channels);
      expect(overlaps.keys).toEqual(new Set(["a", "b", "c"]));
      expect(overlaps.ranges).toEqual({ holding_register: [[10, 11]] });
    });

    it("should join shared ranges that touch", () => {
      const a = holding("a", 10, "float64");
      const b = holding("b", 10, "float32");
      const c = holding("c", 12, "float32");
      expect(Modbus.Task.findOverlaps([a, b, c]).ranges).toEqual({
        holding_register: [[10, 13]],
      });
    });

    it("should mark only register bits that read the same bit", () => {
      const bit = (key: string, address: number, b: number) =>
        Modbus.Task.READ_CHANNEL_SCHEMAS.holding_register_bit.parse({
          type: "holding_register_bit",
          key,
          address,
          bit: b,
        });
      const overlaps = Modbus.Task.findOverlaps([
        bit("a", 10, 3),
        bit("b", 10, 4),
        bit("c", 10, 4),
        bit("d", 11, 3),
        bit("e", 11, 3),
      ]);
      expect(overlaps.keys).toEqual(new Set(["b", "c", "d", "e"]));
      expect(overlaps.ranges).toEqual({ holding_register_bit: [[10, 11]] });
    });
  });
});

describe("Modbus Write Task Types", () => {
  it("should accept a null statusData", () => {
    expect(z.validate(Modbus.Task.WRITE_SCHEMAS.statusData, null)).toBe(true);
  });

  it("should accept a populated statusData", () => {
    expect(
      z.validate(Modbus.Task.WRITE_SCHEMAS.statusData, {
        running: true,
        message: "ok",
      }),
    ).toBe(true);
  });
});

describe("deploy configs", () => {
  it("should reject a read config without a device", () => {
    const result = Modbus.Task.deployReadConfigZ.safeParse(
      Modbus.Task.READ_SCHEMAS.config.parse({}),
    );
    expect(result.success).toBe(false);
  });

  it("should reject a stream rate above the sample rate", () => {
    const result = Modbus.Task.deployReadConfigZ.safeParse({
      ...Modbus.Task.READ_SCHEMAS.config.parse({}),
      device: "my_device",
      sampleRate: 5,
      streamRate: 10,
    });
    expect(result.success).toBe(false);
  });

  it("should reject a write config without a device", () => {
    const result = Modbus.Task.deployWriteConfigZ.safeParse(
      Modbus.Task.WRITE_SCHEMAS.config.parse({}),
    );
    expect(result.success).toBe(false);
  });

  const readConfig = (channels: object[]) =>
    Modbus.Task.READ_SCHEMAS.config.parse({ device: "my_device", channels });
  const writeConfig = (channels: object[]) =>
    Modbus.Task.WRITE_SCHEMAS.config.parse({ device: "my_device", channels });
  const issues = (result: z.ZodSafeParseResult<unknown>) =>
    result.error?.issues.map(({ path, message }) => ({ path, message }));

  it("should reject a register channel that ends past the last address", () => {
    const result = Modbus.Task.deployReadConfigZ.safeParse(
      readConfig([
        { type: "holding_register", address: 65534, dataType: "float32" },
        { type: "holding_register", address: 65533, dataType: "float64" },
      ]),
    );
    expect(issues(result)).toEqual([
      {
        path: ["channels", 1, "address"],
        message: "Channel ends past address 65535",
      },
    ]);
  });

  it.each([0, 251])("should reject a string of %d characters", (stringLength) => {
    const result = Modbus.Task.deployReadConfigZ.safeParse(
      readConfig([
        { type: "input_register", address: 0, dataType: "string", stringLength },
        { type: "input_register", address: 200, dataType: "uint16", stringLength },
      ]),
    );
    expect(issues(result)).toEqual([
      {
        path: ["channels", 0, "stringLength"],
        message: "String length must be from 1 to 250",
      },
    ]);
  });

  it("should reject read channels that bind to one Synnax channel", () => {
    const result = Modbus.Task.deployReadConfigZ.safeParse(
      readConfig([
        { type: "input_register", address: 3, dataType: "uint16" },
        { type: "input_register", address: 3, dataType: "uint16", bytesSwapped: true },
        { type: "input_register", address: 3, dataType: "int16" },
        { type: "holding_register", address: 3, dataType: "uint16" },
      ]),
    );
    expect(issues(result)).toEqual([
      {
        path: ["channels", 1, "address"],
        message: "Another channel reads the same address and data type",
      },
    ]);
  });

  it("should reject write channels whose addresses overlap", () => {
    const result = Modbus.Task.deployWriteConfigZ.safeParse(
      writeConfig([
        { type: "holding_register", address: 0, dataType: "float32" },
        { type: "holding_register", address: 1, dataType: "uint16" },
        { type: "coil", address: 1 },
        { type: "coil", address: 1 },
      ]),
    );
    expect(issues(result)).toEqual([
      {
        path: ["channels", 1, "address"],
        message: "Overlaps the holding register at address 0",
      },
      { path: ["channels", 3, "address"], message: "Overlaps the coil at address 1" },
    ]);
  });

  it("should reject a register bit inside a register value or on a taken bit", () => {
    const result = Modbus.Task.deployWriteConfigZ.safeParse(
      writeConfig([
        { type: "holding_register", address: 0, dataType: "float32" },
        { type: "holding_register_bit", address: 2, bit: 1 },
        { type: "holding_register_bit", address: 2, bit: 2 },
        { type: "coil", address: 2 },
        { type: "holding_register_bit", address: 1, bit: 0 },
        { type: "holding_register_bit", address: 2, bit: 1 },
        { type: "holding_register", address: 2, dataType: "uint16" },
      ]),
    );
    expect(issues(result)).toEqual([
      {
        path: ["channels", 4, "address"],
        message: "Overlaps the holding register at address 0",
      },
      {
        path: ["channels", 5, "address"],
        message: "Overlaps the holding register bit at address 2",
      },
      {
        path: ["channels", 6, "address"],
        message: "Overlaps the holding register bit at address 2",
      },
    ]);
  });

  it("should ignore disabled channels", () => {
    const result = Modbus.Task.deployWriteConfigZ.safeParse(
      writeConfig([
        { type: "coil", address: 1 },
        { type: "coil", address: 1, disabled: true },
      ]),
    );
    expect(result.success).toBe(true);
  });
});

describe("draft configs", () => {
  // Drafts persist server-side before configuration, so the shape schema must
  // accept every default config; retrieve parses with it.
  it("should accept the default read config", () => {
    const config = Modbus.Task.READ_SCHEMAS.config.parse({});
    expect(z.validate(Modbus.Task.READ_SCHEMAS.config, config)).toBe(true);
  });
  it("should accept the default write config", () => {
    const config = Modbus.Task.WRITE_SCHEMAS.config.parse({});
    expect(z.validate(Modbus.Task.WRITE_SCHEMAS.config, config)).toBe(true);
  });
});
