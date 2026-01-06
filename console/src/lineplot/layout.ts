// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { deep, uuid } from "@synnaxlabs/x";

import { type Layout } from "@/layout";
import { internalCreate, type State, ZERO_STATE } from "@/lineplot/slice";

export const LAYOUT_TYPE = "lineplot";
export type LayoutType = typeof LAYOUT_TYPE;

export type CreateArg = Partial<State> & Omit<Partial<Layout.BaseState>, "type">;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Line Plot", location = "mosaic", window, tab, ...rest } = initial;
    const key = lineplot.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return { key, name, location, type: LAYOUT_TYPE, icon: "Visualize", window, tab };
  };
