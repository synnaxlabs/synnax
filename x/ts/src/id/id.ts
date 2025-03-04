// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { customAlphabet } from "nanoid/non-secure";
import { z } from "zod";

const ALPHANUMERIC = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";

const generator = customAlphabet(ALPHANUMERIC, 11);

/**
 * Generates a unique alphanumeric string of length 11.
 *
 * @returns {string} A unique alphanumeric string.
 */
export const generate = (): string => generator();

const REGEX = /^[0-9A-Za-z]{11}$/;

/**
 * Validates an ID.
 *
 * @param {string} id - The ID to validate.
 * @returns {boolean} True if the ID is valid, false otherwise.
 */
export const validate = (id: string): boolean => REGEX.test(id);

/**
 * The Zod schema for an ID.
 */
export const schema = z
  .string()
  .regex(REGEX, "Invalid ID, must be exactly 11 characters and alphanumeric");

/**
 * The type for an ID.
 */
export type ID = z.infer<typeof schema>;
