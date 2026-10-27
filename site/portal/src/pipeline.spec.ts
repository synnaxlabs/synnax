// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute, type MiddlewareHandler } from "astro";
import { eq } from "drizzle-orm";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { POST as unlinkRoute } from "@/pages/api/activations/[key]/unlink";
import { POST as linkRoute } from "@/pages/api/desktop/link";
import { POST as renewRoute } from "@/pages/api/desktop/renew";
import { POST as desktopUnlinkRoute } from "@/pages/api/desktop/unlink";
import { pipeline } from "@/pipeline";
import { schema } from "@/server/db";
import { type Memory, openMemory } from "@/server/db/testutil";
import { HASH_A } from "@/server/license/testutil";
import {
  body,
  type ContextArgs,
  createAPIContext,
  createHarness,
  type Harness,
} from "@/testutil";

const PORTAL = "https://portal.synnaxlabs.com";
const CROSS_SITE = { "sec-fetch-site": "cross-site", origin: "https://evil.example" };
const REFUSED = { error: "Cross-site requests are not allowed" };

interface Linked {
  secret: string;
  activation: string;
}

const form = (fields: Record<string, string>): FormData => {
  const data = new FormData();
  for (const [k, v] of Object.entries(fields)) data.set(k, v);
  return data;
};

describe("pipeline", () => {
  let store: Memory;
  let harness: Harness;
  let handler: MiddlewareHandler;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    harness = createHarness(store);
    harness.directory.people.user_a = { email: "ada@example.com", name: "Ada" };
    handler = pipeline({
      auth: async (_, next) => await next(),
      open: () => harness.portal,
    });
  });

  /** send runs a request through the pipeline, with `route` as the page it reaches. */
  const send = async (route: APIRoute, args: ContextArgs): Promise<Response> => {
    const context = createAPIContext(harness.portal, args);
    const res = await handler(context, async () => await route(context));
    if (res == null) throw new Error("the pipeline answered nothing");
    return res;
  };

  const link = async (headers: Record<string, string>): Promise<Response> =>
    await send(linkRoute, {
      path: "/api/desktop/link",
      body: form({ fingerprint: HASH_A, name: "stand-1" }),
      headers,
    });

  const linkMachine = async (): Promise<Linked> => {
    harness.signIn("user_a");
    const res = await link({ "sec-fetch-site": "same-origin", origin: PORTAL });
    expect(res.status).toBe(200);
    return await body<Linked>(res);
  };

  const secretHash = async (key: string): Promise<string | null> => {
    const [act] = await store.query
      .select()
      .from(schema.activation)
      .where(eq(schema.activation.key, key));
    return act.renewalSecretHash;
  };

  describe("renewal", () => {
    for (const origin of ["tauri://localhost", "http://tauri.localhost"])
      it(`should renew from ${origin} without a content type`, async () => {
        const { secret } = await linkMachine();
        harness.signIn(null);
        const res = await send(renewRoute, {
          path: "/api/desktop/renew",
          headers: {
            authorization: `Bearer ${secret}`,
            origin,
            "sec-fetch-site": "cross-site",
          },
        });
        expect(res.status).toBe(200);
        expect(res.headers.get("access-control-allow-origin")).toBe("*");
        expect(await body(res)).toHaveProperty("key");
      });

    it("should log out from the Desktop app's origin", async () => {
      const { secret } = await linkMachine();
      harness.signIn(null);
      const res = await send(desktopUnlinkRoute, {
        path: "/api/desktop/unlink",
        headers: {
          authorization: `Bearer ${secret}`,
          origin: "tauri://localhost",
          "sec-fetch-site": "cross-site",
        },
      });
      expect(res.status).toBe(204);
    });
  });

  describe("origin check", () => {
    it("should refuse a cross-site form post to a cookie route", async () => {
      const linked = await linkMachine();
      const res = await send(unlinkRoute, {
        path: `/api/activations/${linked.activation}/unlink`,
        params: { key: linked.activation },
        body: form({}),
        headers: CROSS_SITE,
      });
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual(REFUSED);
      expect(await secretHash(linked.activation)).not.toBeNull();
    });

    it("should refuse a JSON post from another site", async () => {
      harness.signIn("user_a");
      const res = await send(linkRoute, {
        path: "/api/desktop/link",
        body: { fingerprint: HASH_A, name: "stand-1" },
        headers: CROSS_SITE,
      });
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual(REFUSED);
    });

    it("should refuse a post from a sibling subdomain", async () => {
      harness.signIn("user_a");
      const res = await link({
        "sec-fetch-site": "same-site",
        origin: "https://www.synnaxlabs.com",
      });
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual(REFUSED);
    });

    it("should refuse a post from another origin without Sec-Fetch-Site", async () => {
      harness.signIn("user_a");
      const res = await link({ origin: "https://evil.example" });
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual(REFUSED);
    });

    it("should pass a same-origin post", async () => {
      harness.signIn("user_a");
      expect(
        (await link({ "sec-fetch-site": "same-origin", origin: PORTAL })).status,
      ).toBe(200);
    });

    it("should pass a post from the portal origin without Sec-Fetch-Site", async () => {
      harness.signIn("user_a");
      expect((await link({ origin: PORTAL })).status).toBe(200);
    });

    it("should pass a post with neither header, as a server sends", async () => {
      harness.signIn("user_a");
      expect((await link({})).status).toBe(200);
    });

    it("should pass a cross-site read", async () => {
      const res = await send(async () => new Response("read"), {
        method: "GET",
        path: "/licenses",
        headers: CROSS_SITE,
      });
      expect(res.status).toBe(200);
    });
  });

  describe("security headers", () => {
    afterEach(() => {
      vi.unstubAllEnvs();
    });

    it("should forbid other sites to frame a page outside development", async () => {
      vi.stubEnv("DEV", false);
      const res = await send(async () => new Response("page"), {
        method: "GET",
        path: "/licenses",
      });
      expect(res.headers.get("content-security-policy")).toContain(
        "frame-ancestors 'none'",
      );
    });
  });
});
