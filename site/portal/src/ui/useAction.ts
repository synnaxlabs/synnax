// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { errorMessage } from "@/ui/clerk";

export interface Action<A extends unknown[] = []> {
  run: (...args: A) => void;
  loading: boolean;
  error: string | null;
}

/** useAction runs an async handler, tracking its loading state and last error. */
export const useAction = <A extends unknown[] = []>(
  handler: (...args: A) => Promise<void>,
): Action<A> => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const run = (...args: A): void => {
    setLoading(true);
    setError(null);
    handler(...args)
      .catch((err: unknown) => setError(errorMessage(err)))
      .finally(() => setLoading(false));
  };
  return { run, loading, error };
};
