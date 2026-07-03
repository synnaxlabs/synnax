// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ShikiTransformer } from "shiki";

// Astro 7 emits syntax-highlighted code fences as a raw `<pre class="astro-code">` that
// bypasses the MDX `pre` component override (Block.astro), so the wrapper and copy
// button that used to live there are gone. This transformer rebuilds that DOM at
// highlight time: it wraps each highlighted `<pre>` in `.astro-code-wrapper` and
// prepends the copy button. The `.copy` / `.check` icon spans and the
// `.astro-code-wrapper button` structure match what `codeUtil.ts` drives at runtime.

interface ElementNode {
  type: "element";
  tagName: string;
  properties: Record<string, unknown>;
  children: ElementNode[];
}

// IoCopy (react-icons/io5), the same icon Block.astro rendered via Icon.Copy.
const COPY_PATHS = [
  "M408 480H184a72 72 0 0 1-72-72V184a72 72 0 0 1 72-72h224a72 72 0 0 1 72 72v224a72 72 0 0 1-72 72z",
  "M160 80h235.88A72.12 72.12 0 0 0 328 32H104a72 72 0 0 0-72 72v224a72.12 72.12 0 0 0 48 67.88V160a80 80 0 0 1 80-80z",
];

// AiOutlineCheck (react-icons/ai), the same icon Block.astro rendered via Icon.Check.
const CHECK_PATHS = [
  "M912 190h-69.9c-9.8 0-19.1 4.5-25.1 12.2L404.7 724.5 207 474a32 32 0 0 0-25.1-12.2H112c-6.7 0-10.4 7.7-6.3 12.9l273.9 347c12.8 16.2 37.4 16.2 50.3 0l488.4-618.9c4.1-5.1.4-12.8-6.3-12.8z",
];

const icon = (
  name: "copy" | "check",
  viewBox: string,
  paths: string[],
): ElementNode => ({
  type: "element",
  tagName: "svg",
  properties: {
    className: ["pluto-icon", name],
    viewBox,
    fill: "currentColor",
    stroke: "currentColor",
    strokeWidth: "0",
    height: "1em",
    width: "1em",
    xmlns: "http://www.w3.org/2000/svg",
    "aria-hidden": "true",
  },
  children: paths.map((d) => ({
    type: "element",
    tagName: "path",
    properties: { d },
    children: [],
  })),
});

const copyButton = (): ElementNode => ({
  type: "element",
  tagName: "button",
  properties: { type: "button", "aria-label": "Copy code" },
  children: [
    icon("copy", "0 0 512 512", COPY_PATHS),
    icon("check", "0 0 1024 1024", CHECK_PATHS),
  ],
});

export const codeBlockWrapper = (): ShikiTransformer => ({
  name: "synnax:code-block-wrapper",
  root(root) {
    const pre = root.children[0];
    root.children = [
      {
        type: "element",
        tagName: "div",
        properties: { className: ["astro-code-wrapper"] },
        children: [copyButton(), pre],
      },
    ] as typeof root.children;
  },
});
