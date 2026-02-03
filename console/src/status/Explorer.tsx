// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Access, Button, Component, Icon, Status } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
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

export const Explorer: Layout.Renderer = () => (
  <View.Frame resourceType="status">
    <Internal />
  </View.Frame>
);

const Internal = () => {
  const listProps = Status.useList({
    initialQuery: View.useContext().getInitialView().query,
  });
  const placeLayout = Layout.usePlacer();
  const handleCreate = useCallback(() => placeLayout(CREATE_LAYOUT), [placeLayout]);
  const canCreate = Access.useCreateGranted(status.TYPE_ONTOLOGY_ID);
  return (
    <View.Form {...listProps}>
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
            tooltip="Create a status"
          >
            <Icon.Add />
          </Button.Button>
        )}
      </View.Toolbar>
      <View.Items>{item}</View.Items>
    </View.Form>
  );
};
