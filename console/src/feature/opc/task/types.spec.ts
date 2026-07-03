// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { OPC } from "@/feature/opc";

describe("OPC Scan Task Types", () => {
  it("should parse null scan config as empty object", () => {
    const result = OPC.Task.SCAN_SCHEMAS.config.safeParse(null);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should parse undefined scan config as empty object", () => {
    const result = OPC.Task.SCAN_SCHEMAS.config.safeParse(undefined);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should accept null statusData", () => {
    expect(OPC.Task.SCAN_SCHEMAS.statusData.safeParse(null).success).toBe(true);
  });

  it("should accept undefined statusData", () => {
    expect(OPC.Task.SCAN_SCHEMAS.statusData.safeParse(undefined).success).toBe(true);
  });
});

describe("OPC Write Task Types", () => {
  it("should validate the write config", () => {
    const config = {
      channels: [
        {
          channel: 432,
          cmdChannel: 0,
          dataType: "float",
          enabled: true,
          key: "432",
          nodeId: "1",
          name: "test",
          nodeName: "test",
        },
      ],
      device: "1",
    };
    const result = OPC.Task.WRITE_SCHEMAS.config.safeParse(config);
    expect(result.success).toBe(true);
    expect(result.data?.channels[0].cmdChannel).toBe(432);
  });
});

describe("OPC Read Task Config Validation", () => {
  const createConfig = (channels: Partial<OPC.Task.InputChannel>[]): unknown => ({
    device: "dev",
    arrayMode: false,
    sampleRate: 50,
    streamRate: 25,
    channels: channels.map((overrides, i) => ({
      key: `k${i}`,
      nodeId: `ns=1;s=n${i}`,
      nodeName: `n${i}`,
      channel: 0,
      enabled: true,
      useAsIndex: false,
      dataType: "float32",
      name: "",
      ...overrides,
    })),
  });

  it("should accept a valid config", () => {
    expect(OPC.Task.READ_SCHEMAS.config.safeParse(createConfig([{}, {}])).success).toBe(
      true,
    );
  });

  it("should reject a node ID used by multiple channels", () => {
    const result = OPC.Task.READ_SCHEMAS.config.safeParse(
      createConfig([{ nodeId: "ns=1;s=dup" }, { nodeId: "ns=1;s=dup" }]),
    );
    expect(result.success).toBe(false);
    expect(
      result.error?.issues.some(({ message }) => message.includes("already been used")),
    ).toBe(true);
  });

  it("should reject multiple channels marked as index", () => {
    const result = OPC.Task.READ_SCHEMAS.config.safeParse(
      createConfig([{ useAsIndex: true }, { useAsIndex: true }]),
    );
    expect(result.success).toBe(false);
    expect(
      result.error?.issues.some(({ message }) =>
        message.includes("Only one channel can be marked as an index"),
      ),
    ).toBe(true);
  });

  it("should reject an array size larger than the sample rate", () => {
    const config = {
      device: "dev",
      arrayMode: true,
      sampleRate: 50,
      arraySize: 100,
      channels: [],
    };
    const result = OPC.Task.READ_SCHEMAS.config.safeParse(config);
    expect(result.success).toBe(false);
    expect(
      result.error?.issues.some(({ message }) =>
        message.includes("Sample rate must be greater than or equal to the array size"),
      ),
    ).toBe(true);
  });
});

describe("OPC Write Task Config Validation", () => {
  it("should reject the same synnax channel used on multiple write channels", () => {
    const channel = (i: number, cmdChannel: number): unknown => ({
      key: `k${i}`,
      nodeId: `ns=1;s=n${i}`,
      nodeName: `n${i}`,
      cmdChannel,
      enabled: true,
      dataType: "float32",
      name: "",
    });
    const result = OPC.Task.WRITE_SCHEMAS.config.safeParse({
      device: "dev",
      channels: [channel(0, 55), channel(1, 55)],
    });
    expect(result.success).toBe(false);
    expect(
      result.error?.issues.some(({ message }) =>
        message.includes("used for multiple channels"),
      ),
    ).toBe(true);
  });
});

describe("Scanned Nodes", () => {
  it("should default a scanned node's key to its node ID", () => {
    const result = OPC.Task.SCAN_SCHEMAS.statusData.parse({
      channels: [
        {
          dataType: "float32",
          isArray: false,
          name: "Node A",
          nodeClass: "Variable",
          nodeId: "ns=2;s=A",
        },
      ],
      connection: OPC.Device.ZERO_CONNECTION_CONFIG,
    });
    expect(result?.channels[0].key).toBe("ns=2;s=A");
  });
});
