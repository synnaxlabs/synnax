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
import {
  internalCreate,
  type PendingUpload,
  type State,
  ZERO_STATE,
} from "@/lineplot/slice";

export const LAYOUT_TYPE = "lineplot";
export type LayoutType = typeof LAYOUT_TYPE;

export type CreateArg = Partial<State> & Omit<Partial<Layout.BaseState>, "type">;

// lineplot.ZERO_NEW carries the wire-input shape where arrays may be null
// (the server's nullishToEmpty transform fills them in). The pendingUpload
// field is the parsed output shape with concrete arrays, so we coerce nulls
// to empty arrays at the source rather than at every dispatch site.
const ZERO_BODY: NonNullable<PendingUpload> = {
  title: lineplot.ZERO_NEW.title,
  legend: lineplot.ZERO_NEW.legend,
  channels: {
    ...lineplot.ZERO_NEW.channels,
    y1: lineplot.ZERO_NEW.channels.y1 ?? [],
    y2: lineplot.ZERO_NEW.channels.y2 ?? [],
    y3: lineplot.ZERO_NEW.channels.y3 ?? [],
    y4: lineplot.ZERO_NEW.channels.y4 ?? [],
  },
  ranges: {
    x1: lineplot.ZERO_NEW.ranges.x1 ?? [],
    x2: lineplot.ZERO_NEW.ranges.x2 ?? [],
  },
  // axesZ.parse coerces the wire-input shape (with unknown bounds) into the
  // concrete output shape PendingUpload expects.
  axes: lineplot.axesZ.parse(lineplot.ZERO_NEW.axes),
  lines: lineplot.ZERO_NEW.lines ?? [],
  rules: lineplot.ZERO_NEW.rules ?? [],
};

// create constructs a Layout for a new or existing line plot. Callers that
// know the plot already exists on the server (link, ontology select) pass
// remote: true to skip pendingUpload; callers that are placing a fresh plot
// or one imported from JSON omit it (default false) so the local pendingUpload
// gets seeded and useAutoUpload sends it to the server on mount.
export interface CreateOptions {
  remote?: boolean;
}

export const create =
  (initial: CreateArg = {}, opts: CreateOptions = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Line Plot", location = "mosaic", window, tab, ...rest } = initial;
    const key = lineplot.keyZ.safeParse(initial.key).data ?? uuid.create();
    const pendingUpload = opts.remote
      ? undefined
      : { ...ZERO_BODY, ...initial.pendingUpload };
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key, pendingUpload }));
    return { key, name, location, type: LAYOUT_TYPE, icon: "Visualize", window, tab };
  };
