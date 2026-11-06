// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { isCalculated, isValidName } from "@/channel/payload";
import { type Where } from "@/channel/types.gen";
import { type Lines, serverComputed as baseServerComputed } from "@/channel/where.gen";

/**
 * The typed lines of a channel Where. A name pattern that is a valid channel name
 * matches that name exactly.
 */
export const LINES: Lines = {
  virtual: (ch, value) => (ch.virtual && !isCalculated(ch)) === value,
  namePatterns: (ch, patterns) => patterns.includes(ch.name),
};

// The Core reads a name pattern that is not a valid channel name as a Go regular
// expression.
const hasRegexNamePattern = (where?: Where): boolean =>
  where != null &&
  (where.namePatterns?.some((p) => !isValidName(p)) === true ||
    where.and?.some(hasRegexNamePattern) === true ||
    where.or?.some(hasRegexNamePattern) === true ||
    hasRegexNamePattern(where.not));

/** @returns whether only the Core can check where. */
export const serverComputed = (where?: Where): boolean =>
  baseServerComputed(LINES, where) || hasRegexNamePattern(where);
