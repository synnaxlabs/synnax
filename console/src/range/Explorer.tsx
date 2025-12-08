// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Explorer.css";

import { ranger } from "@synnaxlabs/client";
import { Access, Button, Component, Icon, Ranger } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Label } from "@/label";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/range/Create";
import { Item } from "@/range/list/Item";
import { View } from "@/view";

export const EXPLORER_LAYOUT_TYPE = "range_explorer";

export const EXPLORER_LAYOUT: Layout.State = {
  key: EXPLORER_LAYOUT_TYPE,
  windowKey: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Range Explorer",
  icon: "Explore",
  location: "mosaic",
};

const item = Component.renderProp(Item);

export const Explorer: Layout.Renderer = () => {
  const listProps = Ranger.useList({
    sort: Ranger.sortByStage,
  });
  const placeLayout = Layout.usePlacer();
  const handleCreate = useCallback(() => placeLayout(CREATE_LAYOUT), [placeLayout]);
  const canCreate = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  return (
    <View.Frame {...listProps} resourceType="range">
      <View.Views />
      <View.Toolbar>
        <View.FilterMenu>
          <Label.Filter.MenuItem />
        </View.FilterMenu>
        <View.Search />
        <Label.Filter.Chips />
        {canCreate && (
          <Button.Button
            onClick={handleCreate}
            tooltipLocation={location.BOTTOM_LEFT}
            tooltip="Create a range"
          >
            <Icon.Add />
          </Button.Button>
        )}
      </View.Toolbar>
      <View.Items>{item}</View.Items>
    </View.Frame>
  );
};
