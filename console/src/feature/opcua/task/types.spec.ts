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

import { OPCUA } from "@/feature/opcua";
import { SERVER_CERTIFICATE } from "@/feature/opcua/testutil";

describe("OPC UA Scan Task Types", () => {
  it("should parse null scan config as empty object", () => {
    const result = OPCUA.Task.SCAN_SCHEMAS.config.safeParse(null);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should parse undefined scan config as empty object", () => {
    const result = OPCUA.Task.SCAN_SCHEMAS.config.safeParse(undefined);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should accept null statusData", () => {
    expect(z.validate(OPCUA.Task.SCAN_SCHEMAS.statusData, null)).toBe(true);
  });

  it("should accept undefined statusData", () => {
    expect(z.validate(OPCUA.Task.SCAN_SCHEMAS.statusData, undefined)).toBe(true);
  });
});

describe("OPC UA Write Task Types", () => {
  it("should validate the write config", () => {
    const config = {
      channels: [
        {
          cmdChannel: 432,
          dataType: "float",
          disabled: false,
          key: "432",
          nodeId: "1",
          name: "test",
          nodeName: "test",
          type: "node",
        },
      ],
      device: "1",
    };
    const result = OPCUA.Task.WRITE_SCHEMAS.config.safeParse(config);
    expect(result.success).toBe(true);
    expect(result.data?.channels[0].cmdChannel).toBe(432);
  });
});

describe("OPC UA Read Task Config Validation", () => {
  const createConfig = (channels: Partial<OPCUA.Task.ReadChannel>[]) => ({
    device: "dev",
    arrayMode: false,
    sampleRate: 50,
    streamRate: 25,
    channels: channels.map((overrides, i) => ({
      key: `k${i}`,
      nodeId: `ns=1;s=n${i}`,
      nodeName: `n${i}`,
      channel: 0,
      disabled: false,
      isIndex: false,
      dataType: "float32",
      name: "",
      ...overrides,
    })),
  });

  it("should accept a valid config", () => {
    expect(z.validate(OPCUA.Task.READ_SCHEMAS.config, createConfig([{}, {}]))).toBe(
      true,
    );
  });

  it("should warn at each node ID used by multiple channels", () => {
    const result = OPCUA.Task.deployReadConfigZ.safeParse(
      createConfig([{ nodeId: "ns=1;s=dup" }, { nodeId: "ns=1;s=dup" }]),
    );
    expect(result.success).toBe(false);
    expect(result.error?.issues).toEqual(
      [0, 1].map((i) =>
        expect.objectContaining({
          path: ["channels", i, "nodeId"],
          message: "This node ID has already been used elsewhere in the configuration",
          params: { variant: "warning" },
        }),
      ),
    );
  });

  it("should accept channels that read different fields of one node", () => {
    const result = OPCUA.Task.deployReadConfigZ.safeParse(
      createConfig([
        { nodeId: "ns=1;s=motor", field: "Speed" },
        { nodeId: "ns=1;s=motor", field: "Position.Y" },
      ]),
    );
    expect(result.success).toBe(true);
  });

  it("should warn at each channel that reads the same field of one node", () => {
    const result = OPCUA.Task.deployReadConfigZ.safeParse(
      createConfig([
        { nodeId: "ns=1;s=motor", field: "Speed" },
        { nodeId: "ns=1;s=motor", field: "Speed" },
      ]),
    );
    expect(result.error?.issues).toEqual(
      [0, 1].map((i) =>
        expect.objectContaining({
          path: ["channels", i, "nodeId"],
          message: "This field has already been used elsewhere in the configuration",
        }),
      ),
    );
  });

  it("should reject each channel marked as index when there are several", () => {
    const result = OPCUA.Task.deployReadConfigZ.safeParse(
      createConfig([{ isIndex: true }, { isIndex: true }]),
    );
    expect(result.success).toBe(false);
    expect(result.error?.issues).toEqual(
      [0, 1].map((i) =>
        expect.objectContaining({
          path: ["channels", i, "isIndex"],
          message: "Only one channel can be marked as an index channel",
        }),
      ),
    );
  });

  it("should reject an index channel in On change mode", () => {
    const result = OPCUA.Task.deployReadConfigZ.safeParse({
      ...createConfig([{}, { isIndex: true, dataType: "timestamp" }]),
      mode: "on_change",
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toEqual([
      expect.objectContaining({
        path: ["channels", 1, "isIndex"],
        message: "On change mode cannot read an index channel",
      }),
    ]);
  });

  it("should accept an index channel in sampled mode", () => {
    const result = OPCUA.Task.deployReadConfigZ.safeParse(
      createConfig([{}, { isIndex: true, dataType: "timestamp" }]),
    );
    expect(result.success).toBe(true);
  });

  it("should reject an array size larger than the sample rate", () => {
    const config = {
      ...createConfig([{}]),
      arrayMode: true,
      arraySize: 100,
    };
    const result = OPCUA.Task.deployReadConfigZ.safeParse(config);
    expect(result.success).toBe(false);
    expect(
      result.error?.issues.some(({ message }) =>
        message.includes("Sample rate must be greater than or equal to the array size"),
      ),
    ).toBe(true);
  });

  it("should reject a non-positive array size", () => {
    const config = {
      ...createConfig([{}]),
      arrayMode: true,
      arraySize: 0,
    };
    const result = OPCUA.Task.deployReadConfigZ.safeParse(config);
    expect(result.success).toBe(false);
    expect(
      result.error?.issues.some(({ message }) =>
        message.includes("Array size must be a positive integer"),
      ),
    ).toBe(true);
  });

  it("should reject a stream rate outside of (0, 10000]", () => {
    const result = OPCUA.Task.deployReadConfigZ.safeParse({
      ...createConfig([{}]),
      streamRate: 20000,
    });
    expect(result.success).toBe(false);
    expect(
      result.error?.issues.some(({ message }) =>
        message.includes("Stream rate must be between 0 and 10000"),
      ),
    ).toBe(true);
  });
});

describe("OPC UA Write Task Config Validation", () => {
  it("should reject the same synnax channel used on multiple write channels", () => {
    const channel = (i: number, cmdChannel: number): unknown => ({
      key: `k${i}`,
      nodeId: `ns=1;s=n${i}`,
      nodeName: `n${i}`,
      cmdChannel,
      disabled: false,
      dataType: "float32",
      name: "",
      type: "node",
    });
    const result = OPCUA.Task.deployWriteConfigZ.safeParse({
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

describe("OPC UA method channel validation", () => {
  const method = (overrides: Partial<OPCUA.Task.WriteChannel> = {}): unknown => ({
    key: "k0",
    nodeId: "ns=1;s=Pump.SetSpeed",
    nodeName: "SetSpeed",
    cmdChannel: 0,
    disabled: false,
    dataType: "float32",
    name: "",
    type: "method",
    objectId: "ns=1;s=Pump",
    arguments: [{ source: "command", name: "speed", dataType: "float64" }],
    ...overrides,
  });
  const issues = (...channels: unknown[]) =>
    OPCUA.Task.deployWriteConfigZ
      .safeParse({ device: "dev", channels })
      .error?.issues.map(({ message, path }) => ({ message, path })) ?? [];

  it("should accept a method channel", () => {
    expect(issues(method())).toEqual([]);
  });

  it("should require an object", () => {
    expect(issues(method({ objectId: "" }))).toEqual([
      { message: "Method must have an object", path: ["channels", 0, "objectId"] },
    ]);
  });

  it("should allow only one argument to take the command", () => {
    const arg = {
      source: "command",
      name: "a",
      dataType: "float64",
    } as const;
    expect(issues(method({ arguments: [arg, { ...arg, name: "b" }] }))).toEqual([
      {
        message: "Only one argument can take the command",
        path: ["channels", 0, "arguments"],
      },
    ]);
  });

  it("should not warn about two channels that call one method", () => {
    expect(issues(method(), method({ key: "k1" }))).toEqual([]);
  });

  it("should not check a disabled method", () => {
    const arg = { source: "command", name: "a", dataType: "float64" } as const;
    const unfinished = method({
      key: "k1",
      disabled: true,
      objectId: "",
      arguments: [arg, { ...arg, name: "b" }],
    });
    expect(issues(method(), unfinished)).toEqual([]);
  });
});

describe("OPC UA event task config validation", () => {
  const issues = (fields: Partial<OPCUA.Task.EventField>[]) =>
    OPCUA.Task.deployEventConfigZ
      .safeParse({
        device: "dev",
        fields: fields.map((f, i) => ({
          key: `f${i}`,
          name: "",
          browsePath: "",
          ...f,
        })),
      })
      .error?.issues.map(({ message, path }) => ({ message, path })) ?? [];

  it("should accept fields with unique names and full browse paths", () => {
    expect(issues([{ name: "level", browsePath: "2:Pump/2:Level" }])).toEqual([]);
  });

  it("should reject a field without a name", () => {
    expect(issues([{ browsePath: "Level" }])).toEqual([
      { message: "Field must have a name", path: ["fields", 0, "name"] },
    ]);
  });

  it("should reject a field that repeats a standard or earlier name", () => {
    expect(
      issues([
        { name: "severity", browsePath: "Level" },
        { name: "level", browsePath: "Level" },
        { name: "level", browsePath: "Level" },
      ]),
    ).toEqual([
      { message: "A standard field has this name", path: ["fields", 0, "name"] },
      { message: "Another field has this name", path: ["fields", 2, "name"] },
    ]);
  });

  it("should reject a browse path with an empty element", () => {
    expect(issues([{ name: "id", browsePath: "EnabledState//Id" }])).toEqual([
      {
        message: "Browse path must name each element",
        path: ["fields", 0, "browsePath"],
      },
    ]);
  });

  it("should accept a namespace URI that holds a slash", () => {
    expect(
      issues([{ name: "level", browsePath: "nsu=http://x.com/UA/;Pump/2:Level" }]),
    ).toEqual([]);
  });

  it("should reject a namespace URI with no semicolon after it", () => {
    expect(issues([{ name: "level", browsePath: "nsu=urn:pump" }])).toEqual([
      {
        message: "Browse path must end each namespace URI with ';'",
        path: ["fields", 0, "browsePath"],
      },
    ]);
  });

  it("should reject a namespace URI with an invalid escape", () => {
    expect(issues([{ name: "level", browsePath: "nsu=urn:%zz;Level" }])).toEqual([
      {
        message: "Browse path has an invalid namespace URI",
        path: ["fields", 0, "browsePath"],
      },
    ]);
  });

  it("should reject a name after a namespace index that is empty", () => {
    expect(issues([{ name: "level", browsePath: "Pump/2:" }])).toEqual([
      {
        message: "Browse path must name each element",
        path: ["fields", 0, "browsePath"],
      },
    ]);
  });
});

describe("Scanned Nodes", () => {
  it("should default a scanned node's key to its node ID", () => {
    const result = OPCUA.Task.browseResponseZ.parse({
      channels: [
        {
          dataType: "float32",
          isArray: false,
          name: "Node A",
          nodeClass: "Variable",
          nodeId: "ns=2;s=A",
        },
      ],
    });
    expect(result).toMatchObject({ channels: [{ key: "ns=2;s=A", field: "" }] });
  });
});

describe("Scan command replies", () => {
  it("should parse a certificate reply", () => {
    const reply = { current: SERVER_CERTIFICATE, next: SERVER_CERTIFICATE };
    expect(OPCUA.Task.certificatesResponseZ.parse(reply)).toEqual(reply);
  });

  it("should parse a failed test connection reply with the server certificate", () => {
    const reply = { serverCertificate: SERVER_CERTIFICATE };
    expect(OPCUA.Task.testConnectionResponseZ.parse(reply)).toEqual(reply);
  });
});

describe("deploy configs", () => {
  it.each([
    ["read", OPCUA.Task.READ_SCHEMAS.config, OPCUA.Task.deployReadConfigZ],
    ["write", OPCUA.Task.WRITE_SCHEMAS.config, OPCUA.Task.deployWriteConfigZ],
  ])("should require an enabled channel to deploy a %s task", (_, draftZ, deployZ) => {
    const issues = deployZ.safeParse(draftZ.parse({})).error?.issues;
    expect(issues).toContainEqual(
      expect.objectContaining({
        path: ["channels"],
        message: "At least one enabled channel is required",
      }),
    );
  });
});
