// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type connection } from "@/connection";
import { type Info, infoZ, type State } from "@/license/types.gen";

/** What each license state means, worded for display. */
export const STATE_MESSAGES: Record<State, string> = {
  ok: "Licensed",
  missing: "No license is active on this Core",
  expired: "The license on this Core has expired",
};

const activateReqZ = z.object({ key: z.string() });

export const RETRIEVE_ENDPOINT = "/license/retrieve";
export const ACTIVATE_ENDPOINT = "/license/activate";

export interface ClientParams {
  unary: UnaryClient;
  /** Re-checked after an activation so the connection leaves the unlicensed state. */
  connection: connection.Handle;
}

export class Client {
  private readonly unary: UnaryClient;
  private readonly connection: connection.Handle;

  constructor({ unary, connection }: ClientParams) {
    this.unary = unary;
    this.connection = connection;
  }

  /**
   * Retrieves the Core's license state.
   * @throws {AccessDeniedError} if the caller lacks permission to read the license.
   */
  async retrieve(): Promise<Info> {
    return await this.unary.send(RETRIEVE_ENDPOINT, undefined, z.void(), infoZ);
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
    this.connection.retryNow();
    return info;
  }
}
