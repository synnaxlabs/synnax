// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { fromPage } from "@/util/markdown";

const PAGE_URL = new URL("https://docs.synnaxlabs.com/reference/client/channels");

const page = (article: string): string =>
  `<html><body><nav><a href="/nav">Nav</a></nav><article>${article}</article></body></html>`;

describe("fromPage", () => {
  it("should convert only the article", () => {
    expect(fromPage(page("<h1>Channels</h1><p>Text.</p>"), PAGE_URL)).toBe(
      "# Channels\n\nText.\n",
    );
  });

  it("should throw when the page has no article", () => {
    expect(() => fromPage("<html><body><p>Hi</p></body></html>", PAGE_URL)).toThrow(
      "/reference/client/channels has no article",
    );
  });

  it("should label each tab panel and drop the tab list", () => {
    const html = page(`
      <div role="tablist"><div role="tab" id="t-py"><svg></svg><p>Python</p></div></div>
      <div role="tabpanel" aria-labelledby="t-py" hidden><p>Install it.</p></div>`);
    expect(fromPage(html, PAGE_URL)).toBe("**Python**\n\nInstall it.\n");
  });

  it("should keep the language of a highlighted code block", () => {
    const html = page(
      `<pre data-language="python"><code><span class="line">import synnax</span></code></pre>`,
    );
    expect(fromPage(html, PAGE_URL)).toBe("```python\nimport synnax\n```\n");
  });

  it("should resolve relative links against the page URL", () => {
    const html = page(`<p><a href="/reference/concepts/channels">channel</a></p>`);
    expect(fromPage(html, PAGE_URL)).toBe(
      "[channel](https://docs.synnaxlabs.com/reference/concepts/channels)\n",
    );
  });

  it("should quote notes and the page description", () => {
    const html = page(
      `<h4 class="article-description">About channels.</h4><div class="pluto-note"><p>Careful.</p></div>`,
    );
    expect(fromPage(html, PAGE_URL)).toBe("> About channels.\n\n> Careful.\n");
  });

  it("should drop an empty page description", () => {
    expect(
      fromPage(page(`<h4 class="article-description"></h4><p>Body.</p>`), PAGE_URL),
    ).toBe("Body.\n");
  });

  it("should drop site controls", () => {
    const html = page(`
      <small class="pluto-breadcrumb"><a href="/reference">Reference</a></small>
      <h2 id="a">Create<a href="#a" class="heading-anchor"><svg></svg></a></h2>
      <details class="table-collapse" open><summary><p>Table</p></summary>
        <table><thead><tr><th>Flag</th></tr></thead><tbody><tr><td>-v</td></tr></tbody></table>
      </details>
      <div class="next-prev"><a href="/next">Next</a></div>
      <script>alert(1)</script><style>p{}</style>`);
    expect(fromPage(html, PAGE_URL)).toBe(
      "## Create\n\n| Flag |\n| ---- |\n| -v   |\n",
    );
  });
});
