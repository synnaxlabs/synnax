// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { type Theming } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Select } from "@/session/select";
import {
  type Action,
  select,
  SLICE_NAME,
  type SliceState,
  type StoreState,
  toggle,
  toggleSyncWithSystem,
} from "@/session/theme/slice";

const selectSlice = (state: StoreState): SliceState => state[SLICE_NAME];

const selectSelected = (state: StoreState) => selectSlice(state).selected;

const selectSyncWithSystem = (state: StoreState) => selectSlice(state).syncWithSystem;

export const useSelectSelected = (): string => Select.useMemo(selectSelected, []);

export const useSelectSyncWithSystem = (): boolean =>
  Select.useMemo(selectSyncWithSystem, []);

export const useToggleSyncWithSystem = (): (() => void) => {
  const dispatch = useDispatch<Dispatch<Action>>();
  return useCallback(() => dispatch(toggleSyncWithSystem()), [dispatch]);
};

export const useProviderProps = (): Theming.ProviderProps => {
  const key = useSelectSelected();
  const dispatch = useDispatch<Dispatch<Action>>();
  return useMemo<Theming.ProviderProps>(
    () => ({
      theme: { key },
      setTheme: (key: string) => dispatch(select(key)),
      toggleTheme: () => dispatch(toggle()),
    }),
    [key],
  );
};
