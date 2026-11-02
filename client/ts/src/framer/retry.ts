// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { breaker, type sync, TimeSpan } from "@synnaxlabs/x";

import { AccessDeniedError, NotFoundError, ValidationError } from "@/errors";

/** @returns true for a rejection of a channel that recurs on every retry. */
export const isDefinitive = (e: unknown): boolean =>
  AccessDeniedError.matches(e) ||
  ValidationError.matches(e) ||
  NotFoundError.matches(e);

/**
 * @returns a breaker for failed reads. Its waits end early when the notifier fires, so
 * a closed reader stops waiting.
 */
export const createBreaker = (
  notifier: sync.Notifier,
  config?: breaker.Config,
): breaker.Breaker =>
  new breaker.Breaker({
    baseInterval: TimeSpan.seconds(1),
    maxInterval: TimeSpan.seconds(30),
    maxRetries: Infinity,
    scale: 2,
    jitter: 0.25,
    sleepFn: async (span) => await notifier.wait(span),
    ...config,
  });
