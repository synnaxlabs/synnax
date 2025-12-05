// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
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

const initialQuery: status.MultiRetrieveArgs = {};

export const Explorer: Layout.Renderer = () => {
  const listProps = Status.useList();
  const placeLayout = Layout.usePlacer();
  const handleCreate = useCallback(() => placeLayout(CREATE_LAYOUT), [placeLayout]);
  const { query, onQueryChange, resetQuery } = View.useQuery(
    initialQuery,
    listProps.retrieve,
  );
  return (
    <View.Frame
      {...listProps}
      resourceType="status"
      onCreate={handleCreate}
      query={query}
      onQueryChange={onQueryChange}
    >
      <View.Toolbar>
        <View.FilterMenu>
          <Label.Filter.MenuItem query={query} onQueryChange={onQueryChange} />
        </View.FilterMenu>
        <Label.Filter.Chips query={query} onQueryChange={onQueryChange} isClosable />
        <View.Search query={query} onQueryChange={onQueryChange} />
      </View.Toolbar>
      <View.Views onQueryChange={onQueryChange} resetQuery={resetQuery} />
      <View.Items>{item}</View.Items>
    </View.Frame>
  );
};
