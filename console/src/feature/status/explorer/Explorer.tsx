// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Flux } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { useState } from "react";

import { contextMenu } from "@/feature/status/list/ContextMenu";
import { Label } from "@/platform/label";
import { type Panel } from "@/platform/panel";
import { Status } from "@/platform/status";
import { View } from "@/platform/view";

const item = Component.renderProp(Status.List.Item);

export const Explorer: Panel.Content = () => (
  <View.Frame resourceType="status" icon="Status">
    <Internal />
  </View.Frame>
);

const Internal = () => {
  const [where, setWhere] = useState(View.useContext().getInitialView().where);
  const listProps = Flux.useList((client) => client.statuses, {
    name: "statuses",
    query: { where },
  });
  const openCreate = Status.useCreateModal();
  const handleCreate = () => openCreate();
  const hasCreatePermission = Access.useCreateGranted(status.TYPE_RESOURCE_ID);
  return (
    <View.Form {...listProps} onWhereChange={setWhere} itemHeight={40}>
      <View.Toolbar>
        <Flex.Box x align="center" gap="small" grow wrap>
          <View.FilterMenu>
            <Label.Filter.MenuItem />
            <Status.Filter.MenuItem />
          </View.FilterMenu>
          <Label.Filter.Chips />
          <Status.Filter.Chips />
        </Flex.Box>
        {hasCreatePermission && (
          <Button.Button
            variant="outlined"
            size="medium"
            tooltip="Create status"
            tooltipLocation={location.BOTTOM_LEFT}
            onClick={handleCreate}
          >
            <Icon.Add />
          </Button.Button>
        )}
      </View.Toolbar>
      <View.Items contextMenu={contextMenu}>{item}</View.Items>
    </View.Form>
  );
};
