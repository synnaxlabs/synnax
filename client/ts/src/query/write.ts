// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { destructor } from "@synnaxlabs/x";

export interface OptimisticParams<R> {
  /**
   * Rollbacks for the already-applied optimistic cache changes, run in
   * reverse order when commit fails.
   */
  rollbacks: destructor.Destructor[];
  /**
   * Notified after the rollbacks are captured and before commit runs.
   * Value-bearing writes close over their optimistic value:
   * `() => opts.onOptimistic?.(value)`.
   */
  onOptimistic?: () => Promise<void> | void;
  /** The network send. Its result is returned when it succeeds. */
  commit: () => Promise<R>;
}

/**
 * Runs an optimistic write. When commit or `onOptimistic` fails the rollbacks
 * run and the error rethrows; on success the cache keeps the optimistic state.
 */
export const optimistic = async <R>({
  rollbacks,
  onOptimistic,
  commit,
}: OptimisticParams<R>): Promise<R> => {
  const rollback = new destructor.Chain();
  rollback.add(...rollbacks);
  return await rollback.guard(async () => {
    await onOptimistic?.();
    return await commit();
  });
};
