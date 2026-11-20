// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";

import {
  type ClientDirectiveConfig,
  type InjectedRoute,
  type InjectedType,
} from "astro";
import { beforeAll, describe, expect, it } from "vitest";

import { integration } from "./integration";

describe("integration", () => {
  const routes: InjectedRoute[] = [];
  const directives: ClientDirectiveConfig[] = [];
  const types: InjectedType[] = [];

  beforeAll(() => {
    const { hooks } = integration();
    hooks["astro:config:setup"]({
      injectRoute: (route) => routes.push(route),
      addClientDirective: (directive) => directives.push(directive),
    });
    hooks["astro:config:done"]({
      injectTypes: (type) => {
        types.push(type);
        return new URL("file:///types.d.ts");
      },
    });
  });

  it("should inject the favicons, the robots.txt, and the consent route", () => {
    expect(routes.map(({ pattern }) => pattern)).toEqual([
      "/favicon.ico",
      "/favicon.svg",
      "/robots.txt",
      "/consent",
    ]);
  });

  it("should prerender every route but the consent route", () => {
    routes.forEach(({ pattern, prerender }) =>
      expect(prerender).toBe(pattern !== "/consent"),
    );
  });

  it("should point every route at an existing entry point", () => {
    routes.forEach(({ entrypoint }) =>
      expect(existsSync(fileURLToPath(entrypoint))).toBe(true),
    );
  });

  it("should add the interaction directive from an existing entry point", () => {
    expect(directives.map(({ name }) => name)).toEqual(["interaction"]);
    expect(existsSync(fileURLToPath(directives[0].entrypoint))).toBe(true);
  });

  it("should declare the interaction directive for type checks", () => {
    expect(types).toHaveLength(1);
    expect(types[0].content).toContain('"client:interaction"');
  });
});
