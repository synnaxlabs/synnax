// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { Account } from "@/feature/account";
import { License } from "@/platform/license";

const LINKED: Account.Linked = {
  state: "s",
  token: "a.b.c",
  secret: "shh",
  activation: "act",
  email: "someone@example.com",
};

const linkOf = (linked: Partial<Account.Linked>): string =>
  `${Account.SCHEME}://activate?${new URLSearchParams(linked).toString()}`;

describe("account portal", () => {
  describe("mintState", () => {
    it("should mint a state the portal accepts that differs each time", () => {
      const a = Account.mintState();
      expect(a).toMatch(/^[A-Za-z0-9_-]{16,128}$/);
      expect(a).not.toBe(Account.mintState());
    });
  });

  describe("signInURL", () => {
    it("should carry the state, fingerprint, name, and version", () => {
      const url = new URL(
        Account.signInURL({
          state: "s",
          fingerprint: ["aa", "bb"],
          name: "my-mac",
          version: "0.58.0",
        }),
      );
      expect(url.origin + url.pathname).toBe(License.PORTAL_SIGN_IN_URL);
      expect(url.searchParams.get("state")).toBe("s");
      expect(url.searchParams.get("fp")).toBe("aa, bb");
      expect(url.searchParams.get("name")).toBe("my-mac");
      expect(url.searchParams.get("v")).toBe("0.58.0");
    });

    it("should leave the version out when unknown", () => {
      const url = new URL(
        Account.signInURL({ state: "s", fingerprint: [], name: "n" }),
      );
      expect(url.searchParams.has("v")).toBe(false);
    });
  });

  describe("parseLink", () => {
    it("should read every field of a sign-in link", () => {
      expect(Account.parseLink(linkOf(LINKED))).toEqual(LINKED);
    });

    it("should refuse a link with another form", () => {
      expect(() => Account.parseLink("synnax://cluster/abc")).toThrow(
        "Sign-in links must be of the form",
      );
    });

    it("should refuse a link that misses a field", () => {
      const { secret: _, ...rest } = LINKED;
      expect(() => Account.parseLink(linkOf(rest))).toThrow(
        "The sign-in link is missing its secret",
      );
    });
  });

  describe("renew", () => {
    afterEach(() => vi.unstubAllGlobals());

    const answer = (status: number, body: unknown): void => {
      vi.stubGlobal(
        "fetch",
        vi.fn(async () => Response.json(body, { status })),
      );
    };

    it("should post the secret as a bearer token and return the new token", async () => {
      answer(200, { token: "x.y.z" });
      const result = await Account.renew("shh");
      expect(result).toEqual({ variant: "renewed", token: "x.y.z" });
      const fetchMock = vi.mocked(fetch);
      expect(fetchMock).toHaveBeenCalledWith(
        License.PORTAL_RENEW_URL,
        expect.objectContaining({
          method: "POST",
          headers: { authorization: "Bearer shh" },
        }),
      );
    });

    it("should report a machine the portal no longer knows as unlinked", async () => {
      answer(403, { error: "This machine was unlinked. Sign in again." });
      expect(await Account.renew("shh")).toEqual({
        variant: "unlinked",
        message: "This machine was unlinked. Sign in again.",
      });
    });

    it("should throw on any other refusal", async () => {
      answer(429, { error: "Too many requests" });
      await expect(Account.renew("shh")).rejects.toThrow(
        "The portal refused the renewal: Too many requests",
      );
    });
  });
});
