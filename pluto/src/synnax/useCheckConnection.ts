// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection } from "@synnaxlabs/client";
import { type CrudeTimeSpan, sleep, TimeSpan } from "@synnaxlabs/x";
import { useState } from "react";

import { useAsyncEffect } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";

const DEFAULT_INTERVAL = TimeSpan.seconds(5);

/**
 * Continuously probes the given cluster address, re-checking every interval
 * while mounted. A params change (by deep equality) restarts the loop; between
 * re-checks the settled status stays in place.
 * @returns null while the first check is pending or params are absent.
 */
export const useCheckConnection = (
  params?: connection.CheckParams | null,
  interval: CrudeTimeSpan = DEFAULT_INTERVAL,
): connection.Status | null => {
  const [status, setStatus] = useState<connection.Status | null>(null);
  const memoParams = useMemoDeepEqual(params ?? null);
  const intervalMS = TimeSpan.fromMilliseconds(interval).milliseconds;
  useAsyncEffect(
    async (signal) => {
      setStatus(null);
      if (memoParams == null) return;
      while (!signal.aborted) {
        const next = await connection.check(memoParams);
        if (signal.aborted) return;
        setStatus(next);
        await sleep.sleep(intervalMS);
      }
    },
    [memoParams, intervalMS],
  );
  return status;
};
