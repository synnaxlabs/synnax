// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@/telem/telem";

export const debounce = <F extends (...args: any[]) => void>(
  func: F,
  waitFor: CrudeTimeSpan,
): F => {
  const ms = new TimeSpan(waitFor).milliseconds;
  if (ms === 0) return func;
  let timeout: ReturnType<typeof setTimeout> | null = null;
  const debounced = (...args: Parameters<F>): void => {
    if (timeout !== null) {
      clearTimeout(timeout);
      timeout = null;
    }
    timeout = setTimeout(() => func(...args), ms);
  };
  return debounced as F;
};
