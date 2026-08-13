// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, debounce, TimeSpan } from "@synnaxlabs/x";
import { type DependencyList, useCallback } from "react";

export const useDebouncedCallback = <Args extends unknown[]>(
  func: (...args: Args) => void,
  waitFor: CrudeTimeSpan,
  deps: DependencyList,
): debounce.DebouncedFn<Args> => {
  const debouncePeriod = new TimeSpan(waitFor).valueOf();
  return useCallback(debounce.debounce(func, debouncePeriod), [
    debouncePeriod,
    ...deps,
  ]);
};
