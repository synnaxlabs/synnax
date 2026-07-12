// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { type Haul, type state } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import {
  type Action,
  setHauled,
  SLICE_NAME,
  type StoreState,
} from "@/session/haul/slice";
import { Select } from "@/session/select";

const selectHauling = (state: StoreState): Haul.DraggingState =>
  state[SLICE_NAME].state;

export const useSelectHauling = (): Haul.DraggingState =>
  Select.useMemo(selectHauling, []);

export const useGetHauling = (): (() => Haul.DraggingState) => {
  const store = useStore<StoreState>();
  return useCallback(() => selectHauling(store.getState()), [store]);
};

const useState: state.PureUse<Haul.DraggingState> = () => {
  const hauled = useSelectHauling();
  const dispatch = useDispatch<Dispatch<Action>>();
  const onHauledChange = useCallback(
    (state: Haul.DraggingState) => dispatch(setHauled(state)),
    [dispatch],
  );
  return [hauled, onHauledChange];
};

export const PROVIDER_PROPS: Haul.ProviderProps = { useState };
