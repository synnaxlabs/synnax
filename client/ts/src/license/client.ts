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
import { licenseZ } from "@/license/types.gen";

export const STATES = ["ok", "missing", "expired"] as const;
export const stateZ = z.enum(STATES);
/** Whether a license applies to the Core. */
export type State = z.infer<typeof stateZ>;

const infoZ = z.object({
  state: stateZ,
  warning: z.string().optional(),
  fingerprint: z
    .string()
    .array()
    .default(() => []),
  license: licenseZ.optional(),
});
/** The Core's license state, its machine fingerprint, and the license that applies. */
export interface Info extends z.infer<typeof infoZ> {}

const activateReqZ = z.object({ token: z.string() });

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

  /** Retrieves the Core's license state. */
  async retrieve(): Promise<Info> {
    return await this.unary.send(RETRIEVE_ENDPOINT, undefined, z.void(), infoZ);
  }

  /**
   * Activates a license token on the Core and returns the resulting state.
   * @throws {InvalidLicenseError} if the token cannot be verified or is malformed.
   * @throws {LicenseFingerprintError} if the token is bound to another machine.
   * @throws {ExpiredLicenseError} if the token no longer applies.
   */
  async activate(token: string): Promise<Info> {
    const info = await this.unary.send(
      ACTIVATE_ENDPOINT,
      { token },
      activateReqZ,
      infoZ,
    );
    this.connection.retryNow();
    return info;
  }
}
