// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, scheduler, TimeSpan } from "@synnaxlabs/x";
import { type DependencyList } from "react";

import { compareDeps, useMemoCompare } from "@/memo";

/**
 * @returns a stable callback that defers `func` until `waitFor` passes with no further
 * calls. The identity changes only when `waitFor` or `deps` change, so a pending call
 * survives a re-render.
 */
export const useDebouncedCallback = <Args extends unknown[]>(
  func: (...args: Args) => void,
  waitFor: CrudeTimeSpan,
  deps: DependencyList,
): scheduler.DebouncedFn<Args> => {
  const debouncePeriod = new TimeSpan(waitFor).valueOf();
  return useMemoCompare(() => scheduler.debounce(func, debouncePeriod), compareDeps, [
    debouncePeriod,
    ...deps,
  ]);
};
