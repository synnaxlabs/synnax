// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Desktop } from "@/components/desktop";

const AGENTS = {
  mac: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15",
  windows:
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
  iphone:
    "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
  ipad: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
  linux:
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
};

describe("Desktop installers", () => {
  it("should pick the macOS installer for a Mac", () => {
    expect(Desktop.installer("1.0.0", AGENTS.mac)?.asset).toEqual(
      "Synnax-Desktop_1.0.0_aarch64.dmg",
    );
  });

  it("should pick the Windows installer for a Windows PC", () => {
    expect(Desktop.installer("1.0.0", AGENTS.windows)?.asset).toEqual(
      "Synnax-Desktop_1.0.0_x64-setup.exe",
    );
  });

  it("should find no installer for a phone that says like Mac OS X", () => {
    expect(Desktop.installer("1.0.0", AGENTS.iphone)).toBeNull();
  });

  it("should find no installer for an iPad that poses as a Mac", () => {
    expect(Desktop.installer("1.0.0", AGENTS.ipad)).toBeNull();
  });

  it("should find no installer for Linux", () => {
    expect(Desktop.installer("1.0.0", AGENTS.linux)).toBeNull();
  });

  it("should name a candidate's installer with a numeric pre-release", () => {
    expect(Desktop.installers("1.1.0-rc.2").map(({ asset }) => asset)).toEqual([
      "Synnax-Desktop_1.1.0-2_aarch64.dmg",
      "Synnax-Desktop_1.1.0-2_x64-setup.exe",
    ]);
  });
});
