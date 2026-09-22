// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { Analytics } from "@/platform/analytics";

const createTransport = (): Analytics.Transport => ({
  capture: vi.fn(),
  describe: vi.fn(),
});

describe("Analytics.createSink", () => {
  it("should report a declared event with its properties", () => {
    const transport = createTransport();
    Analytics.createSink(transport).capture("command_run", { command: "show_logs" });
    expect(transport.capture).toHaveBeenCalledWith("command_run", {
      command: "show_logs",
    });
  });

  it("should drop an event whose number is not a real number", () => {
    // A duration read from a corrupt stored timestamp arrives as NaN. It satisfies the
    // signature, so only the schema stops it from reaching a vendor.
    const transport = createTransport();
    const log = vi.spyOn(console, "error").mockImplementation(() => {});
    Analytics.createSink(transport).capture("core_ready", {
      time_to_ready_ms: Number.NaN,
      starts: 1,
    });
    expect(transport.capture).not.toHaveBeenCalled();
    expect(log).toHaveBeenCalled();
  });

  it("should report a screen at a synthesized address", () => {
    const transport = createTransport();
    Analytics.createSink(transport).screen("schematic");
    expect(transport.capture).toHaveBeenCalledWith("$pageview", {
      $current_url: "https://desktop.synnaxlabs.com/schematic",
      $host: "desktop.synnaxlabs.com",
      $pathname: "/schematic",
    });
  });

  it("should drop a screen whose tab is not an identifier", () => {
    // Panel names are user-written, so one reaching screen would send a name the user
    // typed. Every real tab type is an identifier.
    const transport = createTransport();
    const log = vi.spyOn(console, "error").mockImplementation(() => {});
    Analytics.createSink(transport).screen("Fuel system P&ID");
    expect(transport.capture).not.toHaveBeenCalled();
    expect(log).toHaveBeenCalled();
  });
});
