// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, log, schematic, table } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
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
  const linePlotVisible = Access.useUpdateGranted(lineplot.TYPE_ONTOLOGY_ID);
  const schematicVisible = Access.useUpdateGranted(schematic.TYPE_ONTOLOGY_ID);
  const logVisible = Access.useUpdateGranted(log.TYPE_ONTOLOGY_ID);
  const tableVisible = Access.useUpdateGranted(table.TYPE_ONTOLOGY_ID);
  return linePlotVisible || schematicVisible || logVisible || tableVisible;
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
