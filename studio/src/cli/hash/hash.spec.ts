// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash } from "node:crypto";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { entryHash } from "@/cli/hash";
import { type Entry } from "@/manifest";

describe("cli.hash", () => {
  let root: string;
  const write = async (file: string, text: string): Promise<void> => {
    await mkdir(path.dirname(path.join(root, file)), { recursive: true });
    await writeFile(path.join(root, file), text);
  };
  const entry: Entry = { id: "console/test/shot", script: "scripts/shot.ts" };
  const hashOf = async (files: string[]): Promise<string> => {
    const hash = createHash("sha256");
    for (const file of files) hash.update(await readFile(path.join(root, file)));
    return hash.update(JSON.stringify({ entry, draft: false })).digest("hex");
  };

  beforeEach(async () => {
    root = await mkdtemp(path.join(tmpdir(), "studio-hash-"));
    await write("src/index.ts", "export const engine = 1;\n");
    await write("scripts/common/task.ts", 'export { DRIVER } from "./driver";\n');
    await write("scripts/common/driver.ts", 'export const DRIVER = "a";\n');
    await write("scripts/common/unused.ts", "export const UNUSED = 1;\n");
    await write(
      "scripts/shot.ts",
      'import { capture } from "@/index";\nimport { DRIVER } from "./common/task";\n',
    );
  });

  afterEach(async () => {
    await rm(root, { recursive: true, force: true });
  });

  describe("entryHash", () => {
    it("should hash the script, then its imported modules by path", async () => {
      expect(await entryHash(entry, false, root)).toBe(
        await hashOf([
          "scripts/shot.ts",
          "scripts/common/driver.ts",
          "scripts/common/task.ts",
        ]),
      );
    });

    it("should read every static import and export form", async () => {
      await write("scripts/common/a.ts", "a");
      await write("scripts/common/b.ts", "b");
      await write("scripts/common/c.ts", "c");
      await write("scripts/common/d/index.ts", "d");
      await write(
        "scripts/shot.ts",
        [
          "import {",
          "  type A,",
          "  a,",
          '} from "./common/a";',
          'import "./common/b";',
          'export * from "./common/c";',
          'export { d } from "./common/d";',
        ].join("\n"),
      );
      expect(await entryHash(entry, false, root)).toBe(
        await hashOf([
          "scripts/shot.ts",
          "scripts/common/a.ts",
          "scripts/common/b.ts",
          "scripts/common/c.ts",
          "scripts/common/d/index.ts",
        ]),
      );
    });

    it("should ignore imports inside comments and strings", async () => {
      await write(
        "scripts/shot.ts",
        [
          '// import { gone } from "./common/gone";',
          "/**",
          ' * import { gone } from "./common/gone";',
          " */",
          "const PROGRAM = `",
          "import time",
          "`;",
        ].join("\n"),
      );
      expect(await entryHash(entry, false, root)).toBe(
        await hashOf(["scripts/shot.ts"]),
      );
    });

    it("should skip modules outside the scripts directory", async () => {
      await write("scripts/shot.ts", 'import { engine } from "../src/index";\n');
      expect(await entryHash(entry, false, root)).toBe(
        await hashOf(["scripts/shot.ts"]),
      );
    });

    it("should stop at an import cycle", async () => {
      await write("scripts/common/driver.ts", 'import "./task";\n');
      expect(await entryHash(entry, false, root)).toBe(
        await hashOf([
          "scripts/shot.ts",
          "scripts/common/driver.ts",
          "scripts/common/task.ts",
        ]),
      );
    });

    it("should throw when a relative import names no file", async () => {
      await write("scripts/shot.ts", 'import { gone } from "./common/gone";\n');
      await expect(entryHash(entry, false, root)).rejects.toThrow(
        `imports "./common/gone", which is no file`,
      );
    });

    it("should hash a script with no local imports as before", async () => {
      const script = 'import { capture } from "@/index";\n';
      await write("scripts/shot.ts", script);
      const expected = createHash("sha256")
        .update(script)
        .update(JSON.stringify({ entry, draft: false }))
        .digest("hex");
      expect(await entryHash(entry, false, root)).toBe(expected);
    });

    it("should change when a module the script imports changes", async () => {
      const before = await entryHash(entry, false, root);
      await write("scripts/common/driver.ts", 'export const DRIVER = "b";\n');
      expect(await entryHash(entry, false, root)).not.toBe(before);
    });

    it("should not change when an engine module changes", async () => {
      const before = await entryHash(entry, false, root);
      await write("src/index.ts", "export const engine = 2;\n");
      expect(await entryHash(entry, false, root)).toBe(before);
    });

    it("should not change when a module nobody imports changes", async () => {
      const before = await entryHash(entry, false, root);
      await write("scripts/common/unused.ts", "export const UNUSED = 2;\n");
      expect(await entryHash(entry, false, root)).toBe(before);
    });

    it("should change with the draft flag", async () => {
      expect(await entryHash(entry, true, root)).not.toBe(
        await entryHash(entry, false, root),
      );
    });
  });
});
