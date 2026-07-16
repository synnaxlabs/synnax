// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Payload } from "@/task/types.gen";

/**
 * Reports whether a task's live instance has drifted from its stored task: the task is
 * running and the stored config or rack differs from what the instance was deployed
 * with. Tasks that are not running never drift. Both hashes are server-assigned, so
 * this compares two given values and never hashes a config.
 * @param task - The task payload, including its status.
 * @returns True when a redeploy (start) would change the running instance.
 */
export const drifted = (task: Payload): boolean => {
  const details = task.status?.details;
  if (details == null || !details.running) return false;
  return details.configHash !== task.configHash || details.rack !== task.rack;
};
