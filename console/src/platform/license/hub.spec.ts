// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { License } from "@/platform/license";

describe("hub", () => {
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

  describe("renew", () => {
    it("should post the secret as a bearer token and return the new token", async () => {
      answer(200, { key: "x.y.z" });
      const result = await License.renew("shh");
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
      expect(await License.renew("shh")).toEqual({
        variant: "unlinked",
        message: "This machine was logged out. Log in again.",
      });
    });

    it("should throw on a refusal without the unlinked code", async () => {
      answer(403, { error: "Access denied" });
      await expect(License.renew("shh")).rejects.toThrow(
        "Could not renew the license: Access denied",
      );
    });

    it("should throw on an unauthorized answer without the unlinked code", async () => {
      answer(401, { error: "Sign in to the proxy" });
      await expect(License.renew("shh")).rejects.toThrow(
        "Could not renew the license: Sign in to the proxy",
      );
    });

    it("should throw on a page a proxy answers with", async () => {
      answerPage(403, "Forbidden");
      await expect(License.renew("shh")).rejects.toThrow(
        "Could not renew the license: Forbidden",
      );
    });

    it("should throw on a server error that carries the unlinked code", async () => {
      answer(500, { error: "Internal error", code: "unlinked" });
      await expect(License.renew("shh")).rejects.toThrow(
        "Could not renew the license: Internal error",
      );
    });

    it("should throw on any other refusal", async () => {
      answer(429, { error: "Too many requests" });
      await expect(License.renew("shh")).rejects.toThrow(
        "Could not renew the license: Too many requests",
      );
    });
  });

  describe("unlink", () => {
    it("should post the secret as a bearer token, with a deadline", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn(async () => new Response(null, { status: 204 })),
      );
      await License.unlink("shh");
      expect(vi.mocked(fetch)).toHaveBeenCalledWith(License.UNLINK_URL, {
        method: "POST",
        headers: { authorization: "Bearer shh" },
        signal: expect.any(AbortSignal),
      });
    });

    it("should throw with the message of a refusal", async () => {
      answer(403, { error: "This machine is not logged in", code: "unlinked" });
      await expect(License.unlink("shh")).rejects.toThrow(
        "Could not log out: This machine is not logged in",
      );
    });

    it("should throw on a page a proxy answers with", async () => {
      answerPage(403, "Forbidden");
      await expect(License.unlink("shh")).rejects.toThrow(
        "Could not log out: Forbidden",
      );
    });
  });

  describe("receive", () => {
    it("should post the code as a bearer token and return the key", async () => {
      answer(200, { key: "x.y.z" });
      expect(await License.receive("abc")).toBe("x.y.z");
      expect(vi.mocked(fetch)).toHaveBeenCalledWith(License.HANDOFF_URL, {
        method: "POST",
        headers: { authorization: "Bearer abc" },
        signal: undefined,
      });
    });

    it("should return null while the hub holds no key", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn(async () => new Response(null, { status: 204 })),
      );
      expect(await License.receive("abc")).toBeNull();
    });

    it("should throw with the message of a refusal", async () => {
      answer(401, { error: "Log in first" });
      await expect(License.receive("abc")).rejects.toThrow(
        "Could not get the license key: Log in first",
      );
    });
  });

  describe("activateURL", () => {
    it("should carry the fingerprint and the handoff code", () => {
      const url = new URL(
        License.activateURL({ fingerprint: ["a", "b"], handoff: "c" }),
      );
      expect(url.pathname).toBe("/licenses/activate");
      expect(url.searchParams.get("fingerprint")).toBe("a,b");
      expect(url.searchParams.get("handoff")).toBe("c");
    });
  });
});
