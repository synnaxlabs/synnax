// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Schematic } from "@/schematic";
import { type RootState } from "@/store";
import { Table } from "@/table";

const SELECTABLES: Layout.Selectable[] = [
  ...LinePlot.SELECTABLES,
  ...Schematic.SELECTABLES,
  ...Log.SELECTABLES,
  ...Table.SELECTABLES,
];

export const SELECTOR_LAYOUT_TYPE = "visualizationSelector";

export const SELECTOR_LAYOUT: Layout.BaseState = {
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Visualize",
  location: "mosaic",
  name: "New Visualization",
};

export const getSelectables = (storeState: RootState): Layout.Selectable[] => {
  const canCreateSchematic = Schematic.selectHasPermission(storeState);
  return SELECTABLES.filter((s) =>
    s.key === Schematic.SELECTABLE.key ? canCreateSchematic : true,
  );
};

export const Selector: Layout.Renderer = (props) => {
  const store = useStore<RootState>();
  const selectables = getSelectables(store.getState());
  return (
    <Layout.Selector
      selectables={selectables}
      text="Select a Visualization Type"
      {...props}
    />
  );
};
