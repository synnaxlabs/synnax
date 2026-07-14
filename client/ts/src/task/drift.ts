// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, hash, json } from "@synnaxlabs/x";

import { type Payload } from "@/task/types.gen";

/**
 * Hashes a task config into the shared cross-language form: xxhash64 of the
 * canonical JSON string, as 16 lowercase hex characters. Matches the hashes the
 * server and drivers stamp into status details. Keys are snake-cased first,
 * since that is how the wire codec stores the config.
 * @param config - The task config to hash.
 * @returns The 16-character hex hash.
 */
export const hashConfig = (config: unknown): string =>
  hash.xxHash64(json.canonicalString(caseconv.camelToSnake(config)));

/**
 * Reports whether a task's live instance has drifted from its stored row: the
 * task is running and the stored config or rack differs from what the instance
 * was deployed with. Tasks that are not running never drift.
 * @param task - The task payload, including its status.
 * @returns True when a redeploy (start) would change the running instance.
 */
export const drifted = (task: Payload): boolean => {
  const details = task.status?.details;
  if (details == null || !details.running) return false;
  return details.configHash !== hashConfig(task.config) || details.rack !== task.rack;
};
