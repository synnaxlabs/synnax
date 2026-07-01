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

import { contextMenu } from "@/feature/status/list/ContextMenu";
import { View as ServiceView } from "@/feature/view";
import { Label } from "@/platform/label";
import { type Layout } from "@/platform/layout";
import { Filter } from "@/platform/status/filter";
import { Item } from "@/platform/status/list/Item";
import { useCreateModal } from "@/platform/status/useCreateModal";
import { View } from "@/platform/view";
import { type Session } from "@/session";

export const EXPLORER_LAYOUT_TYPE = "status_explorer";

export const EXPLORER_LAYOUT: Session.Layout.BaseState = {
  key: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Status Explorer",
  icon: "Status",
  location: "mosaic",
};

const item = Component.renderProp(Item);

export const Explorer: Layout.Renderer = () => (
  <ServiceView.Frame resourceType="status" icon="Status">
    <Internal />
  </ServiceView.Frame>
);

const Internal = () => {
  const listProps = Status.useList({
    initialQuery: View.useContext().getInitialView().query,
  });
  const openCreate = useCreateModal();
  const handleCreate = useCallback(() => openCreate(), [openCreate]);
  const hasCreatePermission = Access.useCreateGranted(status.TYPE_ONTOLOGY_ID);
  return (
    <View.Form {...listProps}>
      <View.Toolbar>
        <View.FilterMenu>
          <Label.Filter.MenuItem />
          <Filter.MenuItem />
        </View.FilterMenu>
        <View.Search />
        <Label.Filter.Chips />
        <Filter.Chips />
        {hasCreatePermission && (
          <Button.Button
            onClick={handleCreate}
            tooltipLocation={location.BOTTOM_LEFT}
            tooltip="Create a status"
          >
            <Icon.Add />
          </Button.Button>
        )}
      </View.Toolbar>
      <View.Items contextMenu={contextMenu}>{item}</View.Items>
    </View.Form>
  );
};
