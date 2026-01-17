// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { type Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Schematic } from "@/schematic";
import { Selector as BaseSelector } from "@/selector";
import { Table } from "@/table";

export const SELECTABLES: BaseSelector.Selectable[] = [
  ...LinePlot.SELECTABLES,
  ...Schematic.SELECTABLES,
  ...Log.SELECTABLES,
  ...Table.SELECTABLES,
];

export const SELECTOR_LAYOUT_TYPE = "visualizationSelector";

export const useSelectorVisible = (): boolean => {
  // Call ALL hooks first to maintain consistent hook order across renders.
  // Using .some() directly would short-circuit and skip hooks, violating Rules of Hooks.
  const results = SELECTABLES.map(({ useVisible }) => useVisible?.() ?? true);
  return results.some(Boolean);
};

export const createSelectorLayout = (): Layout.BaseState => ({
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Visualize",
  location: "mosaic",
  name: "New Visualization",
  key: uuid.create(),
});

export const Selector: Layout.Renderer = (props) => (
  <BaseSelector.Selector
    selectables={SELECTABLES}
    text="Select a Visualization Type"
    {...props}
  />
);
