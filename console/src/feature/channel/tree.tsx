// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/channel/tree.css";

import {
  channel,
  isCalculated,
  query,
  ranger,
  resource,
  status,
  type Synnax,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import {
  Access,
  Channel as PChannel,
  type Flux,
  Schematic as PSchematic,
} from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";

import { useOpen } from "@/feature/channel/useOpen";
import { Channel } from "@/platform/channel";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Range } from "@/platform/range";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const haulItems: Tree.HaulItems = ({ name, id: { key } }, client) => {
  const ch = client.channels.watch(Number(key)).getCached();
  if (!query.isLive(ch) || !ch.internal) return [PChannel.createHaulItem(Number(key))];
  return [
    PSchematic.createHaulItem({
      key: id.create(),
      config: { variant: "value", label: { label: name }, channel: Number(key) },
    }),
  ];
};

const useDelete = Tree.createUseDelete({
  type: "Channel",
  useUpdate: PChannel.useDelete,
  convertKey: Number,
});

const beforeSetAlias = async ({
  data,
}: Flux.BeforeUpdateParams<PChannel.UpdateAliasParams>) => {
  if (data.channel == null) return false;
  const [alias, renamed] = await Text.asyncEdit(
    List.itemNameID(resource.idToString(channel.resourceID(data.channel))),
  );
  if (!renamed) return false;
  return { ...data, alias };
};

const useSetAlias = ({
  selection: {
    ids: [firstID],
  },
}: Tree.ContextMenuProps): (() => void) => {
  const activeRange = Session.Range.useSelectSelectedKey();
  const { update } = PChannel.useUpdateAlias({ beforeUpdate: beforeSetAlias });
  return () =>
    update({
      range: activeRange ?? undefined,
      channel: Number(firstID.key),
      alias: "",
    });
};

const useRename = Tree.createUseRename({
  useUpdate: PChannel.useRename,
  resourceID: channel.resourceID,
  convertKey: Number,
});

const useDeleteAlias = ({
  selection: { ids },
}: Tree.ContextMenuProps): (() => void) => {
  const activeRange = Session.Range.useSelectSelectedKey();
  const { update } = PChannel.useDeleteAlias();
  return () =>
    update({
      range: activeRange ?? undefined,
      channels: ids.map((id) => Number(id.key)),
    });
};

const useEditCalculated = () => {
  const open = Channel.useCalculatedModal();
  return ({ selection: { ids }, state: { getResource } }: Tree.ContextMenuProps) => {
    if (ids.length !== 1) return;
    const resource = getResource(ids[0]);
    open({ channelKey: Number(resource.id.key) });
  };
};

const retrieve: Tree.Retrieve = async (c, key) =>
  (await c.channels.retrieve(Number(key))).payload;

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID, parentID },
    state,
  } = props;
  const activeRange = Range.useResolve();
  const groupFromSelection = Group.useCreateFromSelection();
  const handleSetAlias = useSetAlias(props);
  const resources = Tree.useResources(state, ids);
  const channelKeys = ids.map((r) => Number(r.key));
  const channels = PChannel.useMultiple({
    rangeKey: activeRange?.key,
    where: { key: { in: channelKeys } },
  });
  const showDeleteAlias = channels.some((c) => c.alias != null);
  const first = resources[0];
  const handleDeleteAlias = useDeleteAlias(props);
  const handleDelete = useDelete(props);

  const hasUpdatePermission = Access.useUpdateGranted(
    ids.map((id) => channel.resourceID(Number(id.key))),
  );
  const hasDeletePermission = Access.useDeleteGranted(
    ids.map((id) => channel.resourceID(Number(id.key))),
  );
  const hasAliasCreatePermission = Access.useCreateGranted(
    ids.map((id) => ranger.alias.resourceID(activeRange?.key ?? "", Number(id.key))),
  );
  const hasAliasDeletePermission = Access.useDeleteGranted(
    ids.map((id) => ranger.alias.resourceID(activeRange?.key ?? "", Number(id.key))),
  );
  const handleRename = useRename(props);

  const handleLink = Core.useCopyLinkToClipboard();
  const openCalculated = useEditCalculated();
  const singleResource = resources.length === 1;
  const firstChannel = channels.find(({ key }) => key === channelKeys[0]);
  const isCalc =
    singleResource && firstChannel != null && isCalculated(firstChannel.payload);
  const renamable = firstChannel?.internal !== true;

  return (
    <ContextMenu.Menu>
      {isCalc && hasUpdatePermission && (
        <Menu.Item itemKey="openCalculated" onClick={() => openCalculated(props)}>
          <Icon.Edit />
          Edit calculation
        </Menu.Item>
      )}
      <Menu.Divider />
      {singleResource && hasUpdatePermission && renamable && (
        <ContextMenu.RenameItem onClick={handleRename} />
      )}
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          shape={state.shape}
          rootID={rootID}
          parentID={parentID}
          onClick={() => groupFromSelection(props)}
        />
      )}
      <Menu.Divider />
      {activeRange != null &&
        activeRange.variant === "persisted" &&
        (singleResource || showDeleteAlias) &&
        (hasAliasCreatePermission || hasAliasDeletePermission) && (
          <>
            {singleResource && hasAliasCreatePermission && renamable && (
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
          </>
        )}
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      {singleResource && (
        <>
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, resourceID: first.id })}
          />
          <Tree.CopyPropertiesContextMenuItem {...props} retrieve={retrieve} />
        </>
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const Content = ({ resource, icon: _, ...rest }: Tree.ContentProps) => {
  const activeRange = Range.useResolve();
  const params = { key: Number(resource.id.key), rangeKey: activeRange?.key };
  const { data: alias } = PChannel.useResultAlias(params);
  const { data: chStatus } = PChannel.useResultStatus(params);
  const { data: kind } = PChannel.useResultKind(params);
  const name = alias ?? resource.name;
  const DataTypeIcon = kind?.icon ?? Icon.Channel;
  const statusVariant = status.keepVariants(chStatus?.variant, ["error", "warning"]);
  const highlight = Tree.useHighlight();
  return (
    <PTree.Item {...rest}>
      <DataTypeIcon color={10} />
      <Text.MaybeEditable
        id={List.itemNameID(resource.key)}
        allowDoubleClick={false}
        value={name}
        overflow="fade"
        className={CSS.BE("channel-tree-item", "name")}
        grow
        disabled={kind?.internal === true}
        onChange
        highlight={highlight}
      />
      <Tree.MatchGroup id={resource.id} />
      {statusVariant != null && (
        <Tooltip.Dialog location="right">
          <Status.Summary variant={statusVariant} hideIcon level="small" weight={450}>
            {chStatus?.message ?? ""}
          </Status.Summary>
          <Status.Indicator variant={statusVariant} />
        </Tooltip.Dialog>
      )}
      {kind?.virtual === true && <Icon.Virtual color={9} />}
    </PTree.Item>
  );
};

const toResource = (ch: channel.Channel): Tree.Resource =>
  Tree.toResource("channel")(ch.payload);

/** Lists the channels that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "channel",
  children: (client, key) =>
    Tree.createSource(client.channels, { where: { groups: [key] } }, toResource),
};

/** Lists the top level of the channel tree: its groups and the channels in no group. */
export const listTreeRoot = (client: Synnax): Tree.Source[] => [
  Tree.createSource(
    client.groups,
    { scope: "channel", nested: false },
    Tree.toResource("group"),
  ),
  Tree.createSource(
    client.channels,
    { where: { grouped: false, internal: { eq: false } } },
    toResource,
  ),
];

const TreeItem = Tree.createItem({
  linkable: true,
  icon: <Icon.Channel />,
  useOnSelect: useOpen,
  haulItems,
  Content,
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { channel: TreeItem } satisfies Tree.Items;
