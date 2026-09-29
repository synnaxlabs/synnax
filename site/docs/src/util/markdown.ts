// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Element, type ElementContent, type Nodes, type Properties } from "hast";
import { fromHtml } from "hast-util-from-html";
import { toMdast } from "hast-util-to-mdast";
import { gfmToMarkdown } from "mdast-util-gfm";
import { toMarkdown } from "mdast-util-to-markdown";

// Site controls inside an article that carry no page content.
const UI_TAGS = new Set(["button", "script", "style", "svg", "template"]);
const UI_CLASSES = [
  "blog-author__photo",
  "heading-anchor",
  "next-prev",
  "pluto-breadcrumb",
  "pluto-tabs__thumb",
];

const classes = (el: Element): string[] => {
  const { className } = el.properties;
  return Array.isArray(className) ? className.map(String) : [];
};

const text = (node: Nodes): string => {
  if (node.type === "text") return node.value;
  if (!("children" in node)) return "";
  return node.children.map(text).join("");
};

const find = (node: Nodes, match: (el: Element) => boolean): Element[] => {
  if (!("children" in node)) return [];
  return node.children.flatMap((child) => {
    if (child.type !== "element") return [];
    return match(child) ? [child, ...find(child, match)] : find(child, match);
  });
};

const resolve = (props: Properties, key: "href" | "src", base: URL): void => {
  const value = props[key];
  if (typeof value === "string") props[key] = new URL(value, base).href;
};

const tabLabel = (label: string): Element => ({
  type: "element",
  tagName: "p",
  properties: {},
  children: [
    {
      type: "element",
      tagName: "strong",
      properties: {},
      children: [{ type: "text", value: label }],
    },
  ],
});

const clean = (el: Element, base: URL, labels: Map<string, string>): void => {
  el.children = el.children.flatMap((child): ElementContent[] => {
    if (child.type === "comment") return [];
    if (child.type !== "element") return [child];
    if (UI_TAGS.has(child.tagName) || child.properties.role === "tablist") return [];
    const names = classes(child);
    if (names.some((name) => UI_CLASSES.includes(name))) return [];
    if (names.includes("article-description") && text(child).trim() === "") return [];
    // A collapsible table's summary is only its toggle.
    if (child.tagName === "summary" && classes(el).includes("table-collapse"))
      return [];
    clean(child, base, labels);
    const { properties: props } = child;
    if (child.tagName === "a") resolve(props, "href", base);
    if (child.tagName === "img") resolve(props, "src", base);
    if (names.includes("pluto-note") || names.includes("article-description"))
      child.tagName = "blockquote";
    if (child.tagName === "pre" && typeof props.dataLanguage === "string")
      for (const code of find(child, (e) => e.tagName === "code"))
        code.properties.className = [`language-${props.dataLanguage}`];
    if (props.role !== "tabpanel") return [child];
    // Every tab of a switcher renders; the label says which one a panel holds.
    const label = labels.get(String(props.ariaLabelledBy));
    return label == null ? child.children : [tabLabel(label), ...child.children];
  });
};

/**
 * Converts a rendered docs page to Markdown. Keeps the page's article and drops site
 * navigation, controls, and media that render only in the browser. Links resolve to
 * absolute URLs against url.
 * @param html - The full HTML document of the page.
 * @param url - The URL the page was served from.
 * @throws {Error} if the document has no article element.
 */
export const fromPage = (html: string, url: URL): string => {
  const tree = fromHtml(html);
  const [article] = find(tree, (el) => el.tagName === "article");
  if (article == null) throw new Error(`${url.pathname} has no article`);
  const labels = new Map(
    find(article, (el) => el.properties.role === "tab").map((tab) => [
      String(tab.properties.id),
      text(tab).trim(),
    ]),
  );
  clean(article, url, labels);
  return toMarkdown(toMdast(article), { extensions: [gfmToMarkdown()] });
};
