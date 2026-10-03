// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import { type APIContext, type APIRoute, type InjectedRoute } from "astro";
import { beforeAll, describe, expect, it } from "vitest";

import { integration } from "./integration";

const CONTEXT = { site: new URL("https://synnaxlabs.com") } as APIContext;

type Setup = (options: { injectRoute: (route: InjectedRoute) => void }) => void;

describe("integration", () => {
  const routes: InjectedRoute[] = [];

  beforeAll(() => {
    const setup = integration().hooks["astro:config:setup"] as Setup;
    setup({ injectRoute: (route) => routes.push(route) });
  });

  const get = async (pattern: string): Promise<Response> => {
    const route = routes.find((r) => r.pattern === pattern);
    if (route == null) throw new Error(`no route injected at ${pattern}`);
    const { GET } = (await import(/* @vite-ignore */ route.entrypoint.toString())) as {
      GET: APIRoute;
    };
    return await GET(CONTEXT);
  };

  it("should prerender every route it injects", () => {
    expect(routes).toHaveLength(3);
    routes.forEach((route) => expect(route.prerender).toBe(true));
  });

  it("should serve the favicon as an icon", async () => {
    const response = await get("/favicon.ico");
    expect(response.headers.get("Content-Type")).toBe("image/x-icon");
    expect((await response.arrayBuffer()).byteLength).toBeGreaterThan(0);
  });

  it("should serve the favicon as an SVG", async () => {
    const response = await get("/favicon.svg");
    expect(response.headers.get("Content-Type")).toBe("image/svg+xml");
    expect(await response.text()).toContain("<svg");
  });

  it("should serve a robots.txt that points at the site's sitemap", async () => {
    const response = await get("/robots.txt");
    expect(response.headers.get("Content-Type")).toBe("text/plain; charset=utf-8");
    expect(await response.text()).toBe(`User-agent: *
Allow: /

Sitemap: https://synnaxlabs.com/sitemap.xml
`);
  });
});
