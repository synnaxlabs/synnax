// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { url } from "@/url";

describe("URL", () => {
  test("URL - child", () => {
    const endpoint = new url.URL({
      host: "localhost",
      port: 8080,
      protocol: "http",
      pathPrefix: "api",
    });
    expect(endpoint.child("test").toString()).toEqual("http://localhost:8080/api/test");
  });

  test("URL - child with trailing slash", () => {
    const endpoint = new url.URL({
      host: "localhost",
      port: 8080,
      protocol: "http",
      pathPrefix: "api",
    });
    const child = endpoint.child("test/");
    expect(child.toString()).toEqual("http://localhost:8080/api/test");
  });

  test("URL - replacing protocol", () => {
    const endpoint = new url.URL({
      host: "localhost",
      port: 8080,
      protocol: "http",
      pathPrefix: "api",
    });
    expect(endpoint.child("test").replace({ protocol: "https" }).toString()).toEqual(
      "https://localhost:8080/api/test",
    );
  });
});

describe("base64", () => {
  test("encodeBase64 - produces unpadded url-safe output", () => {
    const encoded = url.encodeBase64(JSON.stringify({ is_block: true }));
    expect(encoded).not.toMatch(/[+/=]/);
  });

  test("encodeBase64 / decodeBase64 - round trips", () => {
    const value = JSON.stringify({ is_block: true, name: "valve ⚡" });
    expect(url.decodeBase64(url.encodeBase64(value))).toEqual(value);
  });

  test("encodeBase64 - stable through URI canonicalization", () => {
    const encoded = url.encodeBase64(JSON.stringify({ is_block: true }));
    const uri = `arc://block/test#${encoded}`;
    expect(new globalThis.URL(uri).toString()).toEqual(uri);
  });
});
