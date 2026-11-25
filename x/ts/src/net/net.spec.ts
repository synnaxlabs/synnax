// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { net } from "@/net";

describe("addressZ", () => {
  it("should parse an address with no port", () => {
    expect(net.addressZ.parse({ host: "kafka.example.com" })).toEqual({
      host: "kafka.example.com",
    });
  });

  it("should parse an address with a port", () => {
    expect(net.addressZ.parse({ host: "10.0.0.2", port: 502 })).toEqual({
      host: "10.0.0.2",
      port: 502,
    });
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["no host", {}],
    ["an empty host", { host: "" }],
    ["port 0", { host: "plc", port: 0 }],
    ["a port over 65535", { host: "plc", port: 65536 }],
  ];
  it.each(INVALID)("should reject %s", (_, address) => {
    expect(net.addressZ.safeParse(address).success).toBe(false);
  });
});
