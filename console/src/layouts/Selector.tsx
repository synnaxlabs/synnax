// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useStore } from "react-redux";

import { Hardware } from "@/hardware";
import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { Vis } from "@/vis";

export const SELECTOR_LAYOUT_TYPE = "layoutSelector";

export const SELECTOR_LAYOUT: Layout.BaseState = {
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Visualize",
  location: "mosaic",
  name: "New Component",
};

export const Selector: Layout.Renderer = (props) => {
  const store = useStore<RootState>();
  const selectables = [
    ...Vis.getSelectables(store.getState()),
    ...Hardware.SELECTABLES,
  ];
  return (
    <Layout.Selector
      selectables={selectables}
      text="Select a Component Type"
      {...props}
    />
  );
};
