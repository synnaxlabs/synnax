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

const FIELDS = {
  state: "s",
  key: "a.b.c",
  secret: "shh",
  activation: "act",
  email: "someone@example.com",
  user: "user_a",
};

const linkOf = (params: Record<string, string>): string =>
  `synnax-desktop://activate?${new URLSearchParams(params).toString()}`;

describe("account handoff", () => {
  describe("mintState", () => {
    it("should mint a state the hub accepts that differs each time", () => {
      const a = Account.mintState();
      expect(a).toMatch(/^[A-Za-z0-9_-]{16,128}$/);
      expect(a).not.toBe(Account.mintState());
    });
  });

  describe("loginURL", () => {
    it("should carry the state, fingerprint, name, and version", () => {
      const url = new URL(
        Account.loginURL({
          state: "s",
          fingerprint: ["aa", "bb"],
          name: "my-mac",
          version: "0.58.0",
        }),
      );
      expect(url.origin + url.pathname).toBe(License.LOGIN_URL);
      expect(url.searchParams.get("state")).toBe("s");
      expect(url.searchParams.get("fp")).toBe("aa, bb");
      expect(url.searchParams.get("name")).toBe("my-mac");
      expect(url.searchParams.get("v")).toBe("0.58.0");
    });

    it("should leave the version out when unknown", () => {
      const url = new URL(Account.loginURL({ state: "s", fingerprint: [], name: "n" }));
      expect(url.searchParams.has("v")).toBe(false);
    });
  });

  describe("parseLink", () => {
    it("should read every field of a login link", () => {
      expect(Account.parseLink(linkOf({ ...FIELDS, analytics: "on" }))).toEqual({
        ...FIELDS,
        analyticsAllowed: true,
      });
    });

    it("should refuse analytics when the person said no", () => {
      const linked = Account.parseLink(linkOf({ ...FIELDS, analytics: "off" }));
      expect(linked.analyticsAllowed).toBe(false);
    });

    it("should refuse analytics when the link does not say", () => {
      expect(Account.parseLink(linkOf(FIELDS)).analyticsAllowed).toBe(false);
    });

    it("should refuse analytics for any value other than on", () => {
      const linked = Account.parseLink(linkOf({ ...FIELDS, analytics: "true" }));
      expect(linked.analyticsAllowed).toBe(false);
    });

    it("should refuse a link with another form", () => {
      expect(() => Account.parseLink("synnax://cluster/abc")).toThrow(
        "Login links must be of the form",
      );
    });

    it("should refuse a link that misses a field", () => {
      const { secret: _, ...rest } = FIELDS;
      expect(() => Account.parseLink(linkOf(rest))).toThrow(
        "The login link is missing its secret",
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

    const answerPage = (status: number, statusText: string): void => {
      vi.stubGlobal(
        "fetch",
        vi.fn(async () => new Response("<html>Blocked</html>", { status, statusText })),
      );
    };

    it("should post the secret as a bearer token and return the new token", async () => {
      answer(200, { key: "x.y.z" });
      const result = await Account.renew("shh");
      expect(result).toEqual({ variant: "renewed", key: "x.y.z" });
      const fetchMock = vi.mocked(fetch);
      expect(fetchMock).toHaveBeenCalledWith(
        License.RENEW_URL,
        expect.objectContaining({
          method: "POST",
          headers: { authorization: "Bearer shh" },
        }),
      );
    });

    it("should report a machine the hub logged out as unlinked", async () => {
      answer(403, {
        error: "This machine was logged out. Log in again.",
        code: "unlinked",
      });
      expect(await Account.renew("shh")).toEqual({
        variant: "unlinked",
        message: "This machine was logged out. Log in again.",
      });
    });

    it("should throw on a refusal without the unlinked code", async () => {
      answer(403, { error: "Access denied" });
      await expect(Account.renew("shh")).rejects.toThrow(
        "Could not renew the license: Access denied",
      );
    });

    it("should throw on an unauthorized answer without the unlinked code", async () => {
      answer(401, { error: "Sign in to the proxy" });
      await expect(Account.renew("shh")).rejects.toThrow(
        "Could not renew the license: Sign in to the proxy",
      );
    });

    it("should throw on a page a proxy answers with", async () => {
      answerPage(403, "Forbidden");
      await expect(Account.renew("shh")).rejects.toThrow(
        "Could not renew the license: Forbidden",
      );
    });

    it("should throw on a server error that carries the unlinked code", async () => {
      answer(500, { error: "Internal error", code: "unlinked" });
      await expect(Account.renew("shh")).rejects.toThrow(
        "Could not renew the license: Internal error",
      );
    });

    it("should throw on any other refusal", async () => {
      answer(429, { error: "Too many requests" });
      await expect(Account.renew("shh")).rejects.toThrow(
        "Could not renew the license: Too many requests",
      );
    });
  });
});
