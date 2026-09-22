// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useState } from "react";

import { message } from "@/portal/ui/api";

export interface Action {
  run: () => void;
  loading: boolean;
  error: string | null;
  clear: () => void;
}

/** useAction runs an async handler, tracking its loading state and last error. */
export const useAction = (handler: () => Promise<void>): Action => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const run = useCallback(() => {
    setLoading(true);
    setError(null);
    handler()
      .catch((err: unknown) => setError(message(err)))
      .finally(() => setLoading(false));
  }, [handler]);
  const clear = useCallback(() => setError(null), []);
  return { run, loading, error, clear };
};
