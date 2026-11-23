// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@/channel";

/**
 * @returns the channel that a key of a record frame names. A record keys its columns
 * by string, so a key of all digits is a channel key, and any other key is a name.
 */
export const recordColumn = (key: string): channel.Key | channel.Name =>
  /^\d+$/.test(key) ? Number(key) : key;
