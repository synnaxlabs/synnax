// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The match weights rank an exact word far above a prefix, and a prefix above a word
// within one edit.
const EXACT_WEIGHT = 100;
const PREFIX_WEIGHT = 10;
const FUZZY_WEIGHT = 1;

const MAX_QUERY_WORDS = 64;
const MAX_TOKEN_RUNES = 255;
// A shorter word is one edit from every word of its length, so "2" would match "1".
const MIN_FUZZY_RUNES = 3;

const SEPARATOR = /[\s_\-./]+/u;

const truncate = (token: string): string => {
  const runes = Array.from(token);
  return runes.length > MAX_TOKEN_RUNES
    ? runes.slice(0, MAX_TOKEN_RUNES).join("")
    : token;
};

const tokenize = (text: string): string[] =>
  text
    .toLowerCase()
    .split(SEPARATOR)
    .filter((token) => token !== "")
    .map(truncate);

const queryWords = (term: string): string[] => tokenize(term).slice(0, MAX_QUERY_WORDS);

const withinOneEdit = (a: string[], b: string[]): boolean => {
  if (Math.abs(a.length - b.length) > 1) return false;
  let i = 0;
  let j = 0;
  let edits = 0;
  while (i < a.length && j < b.length) {
    if (a[i] === b[j]) {
      i++;
      j++;
      continue;
    }
    if (++edits > 1) return false;
    if (a.length > b.length) i++;
    else if (a.length < b.length) j++;
    else {
      i++;
      j++;
    }
  }
  return edits + (a.length - i) + (b.length - j) <= 1;
};

const weight = (word: string, wordRunes: string[], token: string): number => {
  if (token === word) return EXACT_WEIGHT;
  if (token.startsWith(word)) return PREFIX_WEIGHT;
  if (
    wordRunes.length >= MIN_FUZZY_RUNES &&
    withinOneEdit(wordRunes, Array.from(token))
  )
    return FUZZY_WEIGHT;
  return 0;
};

interface Score {
  score: number;
  matched: number;
}

const score = (tokens: string[], words: string[]): Score => {
  let total = 0;
  let matched = 0;
  for (const word of words) {
    const runes = Array.from(word);
    let best = 0;
    for (const token of tokens) best = Math.max(best, weight(word, runes, token));
    if (best > 0) {
      total += best;
      matched++;
    }
  }
  return { score: total, matched };
};

/**
 * A full-text index of entries keyed by K. It splits text into lowercase words at
 * whitespace, '_', '-', '.', and '/', and matches each word of a search exactly, by
 * prefix, or, for a word of at least three runes, within one rune edit. The Core's
 * search index uses the same rules, so the two find the same entries.
 */
export class Index<K> {
  private readonly entries = new Map<K, string[]>();

  /** Indexes key under the words of every text, replacing any text it had. */
  set(key: K, ...text: string[]): void {
    this.entries.set(key, [...new Set(text.flatMap(tokenize))]);
  }

  /** Removes key from the index. */
  delete(key: K): void {
    this.entries.delete(key);
  }

  /**
   * Returns the keys that match every word of term, closest match first. When no key
   * matches every word, returns the keys that match any word. Reads only the first 64
   * words of term, and returns at most limit keys, or every key when limit is 0.
   */
  search(term: string, limit: number = 0): K[] {
    const words = queryWords(term);
    if (words.length === 0) return [];
    const matches: Array<{ key: K; tokens: number; order: number } & Score> = [];
    let order = 0;
    for (const [key, tokens] of this.entries) {
      const s = score(tokens, words);
      if (s.matched > 0) matches.push({ key, tokens: tokens.length, order, ...s });
      order++;
    }
    const complete = matches.filter(({ matched }) => matched === words.length);
    const kept = complete.length > 0 ? complete : matches;
    kept.sort((a, b) => b.score - a.score || a.tokens - b.tokens || a.order - b.order);
    const keys = kept.map(({ key }) => key);
    return limit > 0 ? keys.slice(0, limit) : keys;
  }
}

/**
 * Returns true if every word of term matches a word of text by the rules of
 * {@link Index}, or if term has no words.
 */
export const matches = (text: string, term: string): boolean => {
  const words = queryWords(term);
  return words.length === 0 || score(tokenize(text), words).matched === words.length;
};
