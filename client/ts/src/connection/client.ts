// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { ClockSkewCalculator, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { type Info } from "@/connection/status";

export const CHECK_ENDPOINT = "/connectivity/check";

const requestZ = z.void();
const responseZ = z.object({
  clusterKey: z.string(),
  nodeVersion: z.string().optional(),
  nodeTime: TimeStamp.z,
});

/** Probes a cluster's connectivity endpoint. */
export class Client {
  private readonly unary: UnaryClient;

  /**
   * @param unary - Probe transport. Must carry the full middleware chain, auth
   * included, so that a response proves authentication.
   */
  constructor(unary: UnaryClient) {
    this.unary = unary;
  }

  /**
   * Runs a single connectivity probe, measuring clock skew across the round
   * trip.
   * @throws {AuthError} if the cluster rejects the client's credentials.
   * @throws {Unreachable} if the cluster cannot be reached.
   */
  async check(): Promise<Info> {
    const skew = new ClockSkewCalculator();
    skew.start();
    const res = await this.unary.send(CHECK_ENDPOINT, undefined, requestZ, responseZ);
    skew.end(res.nodeTime);
    return {
      clusterKey: res.clusterKey,
      nodeVersion: res.nodeVersion,
      clockSkew: skew.skew,
    };
  }
}
