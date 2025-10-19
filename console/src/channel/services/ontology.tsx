// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, isCalculated, ontology } from "@synnaxlabs/client";
import {
  Channel as PChannel,
  ContextMenu as PContextMenu,
  type Flux,
  type Haul,
  Icon,
  type Schematic as PSchematic,
  telem,
  Text,
  Tree,
} from "@synnaxlabs/pluto";
import { primitive, type record } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { ContextMenu } from "@/components";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";
import { Range } from "@/range";
import { Schematic } from "@/schematic";

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

export const useDelete = createUseDelete({
  type: "Channel",
  query: PChannel.useDelete,
  convertKey: Number,
});

const beforeSetAlias = async ({
  data,
}: Flux.BeforeUpdateParams<PChannel.UpdateAliasParams>) => {
  if (data.channel == null) return false;
  const [alias, renamed] = await Text.asyncEdit(
    ontology.idToString(channel.ontologyID(data.channel)),
  );
  if (!renamed) return false;
  return { ...data, alias };
};

export const useSetAlias = ({
  selection: {
    ids: [firstID],
  },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const activeRange = Range.useSelectActiveKey();
  const { update } = PChannel.useUpdateAlias({ beforeUpdate: beforeSetAlias });
  return useCallback(
    () =>
      update({
        range: activeRange ?? undefined,
        channel: Number(firstID.key),
        alias: "",
      }),
    [update, activeRange, firstID],
  );
};

export const useRename = createUseRename({
  query: PChannel.useRename,
  ontologyID: channel.ontologyID,
  convertKey: Number,
});

export const useDeleteAlias = ({
  selection: { ids },
}: Ontology.TreeContextMenuProps): (() => void) => {
  const activeRange = Range.useSelectActiveKey();
  const { update } = PChannel.useDeleteAlias();
  return useCallback(
    () =>
      update({
        range: activeRange ?? undefined,
        channels: ids.map((id) => Number(id.key)),
      }),
    [update, ids],
  );
};

const useOpenCalculated =
  () =>
  ({
    selection: { ids },
    placeLayout,
    state: { getResource },
  }: Ontology.TreeContextMenuProps) => {
    if (ids.length !== 1) return;
    const resource = getResource(ids[0]);
    return placeLayout(
      Channel.createCalculatedLayout({
        key: Number(resource.id.key),
        name: resource.name,
      }),
    );
  };

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids },
    state: { getResource },
  } = props;
  const activeRange = Range.useSelect();
  const handleSetAlias = useSetAlias(props);
  const resources = getResource(ids);
  const channelKeys = useMemo(() => ids.map((r) => Number(r.key)), [ids]);
  const channels = PChannel.useRetrieveMultiple({
    rangeKey: activeRange?.key,
    keys: channelKeys,
  });
  const showDeleteAlias = channels.data?.some((c) => c.alias != null) ?? false;
  const first = resources[0];
  const handleDeleteAlias = useDeleteAlias(props);
  const handleDelete = useDelete(props);
  const handleRename = useRename(props);
  const copyLink = Cluster.useCopyLinkToClipboard();
  const openCalculated = useOpenCalculated();
  const handleOpenCalculated = () => openCalculated(props);
  const handleLink = () => copyLink({ name: first.name, ontologyID: first.id });
  const singleResource = resources.length === 1;

  const isCalc = singleResource && isCalculated(resources[0].data as channel.Payload);

  return (
    <>
      {singleResource && (
        <ContextMenu.RenameItem onClick={handleRename} showBottomDivider />
      )}
      <Group.ContextMenuItem {...props} showBottomDivider />
      {isCalc && (
        <PContextMenu.Item onClick={handleOpenCalculated} showBottomDivider>
          <Icon.Edit />
          Edit calculation
        </PContextMenu.Item>
      )}
      {activeRange?.persisted === true && (singleResource || showDeleteAlias) && (
        <>
          {singleResource && (
            <PContextMenu.Item onClick={handleSetAlias}>
              <Icon.Rename />
              Set alias under {activeRange.name}
            </PContextMenu.Item>
          )}
          {showDeleteAlias && (
            <PContextMenu.Item onClick={handleDeleteAlias}>
              <Icon.Delete />
              Remove alias under {activeRange.name}
            </PContextMenu.Item>
          )}
          <PContextMenu.Divider />
        </>
      )}
      <ContextMenu.DeleteItem onClick={handleDelete} showBottomDivider />
      {singleResource && (
        <>
          <Link.CopyContextMenuItem onClick={handleLink} />
          <Ontology.CopyContextMenuItem {...props} showBottomDivider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </>
  );
};

export const Item = ({ id, resource, icon: _, ...rest }: Ontology.TreeItemProps) => {
  const activeRange = Range.useSelect();
  const res = PChannel.useRetrieve({
    key: Number(id.key),
    rangeKey: activeRange?.key,
  }).data;
  let name = resource.name;
  if (primitive.isNonZero(res?.alias)) name = res?.alias;
  const data = resource.data as channel.Payload;
  const DataTypeIcon = PChannel.resolveIcon(data);
  return (
    <Tree.Item {...rest}>
      <DataTypeIcon color={10} />
      <Text.MaybeEditable
        id={ontology.idToString(id)}
        allowDoubleClick={false}
        value={name}
        overflow="ellipsis"
        style={{ width: 0 }}
        grow
        disabled={!allowRename(resource)}
        onChange
      />
      {data.virtual && <Icon.Virtual color={8} />}
    </Tree.Item>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "channel",
  icon: <Icon.Channel />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems,
  Item,
  TreeContextMenu,
};
