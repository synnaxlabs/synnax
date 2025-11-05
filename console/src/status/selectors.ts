// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { useMemo } from "react";
import { useSelector } from "react-redux";

import { SLICE_NAME, type SliceState } from "@/status/slice";
import { type RootState } from "@/store";

export const selectSliceState = (state: RootState): SliceState => state[SLICE_NAME];

export const selectFavorites = (state: RootState): status.Key[] =>
  selectSliceState(state).favorites;

export const selectIsFavorite = (state: RootState, key: status.Key): boolean =>
  selectSliceState(state).favorites.includes(key);

export const useSelectFavorites = (): status.Key[] =>
  useSelector(selectFavorites);

export const useSelectIsFavorite = (key: status.Key): boolean =>
  useSelector(useMemo(() => (state: RootState) => selectIsFavorite(state, key), [key]));
