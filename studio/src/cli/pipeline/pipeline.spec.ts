// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { runCapture } from "@/cli/pipeline";

describe("runCapture", () => {
  let dir: string;
  beforeEach(async () => {
    dir = await mkdtemp(path.join(tmpdir(), "studio-pipeline-"));
  });
  afterEach(async () => {
    await rm(dir, { recursive: true, force: true });
  });

  it("should reject a script that exports an OS a capture cannot show", async () => {
    const scriptPath = path.join(dir, "script.mjs");
    await writeFile(
      scriptPath,
      'export default async () => {};\nexport const os = "Linux";\n',
    );
    await expect(
      runCapture({
        scriptPath,
        outDir: dir,
        url: "http://localhost:5173",
        theme: "dark",
        core: "external",
      }),
    ).rejects.toThrow(
      `${scriptPath} exports os = "Linux", but os must be one of macOS, Windows`,
    );
  });
});
