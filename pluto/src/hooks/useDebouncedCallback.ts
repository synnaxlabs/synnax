// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { debounce } from "@synnaxlabs/x";
import { type DependencyList, useCallback } from "react";

export const useDebouncedCallback = <F extends (...args: any[]) => void>(
  func: F,
  waitFor: number,
  deps: DependencyList,
): F => useCallback(debounce(func, waitFor), [waitFor, ...deps]);
