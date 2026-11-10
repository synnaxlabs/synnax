// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { installable } from "@/ui/desktop/platform";

const AGENTS = {
  mac: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Version/18.0 Safari/605.1.15",
  windows: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/130.0 Safari/537.36",
  iphone: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Mobile/15E148",
  android: "Mozilla/5.0 (Linux; Android 14; Pixel 8) Chrome/130.0 Mobile Safari/537.36",
  linux: "Mozilla/5.0 (X11; Linux x86_64) Chrome/130.0 Safari/537.36",
};

describe("installable", () => {
  it("should accept a Mac and a Windows PC", () => {
    expect(installable(AGENTS.mac)).toBe(true);
    expect(installable(AGENTS.windows)).toBe(true);
  });

  it("should refuse a phone, Linux, and an unknown browser", () => {
    expect(installable(AGENTS.iphone)).toBe(false);
    expect(installable(AGENTS.android)).toBe(false);
    expect(installable(AGENTS.linux)).toBe(false);
    expect(installable("")).toBe(false);
  });
});
