// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Label } from "@/label";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/status/Create";
import { Item } from "@/status/list/Item";
import { View } from "@/view";

export const EXPLORER_LAYOUT_TYPE = "status_explorer";

export const EXPLORER_LAYOUT: Layout.BaseState = {
  key: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Status Explorer",
  icon: "Status",
  location: "mosaic",
};

const item = Component.renderProp(Item);

export const Explorer: Layout.Renderer = () => {
  const listProps = Status.useList();
  const placeLayout = Layout.usePlacer();
  const handleCreate = useCallback(() => placeLayout(CREATE_LAYOUT), [placeLayout]);
  return (
    <View.Frame {...listProps} resourceType="status" onCreate={handleCreate}>
      <View.Toolbar>
        <View.FilterMenu>
          <Label.Filter.MenuItem />
        </View.FilterMenu>
        <View.Search />
        <Label.Filter.Chips />
      </View.Toolbar>
      <View.Views />
      <View.Items>{item}</View.Items>
    </View.Frame>
  );
};
