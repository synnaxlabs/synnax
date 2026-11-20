// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { JSDOM } from "jsdom";
import { describe, expect, it } from "vitest";

import Analytics from "./Analytics.astro";
import { hash, HASHES, jitless, policy } from "./csp";

describe("csp", () => {
  describe("hash", () => {
    it("should return the base64 SHA-256 source of the text", () => {
      expect(hash("")).toBe("sha256-47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=");
    });
  });

  describe("HASHES", () => {
    it("should let every inline script of the analytics run", async () => {
      const container = await AstroContainer.create();
      container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
      const html = await container.renderToString(Analytics, {
        props: { privacy: "/privacy" },
      });
      const scripts = [
        ...new JSDOM(html).window.document.querySelectorAll("script:not([src])"),
      ];
      expect(scripts.length).toBeGreaterThan(0);
      for (const script of scripts) expect(HASHES).toContain(hash(script.textContent));
    });
  });

  describe("policy", () => {
    const directives = (p: ReturnType<typeof policy>): string[] => p.directives ?? [];

    it("should refuse plugins and other base URLs", () => {
      expect(directives(policy({}))).toEqual([
        "default-src 'self'",
        "object-src 'none'",
        "base-uri 'self'",
      ]);
    });

    it("should list the sources of each directive", () => {
      const p = policy({
        sources: { "connect-src": ["'self'", "https://a.example"] },
        scripts: ["https://b.example"],
        hashes: ["sha256-x"],
      });
      expect(directives(p)).toContain("connect-src 'self' https://a.example");
      expect(p.scriptDirective).toEqual({
        resources: ["'self'", "https://b.example"],
        hashes: ["sha256-x"],
      });
    });

    it("should allow inline style attributes but not style elements", () => {
      expect(policy({}).styleDirective?.resources).toEqual([
        "'self'",
        { resource: "'unsafe-inline'", kind: "attribute" },
      ]);
      expect(policy({ inlineStyles: true }).styleDirective?.resources).toEqual([
        "'self'",
        "'unsafe-inline'",
      ]);
    });

    it("should leave the Vercel toolbar out of production", () => {
      const p = JSON.stringify(policy({ vercelEnv: "production" }));
      expect(p).not.toContain("vercel");
    });

    it("should allow the Vercel toolbar on a preview", () => {
      const p = policy({
        sources: { "img-src": ["'self'", "data:"] },
        vercelEnv: "preview",
      });
      expect(directives(p)).toEqual([
        "default-src 'self'",
        "connect-src 'self' https://vercel.live wss://ws-us3.pusher.com",
        "font-src 'self' https://vercel.live https://assets.vercel.com",
        "frame-src 'self' https://vercel.live",
        "img-src 'self' data: https://vercel.live https://vercel.com blob:",
        "object-src 'none'",
        "base-uri 'self'",
      ]);
      expect(p.scriptDirective?.resources).toContain("https://vercel.live");
      expect(p.styleDirective?.resources).toEqual([
        "'self'",
        "'unsafe-inline'",
        "https://vercel.live",
      ]);
    });
  });

  describe("jitless", () => {
    it("should turn on jitless before islands hydrate", () => {
      const scripts: [string, string][] = [];
      jitless().hooks["astro:config:setup"]({
        injectScript: (stage, content) => scripts.push([stage, content]),
      });
      expect(scripts).toEqual([
        ["before-hydration", 'import { z } from "zod"; z.config({ jitless: true });'],
      ]);
    });
  });
});
