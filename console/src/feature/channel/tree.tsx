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
  lineplot,
  ontology,
  panel,
  ranger,
  status,
} from "@synnaxlabs/client";
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
import { CSS } from "@/platform/css";
import { Group } from "@/platform/group";
import { LinePlot } from "@/platform/lineplot";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useOnSelect = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const openTab = Panel.useOpenTab();
  const getFocusedTab = Session.Panel.useGetFocusedTab();
  const getSelectedPanel = Session.Panel.useGetSelected();
  const getSelectedProject = Session.Project.useGetSelected();
  const getSelectedRange = Session.Range.useGetSelectedKey();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      const nonVirtualSelection = [resource]
        .filter((s) => s.data?.virtual !== true || s.data.expression != "")
        .map((s) => Number(s.id.key));

      if (nonVirtualSelection.length === 0) return;

      handleError(async () => {
        const focusedTab = getFocusedTab();
        const panelKey = getSelectedPanel();
        if (focusedTab != null && panelKey != null) {
          const doc = await client.panels.retrieve(panelKey);
          const tab = panel.findTab(doc.root, focusedTab);
          if (tab?.variant === "resource" && tab.resource.type === "lineplot") {
            await LinePlot.addChannelsToActivePlot(
              client,
              tab.resource.key,
              nonVirtualSelection,
            );
            return;
          }
        }
        const project = getSelectedProject();
        const selectedRange = getSelectedRange() ?? Session.Range.RECENT_KEY;
        const { key } = await client.lineplots.create(project, {
          name: "Line Plot",
          channels: { y1: nonVirtualSelection },
          ranges: { x1: [selectedRange] },
        });
        store.dispatch(Session.LinePlot.create({ key }));
        openTab({ variant: "resource", resource: lineplot.ontologyID(key) });
      }, "Failed to add channels to plot");
    },
    [
      client,
      openTab,
      getFocusedTab,
      getSelectedPanel,
      getSelectedProject,
      getSelectedRange,
      store,
      handleError,
    ],
  );
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

const allowRename: Tree.AllowRename = ({ data }) => data?.internal !== true;

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

const useEditCalculated = () => {
  const open = Channel.useCalculatedModal();
  return ({ selection: { ids }, state: { getResource } }: Tree.ContextMenuProps) => {
    if (ids.length !== 1) return;
    const resource = getResource(ids[0]);
    open({ channelKey: Number(resource.id.key) });
  };
};

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const activeRange = Session.Range.useSelectState();
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
  const openCalculated = useEditCalculated();
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
          <Tree.CopyPropertiesContextMenuItem {...props} />
        </>
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const Content = ({ resource, icon: _, ...rest }: Tree.ContentProps) => {
  const activeRange = Session.Range.useSelectState();
  const res = PChannel.useRetrieve({
    key: Number(resource.id.key),
    rangeKey: activeRange?.key,
  }).data;
  let name = resource.name;
  if (primitive.isNonZero(res?.alias)) name = res?.alias;
  const data = resource.data as channel.Payload;
  const DataTypeIcon = PChannel.resolveIcon(data);
  const statusVariant = status.keepVariants(res?.status?.variant, ["error", "warning"]);
  return (
    <PTree.Item {...rest}>
      <DataTypeIcon color={10} />
      <Text.MaybeEditable
        id={List.itemNameID(ontology.idToString(resource.id))}
        allowDoubleClick={false}
        value={name}
        overflow="ellipsis"
        className={CSS.BE("channel-tree-item", "name")}
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
