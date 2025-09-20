// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useState } from "react";

import { errorResult, pendingResult, type Result, successResult } from "@/flux/result";

export interface UseActionArgs<F extends (...args: any[]) => Promise<any>> {
  resourceName: string;
  opName: string;
  action: F;
}

export type UseActionReturn<F extends (...args: any[]) => Promise<any>> =
  Result<undefined> & {
    run: (...args: Parameters<F>) => void;
    runAsync: (...args: Parameters<F>) => Promise<ReturnType<F>>;
  };

export const useAction = <F extends (...args: any[]) => Promise<any>>({
  resourceName,
  opName,
  action,
}: UseActionArgs<F>): UseActionReturn<F> => {
  const [status, setStatus] = useState<Result<undefined>>(() =>
    successResult(resourceName, opName, undefined),
  );
  const runAsync = useCallback(
    async (...args: Parameters<F>) => {
      try {
        setStatus(pendingResult(resourceName, opName, undefined));
        const res = await action(...args);
        setStatus(successResult(resourceName, opName, undefined));
        return res;
      } catch (error) {
        setStatus(errorResult(resourceName, opName, error));
      }
    },
    [action],
  );
  const run = useCallback((...args: Parameters<F>) => void runAsync(...args), [action]);
  return { ...status, run, runAsync };
};
