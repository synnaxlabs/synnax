// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { generateKeyPairSync, verify } from "crypto";
import { describe, expect, it, vi } from "vitest";

import { tokenSource } from "@/util/githubApp";

const { privateKey, publicKey } = generateKeyPairSync("rsa", {
  modulusLength: 2048,
  privateKeyEncoding: { type: "pkcs8", format: "pem" },
  publicKeyEncoding: { type: "spki", format: "pem" },
});

const NOW = Date.UTC(2026, 10, 18, 12);
const HOUR = TimeSpan.hours(1).milliseconds;

const json = (body: unknown, status = 200): Response =>
  new Response(JSON.stringify(body), { status });

const accessToken = (token: string, expires: number): Response =>
  json({ token, expires_at: new Date(expires).toISOString() }, 201);

const decode = (part: string): unknown =>
  JSON.parse(Buffer.from(part, "base64url").toString());

const bearer = (init: RequestInit | undefined): string => {
  const headers = new Headers(init?.headers);
  return headers.get("Authorization")?.replace("Bearer ", "") ?? "";
};

describe("tokenSource", () => {
  it("should be undefined with no App and no static token", () => {
    expect(tokenSource({})).toBeUndefined();
  });

  it("should resolve to the static token with no App", async () => {
    expect(await tokenSource({ token: "job" })?.()).toEqual("job");
  });

  it("should throw when only one of the ID and the key is set", () => {
    expect(() => tokenSource({ appID: "1" })).toThrow("both");
    expect(() => tokenSource({ privateKey })).toThrow("both");
  });

  it("should sign an RS256 JWT for the App", async () => {
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(json({ id: 7 }))
      .mockResolvedValueOnce(accessToken("t1", NOW + HOUR));
    await tokenSource({ appID: "42", privateKey, fetch, now: () => NOW })?.();
    const [header, payload, signature] = bearer(fetch.mock.calls[0]?.[1]).split(".");
    expect(decode(header)).toEqual({ alg: "RS256", typ: "JWT" });
    const seconds = NOW / 1000;
    expect(decode(payload)).toEqual({
      iat: seconds - 60,
      exp: seconds + 540,
      iss: "42",
    });
    const signed = Buffer.from(`${header}.${payload}`);
    expect(
      verify("sha256", signed, publicKey, Buffer.from(signature, "base64url")),
    ).toBe(true);
  });

  it("should exchange the installation for an access token", async () => {
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(json({ id: 7 }))
      .mockResolvedValueOnce(accessToken("t1", NOW + HOUR));
    const token = tokenSource({ appID: "42", privateKey, fetch, now: () => NOW });
    expect(await token?.()).toEqual("t1");
    expect(fetch.mock.calls[0]?.[0]).toEqual(
      "https://api.github.com/orgs/synnaxlabs/installation",
    );
    expect(fetch.mock.calls[1]?.[0]).toEqual(
      "https://api.github.com/app/installations/7/access_tokens",
    );
    expect(fetch.mock.calls[1]?.[1]?.method).toEqual("POST");
  });

  it("should cache the token until five minutes before it expires", async () => {
    let time = NOW;
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(json({ id: 7 }))
      .mockResolvedValueOnce(accessToken("t1", NOW + HOUR))
      .mockResolvedValueOnce(json({ id: 7 }))
      .mockResolvedValueOnce(accessToken("t2", NOW + 2 * HOUR));
    const token = tokenSource({ appID: "42", privateKey, fetch, now: () => time });
    expect(await token?.()).toEqual("t1");
    time = NOW + HOUR - TimeSpan.minutes(5).milliseconds - 1;
    expect(await token?.()).toEqual("t1");
    expect(fetch).toHaveBeenCalledTimes(2);
    time += 1;
    expect(await token?.()).toEqual("t2");
    expect(fetch).toHaveBeenCalledTimes(4);
  });

  it("should throw on an error response and retry on the next call", async () => {
    const fetch = vi
      .fn<typeof globalThis.fetch>()
      .mockResolvedValueOnce(json({ message: "Not Found" }, 404))
      .mockResolvedValueOnce(json({ id: 7 }))
      .mockResolvedValueOnce(json({ message: "Unauthorized" }, 401))
      .mockResolvedValueOnce(json({ id: 7 }))
      .mockResolvedValueOnce(accessToken("t1", NOW + HOUR));
    const token = tokenSource({ appID: "42", privateKey, fetch, now: () => NOW });
    await expect(token?.()).rejects.toThrow("404");
    await expect(token?.()).rejects.toThrow("401");
    expect(await token?.()).toEqual("t1");
  });
});
