// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Client, neon } from "@neondatabase/serverless";
import { type ExtractTablesWithRelations } from "drizzle-orm";
import { drizzle as drizzleHTTP } from "drizzle-orm/neon-http";
import { drizzle as drizzleWS } from "drizzle-orm/neon-serverless";
import {
  type PgDatabase,
  type PgQueryResultHKT,
  type PgTransaction,
} from "drizzle-orm/pg-core";

import * as schema from "@/server/db/schema";

type Schema = typeof schema;

export type Query = PgDatabase<PgQueryResultHKT, Schema>;
export type Tx = PgTransaction<
  PgQueryResultHKT,
  Schema,
  ExtractTablesWithRelations<Schema>
>;

/** Reader is what a read runs against: a plain query, or an open transaction. */
export type Reader = Query | Tx;

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
