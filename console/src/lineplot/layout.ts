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

// create constructs a Layout for a new or existing line plot. Callers that
// know the plot already exists on the server (link, ontology select) pass
// remote: true to skip pendingUpload; callers that are placing a fresh plot
// or one imported from JSON omit it (default false) so useAutoUpload sends
// the body to the server on mount. Body field defaults live in useUpload's
// `?? lineplot.ZERO_NEW.X` fallbacks; a pendingUpload with all fields
// undefined is the canonical "fresh empty plot" payload.
export interface CreateOptions {
  remote?: boolean;
}

export const create =
  (initial: CreateArg = {}, opts: CreateOptions = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Line Plot", location = "mosaic", window, tab, ...rest } = initial;
    const key = lineplot.keyZ.safeParse(initial.key).data ?? uuid.create();
    const pendingUpload = opts.remote ? undefined : (initial.pendingUpload ?? {});
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key, pendingUpload }));
    return { key, name, location, type: LAYOUT_TYPE, icon: "Visualize", window, tab };
  };
