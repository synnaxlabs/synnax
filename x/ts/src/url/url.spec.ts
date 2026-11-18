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

describe("buildQueryString", () => {
  test("should join the entries with a leading question mark", () => {
    expect(url.buildQueryString({ a: "1", b: "2" })).toEqual("?a=1&b=2");
  });

  test("should prepend the prefix to each key", () => {
    expect(url.buildQueryString({ a: "1", b: "2" }, "ctx")).toEqual("?ctxa=1&ctxb=2");
  });

  test("should skip null and undefined values", () => {
    expect(url.buildQueryString({ a: null, b: "2", c: undefined })).toEqual("?b=2");
  });

  test("should keep an empty string value", () => {
    expect(url.buildQueryString({ a: "" })).toEqual("?a=");
  });

  test("should percent-encode keys and values", () => {
    expect(
      url.buildQueryString({ "Content-Type": "application/json", "a&b": "x=y" }),
    ).toEqual("?Content-Type=application%2Fjson&a%26b=x%3Dy");
  });

  test("should encode a space as a plus", () => {
    expect(url.buildQueryString({ Authorization: "Bearer token" })).toEqual(
      "?Authorization=Bearer+token",
    );
  });

  test("should round trip through URLSearchParams", () => {
    const record = { token: "a+b c/d?e&f=g" };
    const parsed = new URLSearchParams(url.buildQueryString(record));
    expect(parsed.get("token")).toEqual(record.token);
  });

  test("should return a lone question mark for an empty record", () => {
    expect(url.buildQueryString({})).toEqual("?");
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
