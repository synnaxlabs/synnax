// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type task } from "@synnaxlabs/client";

export interface WhileStoppedParams {
  /** The running tasks to stop. */
  tasks: task.Task[];
  /** Leaves the tasks stopped after the action succeeds. */
  keepStopped?: boolean;
}

/**
 * Stops the tasks, then runs the action. Starts the tasks again after a failure, and
 * after a success unless keepStopped is set. A task that fails to start shows its own
 * error.
 *
 * @throws the error of a stop that fails, or of the action.
 */
export const whileStopped = async (
  { tasks, keepStopped = false }: WhileStoppedParams,
  action: () => Promise<void>,
): Promise<void> => {
  let succeeded = false;
  try {
    await Promise.all(
      tasks.map(async (t) => {
        const res = await t.executeCommandSync({ type: "stop" });
        if (res.variant === "error") throw status.toError(res);
      }),
    );
    await action();
    succeeded = true;
  } finally {
    if (!succeeded || !keepStopped)
      await Promise.allSettled(
        tasks.map(async (t) => await t.executeCommandSync({ type: "start" })),
      );
  }
};
