// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useStore } from "react-redux";

import { SLICE_NAME, type StoreState } from "@/session/color/slice";
import { Select } from "@/session/select";

const selectContext = (state: StoreState): Color.ContextState =>
  Color.contextStateZ.parse(state[SLICE_NAME].context);

export const useSelectContext = (): Color.ContextState =>
  Select.useMemo(selectContext, []);

export const useGetContext = (): (() => Color.ContextState) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectContext(store.getState()), [store]);
};
