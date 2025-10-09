// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Explorer.css";

import { Ranger } from "@synnaxlabs/pluto";

import { type Layout } from "@/layout";
import { List } from "@/range/list/List";

export const EXPLORER_LAYOUT_TYPE = "range_explorer";

export const EXPLORER_LAYOUT: Layout.State = {
  key: EXPLORER_LAYOUT_TYPE,
  windowKey: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Range Explorer",
  icon: "Explore",
  location: "mosaic",
};

export const Explorer: Layout.Renderer = () => {
  const { data, getItem, subscribe, retrieve } = Ranger.useList({
    sort: Ranger.sortByStage,
  });
  return (
    <List
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      retrieve={retrieve}
      enableAddButton
      enableSearch
      enableFilters
    />
  );
};
