// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";

import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, type SliceState } from "@/status/slice";
import { type RootState } from "@/store";

export const selectSliceState = (state: RootState): SliceState => state[SLICE_NAME];

export const selectFavorites = (state: RootState): status.Key[] =>
  selectSliceState(state).favorites;

export const useSelectFavorites = (): status.Key[] =>
  useMemoSelect((state: RootState) => selectFavorites(state), []);

export const selectFavoriteSet = (state: RootState): Set<status.Key> =>
  new Set(selectFavorites(state));

export const useSelectFavoriteSet = (): Set<status.Key> =>
  useMemoSelect((state: RootState) => selectFavoriteSet(state), []);

export const selectIsFavorite = (state: RootState, key: status.Key): boolean =>
  selectSliceState(state).favorites.includes(key);

export const useSelectIsFavorite = (key: status.Key): boolean =>
  useMemoSelect((state: RootState) => selectIsFavorite(state, key), [key]);
