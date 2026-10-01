// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const ENTITIES: Record<string, string> = {
  "&quot;": '"',
  "&amp;": "&",
  "&lt;": "<",
  "&gt;": ">",
  "&#39;": "'",
};

export const unescapeHTML = (s: string): string =>
  s.replace(/&quot;|&amp;|&lt;|&gt;|&#39;/g, (m) => ENTITIES[m]);

/**
 * Matches a double- or single-quoted attribute value; the unmatched quote's capture
 * group is undefined, so read `m[1] ?? m[2]`.
 */
export const QUOTED = `(?:"([^"]*)"|'([^']*)')`;

/** Returns the values of attr across all occurrences of tag, entity-unescaped. */
export const attrValues = (html: string, tag: string, attr: string): string[] => {
  const re = new RegExp(`<${tag}\\b[^>]*\\s${attr}=${QUOTED}`, "g");
  return [...html.matchAll(re)].map((m) => unescapeHTML(m[1] ?? m[2]));
};

/** Returns every id attribute value in the document. */
export const idValues = (html: string): string[] =>
  [...html.matchAll(new RegExp(`\\sid=${QUOTED}`, "g"))].map((m) =>
    unescapeHTML(m[1] ?? m[2]),
  );

/** Returns the value of attribute name in an opening tag, entity-unescaped. */
export const attrOf = (tag: string, name: string): string | undefined => {
  const m = new RegExp(`\\s${name}=${QUOTED}`).exec(tag);
  return m == null ? undefined : unescapeHTML(m[1] ?? m[2]);
};

export interface Element {
  /** The opening tag. */
  tag: string;
  /** The inner HTML. */
  body: string;
}

// Returns the inner HTML up to the close tag matching the element opened before start,
// skipping nested elements of the same name.
const bodyOf = (html: string, name: string, start: number): string => {
  let depth = 1;
  for (let i = start; ;) {
    const open = html.indexOf(`<${name}`, i);
    const close = html.indexOf(`</${name}>`, i);
    if (close === -1) return html.slice(start);
    if (open !== -1 && open < close) {
      depth += 1;
      i = open + name.length + 1;
      continue;
    }
    depth -= 1;
    if (depth === 0) return html.slice(start, close);
    i = close + name.length + 3;
  }
};

/** Returns every name element whose opening tag carries attr="value". */
export const elements = (
  html: string,
  name: string,
  attr: string,
  value: string,
): Element[] =>
  [
    ...html.matchAll(new RegExp(`<${name}\\b[^>]*\\s${attr}="${value}"[^>]*>`, "g")),
  ].map((m) => ({ tag: m[0], body: bodyOf(html, name, (m.index ?? 0) + m[0].length) }));
