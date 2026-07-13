// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, isCalculated, ontology, ranger, status } from "@synnaxlabs/client";
import {
  Access,
  Channel as PChannel,
  type Flux,
  type Haul,
  Icon,
  List,
  Menu,
  Schematic as PSchematic,
  Status,
  Synnax,
  telem,
  Text,
  Tooltip,
  Tree as PTree,
} from "@synnaxlabs/pluto";
import { id, primitive } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";

import { Channel } from "@/platform/channel";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Group } from "@/platform/group";
import { Layout } from "@/platform/layout";
import { LinePlot } from "@/platform/lineplot";
import { Link } from "@/platform/link";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useOnSelect = (): ((entry: Tree.Entry) => void) => {
  const client = Synnax.use();
  const store = Session.useStore();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (entry) => {
      if (client == null) return;
      handleError(async () => {
        const channelKey = Number(entry.id.key);
        const ch = await client.channels.retrieve(channelKey);
        if (ch.virtual && ch.expression == "") return;
        const state = store.getState();
        const layout = Session.Layout.selectActiveMosaicLayout(state);
        if (layout?.type === LinePlot.LAYOUT_TYPE) {
          await LinePlot.addChannelsToActivePlot(client, layout.key, [channelKey]);
          return;
        }
        const project = Session.Project.selectSelected(state);
        const activeRange =
          Session.Range.selectSelectedKey(state) ?? Session.Range.RECENT_KEY;
        const { key, name } = await client.lineplots.create(project, {
          name: "Line Plot",
          channels: { y1: [channelKey] },
          ranges: { x1: [activeRange] },
        });
        placeLayout(LinePlot.create({ key, name }));
      }, "Failed to plot channel");
    },
    [client, store, placeLayout, handleError],
  );
};

const haulItems = ({ name, id: otgID }: Tree.Entry, store: Flux.Store): Haul.Item[] => {
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
  const ch = (store as PChannel.FluxSubStore).channels.get(Number(otgID.key));
  if (ch?.internal === true) return items;
  return [PChannel.createHaulItem(Number(otgID.key))];
};

export const useDelete = Tree.createUseDelete({
  type: "Channel",
  query: PChannel.useDelete,
  convertKey: Number,
});

const beforeSetAlias = async ({
  data,
}: Flux.BeforeUpdateParams<PChannel.UpdateAliasParams>) => {
  if (data.channel == null) return false;
  const [alias, renamed] = await Text.asyncEdit(
    List.itemNameID(ontology.idToString(channel.ontologyID(data.channel))),
  );
  if (!renamed) return false;
  return { ...data, alias };
};

export const useSetAlias = ({
  selection: {
    ids: [firstID],
  },
}: Tree.ContextMenuProps): (() => void) => {
  const activeRange = Session.Range.useSelectSelectedKey();
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

export const useRename = Tree.createUseRename({
  query: PChannel.useRename,
  ontologyID: channel.ontologyID,
  convertKey: Number,
});

export const useDeleteAlias = ({
  selection: { ids },
}: Tree.ContextMenuProps): (() => void) => {
  const activeRange = Session.Range.useSelectSelectedKey();
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

const retrieveProperties = async ({
  client,
  store,
  id,
}: Tree.RetrievePropertiesParams) =>
  (
    await PChannel.retrieveSingle({
      client,
      store: store as PChannel.FluxSubStore,
      query: { key: Number(id.key) },
    })
  ).payload;

const useEditCalculated = () => {
  const open = Channel.useCalculatedModal();
  return ({ selection: { ids } }: Tree.ContextMenuProps) => {
    if (ids.length !== 1) return;
    open({ channelKey: Number(ids[0].key) });
  };
};

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getName, shape },
  } = props;
  const activeRange = Session.Range.useSelectState();
  const groupFromSelection = Group.useCreateFromSelection();
  const handleSetAlias = useSetAlias(props);
  const channelKeys = useMemo(() => ids.map((r) => Number(r.key)), [ids]);
  const channels = PChannel.useRetrieveMultiple({
    rangeKey: activeRange?.key,
    keys: channelKeys,
  });
  const showDeleteAlias = channels.data?.some((c) => c.alias != null) ?? false;
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
  const openCalculated = useEditCalculated();
  const singleResource = ids.length === 1;

  const firstChannel = channels.data?.find((c) => c.key === channelKeys[0]);
  const isCalc =
    singleResource && firstChannel != null && isCalculated(firstChannel.payload);

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
            onClick={() => handleLink({ name: getName(ids[0]), ontologyID: ids[0] })}
          />
          <Tree.CopyPropertiesContextMenuItem
            {...props}
            retrieveProperties={retrieveProperties}
          />
        </>
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const Content = ({ id, name: entryName, icon: _, ...rest }: Tree.ContentProps) => {
  const activeRange = Session.Range.useSelectState();
  const res = PChannel.useRetrieve({
    key: Number(id.key),
    rangeKey: activeRange?.key,
  }).data;
  let name = entryName;
  if (primitive.isNonZero(res?.alias)) name = res?.alias;
  const DataTypeIcon = PChannel.resolveIcon(res?.payload);
  const statusVariant = status.keepVariants(res?.status?.variant, ["error", "warning"]);
  return (
    <PTree.Item {...rest}>
      <DataTypeIcon color={10} />
      <Text.MaybeEditable
        id={List.itemNameID(rest.itemKey)}
        allowDoubleClick={false}
        value={name}
        overflow="ellipsis"
        style={{ width: 0 }}
        grow
        disabled={res?.internal === true}
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
      {res?.virtual === true && <Icon.Virtual color={8} />}
    </PTree.Item>
  );
};

const TreeItem = Tree.createItem({
  type: "channel",
  icon: <Icon.Channel />,
  hasChildren: false,
  useOnSelect,
  haulItems,
  Content,
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { channel: TreeItem } satisfies Tree.Items;
