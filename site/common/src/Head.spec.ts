// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { JSDOM } from "jsdom";
import { describe, expect, it } from "vitest";

import Head from "./Head.astro";

const render = async (props: Record<string, unknown> = {}): Promise<Document> => {
  const container = await AstroContainer.create();
  const html = await container.renderToString(Head, {
    props: { title: "Foundation", canonical: "https://synnaxlabs.com/", ...props },
  });
  return new JSDOM(`<head>${html}</head>`).window.document;
};

const property = (doc: Document, name: string): string | null | undefined =>
  doc.querySelector(`meta[property="${name}"]`)?.getAttribute("content");

describe("Head", () => {
  it("should name the site Synnax by default", async () => {
    expect(property(await render(), "og:site_name")).toBe("Synnax");
  });

  it("should name the site it is given", async () => {
    const doc = await render({ siteName: "Foundation by Synnax" });
    expect(property(doc, "og:site_name")).toBe("Foundation by Synnax");
  });

  it("should link both favicons", async () => {
    const icons = [...(await render()).querySelectorAll('link[rel="icon"]')];
    expect(icons.map((icon) => icon.getAttribute("href"))).toEqual([
      "/favicon.ico",
      "/favicon.svg",
    ]);
  });
});
