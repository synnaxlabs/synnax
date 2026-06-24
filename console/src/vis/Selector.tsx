// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { LinePlot } from "@/layered/service/lineplot";
import { Schematic } from "@/layered/service/schematic";
import { Table } from "@/layered/service/table";
import { type Layout } from "@/layout";
import { Log } from "@/log";
import { Selector as BaseSelector } from "@/selector";

export const SELECTABLES: BaseSelector.Selectable[] = [
  ...LinePlot.SELECTABLES,
  ...Schematic.SELECTABLES,
  ...Log.SELECTABLES,
  ...Table.SELECTABLES,
];

export const SELECTOR_LAYOUT_TYPE = "visualizationSelector";

export const useSelectorVisible = (): boolean =>
  // It's safe to call hooks in map since SELECTABLES is a module-level constant
  // and never changes between renders, ensuring consistent hook order.
  SELECTABLES.map((s) => s.useVisible?.() ?? true).some(Boolean);

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
