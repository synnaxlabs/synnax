// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Synnax } from "@synnaxlabs/client";
import type {
  ChannelKey,
  OntologyID,
  OntologyResource,
  OntologyResourceType,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Hauled } from "@synnaxlabs/pluto";

import { LayoutPlacer, selectActiveMosaicLayout } from "@/layout";
import {
  ZERO_CHANNELS_STATE,
  addLinePlotYChannel,
  createLinePlot,
  setLinePlotYChannels,
} from "@/line/store/slice";
import { RootStore } from "@/store";

export interface ResourceSelectionContext {
  resource: OntologyResource;
  store: RootStore;
  placeLayout: LayoutPlacer;
}

export interface ResourceType {
  type: OntologyResourceType;
  icon: ReactElement;
  hasChildren: boolean;
  onSelect: (ctx: ResourceSelectionContext) => void;
  acceptsDrop: (hauled: Hauled[]) => boolean;
  onDrop: (ctx: ResourceSelectionContext, hauled: Hauled[]) => void;
  contextMenu: (ctx: ResourceSelectionContext, hauled: Hauled[]) => ReactElement;
}

export const resourceTypes: Record<string, ResourceType> = {
  builtin: {
    type: "builtin",
    icon: <Icon.Cluster />,
    hasChildren: true,
    acceptsDrop: () => false,
    onDrop: () => {},
  },
  cluster: {
    type: "cluster",
    icon: <Icon.Cluster />,
    hasChildren: true,
    acceptsDrop: () => false,
    onDrop: () => {},
  },
  node: {
    type: "node",
    icon: <Icon.Node />,
    hasChildren: true,
    acceptsDrop: () => false,
    onDrop: () => {},
  },
  channel: {
    type: "channel",
    icon: <Icon.Channel />,
    hasChildren: false,
    acceptsDrop: () => false,
    onDrop: () => {},
    onSelect: (ctx) => {
      const s = ctx.store.getState();
      const layout = selectActiveMosaicLayout(s);
      if (layout == null) {
        ctx.placeLayout(
          createLinePlot({
            channels: {
              ...ZERO_CHANNELS_STATE,
              y1: [ctx.resource.data.key as ChannelKey],
            },
          })
        );
      }
      switch (layout?.type) {
        case "line":
          ctx.store.dispatch(
            addLinePlotYChannel({
              key: layout?.key,
              axisKey: "y1",
              channels: [ctx.resource.data.key as ChannelKey],
            })
          );
      }
    },
  },
  group: {
    type: "group",
    hasChildren: true,
    acceptsDrop: () => true,
    onDrop: () => {},
  },
  range: {
    type: "range",
    hasChildren: true,
    acceptsDrop: () => true,
    onDrop: () => {},
  },
};
