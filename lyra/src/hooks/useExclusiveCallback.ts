// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useRef } from "react";

/**
 * @returns func, which skips a call while an earlier one has not settled, across
 * re-renders. A call that fails frees the next one.
 */
export const useExclusiveCallback = <Args extends unknown[]>(
  func: (...args: Args) => Promise<void>,
): ((...args: Args) => Promise<void>) => {
  const running = useRef(false);
  return async (...args) => {
    if (running.current) return;
    running.current = true;
    await func(...args).finally(() => {
      running.current = false;
    });
  };
};
