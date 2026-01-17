// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { customAlphabet } from "nanoid/non-secure";

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
const ALPHANUMERIC = `0123456789${ALPHABET}`;

export const LENGTH = 11;

const createPrefix = customAlphabet(ALPHABET, 1);
const createInternal = customAlphabet(ALPHANUMERIC, LENGTH - 1);

/**
 * Creates a unique alphanumeric string of length 11. The returned id always begins
 * with a letter to disambiguate from values that can be potentially interpreted as
 * numbers.
 *
 * @returns {string} A unique alphanumeric string.
 */
export const create = (): string => `${createPrefix()}${createInternal()}`;
