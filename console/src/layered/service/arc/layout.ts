// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";

import { Session } from "@/layered/session";
import { type Layout } from "@/layout";

export const LAYOUT_TYPE = "arc";
export type LayoutType = typeof LAYOUT_TYPE;

export interface CreateArg
  extends Partial<Session.Arc.NewState>, Partial<Layout.BaseState> {
  key?: string;
}

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Arc Editor", location = "mosaic", tab, ...rest } = initial;
    const key = arc.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(Session.Arc.internalCreate({ ...rest, key }));
    return {
      key,
      location,
      name,
      icon: "Arc",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };
