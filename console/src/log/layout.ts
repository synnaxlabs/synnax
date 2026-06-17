// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";

import { type Layout } from "@/layout";
import { internalCreate } from "@/log/session/slice";
import { type Tabs } from "@/panel/tabs/index";

export const LAYOUT_TYPE = "log";
export type LayoutType = typeof LAYOUT_TYPE;

export interface CreateArg extends Partial<Tabs.BaseState> {
  key?: string;
}

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Log", location = "mosaic", window, tab } = initial;
    const key = log.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ key }));
    return {
      key,
      name,
      icon: "Log",
      location,
      type: LAYOUT_TYPE,
      windowKey: key,
      window,
      tab,
    };
  };
