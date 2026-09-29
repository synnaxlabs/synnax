// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { mdxToJs } from "satteri";
import { describe, expect, it } from "vitest";

import { outline } from "@/util/outline";

const compile = async (source: string): Promise<unknown> => {
  const astro = {
    frontmatter: {},
    headings: [],
    localImagePaths: new Set<string>(),
    remoteImagePaths: new Set<string>(),
  };
  const { data } = await mdxToJs(source, { hastPlugins: [outline], data: { astro } });
  return data.astro?.frontmatter.outline;
};

describe("outline", () => {
  it("should restore a heading's inline code in backticks", async () => {
    expect(await compile("## The `read` method")).toEqual({
      "The read method": { text: "The `read` method" },
    });
  });

  it("should leave out headings without code or a step", async () => {
    expect(await compile("## Create a channel\n\n`code` in a paragraph")).toEqual({});
  });

  it("should find code at every heading level", async () => {
    expect(await compile("# `open`\n\n###### `close`")).toEqual({
      open: { text: "`open`" },
      close: { text: "`close`" },
    });
  });

  it("should keep the text of other inline markup and expressions", async () => {
    expect(await compile("## *Use* `x` {'now'}")).toEqual({
      "Use x 'now'": { text: "Use `x` 'now'" },
    });
  });

  it("should label a heading that holds a Step", async () => {
    expect(
      await compile('## <Step step={1} level="h2">Install the Core</Step>'),
    ).toEqual({ "Install the Core": { text: "Install the Core", step: "Step 1" } });
  });

  it("should use a Step's own name and keep its inline code", async () => {
    expect(
      await compile('## <Step step="2" name="Part" level="h2">Run `sy`</Step>'),
    ).toEqual({ "Run sy": { text: "Run `sy`", step: "Part 2" } });
  });

  it("should keep each page's headings apart", async () => {
    await compile("## `first`");
    expect(await compile("## `second`")).toEqual({ second: { text: "`second`" } });
  });
});
