// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ForwardedRef, type RefCallback, useCallback } from "react";

export const useCombinedRefs = <T>(...refs: Array<ForwardedRef<T>>): RefCallback<T> =>
  useCallback(
    (el) =>
      refs.forEach((r) => {
        if (r == null) return;
        if (typeof r === "function") r(el);
        else r.current = el;
      }, el),
    [],
  );
