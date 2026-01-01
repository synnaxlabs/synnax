// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NIL, stringify, v4 as uuid } from "uuid";

/**
 * Creates a new random UUID (Universally Unique Identifier) version 4.
 * This function uses the cryptographically secure random number generator
 * to generate a UUID that is suitable for use as a unique identifier.
 *
 * @returns {string} A new UUID v4 string in the format 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'
 * where x is any hexadecimal digit and y is one of 8, 9, A, or B.
 *
 * @example
 * const id = create(); // Returns something like "123e4567-e89b-12d3-a456-426614174000"
 */
export const create = (): string => uuid();

/**
 * Parses a UUID from a byte array.
 * This function converts a 16-byte array into a standard UUID string format.
 * It can optionally start parsing from a specific offset in the byte array.
 *
 * @param {Uint8Array} bytes - The byte array containing the UUID. Must be at least 16 bytes long.
 * @param {number} [offset=0] - Optional offset in the byte array to start parsing from.
 *
 * @returns {string} A UUID string in the format 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
 *
 * @throws {Error} If the byte array is less than 16 bytes long (accounting for offset)
 *
 * @example
 * const bytes = new Uint8Array([
 *   0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3,
 *   0xa4, 0x56, 0x42, 0x66, 0x14, 0x17, 0x40, 0x00
 * ]);
 * const uuid = parse(bytes); // Returns "123e4567-e89b-12d3-a456-426614174000"
 */
export const parse = (bytes: Uint8Array, offset?: number): string =>
  stringify(bytes, offset);

/** The zero value for a UUID - 00000000-0000-0000-0000-000000000000 */
export const ZERO = NIL;
