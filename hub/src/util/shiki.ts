// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Token, tokens } from "@synnaxlabs/arc";
import { createCssVariablesTheme, type ShikiTransformer, splitTokens } from "shiki";

const color = ({ light, dark }: Token): string => `light-dark(${light}, ${dark})`;

const arcRules = Object.values(tokens).flatMap((token) =>
  token.scopes.map((scope) => ({ scope, settings: { foreground: color(token) } })),
);

/**
 * Astro's css-variables theme, extended with the Arc colors the Console uses. Every Arc
 * scope ends in `.arc`, so the added rules outrank the generic ones and reach no other
 * language. `light-dark()` needs `color-scheme` on the root element.
 */
export const theme = createCssVariablesTheme({ variablePrefix: "--astro-code-" });
theme.tokenColors = [...(theme.tokenColors ?? []), ...arcRules];

const VARIABLE_COLOR = color(tokens.variable);
// The analyzer reports stage and sequence names as functions.
const ROLES = { channels: color(tokens.channel), bodies: color(tokens.function) };
const ROLES_META = new RegExp(`(${Object.keys(ROLES).join("|")})="([^"]*)"`, "g");

/**
 * Colors the identifiers in a fence's `channels="a, b"` metadata as channels, and those
 * in `bodies="c, d"` as stage or sequence names.
 */
export const symbols: ShikiTransformer = {
  name: "arc-symbols",
  tokens(lines) {
    const raw: string = this.options.meta?.__raw ?? "";
    const declared = new Map<string, string>();
    for (const [, role, names] of raw.matchAll(ROLES_META))
      names
        .split(",")
        .map((name) => name.trim())
        .filter(Boolean)
        .forEach((name) => declared.set(name, ROLES[role as keyof typeof ROLES]));
    if (declared.size === 0) return;
    // Shiki merges runs of same-colored tokens, so `a=pressure_1` arrives whole. Split
    // it at every declared name.
    const pattern = new RegExp(`\\b(?:${[...declared.keys()].join("|")})\\b`, "g");
    const breaks = [...this.source.matchAll(pattern)].flatMap(({ 0: name, index }) => [
      index,
      index + name.length,
    ]);
    const split = splitTokens(lines, breaks);
    split.forEach((line) =>
      line.forEach((token) => {
        // A name inside a string or comment keeps that color.
        if (token.color !== VARIABLE_COLOR) return;
        const role = declared.get(token.content);
        if (role != null) token.color = role;
      }),
    );
    return split;
  },
};
