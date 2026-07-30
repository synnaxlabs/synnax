// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/explorer/Explorer.css";

import { ranger } from "@synnaxlabs/client";
import { Access, Button, Component, Icon, Ranger } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { useCallback } from "react";

import { ContextMenu } from "@/feature/range/list/ContextMenu";
import { Item } from "@/feature/range/list/Item";
import { Label } from "@/platform/label";
import { type Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { View } from "@/platform/view";

const item = Component.renderProp(Item);
const contextMenu = Component.renderProp(ContextMenu);

export const Explorer: Panel.Content = () => (
  <View.Frame resourceType="range" icon="Range">
    <Internal />
  </View.Frame>
);

const Internal = () => {
  const listProps = Ranger.useList({
    initialQuery: View.useContext().getInitialView().query,
    sort: Ranger.sortByStage,
  });
  const openCreate = Range.useCreateModal();
  const handleCreate = useCallback(() => openCreate(), [openCreate]);
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  return (
    <View.Form {...listProps}>
      <View.Toolbar>
        <View.FilterMenu>
          <Label.Filter.MenuItem />
        </View.FilterMenu>
        <View.Search />
        <Label.Filter.Chips />
        {hasCreatePermission && (
          <Button.Button
            onClick={handleCreate}
            tooltipLocation={location.BOTTOM_LEFT}
            tooltip="Create a range"
          >
            <Icon.Add />
          </Button.Button>
        )}
      </View.Toolbar>
      <View.Items contextMenu={contextMenu}>{item}</View.Items>
    </View.Form>
  );
};
