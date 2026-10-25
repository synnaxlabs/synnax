// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash } from "node:crypto";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { entryHash, modules } from "@/cli/hash";
import { type Entry } from "@/manifest";

describe("cli.hash", () => {
  let root: string;
  const write = async (file: string, text: string): Promise<void> => {
    await mkdir(path.dirname(path.join(root, file)), { recursive: true });
    await writeFile(path.join(root, file), text);
  };
  const entry: Entry = { id: "console/test/shot", script: "scripts/shot.ts" };

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

  describe("modules", () => {
    it("should list the script, then its imported modules by path", async () => {
      expect(
        await modules(path.join(root, "scripts/shot.ts"), path.join(root, "scripts")),
      ).toEqual([
        path.join(root, "scripts/shot.ts"),
        path.join(root, "scripts/common/driver.ts"),
        path.join(root, "scripts/common/task.ts"),
      ]);
    });

    it("should read every static import and export form", async () => {
      await write("scripts/common/a.ts", "");
      await write("scripts/common/b.ts", "");
      await write("scripts/common/c.ts", "");
      await write("scripts/common/d/index.ts", "");
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
      expect(
        await modules(path.join(root, "scripts/shot.ts"), path.join(root, "scripts")),
      ).toEqual([
        path.join(root, "scripts/shot.ts"),
        path.join(root, "scripts/common/a.ts"),
        path.join(root, "scripts/common/b.ts"),
        path.join(root, "scripts/common/c.ts"),
        path.join(root, "scripts/common/d/index.ts"),
      ]);
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
      expect(
        await modules(path.join(root, "scripts/shot.ts"), path.join(root, "scripts")),
      ).toEqual([path.join(root, "scripts/shot.ts")]);
    });

    it("should skip modules outside the root", async () => {
      await write("scripts/shot.ts", 'import { engine } from "../src/index";\n');
      expect(
        await modules(path.join(root, "scripts/shot.ts"), path.join(root, "scripts")),
      ).toEqual([path.join(root, "scripts/shot.ts")]);
    });

    it("should stop at an import cycle", async () => {
      await write("scripts/common/driver.ts", 'import "./task";\n');
      expect(
        await modules(path.join(root, "scripts/shot.ts"), path.join(root, "scripts")),
      ).toHaveLength(3);
    });

    it("should throw when a relative import names no file", async () => {
      await write("scripts/shot.ts", 'import { gone } from "./common/gone";\n');
      await expect(
        modules(path.join(root, "scripts/shot.ts"), path.join(root, "scripts")),
      ).rejects.toThrow(`imports "./common/gone", which is no file`);
    });
  });

  describe("entryHash", () => {
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
