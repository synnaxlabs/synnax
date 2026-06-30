// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";

import { Session } from "@/session";
import { type Layout } from "@/layout";

export const LAYOUT_TYPE = "lineplot";
export type LayoutType = typeof LAYOUT_TYPE;

export interface CreateArg
  extends Partial<Session.LinePlot.NewState>, Omit<Partial<Layout.BaseState>, "type"> {}

// create constructs a Layout for a line plot, placing it in the mosaic. The plot's
// document must already exist on the server; use useCreate to create a fresh plot.
export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Line Plot", location = "mosaic", window, tab, ...rest } = initial;
    const key = lineplot.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(Session.LinePlot.internalCreate({ ...rest, key }));
    return { key, name, location, type: LAYOUT_TYPE, icon: "Visualize", window, tab };
  };
