// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { effect } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { v4 as uuid } from "uuid";

import { add } from "@/effect/slice";
import { type Effect, ZERO_EFFECT } from "@/effect/types";
import { type Layout } from "@/layout";
import { Slate } from "@/slate";

export const EDIT_LAYOUT_TYPE = "effect_edit";

export const EDIT_LAYOUT: Layout.BaseState = {
  type: EDIT_LAYOUT_TYPE,
  name: "Edit Effect",
  location: "mosaic",
  icon: "Effect",
};

export type CreateArg = Partial<Effect> & Partial<Layout.BaseState>;

export const createEditLayout =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Edit Effect", location = "mosaic", window, tab, ...rest } = initial;
    const key = effect.keyZ.safeParse(initial.key).data ?? uuid();
    const slateKey = uuid();
    dispatch(
      add({
        effects: [{ ...deep.copy(ZERO_EFFECT), ...rest, slate: slateKey, key }],
      }),
    );
    dispatch(Slate.create({ ...Slate.ZERO_STATE, key: slateKey }));
    return {
      key,
      location,
      name,
      icon: "Effect",
      type: EDIT_LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };
