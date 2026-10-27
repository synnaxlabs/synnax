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
    if (cmd === "oidc_wait") return `code=abc&state=${tauri.state}`;
    return undefined;
  }),
  calls: [] as Array<{ cmd: string; args: unknown }>,
  proofs: [] as unknown[],
  opened: [] as string[],
  state: "",
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

import {
  authorizeURL,
  beginBrowserLogin,
  completeBrowserLogin,
  createPKCE,
  loginDesktop,
  parseCallback,
} from "@/platform/core/oidc";

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

  describe("createPKCE", () => {
    it("should derive the challenge from the verifier with S256", async () => {
      const { verifier, challenge } = await createPKCE();
      expect(verifier).toMatch(/^[A-Za-z0-9_-]{43}$/);
      expect(challenge).toBe(await sha256(verifier));
    });
  });

  describe("authorizeURL", () => {
    it("should ask for a code with a PKCE challenge", () => {
      const url = new URL(
        authorizeURL({
          login: LOGIN,
          redirectUri: "http://127.0.0.1:4242/callback",
          challenge: "challenge",
          state: "state",
        }),
      );
      expect(url.origin + url.pathname).toBe("https://idp.example.com/authorize");
      expect(Object.fromEntries(url.searchParams)).toEqual({
        tenant: "a",
        response_type: "code",
        client_id: "synnax",
        redirect_uri: "http://127.0.0.1:4242/callback",
        scope: "openid profile",
        code_challenge: "challenge",
        code_challenge_method: "S256",
        state: "state",
      });
    });
  });

  describe("parseCallback", () => {
    it("should return the code of a matching redirect", () => {
      expect(parseCallback("?code=abc&state=s", "s")).toBe("abc");
    });
    it("should reject a redirect for another login", () => {
      expect(() => parseCallback("?code=abc&state=other", "s")).toThrow(
        "The login redirect does not belong to this login",
      );
    });
    it("should report the provider's refusal", () => {
      expect(() =>
        parseCallback("?error=access_denied&error_description=Denied", "s"),
      ).toThrow("Denied");
    });
  });

  describe("loginDesktop", () => {
    it("should exchange the loopback code and save the session's token", async () => {
      const core = { host: "localhost", port: 9090, secure: false };
      await loginDesktop(core, LOGIN);
      const proof = tauri.proofs[tauri.proofs.length - 1] as Record<string, string>;
      expect(proof).toMatchObject({
        method: "oidc",
        provider: "idp",
        code: "abc",
        redirectUri: "http://127.0.0.1:4242/callback",
      });
      const opened = new URL(tauri.opened[tauri.opened.length - 1]);
      expect(opened.searchParams.get("redirect_uri")).toBe(
        "http://127.0.0.1:4242/callback",
      );
      expect(opened.searchParams.get("code_challenge")).toBe(
        await sha256(proof.verifier),
      );
      expect(tauri.calls).toContainEqual({
        cmd: "keychain_set",
        args: { account: "http://@localhost:9090", secret: "sys_token" },
      });
    });
  });

  describe("beginBrowserLogin", () => {
    it("should save the verifier for the page the provider returns to", async () => {
      await beginBrowserLogin("alpha", LOGIN);
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
      expect(completeBrowserLogin()).toBeUndefined();
    });

    it("should turn the redirect into credentials and clear it", () => {
      pending("s");
      window.history.replaceState(null, "", "/?code=abc&state=s");
      expect(completeBrowserLogin()).toEqual({
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
      expect(() => completeBrowserLogin()).toThrow(
        "The login redirect does not belong to this login",
      );
      expect(window.location.search).toBe("");
    });
  });
});
