// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import type { channel } from "@/channel";

/**
 * @returns the channel that a key of a record frame names. A record keys its columns
 * by string, so a key that is a channel key in canonical decimal, as String writes it,
 * is a key, and any other key is a name.
 */
export const recordColumn = (key: string): channel.Key | channel.Name => {
  const n = Number(key);
  return Number.isInteger(n) && n >= 0 && n <= math.MAX_UINT32 && String(n) === key
    ? n
    : key;
};
