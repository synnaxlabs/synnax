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

export interface Island {
  component: string;
  props: unknown;
  body: string;
}

const ISLAND_OPEN = /<astro-island\b[^>]*>/g;

const attrOf = (tag: string, name: string): string | undefined => {
  const m = new RegExp(`\\s${name}=${QUOTED}`).exec(tag);
  return m == null ? undefined : (m[1] ?? m[2]);
};

// Astro serializes island props as [type, value] pairs; unwrap to plain values.
const unwrapProp = (v: unknown): unknown => {
  if (Array.isArray(v)) {
    if (v.length === 2 && typeof v[0] === "number") return unwrapProp(v[1]);
    return v.map(unwrapProp);
  }
  if (v != null && typeof v === "object")
    return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, unwrapProp(x)]));
  return v;
};

// Returns the island's inner HTML, skipping nested islands to find the matching
// close tag.
const islandBody = (html: string, start: number): string => {
  let depth = 1;
  let i = start;
  while (depth > 0) {
    const open = html.indexOf("<astro-island", i);
    const close = html.indexOf("</astro-island>", i);
    if (close === -1) return html.slice(start);
    if (open !== -1 && open < close) {
      depth += 1;
      i = open + "<astro-island".length;
    } else {
      depth -= 1;
      i = close + "</astro-island>".length;
    }
  }
  return html.slice(start, i);
};

export const islands = (html: string): Island[] =>
  [...html.matchAll(ISLAND_OPEN)].map((match) => {
    const tag = match[0];
    let props: unknown = {};
    const raw = attrOf(tag, "props");
    if (raw != null)
      try {
        props = unwrapProp(JSON.parse(unescapeHTML(raw)));
      } catch {
        // Leave props empty; checks that need them will report the miss.
      }
    return {
      component: attrOf(tag, "component-export") ?? "",
      props,
      body: islandBody(html, (match.index ?? 0) + tag.length),
    };
  });
