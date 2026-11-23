// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Portal } from "@/platform/portal";
import {
  createTestPortal,
  TEST_URL,
  type TestPortal,
} from "@/platform/portal/testutil";

describe("Portal.Client", () => {
  const answer = (status: number, body: unknown): TestPortal =>
    createTestPortal(async () => Response.json(body, { status }));

  const answerEmpty = (): TestPortal =>
    createTestPortal(async () => new Response(null, { status: 204 }));

  const answerPage = (status: number, statusText: string): TestPortal =>
    createTestPortal(
      async () => new Response("<html>Blocked</html>", { status, statusText }),
    );

  describe("renew", () => {
    it("should post the secret as a bearer token and return the new token", async () => {
      const { client, fetch } = answer(200, { key: "x.y.z" });
      expect(await client.renew("shh")).toEqual({ variant: "renewed", key: "x.y.z" });
      expect(fetch).toHaveBeenCalledWith(
        `${TEST_URL}/api/desktop/renew`,
        expect.objectContaining({
          method: "POST",
          headers: { authorization: "Bearer shh" },
        }),
      );
    });

    it("should report a machine the hub logged out as unlinked", async () => {
      const { client } = answer(403, {
        error: "This machine was logged out. Log in again.",
        code: "unlinked",
      });
      expect(await client.renew("shh")).toEqual({
        variant: "unlinked",
        message: "This machine was logged out. Log in again.",
      });
    });

    it("should throw on a refusal without the unlinked code", async () => {
      const { client } = answer(403, { error: "Access denied" });
      await expect(client.renew("shh")).rejects.toThrow(
        "Could not renew the license: Access denied",
      );
    });

    it("should throw on an unauthorized answer without the unlinked code", async () => {
      const { client } = answer(401, { error: "Sign in to the proxy" });
      await expect(client.renew("shh")).rejects.toThrow(
        "Could not renew the license: Sign in to the proxy",
      );
    });

    it("should throw on a page a proxy answers with", async () => {
      const { client } = answerPage(403, "Forbidden");
      await expect(client.renew("shh")).rejects.toThrow(
        "Could not renew the license: Forbidden",
      );
    });

    it("should throw on a server error that carries the unlinked code", async () => {
      const { client } = answer(500, { error: "Internal error", code: "unlinked" });
      await expect(client.renew("shh")).rejects.toThrow(
        "Could not renew the license: Internal error",
      );
    });

    it("should throw on any other refusal", async () => {
      const { client } = answer(429, { error: "Too many requests" });
      await expect(client.renew("shh")).rejects.toThrow(
        "Could not renew the license: Too many requests",
      );
    });
  });

  describe("unlink", () => {
    it("should post the secret as a bearer token, with a deadline", async () => {
      const { client, fetch } = answerEmpty();
      await client.unlink("shh");
      expect(fetch).toHaveBeenCalledWith(`${TEST_URL}/api/desktop/unlink`, {
        method: "POST",
        headers: { authorization: "Bearer shh" },
        signal: expect.any(AbortSignal),
      });
    });

    it("should throw with the message of a refusal", async () => {
      const { client } = answer(403, {
        error: "This machine is not logged in",
        code: "unlinked",
      });
      await expect(client.unlink("shh")).rejects.toThrow(
        "Could not log out: This machine is not logged in",
      );
    });

    it("should throw on a page a proxy answers with", async () => {
      const { client } = answerPage(403, "Forbidden");
      await expect(client.unlink("shh")).rejects.toThrow(
        "Could not log out: Forbidden",
      );
    });
  });

  describe("receive", () => {
    it("should post the code as a bearer token and return the key", async () => {
      const { client, fetch } = answer(200, { key: "x.y.z" });
      expect(await client.receive("abc")).toBe("x.y.z");
      expect(fetch).toHaveBeenCalledWith(`${TEST_URL}/api/handoff`, {
        method: "POST",
        headers: { authorization: "Bearer abc" },
        signal: expect.any(AbortSignal),
      });
    });

    it("should return null while the hub holds no key", async () => {
      const { client } = answerEmpty();
      expect(await client.receive("abc")).toBeNull();
    });

    it("should throw with the message of a refusal", async () => {
      const { client } = answer(401, { error: "Log in first" });
      await expect(client.receive("abc")).rejects.toThrow(
        "Could not get the license key: Log in first",
      );
    });

    const answerNever = (): TestPortal =>
      createTestPortal(
        async (_, init) =>
          await new Promise<Response>((_resolve, reject) =>
            init?.signal?.addEventListener("abort", () => reject(new Error("Aborted"))),
          ),
      );

    describe("when the hub does not answer in ten seconds", () => {
      beforeEach(() => vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] }));
      afterEach(() => vi.useRealTimers());

      it("should throw a TypeError", async () => {
        const result = answerNever().client.receive("abc");
        vi.advanceTimersByTime(10_000);
        await expect(result).rejects.toThrow(TypeError);
      });

      it("should throw a TypeError when the caller signal is not aborted", async () => {
        const { client } = answerNever();
        const result = client.receive("abc", new AbortController().signal);
        vi.advanceTimersByTime(10_000);
        await expect(result).rejects.toThrow(TypeError);
      });
    });

    it("should stop the request when the caller aborts", async () => {
      const controller = new AbortController();
      const result = answerNever().client.receive("abc", controller.signal);
      controller.abort();
      await expect(result).rejects.toThrow("Aborted");
    });
  });

  describe("activateURL", () => {
    it("should carry the fingerprint and the handoff code", () => {
      const { client } = createTestPortal();
      const url = new URL(
        client.activateURL({ fingerprint: ["a", "b"], handoff: "c" }),
      );
      expect(url.origin).toBe(TEST_URL);
      expect(url.pathname).toBe("/licenses/activate");
      expect(url.searchParams.get("fingerprint")).toBe("a,b");
      expect(url.searchParams.get("handoff")).toBe("c");
    });
  });

  describe("loginURL", () => {
    it("should carry the state, fingerprint, name, and version", () => {
      const { client } = createTestPortal();
      const url = new URL(
        client.loginURL({
          state: "s",
          fingerprint: ["aa", "bb"],
          name: "my-mac",
          version: "0.58.0",
        }),
      );
      expect(url.origin + url.pathname).toBe(`${TEST_URL}/desktop/login`);
      expect(url.searchParams.get("state")).toBe("s");
      expect(url.searchParams.get("fp")).toBe("aa, bb");
      expect(url.searchParams.get("name")).toBe("my-mac");
      expect(url.searchParams.get("v")).toBe("0.58.0");
    });

    it("should leave the version out when unknown", () => {
      const { client } = createTestPortal();
      const url = new URL(client.loginURL({ state: "s", fingerprint: [], name: "n" }));
      expect(url.searchParams.has("v")).toBe(false);
    });
  });
});

describe("Portal.joinFingerprint", () => {
  it("should join the host hashes with commas", () => {
    expect(Portal.joinFingerprint(["a", "b"])).toBe("a, b");
  });
});
