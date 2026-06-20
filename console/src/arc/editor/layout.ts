// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { deep, uuid } from "@synnaxlabs/x";

import { internalCreate, type State, TYPE, ZERO_STATE } from "@/arc/slice";
import { type Layout } from "@/layout";

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const {
      name = "Arc Editor",
      location = "mosaic",
      tab,
      mode = "graph",
      ...rest
    } = initial;
    const key = arc.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key, mode }));
    return {
      key,
      location,
      name,
      icon: "Arc",
      type: TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };
