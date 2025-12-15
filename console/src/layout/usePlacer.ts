// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { id } from "@synnaxlabs/x";
import { type Dispatch, useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { place, type State } from "@/layout/slice";
import { type RootAction, type RootState, type RootStore } from "@/store";

export interface CreatorProps {
  dispatch: Dispatch<PayloadAction<unknown>>;
  store: RootStore;
  windowKey: string;
}

export interface BaseState<A = unknown>
  extends Omit<State<A>, "windowKey" | "key">, Partial<Pick<State<A>, "key">> {}

/** A function that creates a layout given a set of utilities. */
export interface Creator<A = unknown> {
  (props: CreatorProps): BaseState<A>;
}

export type PlacerArgs<A = unknown> = BaseState<A> | Creator<A>;

/** A function that places a layout using the given properties or creation func. */
export interface Placer<A = unknown> {
  (layout: PlacerArgs<A>): { windowKey: string; key: string };
}

/**
 * useLayoutPlacer is a hook that returns a function that allows the caller to place
 * a layout in the central mosaic or in a window.
 *
 * @returns A layout placer function that allows the caller to open a layout using one
 * of two methods. The first is to pass a layout object with the layout's key, type,
 * title, location, and window properties. The second is to pass a layout creator function
 * that accepts a few utilities and returns a layout object. Prefer the first method
 * when possible, but feel free to use the second method for more dynamic layout creation.
 */
export const usePlacer = <A = unknown>(): Placer<A> => {
  const dispatch = useDispatch();
  const store = useStore<RootState, RootAction>();
  const windowKey = useSelectWindowKey();
  return useCallback(
    (base) => {
      if (windowKey == null) throw new Error("windowKey is null");
      const layout =
        typeof base === "function" ? base({ dispatch, store, windowKey }) : base;
      const key = layout.key ?? id.create();
      dispatch(place({ ...layout, windowKey, key }));
      return { windowKey, key };
    },
    [dispatch, store, windowKey],
  );
};
