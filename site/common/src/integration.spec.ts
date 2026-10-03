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

import { type InjectedRoute } from "astro";
import { beforeAll, describe, expect, it } from "vitest";

import { integration } from "./integration";

describe("integration", () => {
  const routes: InjectedRoute[] = [];

  beforeAll(() => {
    integration().hooks["astro:config:setup"]({
      injectRoute: (route) => routes.push(route),
    });
  });

  it("should inject the favicons and the robots.txt", () => {
    expect(routes.map(({ pattern }) => pattern)).toEqual([
      "/favicon.ico",
      "/favicon.svg",
      "/robots.txt",
    ]);
  });

  it("should prerender every route", () => {
    routes.forEach(({ prerender }) => expect(prerender).toBe(true));
  });

  it("should point every route at an existing entry point", () => {
    routes.forEach(({ entrypoint }) =>
      expect(existsSync(fileURLToPath(entrypoint))).toBe(true),
    );
  });
});
