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
