// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection } from "@synnaxlabs/client";
import { useState } from "react";

import { useAsyncEffect } from "@/hooks";
import { useMemoDeepEqual } from "@/memo";

/**
 * Runs a one-shot connectivity check against the given cluster address,
 * re-running whenever the params change by deep equality.
 * @returns null while the check is pending or params are absent.
 */
export const useCheckConnection = (
  params?: connection.CheckParams | null,
): connection.Status | null => {
  const [status, setStatus] = useState<connection.Status | null>(null);
  const memoParams = useMemoDeepEqual(params ?? null);
  useAsyncEffect(
    async (signal) => {
      setStatus(null);
      if (memoParams == null) return;
      const next = await connection.check(memoParams);
      if (!signal.aborted) setStatus(next);
    },
    [memoParams],
  );
  return status;
};
