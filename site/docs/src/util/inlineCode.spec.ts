// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { markdownToHtml } from "satteri";
import { describe, expect, it } from "vitest";

import { inlineCode } from "@/util/inlineCode";

const LIMIT = 32;

const html = async (source: string): Promise<string> =>
  (await markdownToHtml(source, { hastPlugins: [inlineCode] })).html;

describe("inlineCode", () => {
  it("should mark short inline code", async () => {
    expect(await html("Use `--port`.")).toBe(
      '<p>Use <code data-short="">--port</code>.</p>\n',
    );
  });

  it("should mark inline code at the length limit", async () => {
    expect(await html(`\`${"a".repeat(LIMIT)}\``)).toContain("data-short");
  });

  it("should leave long inline code free to wrap", async () => {
    expect(await html(`\`${"a".repeat(LIMIT + 1)}\``)).not.toContain("data-short");
  });

  it("should mark a single token in a heading", async () => {
    expect(await html("## Use `->` for values")).toContain("data-short");
  });

  it("should leave code with spaces in a heading free to wrap", async () => {
    expect(await html("## `Synnax Driver on ... not running`")).not.toContain(
      "data-short",
    );
  });

  it("should leave code blocks alone", async () => {
    expect(await html("```\nx\n```")).not.toContain("data-short");
  });
});
