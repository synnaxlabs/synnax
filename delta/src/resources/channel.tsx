// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { type Haul } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";

import { type Service, type SelectionContext } from "./service";

const TYPE = "channel";

const ICON = <Icon.Channel />;

const HAS_CHILDREN = false;

const canDrop = (): boolean => false;

const onSelect = ({ store, placeLayout, selection }: SelectionContext): void => {
  const state = store.getState();
  const layout = Layout.selectActiveMosaicTab(state);
  if (selection.length === 0) return;

  // If no layout is selected, create a new line plot and add the selected channels
  // to it.
  if (layout == null) {
    placeLayout(
      LinePlot.createLinePlot({
        channels: {
          ...LinePlot.ZERO_CHANNELS_STATE,
          y1: selection.map((s) => Number(s.id.key)),
        },
      })
    );
    return;
  }

  // Otherwise, update the layout with the selected channels.
  switch (layout.type) {
    case LinePlot.LAYOUT_TYPE:
      store.dispatch(
        LinePlot.setYChannels({
          key: layout.key,
          mode: "add",
          axisKey: "y1",
          channels: selection.map((s) => Number(s.id.key)),
        })
      );
  }
};

const haulItems = ({ id }: ontology.Resource): Haul.Item[] => [
  {
    type: "channel",
    key: Number(id.key),
  },
];

const allowRename = (): boolean => false;

const TreeContextMenu = (): ReactElement => <></>;

export const CHANNEL_SERVICE: Service = {
  type: TYPE,
  icon: ICON,
  hasChildren: HAS_CHILDREN,
  allowRename,
  canDrop,
  onSelect,
  haulItems,
  TreeContextMenu,
};
