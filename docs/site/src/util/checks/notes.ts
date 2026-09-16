// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Check } from "./check.ts";
import { locate } from "./crawl.ts";
import { QUOTED, unescapeHTML } from "./html.ts";

const VARIANTS = ["info", "warning", "error"];
const DIV_OPEN = new RegExp(`<div\\b[^>]*\\sclass=${QUOTED}[^>]*>`, "g");
const TAG = /<\/?[a-zA-Z][\w-]*\b[^>]*>/g;
const NEEDLE_LENGTH = 40;

// Returns the note's inner HTML, matching nested divs to find its close tag.
const noteBody = (html: string, start: number): string => {
  let depth = 1;
  let i = start;
  while (depth > 0) {
    const open = html.indexOf("<div", i);
    const close = html.indexOf("</div>", i);
    if (close === -1) return html.slice(start);
    if (open !== -1 && open < close) {
      depth += 1;
      i = open + "<div".length;
    } else {
      depth -= 1;
      i = close + "</div>".length;
    }
  }
  return html.slice(start, i - "</div>".length);
};

// The note's leading words, for resolving a source line.
const needleOf = (body: string): string => {
  const text = unescapeHTML(body.replace(TAG, "")).replace(/\s+/g, " ").trim();
  if (text.length <= NEEDLE_LENGTH) return text;
  const cut = text.lastIndexOf(" ", NEEDLE_LENGTH);
  return text.slice(0, cut > 0 ? cut : NEEDLE_LENGTH);
};

// Every note must carry a variant the stylesheet knows and have content. The
// zero-notes canary needs a full crawl.
export const notes = (fullCrawl: boolean): Check => {
  let seen = 0;
  return {
    name: "notes",
    page: ({ route, html }) => {
      const failures: string[] = [];
      for (const m of html.matchAll(DIV_OPEN)) {
        const classes = (m[1] ?? m[2]).split(/\s+/);
        if (!classes.includes("pluto-note")) continue;
        seen += 1;
        const body = noteBody(html, (m.index ?? 0) + m[0].length);
        const needle = needleOf(body);
        const where = needle === "" ? route : locate(route, needle);
        const variant = classes
          .filter((c) => c.startsWith("pluto--"))
          .map((c) => c.slice("pluto--".length))
          .find((v) => VARIANTS.includes(v));
        if (variant == null)
          failures.push(`${where} - note variant is not one of ${VARIANTS.join(", ")}`);
        if (needle === "") failures.push(`${where} - note is empty`);
      }
      return failures;
    },
    finish: async (_, report) => {
      if (fullCrawl && seen === 0) report("no notes found: markup has changed");
    },
  };
};
