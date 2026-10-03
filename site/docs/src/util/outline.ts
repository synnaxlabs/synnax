// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SatteriAstroData } from "@astrojs/markdown-satteri";
import {
  type HastNode,
  type HastPluginDefinition,
  type HastVisitorContext,
} from "satteri";

/** How the page outline shows a heading that Astro's heading list flattens. */
export interface Entry {
  /** The heading text with its inline code in backticks. */
  text: string;
  /** The label of a Step component in the heading, such as "Step 1". */
  step?: string;
}

interface Heading {
  type: string;
  tagName?: string;
  name?: string | null;
  attributes?: { name?: string; value?: unknown }[];
  children?: Heading[];
}

const marked = (node: Heading, textContent: (node: Heading) => string): string =>
  node.type === "element" && node.tagName === "code"
    ? `\`${textContent(node)}\``
    : node.children == null
      ? textContent(node)
      : node.children.map((child) => marked(child, textContent)).join("");

// An attribute holds a string, or an expression such as step={1}.
const attribute = (node: Heading, name: string): string | undefined => {
  const value = node.attributes?.find((a) => a.name === name)?.value;
  if (typeof value === "string") return value;
  if (value != null && typeof value === "object" && "value" in value)
    return String(value.value);
  return undefined;
};

// Matches Step alone or through a namespace, such as Text.Step.
const STEP = /(?:^|\.)Step$/;

const step = (node: Heading): string | undefined => {
  const found = node.children?.find(
    (child) => child.type === "mdxJsxTextElement" && STEP.test(child.name ?? ""),
  );
  if (found == null) return undefined;
  return `${attribute(found, "name") ?? "Step"} ${attribute(found, "step")}`;
};

const astro = (ctx: HastVisitorContext): SatteriAstroData => {
  if (ctx.data.astro == null) throw new Error("outline runs only in Astro");
  return ctx.data.astro;
};

/**
 * Stores `frontmatter.outline`, mapping the text of each heading in Astro's heading
 * list to an entry that restores the inline code and step label the list drops. Only
 * headings with either appear.
 */
export const outline = (): HastPluginDefinition => {
  const found: Record<string, Entry> = {};
  return {
    name: "outline",
    before: (_, ctx) => {
      astro(ctx).frontmatter.outline = found;
    },
    element: {
      filter: ["h1", "h2", "h3", "h4", "h5", "h6"],
      visit: (node, ctx) => {
        const textContent = (n: Heading): string => ctx.textContent(n as HastNode);
        const text = textContent(node);
        const entry: Entry = { text: marked(node, textContent), step: step(node) };
        if (entry.text !== text || entry.step != null) found[text] = entry;
      },
    },
  };
};
