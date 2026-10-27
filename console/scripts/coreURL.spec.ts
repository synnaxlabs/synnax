// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { parse } from "./coreURL.ts";

describe("parse", () => {
  it("should read a bare port as an insecure localhost Core", () => {
    expect(parse("9097")).toEqual({ host: "localhost", port: 9097, secure: false });
  });

  it("should read an https URL with no port as a secure Core on 443", () => {
    expect(parse("https://pr-1.52-44-81-137.sslip.io")).toEqual({
      host: "pr-1.52-44-81-137.sslip.io",
      port: 443,
      secure: true,
    });
  });

  it("should read an http URL with no port as an insecure Core on 80", () => {
    expect(parse("http://core.local")).toEqual({
      host: "core.local",
      port: 80,
      secure: false,
    });
  });

  it("should keep an explicit port", () => {
    expect(parse("https://localhost:9090/")).toEqual({
      host: "localhost",
      port: 9090,
      secure: true,
    });
  });

  it("should reject a URL with another scheme", () => {
    expect(() => parse("ws://localhost:9090")).toThrow("not an http or https URL");
  });

  it("should reject an address that is not a URL", () => {
    expect(() => parse("localhost")).toThrow();
  });
});
