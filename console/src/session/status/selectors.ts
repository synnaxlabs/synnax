// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";

import { Select } from "@/session/select";
import { SLICE_NAME, type SliceState } from "@/session/status/slice";
import { type State } from "@/session/store";

const selectSliceState = (state: State): SliceState => state[SLICE_NAME];

export const selectFavorites = (state: State): status.Key[] =>
  selectSliceState(state).favorites;

export const useSelectFavorites = (): status.Key[] =>
  Select.useMemo((state: State) => selectFavorites(state), []);

const selectFavoriteSet = (state: State): Set<status.Key> =>
  new Set(selectFavorites(state));

export const useSelectFavoriteSet = (): Set<status.Key> =>
  Select.useMemo((state: State) => selectFavoriteSet(state), []);

export const selectIsFavorite = (state: State, key: status.Key): boolean =>
  selectSliceState(state).favorites.includes(key);

export const useSelectIsFavorite = (key: status.Key): boolean =>
  Select.useMemo((state: State) => selectIsFavorite(state, key), [key]);
