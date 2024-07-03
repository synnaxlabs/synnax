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
import { errors, UnknownRecord } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { type ReactElement } from "react";

import { Menu } from "@/components/menu";
import { Confirm } from "@/confirm";
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

const haulItems = ({ name, id, data }: ontology.Resource): Haul.Item[] => {
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
  const items = [
    {
      type: Schematic.HAUL_TYPE,
      key: "value",
      data: schematicSymbolProps as UnknownRecord,
    },
  ];
  if (data?.internal === true) return items;
  return [
    {
      type: "channel",
      key: Number(id.key),
    },
  ];
};

const allowRename: Ontology.AllowRename = (res) => {
  if (res.data?.internal === true) return false;
  return true;
};

export const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = Confirm.useModal();

  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ state: { nodes, setNodes }, selection: { resources } }) => {
      const prevNodes = Tree.deepCopy(nodes);
      let message = `Are you sure you want to delete ${resources.length} channels?`;
      if (resources.length === 1)
        message = `Are you sure you want to delete ${resources[0].name}?`;
      if (
        !(await confirm(
          {
            message,
            description:
              "Deleting channels will also delete all of their associated data. This action cannot be undone.",
            confirm: { variant: "error", label: "Delete" },
            cancel: { label: "Cancel" },
          },
          { name: "Channel.Delete", icon: "Channel" },
        ))
      )
        throw errors.CANCELED;
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => id.toString()),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection: { resources } }) =>
      await client.channels.delete(resources.map(({ id }) => Number(id.key))),
    onError: (
      e,
      { selection: { resources }, addStatus, state: { setNodes } },
      prevNodes,
    ) => {
      if (errors.CANCELED.matches(e)) return;
      if (prevNodes != null) setNodes(prevNodes);
      let message = "Failed to delete channels";
      if (resources.length === 1)
        message = `Failed to delete channel ${resources[0].name}`;
      addStatus({
        key: nanoid(),
        variant: "error",
        message,
        description: e.message,
      });
    },
  }).mutate;
};

export const useSetAlias = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: ({ state: { nodes } }) => Tree.deepCopy(nodes),
    mutationFn: async ({ client, store, selection: { resources, nodes } }) => {
      const [value, renamed] = await Tree.asyncRename(nodes[0].key);
      if (!renamed) return;
      const activeRange = Range.select(store.getState());
      if (activeRange == null) return;
      const rng = await client.ranges.retrieve(activeRange.key);
      await rng.setAlias(Number(resources[0].id.key), value);
    },
    onError: (
      e: Error,
      { selection: { resources }, addStatus, state: { setNodes } },
      prevNodes,
    ) => {
      if (prevNodes != null) setNodes(prevNodes);
      const first = resources[0];
      addStatus({
        key: nanoid(),
        variant: "error",
        message: `Failed to set alias for ${first.name}`,
        description: e.message,
      });
    },
  }).mutate;

export const useRename = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: ({ state: { nodes } }) => Tree.deepCopy(nodes),
    mutationFn: async ({ client, selection: { resources, nodes } }) => {
      const [value, renamed] = await Tree.asyncRename(nodes[0].key);
      if (!renamed) return;
      await client.channels.rename(Number(resources[0].id.key), value);
    },
    onError: (
      e: Error,
      { selection: { resources }, addStatus, state: { setNodes } },
      prevNodes,
    ) => {
      if (prevNodes != null) setNodes(prevNodes);
      const first = resources[0];
      addStatus({
        key: nanoid(),
        variant: "error",
        message: `Failed to rename ${first.name}`,
        description: e.message,
      });
    },
  }).mutate;

export const useDeleteAlias = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: ({ state: { nodes } }) => Tree.deepCopy(nodes),
    mutationFn: async ({ client, store, selection: { resources } }) => {
      const activeRange = Range.select(store.getState());
      if (activeRange == null) return;
      const rng = await client.ranges.retrieve(activeRange.key);
      await rng.deleteAlias(...resources.map((r) => Number(r.id.key)));
    },
    onError: (
      e: Error,
      { selection: { resources }, addStatus, state: { setNodes } },
      prevNodes,
    ) => {
      if (prevNodes != null) setNodes(prevNodes);
      const first = resources[0];
      addStatus({
        key: nanoid(),
        variant: "error",
        message: `Failed to remove alias on ${first.name}`,
        description: e.message,
      });
    },
  }).mutate;

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { store, selection } = props;
  const activeRange = Range.select(store.getState());
  const groupFromSelection = Group.useCreateFromSelection();
  const setAlias = useSetAlias();
  const delAlias = useDeleteAlias();
  const del = useDelete();
  const handleRename = useRename();
  const handleSelect = {
    group: () => groupFromSelection(props),
    delete: () => del(props),
    deleteAlias: () => delAlias(props),
    alias: () => setAlias(props),
    rename: () => handleRename(props),
  };
  const singleResource = selection.resources.length === 1;
  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={handleSelect}>
      {singleResource && <Menu.RenameItem />}
      <Group.GroupMenuItem selection={selection} />
      {activeRange != null && activeRange.persisted && (
        <>
          <PMenu.Divider />
          {singleResource && (
            <PMenu.Item itemKey="alias" startIcon={<Icon.Rename />}>
              Set Alias Under {activeRange.name}
            </PMenu.Item>
          )}
          <PMenu.Item itemKey="deleteAlias" startIcon={<Icon.Delete />}>
            Remove Alias Under {activeRange.name}
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {singleResource && <Link.CopyMenuItem />}
      <PMenu.Divider />
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
  onRename: undefined,
  canDrop,
  onSelect: handleSelect,
  haulItems,
  Item,
  TreeContextMenu,
};
