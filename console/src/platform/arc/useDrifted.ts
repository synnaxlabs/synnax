// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { Arc } from "@synnaxlabs/pluto";

/**
 * Whether the running instance was deployed with a different program, config, or rack
 * than the arc now holds. Arcs that are not running never drift. An unstamped arc hash
 * or an empty deployed hash never drifts: the comparison side is unknown, not
 * different. Callers must render beneath a boundary that has retrieved the arc.
 */
export const useDrifted = (key: arc.Key): boolean => {
  const tsk = Arc.useRetrieveTask({ arcKey: key });
  const { data: arcData } = Arc.useRetrieve({ key });
  const stored = tsk.data;
  const deployed = stored?.status?.details;
  if (stored == null || deployed == null || !deployed.running) return false;
  if (deployed.configHash !== "" && stored.configHash !== deployed.configHash)
    return true;
  if (stored.rack !== deployed.rack) return true;
  const hash = arcData?.hash;
  return hash != null && stored.config.hash !== hash;
};
