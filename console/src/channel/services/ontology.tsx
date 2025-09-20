// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, isCalculated, ontology } from "@synnaxlabs/client";
import {
  Channel as PChannel,
  Flex,
  type Haul,
  Icon,
  Menu as PMenu,
  type Schematic as PSchematic,
  telem,
  Text,
  Tree,
} from "@synnaxlabs/pluto";
import { errors, primitive, type record } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useMemo } from "react";

import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { Range } from "@/range";
import { Schematic } from "@/schematic";

const canDrop = (): boolean => false;

const handleSelect: Ontology.HandleSelect = ({
  store,
  placeLayout,
  selection,
}): void => {
  const state = store.getState();
  const layout = Layout.selectActiveMosaicLayout(state);
  if (selection.length === 0) return;

  const nonVirtualSelection = selection
    .filter((s) => s.data?.virtual !== true || s.data.expression != "")
    .map((s) => Number(s.id.key));

  if (nonVirtualSelection.length === 0) return;

  // Otherwise, update the layout with the selected channels.
  switch (layout?.type) {
    case LinePlot.LAYOUT_TYPE:
      store.dispatch(
        LinePlot.setYChannels({
          key: layout.key,
          mode: "add",
          axisKey: "y1",
          channels: nonVirtualSelection,
        }),
      );
      break;
    default:
      placeLayout(
        LinePlot.create({
          channels: {
            ...LinePlot.ZERO_CHANNELS_STATE,
            y1: nonVirtualSelection,
          },
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
  const schematicSymbolProps: PSchematic.Symbol.ValueProps = {
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
      data: schematicSymbolProps as record.Unknown,
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

const allowRename: Ontology.AllowRename = ({ data }) => data?.internal !== true;

export const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({
    type: "Channel",
  });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({
      state: { nodes, setNodes, getResource },
      selection: { resourceIDs },
    }) => {
      const prevNodes = Tree.deepCopy(nodes);
      const resources = getResource(resourceIDs);
      if (!(await confirm(resources))) throw new errors.Canceled();
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => ontology.idToString(id)),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ client, selection: { resourceIDs } }) =>
      await client.channels.delete(resourceIDs.map(({ key }) => Number(key))),
    onError: (
      e,
      { selection: { resourceIDs }, handleError, state: { setNodes, getResource } },
      prevNodes,
    ) => {
      if (errors.Canceled.matches(e)) return;
      if (prevNodes != null) setNodes(prevNodes);
      let message = "Failed to delete channels";
      if (resourceIDs.length === 1) {
        const resource = getResource(resourceIDs[0]);
        message = `Failed to delete channel ${resource.name}`;
      }
      handleError(e, message);
    },
  }).mutate;
};

export const useSetAlias = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: ({ state: { nodes } }) => Tree.deepCopy(nodes),
    mutationFn: async ({
      client,
      store,
      selection: { resourceIDs },
      state: { getResource },
    }) => {
      const resources = getResource(resourceIDs);
      const [value, renamed] = await Text.asyncEdit(
        ontology.idToString(resourceIDs[0]),
      );
      if (!renamed) return;
      const activeRange = Range.select(store.getState());
      if (activeRange == null) return;
      const rng = await client.ranges.retrieve(activeRange.key);
      await rng.setAlias(Number(resources[0].id.key), value);
    },
    onError: (
      e: Error,
      { selection: { resourceIDs }, handleError, state: { setNodes, getResource } },
      prevNodes,
    ) => {
      if (prevNodes != null) setNodes(prevNodes);
      const first = getResource(resourceIDs[0]);
      handleError(e, `Failed to set alias for ${first.name}`);
    },
  }).mutate;

export const useRename = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: ({ state: { nodes } }) => Tree.deepCopy(nodes),
    mutationFn: async ({
      client,
      selection: { resourceIDs },
      state: { getResource },
    }) => {
      const resources = getResource(resourceIDs);
      const [value, renamed] = await Text.asyncEdit(
        ontology.idToString(resourceIDs[0]),
      );
      if (!renamed) return;
      await client.channels.rename(Number(resources[0].id.key), value);
    },
    onError: (
      e: Error,
      { selection: { resourceIDs }, handleError, state: { setNodes, getResource } },
      prevNodes,
    ) => {
      if (prevNodes != null) setNodes(prevNodes);
      const first = getResource(resourceIDs[0]);
      handleError(e, `Failed to rename ${first.name}`);
    },
  }).mutate;

export const useDeleteAlias = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: ({ state: { nodes } }) => Tree.deepCopy(nodes),
    mutationFn: async ({
      client,
      store,
      selection: { resourceIDs },
      state: { getResource },
    }) => {
      const resources = getResource(resourceIDs);
      const activeRange = Range.select(store.getState());
      if (activeRange == null) return;
      const rng = await client.ranges.retrieve(activeRange.key);
      await rng.deleteAlias(...resources.map((r) => Number(r.id.key)));
    },
    onError: (
      e: Error,
      { selection: { resourceIDs }, handleError, state: { setNodes, getResource } },
      prevNodes,
    ) => {
      if (prevNodes != null) setNodes(prevNodes);
      const first = getResource(resourceIDs[0]);
      handleError(e, `Failed to remove alias on ${first.name}`);
    },
  }).mutate;

const useOpenCalculated =
  () =>
  ({
    selection: { resourceIDs },
    placeLayout,
    state: { getResource },
  }: Ontology.TreeContextMenuProps) => {
    if (resourceIDs.length !== 1) return;
    const resource = getResource(resourceIDs[0]);
    return placeLayout(
      Channel.createCalculatedLayout({
        key: Number(resource.id.key),
        name: resource.name,
      }),
    );
  };

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resourceIDs, rootID },
    state: { getResource, shape },
  } = props;
  const activeRange = Range.useSelect();
  const groupFromSelection = Group.useCreateFromSelection();
  const setAlias = useSetAlias();
  const resources = getResource(resourceIDs);
  const channelKeys = useMemo(
    () => resourceIDs.map((r) => Number(r.key)),
    [resourceIDs],
  );
  const channels = PChannel.useRetrieveMany({
    rangeKey: activeRange?.key,
    keys: channelKeys,
  });
  const showDeleteAlias = channels.data?.some((c) => c.alias != null) ?? false;
  const first = resources[0];
  const delAlias = useDeleteAlias();
  const del = useDelete();
  const handleRename = useRename();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const openCalculated = useOpenCalculated();
  const handleSelect = {
    group: () => groupFromSelection(props),
    delete: () => del(props),
    deleteAlias: () => delAlias(props),
    alias: () => setAlias(props),
    rename: () => handleRename(props),
    link: () => handleLink({ name: first.name, ontologyID: first.id }),
    openCalculated: () => openCalculated(props),
  };
  const singleResource = resources.length === 1;

  const isCalc = singleResource && isCalculated(resources[0].data as channel.Payload);

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {singleResource && <Menu.RenameItem />}
      <Group.MenuItem resourceIDs={resourceIDs} shape={shape} rootID={rootID} />
      {isCalc && (
        <>
          <PMenu.Divider />
          <PMenu.Item itemKey="openCalculated">
            <Icon.Edit />
            Edit Calculation
          </PMenu.Item>
        </>
      )}
      {activeRange != null &&
        activeRange.persisted &&
        (singleResource || showDeleteAlias) && (
          <>
            <PMenu.Divider />
            {singleResource && (
              <PMenu.Item itemKey="alias">
                <Icon.Rename />
                Set Alias Under {activeRange.name}
              </PMenu.Item>
            )}
            {showDeleteAlias && (
              <PMenu.Item itemKey="deleteAlias">
                <Icon.Delete />
                Remove Alias Under {activeRange.name}
              </PMenu.Item>
            )}
            <PMenu.Divider />
          </>
        )}
      <PMenu.Item itemKey="delete">
        <Icon.Delete />
        Delete
      </PMenu.Item>
      {singleResource && (
        <>
          <PMenu.Divider />
          <Link.CopyMenuItem />
        </>
      )}
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const Item = ({
  id,
  resource,
  icon: _,
  onRename,
  ...rest
}: Ontology.TreeItemProps) => {
  const activeRange = Range.useSelect();
  const res = PChannel.useRetrieve({
    key: Number(id.key),
    rangeKey: activeRange?.key,
  }).data;
  let name = resource.name;
  if (primitive.isNonZero(res?.alias)) name = res?.alias;
  const data = resource.data as channel.Payload;
  const I = PChannel.resolveIcon(data);
  return (
    <Tree.Item {...rest}>
      <Flex.Box gap="small" x align="center">
        <I style={{ color: "var(--pluto-gray-l10" }} />
        <Text.MaybeEditable
          id={ontology.idToString(id)}
          allowDoubleClick={false}
          value={name}
          disabled={!allowRename(resource)}
          onChange={onRename}
        />
      </Flex.Box>
      {data.virtual && (
        <Icon.Virtual
          style={{ color: "var(--pluto-gray-l8)", transform: "scale(1)" }}
        />
      )}
    </Tree.Item>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "channel",
  icon: <Icon.Channel />,
  hasChildren: false,
  onSelect: handleSelect,
  canDrop,
  haulItems,
  allowRename,
  Item,
  TreeContextMenu,
};
