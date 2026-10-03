// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { beforeAll, describe, expect, it } from "vitest";

import SDKs from "@/components/sdks/SDKs.astro";
import { islandProps, renderAstro } from "@/testutil";

const text = (html: string): string =>
  html
    .replace(/<[^>]+>/g, "")
    .replaceAll("&quot;", '"')
    .replaceAll("&#x3C;", "<");

describe("SDKs", () => {
  let codeHtmls: string[][];

  beforeAll(async () => {
    const doc = await renderAstro(SDKs);
    ({ codeHtmls } = islandProps<{ codeHtmls: string[][] }>(
      doc,
      "@/components/sdks/SDKShowcase",
    ));
  });

  it("should give each language a stream, write, and read sample", () => {
    expect(codeHtmls.map((samples) => samples.length)).toEqual([3, 3, 3]);
  });

  it("should order the languages as the tabs do", () => {
    const [python, typescript, cpp] = codeHtmls;
    for (const html of python) expect(text(html)).toContain("import synnax as sy");
    for (const html of typescript)
      expect(text(html)).toContain('import Synnax from "@synnaxlabs/client"');
    for (const html of cpp)
      expect(text(html)).toContain('#include "client/cpp/synnax.h"');
  });

  it("should order the operations as the panels do", () => {
    for (const [stream, write, read] of codeHtmls) {
      expect(text(stream).toLowerCase()).toContain("stream");
      expect(text(write).toLowerCase()).toContain("write");
      expect(text(read).toLowerCase()).toContain("read");
    }
  });
});
