// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { type Color } from "@synnaxlabs/lyra/color";
import { useDispatch } from "react-redux";

import {
  type Action,
  setContext,
  SLICE_NAME,
  type StoreState,
} from "@/session/color/slice";
import { Select } from "@/session/select";

const selectContext = (state: StoreState): Color.ContextState =>
  state[SLICE_NAME].context;

const useSelectContext = (): Color.ContextState => Select.useMemo(selectContext, []);

/** @returns props that keep the Color provider's color state in the session. */
export const useProviderProps = (): Color.ProviderProps => {
  const value = useSelectContext();
  const dispatch = useDispatch<Dispatch<Action>>();
  return { value, onChange: (state) => dispatch(setContext(state)) };
};
