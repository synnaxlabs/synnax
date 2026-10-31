// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

const tauri = vi.hoisted(() => ({
  invoke: vi.fn(async (cmd: string, args?: unknown): Promise<unknown> => {
    tauri.calls.push({ cmd, args });
    if (cmd === "oidc_listen") return 4242;
    if (cmd === "oidc_wait") return tauri.query ?? `code=abc&state=${tauri.state}`;
    return undefined;
  }),
  calls: [] as Array<{ cmd: string; args: unknown }>,
  proofs: [] as unknown[],
  opened: [] as string[],
  state: "",
  query: undefined as string | undefined,
}));
const runtime = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));
vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, runtime);
});
vi.mock("@synnaxlabs/client", async (importOriginal) => {
  const client = await importOriginal<{ auth: object }>();
  return {
    ...client,
    auth: {
      ...client.auth,
      createSession: async (_: unknown, proof: unknown) => {
        tauri.proofs.push(proof);
        return { token: "sys_token", session: {} };
      },
    },
  };
});
vi.mock("@tauri-apps/api/core", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  invoke: tauri.invoke,
}));
vi.mock("@/platform/runtime/openExternal", () => ({
  openExternal: async (url: string) => {
    tauri.opened.push(url);
    tauri.state = new URL(url).searchParams.get("state") ?? "";
  },
}));

import { Core } from "@/platform/core";

const LOGIN = {
  provider: "idp",
  authorizationEndpoint: "https://idp.example.com/authorize?tenant=a",
  clientId: "synnax",
};

const sha256 = async (s: string): Promise<string> =>
  btoa(
    String.fromCharCode(
      ...new Uint8Array(
        await crypto.subtle.digest("SHA-256", new TextEncoder().encode(s)),
      ),
    ),
  )
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");

describe("oidc", () => {
  afterEach(() => {
    sessionStorage.clear();
    window.history.replaceState(null, "", "/");
  });

  describe("loginDesktop", () => {
    const core = { key: "alpha", host: "localhost", port: 9090, secure: false };
    afterEach(() => {
      tauri.query = undefined;
    });

    it("should exchange the loopback code and save the session's token", async () => {
      await Core.loginDesktop(core, LOGIN);
      const proof = tauri.proofs[tauri.proofs.length - 1] as Record<string, string>;
      expect(proof).toMatchObject({
        method: "oidc",
        provider: "idp",
        code: "abc",
        redirectUri: "http://127.0.0.1:4242/callback",
      });
      expect(proof.verifier).toMatch(/^[A-Za-z0-9_-]{43}$/);
      const opened = new URL(tauri.opened[tauri.opened.length - 1]);
      expect(opened.origin + opened.pathname).toBe("https://idp.example.com/authorize");
      expect(Object.fromEntries(opened.searchParams)).toEqual({
        tenant: "a",
        response_type: "code",
        client_id: "synnax",
        redirect_uri: "http://127.0.0.1:4242/callback",
        scope: "openid profile",
        code_challenge: await sha256(proof.verifier),
        code_challenge_method: "S256",
        state: tauri.state,
      });
      expect(tauri.calls).toContainEqual({
        cmd: "oidc_wait",
        args: { state: tauri.state },
      });
      expect(tauri.calls).toContainEqual({
        cmd: "keychain_set",
        args: { account: "http://@localhost:9090#alpha", secret: "sys_token" },
      });
    });

    it("should reject a redirect for another login", async () => {
      tauri.query = "code=abc&state=other";
      await expect(Core.loginDesktop(core, LOGIN)).rejects.toThrow(
        "The login redirect does not belong to this login",
      );
    });

    it("should report the provider's refusal", async () => {
      tauri.query = "error=access_denied&error_description=Denied";
      await expect(Core.loginDesktop(core, LOGIN)).rejects.toThrow("Denied");
    });
  });

  describe("beginBrowserLogin", () => {
    it("should save the verifier for the page the provider returns to", async () => {
      await Core.beginBrowserLogin("alpha", LOGIN);
      const pending = JSON.parse(sessionStorage.getItem("synnax-oidc-login") ?? "{}");
      expect(pending).toMatchObject({
        core: "alpha",
        provider: "idp",
        redirectUri: `${window.location.origin}/`,
      });
    });
  });

  describe("completeBrowserLogin", () => {
    const pending = (state: string) =>
      sessionStorage.setItem(
        "synnax-oidc-login",
        JSON.stringify({
          core: "alpha",
          provider: "idp",
          verifier: "verifier",
          state,
          redirectUri: "http://localhost:3000/",
        }),
      );

    it("should return nothing on a page that is not a redirect", () => {
      pending("s");
      expect(Core.completeBrowserLogin()).toBeUndefined();
    });

    it("should turn the redirect into credentials and clear it", () => {
      pending("s");
      window.history.replaceState(null, "", "/?code=abc&state=s");
      expect(Core.completeBrowserLogin()).toEqual({
        core: "alpha",
        credentials: {
          provider: "idp",
          code: "abc",
          verifier: "verifier",
          redirectUri: "http://localhost:3000/",
        },
      });
      expect(window.location.search).toBe("");
      expect(sessionStorage.getItem("synnax-oidc-login")).toBeNull();
    });

    it("should reject a redirect for another login", () => {
      pending("s");
      window.history.replaceState(null, "", "/?code=abc&state=other");
      expect(() => Core.completeBrowserLogin()).toThrow(
        "The login redirect does not belong to this login",
      );
      expect(window.location.search).toBe("");
    });

    it("should report the provider's refusal", () => {
      pending("s");
      window.history.replaceState(
        null,
        "",
        "/?error=access_denied&error_description=Denied&state=s",
      );
      expect(() => Core.completeBrowserLogin()).toThrow("Denied");
    });
  });
});
