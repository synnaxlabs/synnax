// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

const SCRIPT = join(import.meta.dirname, "surface.sh");

const TSCONFIG = {
  compilerOptions: {
    declaration: true,
    emitDeclarationOnly: true,
    module: "esnext",
    moduleResolution: "bundler",
    rootDir: ".",
    strict: true,
    target: "es2022",
  },
  include: ["src/**/*"],
};

const TIME = `
export class TimeStamp { readonly ns = 0; }
export class TimeSpan { readonly span = 0; }
`;

const INDEX = `
import { TimeStamp as T } from "./time";

/** Opens a valve. */
export const open = (at: T): boolean => at !== undefined;

export class Valve {
  private secret = 1;
  close(): void {}
}
`;

// The script is Bash, and CI runs it on Linux only.
describe.skipIf(process.platform === "win32")("surface.sh", () => {
  let dir: string;
  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), "surface-"));
  });
  afterEach(() => rmSync(dir, { recursive: true, force: true }));

  const report = (index: string = INDEX): string => {
    mkdirSync(join(dir, "src"), { recursive: true });
    writeFileSync(join(dir, "tsconfig.build.json"), JSON.stringify(TSCONFIG));
    writeFileSync(join(dir, "src", "time.ts"), TIME);
    writeFileSync(join(dir, "src", "index.ts"), index);
    return execFileSync(SCRIPT, [dir], { encoding: "utf8" });
  };

  it("should print the public declarations", () => {
    const out = report();
    expect(out).toContain("# src/index.d.ts\n");
    expect(out).toContain("export declare const open: (at: T) => boolean;");
    expect(out).toContain("close(): void;");
    expect(out).not.toContain("secret");
  });

  it("should ignore comments and private members", () => {
    const before = report();
    const after = report(
      INDEX.replace("Opens a valve.", "Opens it.").replace("secret = 1", "hidden = 2"),
    );
    expect(after).toEqual(before);
  });

  it("should change when an import alias changes", () => {
    const before = report();
    const after = report(INDEX.replace("TimeStamp as T", "TimeSpan as T"));
    expect(after).not.toEqual(before);
  });
});
