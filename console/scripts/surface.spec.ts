// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { writeFileSync } from "node:fs";

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { PERSIST_SCOPES } from "@/session/store";

import { report } from "./surface.ts";

interface State {
  stored: unknown;
  scratch: unknown;
}

const scopes = (stored: z.ZodType<unknown>) => ({
  global: { stored },
  core: {},
  project: {},
  window: {},
  transient: ["scratch" as const],
});

describe("report", () => {
  it("should list each field of a stored slice and each transient slice", () => {
    const schema = z.object({ version: z.literal(0).default(0), name: z.string() });
    expect(report<State>(scopes(schema)).split("\n")).toEqual([
      'global.stored.properties.name.type "string"',
      "global.stored.properties.version.const 0",
      "global.stored.properties.version.default 0",
      'global.stored.properties.version.type "number"',
      'global.stored.required ["name"]',
      'global.stored.type "object"',
      "transient scratch",
      "",
    ]);
  });

  it("should change when a stored field changes type", () => {
    const before = report<State>(scopes(z.object({ size: z.number() })));
    const after = report<State>(scopes(z.object({ size: z.string() })));
    expect(after).not.toEqual(before);
  });

  it("should report a field whose schema has no JSON form", () => {
    const before = report<State>(scopes(z.object({})));
    const after = report<State>(scopes(z.object({ extra: z.unknown() })));
    expect(after).toContain("global.stored.properties.extra {}");
    expect(after).not.toEqual(before);
  });

  it("should not change when fields change order", () => {
    const before = report<State>(scopes(z.object({ a: z.number(), b: z.number() })));
    const after = report<State>(scopes(z.object({ b: z.number(), a: z.number() })));
    expect(after).toEqual(before);
  });

  it("should give the same report of the Console on each run", () => {
    const text = report(PERSIST_SCOPES);
    expect(text).toEqual(report(PERSIST_SCOPES));
    const out = process.env.SURFACE_OUT;
    if (out != null) writeFileSync(out, text);
  });
});
