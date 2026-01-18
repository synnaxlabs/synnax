// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, lineplot, log, schematic, table, task } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";

import { Arc } from "@/arc";
import { Hardware } from "@/hardware";
import { type Layout } from "@/layout";
import { Selector as BaseSelector } from "@/selector";
import { Vis } from "@/vis";

export const useSelectorVisible = (): boolean => {
  const linePlotVisible = Access.useUpdateGranted(lineplot.TYPE_ONTOLOGY_ID);
  const schematicVisible = Access.useUpdateGranted(schematic.TYPE_ONTOLOGY_ID);
  const logVisible = Access.useUpdateGranted(log.TYPE_ONTOLOGY_ID);
  const tableVisible = Access.useUpdateGranted(table.TYPE_ONTOLOGY_ID);
  const taskVisible = Access.useUpdateGranted(task.TYPE_ONTOLOGY_ID);
  const arcVisible = Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID);
  return (
    linePlotVisible ||
    schematicVisible ||
    logVisible ||
    tableVisible ||
    taskVisible ||
    arcVisible
  );
};

export const SELECTOR_LAYOUT_TYPE = "layoutSelector";

export interface CreateSelectorLayoutArgs extends Omit<
  Layout.BaseState,
  "type" | "icon" | "location" | "name" | "key"
> {}

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

const SELECTABLES: BaseSelector.Selectable[] = [
  ...Vis.SELECTABLES,
  ...Hardware.SELECTABLES,
  ...Arc.SELECTABLES,
];

export const Selector: Layout.Renderer = (props) => (
  <BaseSelector.Selector
    selectables={SELECTABLES}
    text="Select a Component Type"
    {...props}
  />
);
