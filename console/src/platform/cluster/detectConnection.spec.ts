// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Cluster } from "@/platform/cluster";
import { pinLocationOrigin } from "@/testutil";

describe("detectConnection", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

  it("should return null in the tauri engine", () => {
    mocks.engine = "tauri";
    expect(Cluster.detectConnection()).toBeNull();
  });

  it("should parse an insecure origin with an explicit port", () => {
    pinLocationOrigin("http://example.com:8080");
    expect(Cluster.detectConnection()).toEqual({
      name: "Core",
      host: "example.com",
      port: 8080,
      secure: false,
    });
  });

  it("should default an https origin without a port to 443 and mark it secure", () => {
    pinLocationOrigin("https://example.com");
    expect(Cluster.detectConnection()).toEqual({
      name: "Core",
      host: "example.com",
      port: 443,
      secure: true,
    });
  });

  it("should default an http origin without a port to 80", () => {
    pinLocationOrigin("http://example.com");
    expect(Cluster.detectConnection()).toEqual({
      name: "Core",
      host: "example.com",
      port: 80,
      secure: false,
    });
  });
});
