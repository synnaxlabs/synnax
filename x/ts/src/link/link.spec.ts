// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { is } from "@/link/link";

describe("is", () => {
  // Valid IRIs
  it("should return true for a simple HTTP URL", () => {
    expect(is("http://example.com")).toBe(true);
  });

  it("should return true for an HTTP URL with a port", () => {
    expect(is("http://example.com:8080")).toBe(true);
  });

  it("should return true for an HTTPS URL with path, query, and fragment", () => {
    expect(is("https://example.com/path?query=param#fragment")).toBe(true);
  });

  it("should return true for an IRI with IPv4", () => {
    expect(is("http://192.168.1.1")).toBe(true);
  });

  it("should return true for an IRI with percent-encoded spaces", () => {
    expect(is("http://example.com/path%20with%20spaces")).toBe(true);
  });

  it("should return true for an IRI with query and fragment", () => {
    expect(is("http://example.com/path?query=123#section")).toBe(true);
  });

  // Invalid IRIs
  it("should return false for a URL with invalid characters", () => {
    expect(is("http://example.com/invalid|character")).toBe(false);
  });

  it("should return false for a scheme with spaces", () => {
    expect(is("ht tp://example.com")).toBe(false);
  });

  it("should return false for a URL with a missing host", () => {
    expect(is("http://")).toBe(false);
  });

  it("should return false for a URL with an unsupported scheme", () => {
    expect(is("xyz://example.com")).toBe(false);
  });

  it("should return false for an IPv6 with double colons", () => {
    expect(is("http://[2001:db8:::1]")).toBe(false);
  });

  it("should return false for a scheme with a trailing colon", () => {
    expect(is("http:/example.com")).toBe(false);
  });

  it("should return false for a fragment with spaces", () => {
    expect(is("http://example.com#invalid fragment")).toBe(false);
  });
});
