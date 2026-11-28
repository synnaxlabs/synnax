// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Device } from "@/platform/device";

describe("Device.formatAddress", () => {
  it("should join a host and a port with a colon", () => {
    expect(Device.formatAddress("192.168.1.20", 502)).toBe("192.168.1.20:502");
  });

  it("should put brackets around an IPv6 host", () => {
    expect(Device.formatAddress("fe80::1", 502)).toBe("[fe80::1]:502");
  });

  it("should keep the brackets of an IPv6 host that has them", () => {
    expect(Device.formatAddress("[fe80::1]", 502)).toBe("[fe80::1]:502");
  });

  it("should give only the port for an empty host", () => {
    expect(Device.formatAddress("", 5032)).toBe(":5032");
  });
});
