// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { type Key, type User } from "@/user/types.gen";

export const STORE_KEY = "users";

/** Registers the user store on the given engine. */
export const bindStore = (engine: cache.Engine): void => {
  engine.registerStore<Key, User>(STORE_KEY, { listeners: [] });
};
