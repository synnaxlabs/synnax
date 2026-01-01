// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { useDebouncedCallback } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

export const useDispatchEffect = <P>(
  f: () => void,
  debounce: number = 0,
  dispatch?: Dispatch<PayloadAction<P>>,
): Dispatch<PayloadAction<P>> => {
  const coreDispatch = useDispatch();
  dispatch ??= coreDispatch;
  const update = useDebouncedCallback(f, debounce, [f]);
  return useCallback(
    (a) => {
      const r = dispatch(a);
      update();
      return r;
    },
    [update, dispatch],
  );
};
