// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Client, neon } from "@neondatabase/serverless";
import { drizzle as drizzleHTTP, type NeonHttpDatabase } from "drizzle-orm/neon-http";
import { drizzle as drizzleWS, type NeonDatabase } from "drizzle-orm/neon-serverless";

import * as schema from "@/server/db/schema";

export type Query = NeonHttpDatabase<typeof schema>;
export type Tx = Parameters<
  Parameters<NeonDatabase<typeof schema>["transaction"]>[0]
>[0];

/**
 * Store is the portal's database. Plain reads and writes go over Neon's stateless
 * HTTP driver through `query`. A read-then-write that must be atomic runs inside
 * `transact`, which opens one WebSocket connection for the transaction and closes it
 * when the callback settles.
 */
export interface Store {
  query: Query;
  transact: <T>(fn: (tx: Tx) => Promise<T>) => Promise<T>;
}

export const open = (url: string): Store => ({
  query: drizzleHTTP(neon(url), { schema }),
  transact: async (fn) => {
    const client = new Client(url);
    await client.connect();
    try {
      return await drizzleWS(client, { schema }).transaction(fn);
    } finally {
      await client.end();
    }
  },
});
