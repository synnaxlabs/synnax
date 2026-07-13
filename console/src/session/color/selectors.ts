// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { Color, type state } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import {
  type Action,
  setContext,
  SLICE_NAME,
  type StoreState,
} from "@/session/color/slice";
import { Select } from "@/session/select";

const selectContext = (state: StoreState): Color.ContextState =>
  Color.contextStateZ.parse(state[SLICE_NAME].context);

export const useSelectContext = (): Color.ContextState =>
  Select.useMemo(selectContext, []);

export const useGetContext = (): (() => Color.ContextState) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectContext(store.getState()), [store]);
};

const useState: state.PureUse<Color.ContextState> = () => {
  const colorContext = useSelectContext();
  const dispatch = useDispatch<Dispatch<Action>>();
  const onColorContextChange = useCallback(
    (state: Color.ContextState) => dispatch(setContext(state)),
    [dispatch],
  );
  return [colorContext, onColorContextChange];
};

export const PROVIDER_PROPS: Color.ProviderProps = { useState };
