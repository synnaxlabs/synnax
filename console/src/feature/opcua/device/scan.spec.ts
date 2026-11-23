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
    const result = OPCUA.Device.SCAN_SCHEMAS.config.safeParse(null);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should parse undefined scan config as empty object", () => {
    const result = OPCUA.Device.SCAN_SCHEMAS.config.safeParse(undefined);
    expect(result.success).toBe(true);
    expect(result.data).toEqual({});
  });

  it("should accept null statusData", () => {
    expect(z.validate(OPCUA.Device.SCAN_SCHEMAS.statusData, null)).toBe(true);
  });

  it("should accept undefined statusData", () => {
    expect(z.validate(OPCUA.Device.SCAN_SCHEMAS.statusData, undefined)).toBe(true);
  });
});

describe("Scanned Nodes", () => {
  it("should default a scanned node's key to its node ID", () => {
    const result = OPCUA.Device.browseResponseZ.parse({
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
    expect(OPCUA.Device.certificatesResponseZ.parse(reply)).toEqual(reply);
  });

  it("should parse a failed test connection reply with the server certificate", () => {
    const reply = { serverCertificate: SERVER_CERTIFICATE };
    expect(OPCUA.Device.testConnectionResponseZ.parse(reply)).toEqual(reply);
  });
});
