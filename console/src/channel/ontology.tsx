// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Channel,
  type Haul,
  Menu as PMenu,
  type Schematic as PSchematic,
  telem,
} from "@synnaxlabs/pluto";
import { Tree } from "@synnaxlabs/pluto/tree";
import { nanoid } from "nanoid";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { Range } from "@/range";
import { Schematic } from "@/schematic";

const canDrop = (): boolean => false;

const handleSelect: Ontology.HandleSelect = ({
  store,
  placeLayout,
  selection,
}): void => {
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
      }),
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
        }),
      );
  }
};

const haulItems = ({ name, id }: ontology.Resource): Haul.Item[] => {
  const t = telem.sourcePipeline("string", {
    connections: [
      {
        from: "valueStream",
        to: "stringifier",
      },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel: Number(id.key) }),
      stringifier: telem.stringifyNumber({ precision: 2 }),
    },
    outlet: "stringifier",
  });
  const schematicSymbolProps: PSchematic.ValueProps = {
    label: {
      label: name,
      level: "p",
    },
    telem: t,
  };
  return [
    {
      type: "channel",
      key: Number(id.key),
    },
    {
      type: Schematic.HAUL_TYPE,
      key: "value",
      data: schematicSymbolProps,
    },
  ];
};

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
  await rng.setAlias(Number(id.key), name);
};

const handleRename: Ontology.HandleTreeRename = ({ client, name, id, addStatus }) => {
  void (async () => {
    try {
      const channelKey = Number(id.key);
      await client.channels.rename(channelKey, name);
    } catch (e) {
      addStatus({
        variant: "error",
        key: `renameChannelError-${id.key}`,
        message: (e as Error).message,
      });
    }
  })();
};

const handleDeleteAlias = async ({
  selection: { resources },
  client,
  store,
}: Ontology.TreeContextMenuProps): Promise<void> => {
  const activeRange = Range.select(store.getState());
  if (activeRange == null) return;
  const rng = await client.ranges.retrieve(activeRange.key);
  await rng.deleteAlias(...resources.map((r) => Number(r.id.key)));
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { store, selection, client, addStatus } = props;
  const activeRange = Range.select(store.getState());
  const { nodes, resources } = selection;

  const handleSelect = (itemKey: string): void => {
    switch (itemKey) {
      case "alias":
        Tree.startRenaming(nodes[0].key, (name) =>
          handleSetAlias({ ...props, name, id: resources[0].id }),
        );
        break;
      case "rename":
        Tree.startRenaming(nodes[0].key, (name) => {
          handleRename({ ...props, name, id: resources[0].id });
        });
        break;
      case "deleteAlias":
        handleDeleteAlias(props).catch((e: Error) => {
          addStatus({
            variant: "error",
            key: "deleteAliasError",
            message: e.message,
          });
        });
        break;
      case "delete":
        client.channels
          .delete(resources.map(({ id }) => Number(id.key)))
          .catch((e: Error) => {
            addStatus({
              variant: "error",
              key: nanoid(),
              message: e.message,
            });
          });
        break;
      case "group":
        void Group.fromSelection(props);
        break;
      case "link": {
        Link.CopyLinkToClipboard({
          clusterKey: client.key,
          resource: {
            type: "channel",
            key: resources[0].id.key,
          },
          addStatus,
        });
      }
    }
  };

  const singleResource = selection.resources.length === 1;

  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={handleSelect}>
      <Group.GroupMenuItem selection={selection} />
      {singleResource && <Ontology.RenameMenuItem />}
      {activeRange != null && activeRange.persisted && (
        <>
          {singleResource && (
            <PMenu.Item itemKey="alias" startIcon={<Icon.Rename />}>
              Set Alias Under {activeRange.name}
            </PMenu.Item>
          )}
          <PMenu.Item itemKey="deleteAlias" startIcon={<Icon.Delete />}>
            Clear Alias Under {activeRange.name}
          </PMenu.Item>
        </>
      )}
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {singleResource && <Link.CopyMenuItem />}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const Item: Tree.Item = (props: Tree.ItemProps): ReactElement => {
  const alias = Channel.useAlias(Number(new ontology.ID(props.entry.key).key));
  return (
    <Tree.DefaultItem
      {...props}
      entry={{ ...props.entry, name: alias ?? props.entry.name }}
    />
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "channel",
  icon: <Icon.Channel />,
  hasChildren: false,
  allowRename,
  onRename: () => {},
  canDrop,
  onSelect: handleSelect,
  haulItems,
  Item,
  TreeContextMenu,
};
