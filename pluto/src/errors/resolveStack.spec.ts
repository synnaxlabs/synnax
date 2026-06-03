// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

import { resolveStack } from "@/errors/resolveStack";

const { fromError } = vi.hoisted(() => ({ fromError: vi.fn() }));

vi.mock("stacktrace-js", () => ({ default: { fromError } }));

describe("resolveStack", () => {
  beforeEach(() => {
    fromError.mockReset();
  });

  it("resolves both error and component stacks into formatted source-mapped frames", async () => {
    const error = new Error("boom");
    error.stack = "ERROR_STACK";
    fromError.mockImplementation(async (err: Error) => {
      if (err.stack === "ERROR_STACK")
        return [
          {
            functionName: "decode",
            fileName: "client/ts/src/channel/payload.ts",
            lineNumber: 42,
            columnNumber: 8,
          },
          {
            functionName: "loadSchematic",
            fileName: "console/src/schematic/retrieve.ts",
            lineNumber: 67,
            columnNumber: 14,
          },
        ];
      if (err.stack === "COMPONENT_STACK")
        return [
          {
            functionName: "SchematicLayout",
            fileName: "console/src/schematic/Schematic.tsx",
            lineNumber: 155,
          },
        ];
      throw new Error(`unexpected stack: ${err.stack}`);
    });
    const result = await resolveStack(error, "COMPONENT_STACK");
    expect(result.stack).toBe(
      "  at decode (client/ts/src/channel/payload.ts:42:8)\n" +
        "  at loadSchematic (console/src/schematic/retrieve.ts:67:14)",
    );
    expect(result.componentStack).toBe(
      "  at SchematicLayout (console/src/schematic/Schematic.tsx:155)",
    );
  });

  it("leaves componentStack null when none is provided", async () => {
    fromError.mockResolvedValue([
      {
        functionName: "decode",
        fileName: "client/ts/src/channel/payload.ts",
        lineNumber: 42,
        columnNumber: 8,
      },
    ]);
    const result = await resolveStack(new Error("boom"), null);
    expect(result.stack).toBe("  at decode (client/ts/src/channel/payload.ts:42:8)");
    expect(result.componentStack).toBeNull();
  });

  it("rejects when stacktrace-js throws so the caller can surface the failure", async () => {
    fromError.mockRejectedValue(new Error("parse failed"));
    await expect(resolveStack(new Error("boom"), "anything")).rejects.toThrow(
      "parse failed",
    );
  });

  it("formats frames missing function names as <anonymous>", async () => {
    const error = new Error("boom");
    error.stack = "ERROR_STACK";
    fromError.mockImplementation(async (err: Error) => {
      if (err.stack === "ERROR_STACK")
        return [{ fileName: "a.ts", lineNumber: 1, columnNumber: 1 }];
      if (err.stack === "COMPONENT_STACK")
        return [{ fileName: "b.tsx", lineNumber: 2, columnNumber: 3 }];
      throw new Error(`unexpected stack: ${err.stack}`);
    });
    const result = await resolveStack(error, "COMPONENT_STACK");
    expect(result.stack).toBe("  at <anonymous> (a.ts:1:1)");
    expect(result.componentStack).toBe("  at <anonymous> (b.tsx:2:3)");
  });

  it("strips webpack:/// and leading ../ from file names", async () => {
    const error = new Error("boom");
    error.stack = "ERROR_STACK";
    fromError.mockImplementation(async (err: Error) => {
      if (err.stack === "ERROR_STACK")
        return [
          {
            functionName: "fn",
            fileName: "webpack:///../../console/src/foo.ts",
            lineNumber: 10,
            columnNumber: 5,
          },
        ];
      if (err.stack === "COMPONENT_STACK")
        return [
          {
            functionName: "Comp",
            fileName: "webpack-internal:///../pluto/src/bar.tsx",
            lineNumber: 20,
            columnNumber: 1,
          },
        ];
      throw new Error(`unexpected stack: ${err.stack}`);
    });
    const result = await resolveStack(error, "COMPONENT_STACK");
    expect(result.stack).toBe("  at fn (console/src/foo.ts:10:5)");
    expect(result.componentStack).toBe("  at Comp (pluto/src/bar.tsx:20:1)");
  });
});
