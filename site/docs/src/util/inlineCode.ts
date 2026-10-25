// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type HastPluginDefinition } from "satteri";

/** The longest inline code, in characters, that stays on one line. */
const SHORT_LENGTH = 32;

const HEADING = /^h[1-6]$/;

/** Returns true if the text is short and has no spaces, so it can never wrap. */
export const isToken = (text: string): boolean =>
  !/\s/.test(text) && text.length <= SHORT_LENGTH;

/**
 * Marks inline code of at most {@link SHORT_LENGTH} characters with `data-short`, so
 * CSS keeps it on one line and a chip never splits at a hyphen, as in `--port`. Longer
 * code, such as an error message, still wraps. In a heading, which is wider per
 * character, only a single token is marked.
 */
export const inlineCode = (): HastPluginDefinition => ({
  name: "inline-code",
  element: {
    filter: ["code"],
    visit: (node, ctx) => {
      const parent = ctx.parent(node);
      const tag = parent.type === "element" ? parent.tagName : "";
      if (tag === "pre") return;
      const text = ctx.textContent(node);
      const short = HEADING.test(tag) ? isToken(text) : text.length <= SHORT_LENGTH;
      if (short) ctx.setProperty(node, "dataShort", "");
    },
  },
});
