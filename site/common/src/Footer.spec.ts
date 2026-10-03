// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { JSDOM } from "jsdom";
import { beforeAll, describe, expect, it } from "vitest";

import Footer from "./Footer.astro";

const COLUMNS = [
  {
    title: "Documentation",
    href: "/reference/",
    links: [{ label: "Concepts", href: "/reference/concepts/overview" }],
  },
  {
    title: "Company",
    links: [{ label: "About", href: "https://synnaxlabs.com/company" }],
  },
];

describe("Footer", () => {
  let doc: Document;

  beforeAll(async () => {
    const container = await AstroContainer.create();
    container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
    const html = await container.renderToString(Footer, {
      props: { home: "https://synnaxlabs.com", columns: COLUMNS },
    });
    doc = new JSDOM(html).window.document;
  });

  const hrefs = (): string[] =>
    [...doc.querySelectorAll("a")].map((a) => a.getAttribute("href") ?? "");

  it("should link the logo to the home page", () => {
    expect(doc.querySelector("a.footer-logo-link")?.getAttribute("href")).toBe(
      "https://synnaxlabs.com",
    );
  });

  it("should render each column's links", () => {
    expect(hrefs()).toContain("/reference/concepts/overview");
    expect(hrefs()).toContain("https://synnaxlabs.com/company");
  });

  it("should link a column header only when the column has an href", () => {
    const headers = [...doc.querySelectorAll(".footer-column-header")];
    const documentation = headers.find((h) => h.textContent?.includes("Documentation"));
    const company = headers.find((h) => h.textContent?.includes("Company"));
    expect(documentation?.querySelector("a")?.getAttribute("href")).toBe("/reference/");
    expect(company?.querySelector("a")).toBeNull();
  });
});
