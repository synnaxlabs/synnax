// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";

import { Session } from "@/session";

export const LAYOUT_TYPE = "table";
export type LayoutType = typeof LAYOUT_TYPE;

export interface CreateArg
  extends
    Partial<Session.Layout.BaseState>,
    Partial<Pick<Session.Table.State, "editable">> {
  key?: string;
}

export const create =
  (initial: CreateArg = {}): Session.Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Table", location = "mosaic", tab, ...rest } = initial;
    const key = table.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(Session.Table.internalCreate({ key, ...rest }));
    return {
      key,
      location,
      name,
      icon: "Table",
      type: LAYOUT_TYPE,
      tab,
    };
  };
