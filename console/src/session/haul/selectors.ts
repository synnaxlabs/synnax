// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { useDispatch } from "react-redux";

import {
  type Action,
  setHauled,
  SLICE_NAME,
  type StoreState,
} from "@/session/haul/slice";
import { Select } from "@/session/select";

const selectHauling = (state: StoreState): Haul.DraggingState =>
  state[SLICE_NAME].state;

const useSelectHauling = (): Haul.DraggingState => Select.useMemo(selectHauling, []);

/** @returns props that keep the Haul provider's drag state in the session. */
export const useProviderProps = (): Haul.ProviderProps => {
  const value = useSelectHauling();
  const dispatch = useDispatch<Dispatch<Action>>();
  return { value, onChange: (state) => dispatch(setHauled(state)) };
};
