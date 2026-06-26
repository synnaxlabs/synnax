// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Theming } from "@synnaxlabs/pluto";
import { useMemo } from "react";
import { useDispatch } from "react-redux";

import { useMemoSelect } from "@/hooks";
import {
  select,
  SLICE_NAME,
  type SliceState,
  type StoreState,
  toggle,
} from "@/layered/session/theme/slice";

const selectSlice = (state: StoreState): SliceState => state[SLICE_NAME];

const selectSelected = (state: StoreState) => selectSlice(state).selected;

export const useSelectSelected = (): string => useMemoSelect(selectSelected, []);

export const useProviderProps = (): Theming.ProviderProps => {
  const key = useSelectSelected();
  const dispatch = useDispatch();
  return useMemo<Theming.ProviderProps>(
    () => ({
      theme: { key },
      setTheme: (key: string) => dispatch(select(key)),
      toggleTheme: () => dispatch(toggle()),
    }),
    [key],
  );
};
