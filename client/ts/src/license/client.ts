// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { type connection } from "@/connection";
import { type Info, infoZ, type State } from "@/license/types.gen";
import { type ontology } from "@/ontology";
import { query } from "@/query";

export const STATE_MESSAGES: Record<State, string> = {
  ok: "Licensed",
  missing: "No license is active on this Core",
  expired: "The license on this Core has expired",
};

const activateReqZ = z.object({ key: z.string() });

const KEY = "license";

/** The license as an access control object. Reading it takes a retrieve grant. */
export const ONTOLOGY_ID: ontology.ID = { type: "builtin", key: KEY };

export const RETRIEVE_ENDPOINT = "/license/retrieve";
export const ACTIVATE_ENDPOINT = "/license/activate";

/**
 * The channel the Core announces license changes on. A sample names the license, never
 * carries it, so the client rereads it through the retrieve endpoint's permission check.
 */
export const SET_CHANNEL_NAME = "sy_license_set";

interface Entry extends Info {
  key: typeof KEY;
}

export interface ClientParams {
  unary: UnaryClient;
  connection: connection.Handle;
  cache: query.Cache;
}

export class Client {
  private readonly unary: UnaryClient;
  private readonly connection: connection.Handle;
  private readonly table: query.Table<typeof KEY, Entry>;
  private readonly space: query.Retrieves<typeof KEY, Info>;

  constructor({ unary, connection, cache }: ClientParams) {
    this.unary = unary;
    this.connection = connection;
    const table = cache.createTable<typeof KEY, Entry>({
      name: "license",
      fetch: async () => [await this.fetch()],
      listen: [query.createFetchListener(SET_CHANNEL_NAME, z.literal(KEY))],
    });
    this.table = table;
    this.space = cache.queries<typeof KEY, Info, typeof KEY, Entry>({
      name: "license",
      table,
      fetch: async () => {
        table.ingest(await this.fetch());
        return [KEY];
      },
      compose: ([{ key: _, ...info }]) => info,
      keyOf: (key) => key,
      single: true,
    });
  }

  /**
   * Retrieves the Core's license state.
   * @throws {AccessDeniedError} if the caller lacks permission to read the license.
   */
  async retrieve(): Promise<Info> {
    return await this.space.retrieve(KEY);
  }

  /**
   * Calls the handler each time the cached license state changes, and keeps it current
   * until the returned destructor runs.
   */
  onChange(handler: query.ChangeHandler<Info>): destructor.Destructor {
    return this.space.onChange(KEY, handler);
  }

  /** @returns The cached license state, or undefined when none is cached. */
  getCached(): query.Cached<Info> | undefined {
    return this.space.getCached(KEY);
  }

  /**
   * Activates a license key on the Core and returns the resulting state.
   * @throws {InvalidLicenseError} if the key cannot be verified or is malformed.
   * @throws {LicenseFingerprintError} if the key is bound to another machine.
   * @throws {ExpiredLicenseError} if the key no longer applies.
   * @throws {AccessDeniedError} if the caller lacks permission to activate a license.
   */
  async activate(key: string): Promise<Info> {
    const info = await this.unary.send(ACTIVATE_ENDPOINT, { key }, activateReqZ, infoZ);
    this.table.set(KEY, { key: KEY, ...info });
    this.connection.retryNow();
    return info;
  }

  private async fetch(): Promise<Entry> {
    const info = await this.unary.send(RETRIEVE_ENDPOINT, undefined, z.void(), infoZ);
    return { key: KEY, ...info };
  }
}
