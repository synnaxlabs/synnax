// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import type { z } from "zod";

import type { PlanChangedError } from "@/errors";
import {
  applyRequestZ,
  type Change,
  type Plan,
  planRequestZ,
  planZ,
} from "@/impact/types.gen";

/** The picks of {@link Client.plan}. */
export interface PlanOptions extends Omit<z.input<typeof planRequestZ>, "change"> {}

/** The token and picks of {@link Client.apply}. A missing token skips the check. */
export interface ApplyOptions extends Omit<z.input<typeof applyRequestZ>, "change"> {}

/** The transports of a {@link Client}. */
export interface ClientConfig {
  /** unary sends plans, and retries one that does not reach the Core. */
  unary: UnaryClient;
  /** unaryNoRetry sends applies, so an apply makes one attempt. */
  unaryNoRetry: UnaryClient;
}

/** Lists every record that a change to channels touches, and carries the change out. */
export class Client {
  private readonly unary: UnaryClient;
  private readonly unaryNoRetry: UnaryClient;

  constructor({ unary, unaryNoRetry }: ClientConfig) {
    this.unary = unary;
    this.unaryNoRetry = unaryNoRetry;
  }

  /**
   * Returns every record that change touches with the picks of options: an entry for
   * each record that the caller can see, a count of the others, and the token that
   * {@link Client.apply} checks. A delete pick on a channel adds the records that hold
   * it. Writes nothing.
   */
  async plan(change: Change, options: PlanOptions = {}): Promise<Plan> {
    return await this.unary.send(
      "/impact/plan",
      { ...options, change },
      planRequestZ,
      planZ,
    );
  }

  /**
   * Carries out change with the picks of options and returns the entries that it
   * applied. With a token, it throws a {@link PlanChangedError} before any write when
   * the plan is no longer the one that the token names. It makes one attempt, so a
   * caller waits for at most one request.
   */
  async apply(change: Change, options: ApplyOptions = {}): Promise<Plan> {
    return await this.unaryNoRetry.send(
      "/impact/apply",
      { ...options, change },
      applyRequestZ,
      planZ,
    );
  }
}

/** Returns the plan as it is now, which error carries. */
export const planOf = (error: PlanChangedError): Plan =>
  planZ.parse(JSON.parse(error.plan));
