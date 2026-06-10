// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, isCalculated, ontology, ranger } from "@synnaxlabs/client";
import {
  Access,
  Channel as PChannel,
  type Flux,
  type Haul,
  Icon,
  Menu,
  Schematic as PSchematic,
  Status,
  telem,
  Text,
  Tooltip,
  Tree,
} from "@synnaxlabs/pluto";
import { id, primitive, status, uuid } from "@synnaxlabs/x";
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
import { Workspace } from "@/workspace";

const handleSelect: Ontology.HandleSelect = ({
  client,
  store,
  placeLayout,
  selection,
  handleError,
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
    case LinePlot.LAYOUT_TYPE: {
      // A plot still staging a pendingUpload does not exist on the server yet,
      // so an add would fail with not found. Skip until useAutoUpload lands it;
      // the user can retry once the plot is created.
      if (LinePlot.selectPendingUpload(state, layout.key) != null) return;
      handleError(
        () => LinePlot.addChannelsToActivePlot(client, layout.key, nonVirtualSelection),
        "Failed to add channels to plot",
      );
      break;
    }
    default: {
      const workspace = Workspace.selectActiveKey(state) ?? uuid.ZERO;
      const activeRange = Range.selectActiveKey(state) ?? Range.RECENT_KEY;
      handleError(async () => {
        const { key, name } = await client.lineplots.create(workspace, {
          name: "Line Plot",
          channels: { y1: nonVirtualSelection },
          ranges: { x1: [activeRange] },
        });
        placeLayout(LinePlot.create({ key, name }));
      }, "Failed to create plot");
    }
  }
};

const haulItems = ({ name, id: otgID, data }: ontology.Resource): Haul.Item[] => {
  const t = telem.sourcePipeline("string", {
    connections: [
      {
        from: "valueStream",
        to: "stringifier",
      },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel: Number(otgID.key) }),
      stringifier: telem.stringifyNumber({ precision: 2 }),
    },
    outlet: "stringifier",
  });
  const nodeConfig: PSchematic.Node.ConfigOf<"value"> = {
    variant: "value",
    label: { label: name, level: "p" },
    telem: t,
  };
  const items = [
    PSchematic.createHaulItem({
      key: id.create(),
      variant: "value",
      config: nodeConfig,
    }),
  ];
  if (data?.internal === true) return items;
  return [PChannel.createHaulItem(Number(otgID.key))];
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
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const activeRange = Range.useSelect();
  const groupFromSelection = Group.useCreateFromSelection();
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

  const hasUpdatePermission = Access.useUpdateGranted(
    ids.map((id) => channel.ontologyID(Number(id.key))),
  );
  const hasDeletePermission = Access.useDeleteGranted(
    ids.map((id) => channel.ontologyID(Number(id.key))),
  );
  const hasAliasCreatePermission = Access.useCreateGranted(
    ids.map((id) => ranger.alias.ontologyID(activeRange?.key ?? "", Number(id.key))),
  );
  const hasAliasDeletePermission = Access.useDeleteGranted(
    ids.map((id) => ranger.alias.ontologyID(activeRange?.key ?? "", Number(id.key))),
  );
  const handleRename = useRename(props);

  const handleLink = Cluster.useCopyLinkToClipboard();
  const openCalculated = useOpenCalculated();
  const singleResource = resources.length === 1;

  const isCalc = singleResource && isCalculated(resources[0].data as channel.Payload);

  return (
    <ContextMenu.Menu>
      {singleResource && hasUpdatePermission && (
        <ContextMenu.RenameItem onClick={handleRename} />
      )}
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          shape={shape}
          rootID={rootID}
          onClick={() => groupFromSelection(props)}
        />
      )}
      {isCalc && hasUpdatePermission && (
        <>
          <Menu.Divider />
          <Menu.Item itemKey="openCalculated" onClick={() => openCalculated(props)}>
            <Icon.Edit />
            Edit calculation
          </Menu.Item>
        </>
      )}
      {activeRange != null &&
        activeRange.persisted &&
        (singleResource || showDeleteAlias) &&
        (hasAliasCreatePermission || hasAliasDeletePermission) && (
          <>
            <Menu.Divider />
            {singleResource && hasAliasCreatePermission && (
              <Menu.Item itemKey="alias" onClick={handleSetAlias}>
                <Icon.Rename />
                Set alias under {activeRange.name}
              </Menu.Item>
            )}
            {showDeleteAlias && hasAliasDeletePermission && (
              <Menu.Item itemKey="deleteAlias" onClick={handleDeleteAlias}>
                <Icon.Delete />
                Remove alias under {activeRange.name}
              </Menu.Item>
            )}
            <Menu.Divider />
          </>
        )}
      {hasDeletePermission && (
        <>
          <ContextMenu.DeleteItem onClick={handleDelete} />
          <Menu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, ontologyID: first.id })}
          />
          <Ontology.CopyPropertiesContextMenuItem {...props} />
        </>
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
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
  const statusVariant = status.keepVariants(res?.status?.variant, ["error", "warning"]);
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
      {statusVariant != null && (
        <Tooltip.Dialog location="right">
          <Status.Summary variant={statusVariant} hideIcon level="small" weight={450}>
            {res?.status?.message ?? ""}
          </Status.Summary>
          <Status.Indicator variant={statusVariant} />
        </Tooltip.Dialog>
      )}
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
