// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare } from "@synnaxlabs/x";

import KINDS from "../../../../../arc/go/analyzer/codes/codes.gen.json";

/** An Arc diagnostic kind, as the docs show it. */
export interface Kind {
  code: string;
  severity: "error" | "warning";
  /** The headline that each message of the kind starts with. */
  message: string;
  /** The description of the problem, in general terms. */
  problem: string;
  /** Source that reports the kind. Empty when the kind has no example. */
  example: string;
}

/** The name of each band that the docs show, by the first digit of its codes. */
export const BANDS: Record<string, string> = {
  1: "Syntax",
  2: "Types and expressions",
  3: "Functions and statements",
  4: "Names and modules",
  5: "Flows, sequences, and stages",
};

// The band is first, then W before E.
const order = (code: string): string =>
  code[1] + (code[0] === "W" ? "0" : "1") + code.slice(2);

/**
 * Returns the Arc diagnostic kinds that the docs show, by band, warnings before errors.
 */
export const kinds = (): Kind[] =>
  Object.entries(KINDS as Record<string, Omit<Kind, "code">>)
    .filter(([code]) => code[1] in BANDS)
    .map(([code, kind]) => ({ code, ...kind }))
    .sort(compare.by((k) => order(k.code)));
