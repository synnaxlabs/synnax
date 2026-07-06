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

describe("checkConfigured", () => {
  it("should not throw when the device is configured", () => {
    expect(() =>
      Device.checkConfigured({ configured: true, name: "Sensor" }),
    ).not.toThrow();
  });

  it("should throw when the device is not configured", () => {
    expect(() => Device.checkConfigured({ configured: false, name: "Sensor" })).toThrow(
      "Sensor is not configured.",
    );
  });

  it("should throw when the configured flag is absent", () => {
    expect(() => Device.checkConfigured({ name: "My Device" })).toThrow(
      "My Device is not configured.",
    );
  });
});
