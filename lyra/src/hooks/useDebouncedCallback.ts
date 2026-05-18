// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { debounce } from "@synnaxlabs/x/debounce";
import { telem } from "@synnaxlabs/x/telem";
import { type DependencyList, useCallback } from "react";

export const useDebouncedCallback = <Args extends unknown[]>(
  func: (...args: Args) => void,
  waitFor: telem.CrudeTimeSpan,
  deps: DependencyList,
): ((...args: Args) => void) => {
  const debouncePeriod = new telem.TimeSpan(waitFor).valueOf();
  return useCallback(debounce(func, debouncePeriod), [debouncePeriod, ...deps]);
};
