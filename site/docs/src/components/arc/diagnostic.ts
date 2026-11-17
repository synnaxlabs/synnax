// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import KINDS from "../../../../../arc/go/analyzer/codes/codes.gen.json";

/** An Arc diagnostic kind, as the docs show it. */
export interface Kind {
  code: string;
  severity: "error" | "warning";
  /** The message that the example reports. Empty when the kind has no example. */
  message: string;
  problem: string;
  example: string;
}

/**
 * Returns the Arc diagnostic kinds of a band, errors before warnings.
 * @param band - The first digit of the code.
 * @throws If no kind is in the band.
 */
export const kinds = (band: number): Kind[] => {
  const found = Object.entries(KINDS as Record<string, Omit<Kind, "code">>)
    .filter(([code]) => code[1] === band.toString())
    .map(([code, kind]) => ({ code, ...kind }))
    .sort((a, b) => a.code.localeCompare(b.code));
  if (found.length === 0) throw new Error(`no Arc diagnostic kind is in band ${band}`);
  return found;
};
