// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { customAlphabet } from "nanoid/non-secure";

const ALPHANUMERIC = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";

export const LENGTH = 11;

const createInternal = customAlphabet(ALPHANUMERIC, 11);

/**
 * Creates a unique alphanumeric string of length 11.
 *
 * @returns {string} A unique alphanumeric string.
 */
export const create = (): string => createInternal();
