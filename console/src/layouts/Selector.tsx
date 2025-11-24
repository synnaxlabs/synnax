// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { Arc } from "@/arc";
import { Hardware } from "@/hardware";
import { type Layout } from "@/layout";
import { Selector as CoreSelector } from "@/selector";
import { Vis } from "@/vis";

// It's OK to call this in a map as SELECTABLES is a global constant. All hooks are
// always called and always in the same order.
export const useSelectorVisible = () => {
  const visible = SELECTABLES.map(({ useVisible }) => useVisible?.() ?? true);
  return visible.some((v) => v);
};

export const SELECTOR_LAYOUT_TYPE = "layoutSelector";

export interface CreateSelectorLayoutArgs
  extends Omit<Layout.BaseState, "type" | "icon" | "location" | "name" | "key"> {}

export const createSelectorLayout = (
  args: CreateSelectorLayoutArgs = {},
): Layout.BaseState => ({
  ...args,
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Visualize",
  location: "mosaic",
  name: "New Component",
  key: uuid.create(),
});

const SELECTABLES: CoreSelector.Selectable[] = [
  ...Vis.SELECTABLES,
  ...Hardware.SELECTABLES,
  ...Arc.SELECTABLES,
];

export const Selector: Layout.Renderer = (props) => (
  <CoreSelector.Selector
    selectables={SELECTABLES}
    text="Select a Component Type"
    {...props}
  />
);
