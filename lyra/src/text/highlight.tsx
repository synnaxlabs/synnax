// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";

interface Run {
  text: string;
  match: boolean;
}

// The runs share one wrapper, so a flex parent lays them out as one item, with no gap.
const RUNS_CLASS = CSS.BE("text", "runs");
const MARK_CLASS = CSS.BE("text", "mark");

// Splits words the same way as the Core search tokenizer.
const SEPARATORS = /[\s_.-]+/;
const WORD = /[^\s_.-]+/g;

const tokenize = (term: string): string[] =>
  term
    .toLowerCase()
    .split(SEPARATORS)
    .filter((token) => token.length > 0);

// Returns null when no word matches, so the text renders as it would with no term.
const split = (text: string, term: string): Run[] | null => {
  const tokens = tokenize(term);
  const runs: Run[] = [];
  let plainStart = 0;
  let lastWordEnd = 0;
  for (const { 0: word, index } of text.matchAll(WORD)) {
    const length = tokens.reduce(
      (longest, token) =>
        word.slice(0, token.length).toLowerCase() === token
          ? Math.max(longest, token.length)
          : longest,
      0,
    );
    const previous = runs.at(-1);
    const joined = previous?.match === true && plainStart === lastWordEnd;
    lastWordEnd = index + word.length;
    if (length === 0) continue;
    if (joined) previous.text += text.slice(plainStart, index + length);
    else {
      if (index > plainStart)
        runs.push({ text: text.slice(plainStart, index), match: false });
      runs.push({ text: text.slice(index, index + length), match: true });
    }
    plainStart = index + length;
  }
  if (runs.length === 0) return null;
  if (plainStart < text.length)
    runs.push({ text: text.slice(plainStart), match: false });
  return runs;
};

/**
 * Renders text with the start of each word that a token of the term begins tinted. A
 * separator between two wholly tinted words is tinted too, so press_pt_0 tints as one
 * run in press_pt_02. Returns the text unchanged when nothing matches.
 */
export const renderHighlight = (text: string, term: string): ReactElement | string => {
  const runs = split(text, term);
  if (runs == null) return text;
  return (
    <span className={RUNS_CLASS}>
      {runs.map(({ text, match }, i) =>
        match ? (
          <mark key={i} className={MARK_CLASS}>
            {text}
          </mark>
        ) : (
          text
        ),
      )}
    </span>
  );
};

/**
 * Writes the same markup as {@link renderHighlight} into an element React does not
 * manage the children of.
 */
export const paintHighlight = (el: HTMLElement, text: string, term?: string): void => {
  const runs = term == null ? null : split(text, term);
  if (runs == null) {
    el.textContent = text;
    return;
  }
  const wrapper = document.createElement("span");
  wrapper.className = RUNS_CLASS;
  wrapper.append(
    ...runs.map(({ text, match }) => {
      if (!match) return text;
      const mark = document.createElement("mark");
      mark.className = MARK_CLASS;
      mark.textContent = text;
      return mark;
    }),
  );
  el.replaceChildren(wrapper);
};
