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

export type UseAsyncOperationReturn<F extends (...args: any[]) => Promise<any>> =
  Result<undefined> & {
    run: () => void;
    runAsync: () => Promise<ReturnType<F>>;
  };

export const useAsyncOperation = <F extends (...args: any[]) => Promise<any>>(
  resourceName: string,
  opName: string,
  operation: F,
): UseAsyncOperationReturn<F> => {
  const [status, setStatus] = useState<Result<undefined>>(
    successResult(resourceName, opName, undefined),
  );
  const runAsync = useCallback(async () => {
    try {
      setStatus(pendingResult(resourceName, opName, undefined));
      const res = await operation();
      setStatus(successResult(resourceName, opName, undefined));
      return res;
    } catch (error) {
      setStatus(errorResult(resourceName, opName, error));
    }
  }, [operation]);
  const run = useCallback(() => void runAsync(), [operation]);
  return { ...status, run, runAsync };
};
