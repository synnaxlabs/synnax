// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "path";
import { describe, expect, it } from "vitest";

import { warning } from "@/util/checks/annotation";

describe("warning", () => {
  it("should name the file relative to the repository root", () => {
    const root = path.resolve("../..");
    const at = { file: "src/pages/guide.mdx", line: 25 };
    expect(warning("dead link", at, root)).toBe(
      "::warning file=site/docs/src/pages/guide.mdx,line=25::dead link",
    );
  });

  it("should leave out the location of a warning without a source", () => {
    expect(warning("dead link", null, process.cwd())).toBe("::warning::dead link");
  });

  it("should escape the characters that end a command or a property", () => {
    const at = { file: "src/pages/a,b:c.mdx", line: 1 };
    expect(warning("50%\r\ndone", at, process.cwd())).toBe(
      "::warning file=src/pages/a%2Cb%3Ac.mdx,line=1::50%25%0D%0Adone",
    );
  });
});
