// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SourceMapGenerator } from "source-map-js";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { resolveStack } from "@/errors/resolveStack";

// The consumer cache inside resolveStack is keyed by script URL and lives for the
// whole session, so every spec uses a distinct script URL to stay isolated.
const HOST = "https://app.test";

const buildMap = (
  mappings: {
    generated: { line: number; column: number };
    original: { line: number; column: number };
    source: string;
    name?: string;
  }[],
): string => {
  const gen = new SourceMapGenerator({ file: "bundle.js" });
  mappings.forEach((m) => gen.addMapping(m));
  return gen.toString();
};

interface Fixture {
  bundleURL: string;
  bundle: string;
  map: string | null;
}

const fixtures: Fixture[] = [];

const addFixture = (key: string, map: string | null): string => {
  const bundleURL = `${HOST}/${key}.js`;
  const bundle =
    map != null ? `var x = 1;\n//# sourceMappingURL=${key}.js.map\n` : "var x = 1;\n";
  fixtures.push({ bundleURL, bundle, map });
  return bundleURL;
};

const fetchMock = vi.fn(async (url: string) => {
  for (const f of fixtures) {
    if (url === f.bundleURL) return new Response(f.bundle, { status: 200 });
    if (f.map != null && url === f.bundleURL.replace(/\.js$/, ".js.map"))
      return new Response(f.map, { status: 200 });
  }
  return new Response("not found", { status: 404 });
});

describe("resolveStack", () => {
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeAll(() => {
    vi.stubGlobal("fetch", fetchMock);
  });

  afterAll(() => {
    vi.unstubAllGlobals();
  });

  beforeEach(() => {
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    return () => {
      warnSpy.mockRestore();
    };
  });

  it("resolves both error and component stacks into source-mapped frames", async () => {
    const url = addFixture(
      "resolve-both",
      buildMap([
        {
          generated: { line: 1, column: 10 },
          original: { line: 42, column: 8 },
          source: "client/ts/src/channel/payload.ts",
          name: "decode",
        },
        {
          generated: { line: 1, column: 20 },
          original: { line: 155, column: 4 },
          source: "console/src/schematic/Schematic.tsx",
          name: "SchematicLayout",
        },
      ]),
    );
    const error = new Error("boom");
    // Stack columns are 1-based, so generated column 10 is stack column 11.
    error.stack = `Error: boom\n    at yte (${url}:1:11)`;
    const result = await resolveStack(error, `    at Cmp (${url}:1:21)`);
    expect(result.stack).toBe("  at decode (client/ts/src/channel/payload.ts:42:9)");
    expect(result.componentStack).toBe(
      "  at SchematicLayout (console/src/schematic/Schematic.tsx:155:5)",
    );
  });

  it("falls back to the raw minified name when the map carries none", async () => {
    const url = addFixture(
      "no-name",
      buildMap([
        {
          generated: { line: 1, column: 10 },
          original: { line: 7, column: 2 },
          source: "console/src/foo.ts",
        },
      ]),
    );
    const error = new Error("boom");
    error.stack = `Error: boom\n    at yte (${url}:1:11)`;
    const result = await resolveStack(error, null);
    expect(result.stack).toBe("  at yte (console/src/foo.ts:7:3)");
    expect(result.componentStack).toBeNull();
  });

  it("leaves componentStack null when none is provided", async () => {
    const url = addFixture(
      "no-component",
      buildMap([
        {
          generated: { line: 1, column: 10 },
          original: { line: 42, column: 8 },
          source: "client/ts/src/channel/payload.ts",
          name: "decode",
        },
      ]),
    );
    const error = new Error("boom");
    error.stack = `Error: boom\n    at yte (${url}:1:11)`;
    const result = await resolveStack(error, null);
    expect(result.stack).toBe("  at decode (client/ts/src/channel/payload.ts:42:9)");
    expect(result.componentStack).toBeNull();
  });

  it("resolves through a base64 data: URL sourcemap without a map fetch", async () => {
    const map = buildMap([
      {
        generated: { line: 1, column: 10 },
        original: { line: 3, column: 6 },
        source: "console/src/inline.ts",
        name: "inlineFn",
      },
    ]);
    const bundleURL = `${HOST}/data-url.js`;
    const bundle =
      "var x = 1;\n" +
      `//# sourceMappingURL=data:application/json;base64,${btoa(map)}\n`;
    fixtures.push({ bundleURL, bundle, map: null });
    const error = new Error("boom");
    error.stack = `Error: boom\n    at yte (${bundleURL}:1:11)`;
    fetchMock.mockClear();
    const result = await resolveStack(error, null);
    expect(result.stack).toBe("  at inlineFn (console/src/inline.ts:3:7)");
    expect(fetchMock.mock.calls.map((call) => call[0])).toEqual([bundleURL]);
  });

  it("keeps raw frames when the script has no sourcemap", async () => {
    const url = addFixture("no-map", null);
    const error = new Error("boom");
    error.stack = `Error: boom\n    at fn (${url}:1:11)`;
    const result = await resolveStack(error, null);
    expect(result.stack).toBe(`  at fn (${HOST}/no-map.js:1:11)`);
  });

  it("keeps raw frames and warns when the script cannot be fetched", async () => {
    const error = new Error("boom");
    error.stack = `Error: boom\n    at fn (${HOST}/missing.js:1:11)`;
    const result = await resolveStack(error, null);
    expect(result.stack).toBe(`  at fn (${HOST}/missing.js:1:11)`);
    expect(warnSpy).toHaveBeenCalledWith(
      expect.stringContaining("no usable sourcemap"),
      expect.any(Error),
    );
  });

  it("keeps the raw stack and warns when the stack cannot be parsed", async () => {
    const error = new Error("boom");
    error.stack = "";
    const result = await resolveStack(error, null);
    expect(result.stack).toBe("");
    expect(warnSpy).toHaveBeenCalledWith(
      "resolveStack: failed to resolve error stack",
      expect.anything(),
    );
  });

  it("keeps the resolved error stack when only the component leg fails", async () => {
    const url = addFixture(
      "component-fails",
      buildMap([
        {
          generated: { line: 1, column: 10 },
          original: { line: 42, column: 8 },
          source: "client/ts/src/channel/payload.ts",
          name: "decode",
        },
      ]),
    );
    const error = new Error("boom");
    error.stack = `Error: boom\n    at yte (${url}:1:11)`;
    // An unparseable component stack fails its leg while the error leg resolves.
    const result = await resolveStack(error, "");
    expect(result.stack).toBe("  at decode (client/ts/src/channel/payload.ts:42:9)");
    expect(result.componentStack).toBe("");
    expect(warnSpy).toHaveBeenCalledWith(
      "resolveStack: failed to resolve component stack",
      expect.anything(),
    );
  });

  it("fetches each script and map exactly once across resolutions", async () => {
    const url = addFixture(
      "cached",
      buildMap([
        {
          generated: { line: 1, column: 10 },
          original: { line: 1, column: 0 },
          source: "console/src/foo.ts",
          name: "fn",
        },
      ]),
    );
    const makeError = (): Error => {
      const error = new Error("boom");
      error.stack = `Error: boom\n    at a (${url}:1:11)\n    at b (${url}:1:11)`;
      return error;
    };
    fetchMock.mockClear();
    await resolveStack(makeError(), `    at Cmp (${url}:1:11)`);
    await resolveStack(makeError(), null);
    const fetched = fetchMock.mock.calls.map((call) => call[0]);
    expect(fetched).toEqual([url, url.replace(/\.js$/, ".js.map")]);
  });
});
