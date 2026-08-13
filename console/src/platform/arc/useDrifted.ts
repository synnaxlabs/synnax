// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, task } from "@synnaxlabs/client";
import { Arc } from "@synnaxlabs/pluto";

/**
 * Whether the arc's running instance was deployed from different content, config, or
 * rack than its task now holds. The Core rewrites the task config whenever the arc's
 * semantic content changes, so content drift surfaces as ordinary task config drift.
 * Arcs that are not running never drift.
 */
export const useDrifted = (key: arc.Key): boolean => {
  const { data: stored } = Arc.useResultTask({ arcKey: key });
  return stored != null && task.drifted(stored);
};
