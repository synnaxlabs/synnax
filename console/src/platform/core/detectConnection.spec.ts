// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { Session } from "@/session";
import { pinLocationOrigin } from "@/testutil";

describe("detectConnection", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

  it("should return null in the tauri engine", () => {
    mocks.engine = "tauri";
    expect(Core.detectConnection()).toBeNull();
  });

  it("should return null when the select-core param is set", () => {
    pinLocationOrigin("http://example.com", "?select-core");
    expect(Core.detectConnection()).toBeNull();
  });

  it("should parse an insecure origin with an explicit port", () => {
    pinLocationOrigin("http://example.com:8080");
    expect(Core.detectConnection()).toEqual({
      key: Session.Core.SERVED_KEY,
      name: "Core",
      host: "example.com",
      port: 8080,
      secure: false,
    });
  });

  it("should default an https origin without a port to 443 and mark it secure", () => {
    pinLocationOrigin("https://example.com");
    expect(Core.detectConnection()).toEqual({
      key: Session.Core.SERVED_KEY,
      name: "Core",
      host: "example.com",
      port: 443,
      secure: true,
    });
  });

  it("should default an http origin without a port to 80", () => {
    pinLocationOrigin("http://example.com");
    expect(Core.detectConnection()).toEqual({
      key: Session.Core.SERVED_KEY,
      name: "Core",
      host: "example.com",
      port: 80,
      secure: false,
    });
  });
});

describe("detectConnection on the dev server", () => {
  beforeEach(() => {
    mocks.engine = "web";
    vi.stubGlobal("IS_DEV", true);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  it("should return null, so the Console starts on the Cores list", () => {
    expect(Core.detectConnection()).toBeNull();
  });

  it("should pin the Console to the Core on the stored dev port", () => {
    localStorage.setItem("synnax-dev-connection-port", "9401");
    expect(Core.detectConnection()).toEqual({
      key: Session.Core.SERVED_KEY,
      name: "Core",
      host: "localhost",
      port: 9401,
      secure: false,
    });
  });

  it("should drop the pin when the detach key is set", () => {
    localStorage.setItem("synnax-dev-connection-port", "9401");
    localStorage.setItem("synnax-dev-connection-detach", "");
    expect(Core.detectConnection()).toBeNull();
  });
});

describe("servesPage", () => {
  const ORIGIN = "http://example.com:9090";

  beforeEach(() => {
    mocks.engine = "web";
  });

  it("should return true when the page came from the Core", () => {
    expect(Core.servesPage({ host: "example.com", port: 9090 }, ORIGIN)).toBe(true);
  });

  it("should return false when another Core served the page", () => {
    expect(Core.servesPage({ host: "example.com", port: 9091 }, ORIGIN)).toBe(false);
    expect(Core.servesPage({ host: "other.com", port: 9090 }, ORIGIN)).toBe(false);
  });

  it("should return false in the tauri engine", () => {
    mocks.engine = "tauri";
    expect(Core.servesPage({ host: "example.com", port: 9090 }, ORIGIN)).toBe(false);
  });
});
