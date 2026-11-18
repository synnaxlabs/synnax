// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { runtime } from "@/runtime";

const stubUserAgent = (userAgent: string): void => {
  vi.stubGlobal("window", { navigator: { userAgent } });
};

describe("getOS", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("should return undefined outside a browser", () => {
    vi.stubGlobal("window", undefined);
    expect(runtime.getOS()).toBeUndefined();
  });

  it("should read the OS from the user agent", () => {
    stubUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)");
    expect(runtime.getOS()).toEqual("macOS");
    stubUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64)");
    expect(runtime.getOS()).toEqual("Windows");
    stubUserAgent("Mozilla/5.0 (X11; Linux x86_64)");
    expect(runtime.getOS()).toEqual("Linux");
  });

  it("should return undefined for a user agent that names no OS", () => {
    stubUserAgent("Mozilla/5.0 (PlayStation)");
    expect(runtime.getOS()).toBeUndefined();
  });
});
