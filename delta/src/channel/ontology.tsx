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
import { Menu } from "@synnaxlabs/pluto";

import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { type Ontology } from "@/ontology";
import { PID } from "@/pid";
import { Range } from "@/range";

const canDrop = (): boolean => false;

const onSelect: Ontology.HandleSelect = ({ store, placeLayout, selection }): void => {
  const state = store.getState();
  const layout = Layout.selectActiveMosaicTab(state);
  if (selection.length === 0) return;

  // If no layout is selected, create a new line plot and add the selected channels
  // to it.
  if (layout == null) {
    placeLayout(
      LinePlot.create({
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

const haulItems = ({ name, id }: ontology.Resource): Haul.Item[] => [
  {
    type: "channel",
    key: Number(id.key),
  },
  {
    type: PID.HAUL_TYPE,
    key: "value",
    data: { telem: { channel: Number(id.key) }, label: name },
  },
];

const allowRename = (): boolean => true;

const handleSetAlias = async ({
  id,
  name,
  client,
  store,
}: Ontology.HandleTreeRenameProps): Promise<void> => {
  const activeRange = Range.select(store.getState());
  if (activeRange == null) return;
  const rng = await client.ranges.retrieve(activeRange.key);
  await rng.setAlias(id.key, name);
};

const handleRename: Ontology.HandleTreeRename = (p) => {
  void handleSetAlias(p);
};

const TreeContextMenu: Ontology.TreeContextMenu = ({
  store,
  selection,
}): ReactElement => {
  const activeRange = Range.select(store.getState());
  return (
    <Menu.Menu level="small" iconSpacing="small">
      <Group.GroupMenuItem selection={selection} />
      {activeRange != null && (
        <>
          {selection.resources.length === 1 && (
            <Menu.Item itemKey="alias" startIcon={<Icon.Rename />}>
              Set Alias Under {activeRange.name}
            </Menu.Item>
          )}
          <Menu.Item itemKey="plot" startIcon={<Icon.Visualize />}>
            Plot for {activeRange.name}
          </Menu.Item>
        </>
      )}
    </Menu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "channel",
  icon: <Icon.Channel />,
  hasChildren: false,
  allowRename,
  onRename: handleRename,
  canDrop,
  onSelect,
  haulItems,
  TreeContextMenu,
};
